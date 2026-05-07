# behavior_monitoring.test.ps1
# Vérifie si la règle defender.behavior_monitoring est déjà conforme.
# Output stdout : { "compliant": <bool>, "current": {...} }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$current = (Get-MpPreference).DisableBehaviorMonitoring
$compliant = -not $current

@{
    compliant = $compliant
    current   = @{ DisableBehaviorMonitoring = $current }
} | ConvertTo-Json -Compress -Depth 10
