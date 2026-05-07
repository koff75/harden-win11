# controlled_folder_access.test.ps1
# 0=Disabled, 1=Enabled, 2=AuditMode, 3=BlockDiskModification, 4=AuditDiskModification

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$raw = [int](Get-MpPreference).EnableControlledFolderAccess
$compliant = $raw -eq 1
$names = @{ 0 = 'Disabled'; 1 = 'Enabled'; 2 = 'AuditMode'; 3 = 'BlockDiskModification'; 4 = 'AuditDiskModification' }
$mode = if ($names.ContainsKey($raw)) { $names[$raw] } else { "Unknown($raw)" }

@{
    compliant = $compliant
    current   = @{ EnableControlledFolderAccess = $mode }
} | ConvertTo-Json -Compress -Depth 10
