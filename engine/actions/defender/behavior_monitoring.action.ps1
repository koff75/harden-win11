# behavior_monitoring.action.ps1
# Active l'analyse comportementale de Microsoft Defender.
# Output stdout : { "ok": true, "before": {...}, "after": {...} }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$before = @{
    DisableBehaviorMonitoring = (Get-MpPreference).DisableBehaviorMonitoring
}

Set-MpPreference -DisableBehaviorMonitoring $false

$after = @{
    DisableBehaviorMonitoring = (Get-MpPreference).DisableBehaviorMonitoring
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
