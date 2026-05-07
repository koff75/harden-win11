# rename_admin.test.ps1
# Conforme = le compte SID-500 ne s'appelle pas 'Administrator'.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$admin = Get-LocalUser | Where-Object { $_.SID.Value -like 'S-1-5-*-500' }
$currentName = if ($admin) { $admin.Name } else { $null }
$compliant = ($null -ne $admin) -and ($admin.Name -ne 'Administrator')

@{
    compliant = $compliant
    current   = @{
        AdminCurrentName = $currentName
        SID500Found      = [bool]$admin
    }
} | ConvertTo-Json -Compress -Depth 10
