# script_scanning.action.ps1
# Active l'analyse antivirus des scripts (PS, JS, VBS) avant exécution.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$before = @{
    DisableScriptScanning = (Get-MpPreference).DisableScriptScanning
}

Set-MpPreference -DisableScriptScanning $false

$after = @{
    DisableScriptScanning = (Get-MpPreference).DisableScriptScanning
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
