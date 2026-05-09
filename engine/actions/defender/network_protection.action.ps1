# network_protection.action.ps1
# Active Network Protection : bloque la résolution DNS / connexions vers
# domaines/IP réputés malveillants.
#
# Mode :
#   - Enabled (block) par défaut
#   - AuditMode si HARDEN_ASR_MODE=audit (mode audit GUI)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$names = @{ 0 = 'Disabled'; 1 = 'Enabled'; 2 = 'AuditMode' }
$targetMode = if ($env:HARDEN_ASR_MODE -eq 'audit') { 'AuditMode' } else { 'Enabled' }

$rawBefore = [int](Get-MpPreference).EnableNetworkProtection
$before = @{
    EnableNetworkProtection = if ($names.ContainsKey($rawBefore)) { $names[$rawBefore] } else { "Unknown($rawBefore)" }
}

Set-MpPreference -EnableNetworkProtection $targetMode

$rawAfter = [int](Get-MpPreference).EnableNetworkProtection
$after = @{
    EnableNetworkProtection = if ($names.ContainsKey($rawAfter)) { $names[$rawAfter] } else { "Unknown($rawAfter)" }
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
