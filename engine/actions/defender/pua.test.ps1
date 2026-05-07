# pua.test.ps1
# PUAProtection : 0=Disabled, 1=Enabled, 2=AuditMode

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$raw = [int](Get-MpPreference).PUAProtection
$compliant = $raw -eq 1
$names = @{ 0 = 'Disabled'; 1 = 'Enabled'; 2 = 'AuditMode' }
$mode = if ($names.ContainsKey($raw)) { $names[$raw] } else { "Unknown($raw)" }

@{
    compliant = $compliant
    current   = @{ PUAProtection = $mode }
} | ConvertTo-Json -Compress -Depth 10
