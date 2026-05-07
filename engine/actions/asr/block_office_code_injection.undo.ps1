# block_office_code_injection.undo.ps1
# Restaure l'Ã©tat AsrAction de la rÃ¨gle 75668C1F-73B5-4CF0-BB93-3ECF5CB7CC84 selon 'before'.
# Input : { "AsrAction": <int|null> }
# Si AsrAction Ã©tait null (rÃ¨gle absente), on Remove-MpPreference.
# Sinon on rÃ©-Add-MpPreference avec la valeur prÃ©cÃ©dente.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

$guid = '75668C1F-73B5-4CF0-BB93-3ECF5CB7CC84'

# Toujours retirer d'abord (pour ne pas accumuler de doublons cÃ´tÃ© Defender)
Remove-MpPreference -AttackSurfaceReductionRules_Ids $guid -ErrorAction SilentlyContinue

if ($null -ne $state.AsrAction) {
    Add-MpPreference -AttackSurfaceReductionRules_Ids $guid -AttackSurfaceReductionRules_Actions ([int]$state.AsrAction) -ErrorAction Stop
}

@{ ok = $true } | ConvertTo-Json -Compress