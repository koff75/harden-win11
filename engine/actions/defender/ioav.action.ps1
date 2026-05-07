# ioav.action.ps1
# Active IOAV (analyse des fichiers téléchargés / pièces jointes Office).
# Output stdout : { "ok": true, "before": {...}, "after": {...} }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$before = @{
    DisableIOAVProtection = (Get-MpPreference).DisableIOAVProtection
}

Set-MpPreference -DisableIOAVProtection $false

$after = @{
    DisableIOAVProtection = (Get-MpPreference).DisableIOAVProtection
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
