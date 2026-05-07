# realtime.undo.ps1
# Revient à l'état "before" passé en stdin.
# Input stdin : { "DisableRealtimeMonitoring": <bool> }
# Output stdout : { "ok": true }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$inputJson = [Console]::In.ReadToEnd()
if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input on stdin with DisableRealtimeMonitoring field"
    exit 1
}
$input = $inputJson | ConvertFrom-Json

Set-MpPreference -DisableRealtimeMonitoring ([bool]$input.DisableRealtimeMonitoring)

@{ ok = $true } | ConvertTo-Json -Compress
