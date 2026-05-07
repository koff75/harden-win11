# nis.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$current = (Get-MpPreference).DisableIntrusionPreventionSystem
$compliant = -not $current

@{
    compliant = $compliant
    current   = @{ DisableIntrusionPreventionSystem = $current }
} | ConvertTo-Json -Compress -Depth 10
