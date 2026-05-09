# advanced_ransomware_protection.undo.ps1
# Restaure l'état AsrAction de la règle C1DB55AB-C21A-4637-BB3F-A12568109D35 selon 'before'.
# Input : { "AsrAction": <int|null> }
# Si AsrAction était null (règle absente), on Remove-MpPreference.
# Sinon on ré-Add-MpPreference avec la valeur précédente.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

$guid = 'C1DB55AB-C21A-4637-BB3F-A12568109D35'

# Toujours retirer d'abord (pour ne pas accumuler de doublons côté Defender)
Remove-MpPreference -AttackSurfaceReductionRules_Ids $guid -ErrorAction SilentlyContinue

if ($null -ne $state.AsrAction) {
    Add-MpPreference -AttackSurfaceReductionRules_Ids $guid -AttackSurfaceReductionRules_Actions ([int]$state.AsrAction) -ErrorAction Stop
}

@{ ok = $true } | ConvertTo-Json -Compress