# network_protection.action.ps1
# Active Network Protection : bloque la résolution DNS / connexions vers
# domaines/IP réputés malveillants.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$names = @{ 0 = 'Disabled'; 1 = 'Enabled'; 2 = 'AuditMode' }

$rawBefore = [int](Get-MpPreference).EnableNetworkProtection
$before = @{
    EnableNetworkProtection = if ($names.ContainsKey($rawBefore)) { $names[$rawBefore] } else { "Unknown($rawBefore)" }
}

Set-MpPreference -EnableNetworkProtection Enabled

$rawAfter = [int](Get-MpPreference).EnableNetworkProtection
$after = @{
    EnableNetworkProtection = if ($names.ContainsKey($rawAfter)) { $names[$rawAfter] } else { "Unknown($rawAfter)" }
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
