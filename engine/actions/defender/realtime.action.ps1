# realtime.action.ps1
# Active la protection temps réel de Microsoft Defender.
# Input stdin : { "before": { "DisableRealtimeMonitoring": <bool> } }
# Output stdout : { "ok": true, "before": {...}, "after": {...} }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# État avant
$before = @{
    DisableRealtimeMonitoring = (Get-MpPreference).DisableRealtimeMonitoring
}

# Action
Set-MpPreference -DisableRealtimeMonitoring $false

# État après
$after = @{
    DisableRealtimeMonitoring = (Get-MpPreference).DisableRealtimeMonitoring
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
