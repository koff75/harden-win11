# disable_unused.undo.ps1
# Restaure l'état Enabled de chaque compte selon le 'before' fourni.
# Input : { "Administrator": {"exists":bool,"enabled":bool}, ... }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with Administrator/Guest/WsiAccount/DefaultAccount fields"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

$accounts = @('Administrator', 'Guest', 'WsiAccount', 'DefaultAccount')
foreach ($name in $accounts) {
    $info = $state.$name
    if (-not $info -or -not $info.exists) { continue }
    $u = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if (-not $u) { continue }
    if ($info.enabled -and -not $u.Enabled) {
        Enable-LocalUser -Name $name
    } elseif (-not $info.enabled -and $u.Enabled) {
        Disable-LocalUser -Name $name
    }
}

@{ ok = $true } | ConvertTo-Json -Compress
