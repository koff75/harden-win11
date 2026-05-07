# behavior_monitoring.undo.ps1
# Restaure DisableBehaviorMonitoring à la valeur passée en entrée.
# Input : { "DisableBehaviorMonitoring": <bool> }
# Output stdout : { "ok": true }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with DisableBehaviorMonitoring field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -DisableBehaviorMonitoring ([bool]$state.DisableBehaviorMonitoring)

@{ ok = $true } | ConvertTo-Json -Compress
