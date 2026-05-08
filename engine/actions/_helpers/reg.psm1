# reg.psm1 — Helpers pour les règles de hardening basées sur le registre.
#
# Factorise le pattern récurrent des snippets registry (get before, set,
# get after, capture pour undo) afin que chaque règle .action.ps1 / .test.ps1
# / .undo.ps1 ne soit plus que 5 lignes au lieu de 30.
#
# Convention de retour (cohérente avec le contrat moteur Go) :
#
#   Invoke-RegSetAction  → { ok, before: {exists, value}, after: {exists, value} }
#   Invoke-RegTestAction → { compliant, current: { <Name>: <value> } }
#   Invoke-RegUndoAction → { ok }
#
# Tous les helpers émettent leur JSON sur stdout via ConvertTo-Json -Compress.

Set-StrictMode -Version Latest

function Get-RegState {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)] [string] $Path,
        [Parameter(Mandatory)] [string] $Name
    )
    $existing = Get-ItemProperty -Path $Path -Name $Name -ErrorAction SilentlyContinue
    if ($existing) {
        @{ exists = $true; value = $existing.$Name }
    } else {
        @{ exists = $false; value = $null }
    }
}

function Set-RegValue {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)] [string] $Path,
        [Parameter(Mandatory)] [string] $Name,
        [Parameter(Mandatory)] $Value,
        [string] $Type = 'DWord'
    )
    if (-not (Test-Path $Path)) { New-Item -Path $Path -Force | Out-Null }
    Set-ItemProperty -Path $Path -Name $Name -Value $Value -Type $Type -Force
}

function Remove-RegValueIfPresent {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)] [string] $Path,
        [Parameter(Mandatory)] [string] $Name
    )
    if (Get-ItemProperty -Path $Path -Name $Name -ErrorAction SilentlyContinue) {
        Remove-ItemProperty -Path $Path -Name $Name -ErrorAction SilentlyContinue
    }
}

# Invoke-RegSetAction : équivalent .action.ps1 pour une règle registry simple.
# Lit le before, set, lit le after, émet le JSON.
function Invoke-RegSetAction {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)] [string] $Path,
        [Parameter(Mandatory)] [string] $Name,
        [Parameter(Mandatory)] $Value,
        [string] $Type = 'DWord'
    )
    $before = Get-RegState -Path $Path -Name $Name
    Set-RegValue -Path $Path -Name $Name -Value $Value -Type $Type
    $after  = Get-RegState -Path $Path -Name $Name

    @{
        ok     = $true
        before = $before
        after  = $after
    } | ConvertTo-Json -Compress -Depth 10
}

# Invoke-RegTestAction : équivalent .test.ps1 pour une règle registry simple.
# Conforme = la valeur actuelle correspond à la valeur attendue.
# Le paramètre -CurrentLabel permet de personnaliser la clé exposée dans
# 'current' pour rester compatible avec les tests Pester qui valident des
# noms de clés précis (ex: 'EnableLUA' au lieu de 'value').
function Invoke-RegTestAction {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)] [string] $Path,
        [Parameter(Mandatory)] [string] $Name,
        [Parameter(Mandatory)] $Expected,
        [string] $CurrentLabel
    )
    $state = Get-RegState -Path $Path -Name $Name
    $compliant = ($state.value -eq $Expected)

    $label = if ($CurrentLabel) { $CurrentLabel } else { $Name }
    $current = @{}
    $current[$label] = $state.value

    @{
        compliant = $compliant
        current   = $current
    } | ConvertTo-Json -Compress
}

# Invoke-RegUndoAction : équivalent .undo.ps1 pour une règle registry simple.
# Lit l'état before depuis stdin (pipeline ou Console::In), restaure ou supprime
# la valeur selon que la clé existait avant.
function Invoke-RegUndoAction {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)] [string] $Path,
        [Parameter(Mandatory)] [string] $Name,
        [string] $Type = 'DWord'
    )

    if ($MyInvocation.ExpectingInput) {
        $inputJson = ($input | Out-String).Trim()
    } else {
        $inputJson = [Console]::In.ReadToEnd()
    }

    if (-not $inputJson.Trim()) {
        Write-Error "undo requires JSON input with {exists, value} fields"
        exit 1
    }
    $state = $inputJson | ConvertFrom-Json

    if ($state.exists) {
        Set-RegValue -Path $Path -Name $Name -Value ([int]$state.value) -Type $Type
    } else {
        Remove-RegValueIfPresent -Path $Path -Name $Name
    }

    @{ ok = $true } | ConvertTo-Json -Compress
}

Export-ModuleMember -Function Get-RegState, Set-RegValue, Remove-RegValueIfPresent,
    Invoke-RegSetAction, Invoke-RegTestAction, Invoke-RegUndoAction
