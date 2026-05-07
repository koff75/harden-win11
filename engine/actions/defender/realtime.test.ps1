# realtime.test.ps1
# Vérifie si la règle defender.realtime est déjà conforme.
# Output stdout : { "compliant": <bool>, "current": {...} }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$current = (Get-MpPreference).DisableRealtimeMonitoring
$compliant = -not $current   # conforme si DisableRealtimeMonitoring = $false

@{
    compliant = $compliant
    current   = @{ DisableRealtimeMonitoring = $current }
} | ConvertTo-Json -Compress -Depth 10
