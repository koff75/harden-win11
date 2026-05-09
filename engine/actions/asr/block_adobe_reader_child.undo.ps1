# block_adobe_reader_child.undo.ps1
# Restaure l'état AsrAction de la règle 7674BA52-37EB-4A4F-A9A1-F0F9A1619A2C selon 'before'.
# Input : { "AsrAction": <int|null> }
# Si AsrAction était null (règle absente), on Remove-MpPreference.
# Sinon on ré-Add-MpPreference avec la valeur précédente.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

$guid = '7674BA52-37EB-4A4F-A9A1-F0F9A1619A2C'

# Toujours retirer d'abord (pour ne pas accumuler de doublons côté Defender)
Remove-MpPreference -AttackSurfaceReductionRules_Ids $guid -ErrorAction SilentlyContinue

if ($null -ne $state.AsrAction) {
    Add-MpPreference -AttackSurfaceReductionRules_Ids $guid -AttackSurfaceReductionRules_Actions ([int]$state.AsrAction) -ErrorAction Stop
}

@{ ok = $true } | ConvertTo-Json -Compress