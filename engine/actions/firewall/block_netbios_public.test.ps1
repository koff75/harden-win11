# block_netbios_public.test.ps1
# Conforme = les 2 rules existent ET sont Enabled.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$udpName = 'Block NetBIOS UDP Inbound (Public) [Hardening]'
$tcpName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'

function Get-State($name) {
    $r = Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue
    if ($r) { @{ exists = $true; enabled = $r.Enabled.ToString() } }
    else    { @{ exists = $false; enabled = 'NotPresent' } }
}

$udp = Get-State $udpName
$tcp = Get-State $tcpName

$compliant = $udp.exists -and ($udp.enabled -eq 'True') -and $tcp.exists -and ($tcp.enabled -eq 'True')

@{
    compliant = $compliant
    current   = @{
        udp = $udp
        tcp = $tcp
    }
} | ConvertTo-Json -Compress -Depth 10
