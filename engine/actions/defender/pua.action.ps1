# pua.action.ps1
# Active la protection PUA (Potentially Unwanted Apps).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$names = @{ 0 = 'Disabled'; 1 = 'Enabled'; 2 = 'AuditMode' }

$rawBefore = [int](Get-MpPreference).PUAProtection
$before = @{
    PUAProtection = if ($names.ContainsKey($rawBefore)) { $names[$rawBefore] } else { "Unknown($rawBefore)" }
}

Set-MpPreference -PUAProtection Enabled

$rawAfter = [int](Get-MpPreference).PUAProtection
$after = @{
    PUAProtection = if ($names.ContainsKey($rawAfter)) { $names[$rawAfter] } else { "Unknown($rawAfter)" }
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
