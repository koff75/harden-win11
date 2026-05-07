# nis.action.ps1
# Active Network Inspection System (NIS) de Defender.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$before = @{
    DisableIntrusionPreventionSystem = (Get-MpPreference).DisableIntrusionPreventionSystem
}

Set-MpPreference -DisableIntrusionPreventionSystem $false

$after = @{
    DisableIntrusionPreventionSystem = (Get-MpPreference).DisableIntrusionPreventionSystem
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
