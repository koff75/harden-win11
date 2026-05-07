# netbios_off.test.ps1
# Conforme = toutes les interfaces NetBT ont NetbiosOptions=2 (Disable).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ifaceRoot = 'HKLM:\SYSTEM\CurrentControlSet\Services\NetBT\Parameters\Interfaces'
$ifaces = @{}
$compliant = $true
$total = 0

Get-ChildItem $ifaceRoot -ErrorAction SilentlyContinue | ForEach-Object {
    $total++
    $val = (Get-ItemProperty -Path $_.PSPath -Name NetbiosOptions -ErrorAction SilentlyContinue).NetbiosOptions
    $ifaces[$_.PSChildName] = $val
    if ($val -ne 2) { $compliant = $false }
}

if ($total -eq 0) { $compliant = $true }   # pas d'interfaces NetBT = OK

@{
    compliant = $compliant
    current   = @{
        InterfaceCount = $total
        Interfaces     = $ifaces
    }
} | ConvertTo-Json -Compress -Depth 10
