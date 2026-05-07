# rename_admin.action.ps1
# Renomme le compte SID-500 (Administrator built-in) en AdminLocal_<COMPUTERNAME>
# pour casser les attaques qui ciblent le nom 'Administrator' par défaut.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$admin = Get-LocalUser | Where-Object { $_.SID.Value -like 'S-1-5-*-500' }
$before = @{
    AdminCurrentName = if ($admin) { $admin.Name } else { $null }
    SID500Found      = [bool]$admin
}

$targetName = "AdminLocal_$env:COMPUTERNAME"

if ($admin -and $admin.Name -eq 'Administrator') {
    Rename-LocalUser -Name 'Administrator' -NewName $targetName
}

$admin = Get-LocalUser | Where-Object { $_.SID.Value -like 'S-1-5-*-500' }
$after = @{
    AdminCurrentName = if ($admin) { $admin.Name } else { $null }
    SID500Found      = [bool]$admin
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
