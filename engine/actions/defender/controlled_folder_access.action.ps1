# controlled_folder_access.action.ps1
# Active Controlled Folder Access (anti-ransomware).
# 0=Disabled, 1=Enabled, 2=AuditMode, 3=BlockDiskModification, 4=AuditDiskModification

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$names = @{ 0 = 'Disabled'; 1 = 'Enabled'; 2 = 'AuditMode'; 3 = 'BlockDiskModification'; 4 = 'AuditDiskModification' }

$rawBefore = [int](Get-MpPreference).EnableControlledFolderAccess
$before = @{
    EnableControlledFolderAccess = if ($names.ContainsKey($rawBefore)) { $names[$rawBefore] } else { "Unknown($rawBefore)" }
}

Set-MpPreference -EnableControlledFolderAccess Enabled

$rawAfter = [int](Get-MpPreference).EnableControlledFolderAccess
$after = @{
    EnableControlledFolderAccess = if ($names.ContainsKey($rawAfter)) { $names[$rawAfter] } else { "Unknown($rawAfter)" }
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
