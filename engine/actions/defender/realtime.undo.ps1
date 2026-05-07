# realtime.undo.ps1
# Revient à l'état "before" passé en entrée.
# Input stdin (ou pipeline PowerShell) : { "DisableRealtimeMonitoring": <bool> }
# Output stdout : { "ok": true }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Dual-source : pipeline PowerShell ($input) en mode Pester, stdin du process en prod.
if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with DisableRealtimeMonitoring field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -DisableRealtimeMonitoring ([bool]$state.DisableRealtimeMonitoring)

@{ ok = $true } | ConvertTo-Json -Compress
