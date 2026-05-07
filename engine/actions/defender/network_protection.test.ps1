# network_protection.test.ps1
# EnableNetworkProtection : 0=Disabled, 1=Enabled, 2=AuditMode

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$raw = [int](Get-MpPreference).EnableNetworkProtection
$compliant = $raw -eq 1
$names = @{ 0 = 'Disabled'; 1 = 'Enabled'; 2 = 'AuditMode' }
$mode = if ($names.ContainsKey($raw)) { $names[$raw] } else { "Unknown($raw)" }

@{
    compliant = $compliant
    current   = @{ EnableNetworkProtection = $mode }
} | ConvertTo-Json -Compress -Depth 10
