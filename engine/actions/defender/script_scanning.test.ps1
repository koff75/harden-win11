# script_scanning.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$current = (Get-MpPreference).DisableScriptScanning
$compliant = -not $current

@{
    compliant = $compliant
    current   = @{ DisableScriptScanning = $current }
} | ConvertTo-Json -Compress -Depth 10
