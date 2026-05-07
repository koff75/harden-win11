# netbios_off.action.ps1
# Désactive NetBIOS over TCP/IP sur tous les adaptateurs réseau via WMI
# (TcpipNetbiosOptions=2) ET dans le registre NetBT\Interfaces (NetbiosOptions=2).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# État avant : valeurs registre NetBT\Interfaces
$ifaceRoot = 'HKLM:\SYSTEM\CurrentControlSet\Services\NetBT\Parameters\Interfaces'
$beforeIfaces = @{}
Get-ChildItem $ifaceRoot -ErrorAction SilentlyContinue | ForEach-Object {
    $val = (Get-ItemProperty -Path $_.PSPath -Name NetbiosOptions -ErrorAction SilentlyContinue).NetbiosOptions
    $beforeIfaces[$_.PSChildName] = $val
}

# Action via WMI : adaptateurs IP-enabled
$adapters = Get-CimInstance -ClassName Win32_NetworkAdapterConfiguration -Filter 'IPEnabled=TRUE'
foreach ($a in $adapters) {
    $null = Invoke-CimMethod -InputObject $a -MethodName SetTcpipNetbios -Arguments @{TcpipNetbiosOptions = [uint32]2}
}

# Action via registre : tous les NetBT\Interfaces
Get-ChildItem $ifaceRoot -ErrorAction SilentlyContinue | ForEach-Object {
    Set-ItemProperty -Path $_.PSPath -Name 'NetbiosOptions' -Value 2 -Force -ErrorAction SilentlyContinue
}

# État après
$afterIfaces = @{}
Get-ChildItem $ifaceRoot -ErrorAction SilentlyContinue | ForEach-Object {
    $val = (Get-ItemProperty -Path $_.PSPath -Name NetbiosOptions -ErrorAction SilentlyContinue).NetbiosOptions
    $afterIfaces[$_.PSChildName] = $val
}

@{
    ok     = $true
    before = @{ interfaces = $beforeIfaces }
    after  = @{ interfaces = $afterIfaces }
} | ConvertTo-Json -Compress -Depth 10
