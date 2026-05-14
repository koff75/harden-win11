# block_netbios_public.test.ps1
# Conforme = les 2 rules existent ET sont Enabled.
#
# Note : les crochets [ ] dans le DisplayName sont interpretes comme
# wildcards par Get-NetFirewallRule -DisplayName. On filtre via Where-Object
# pour matcher litteralement.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$udpName = 'Block NetBIOS UDP Inbound (Public) [Hardening]'
$tcpName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'

function Get-State($name) {
    $rules = @(Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -eq $name })
    if ($rules.Count -gt 0) { @{ exists = $true; enabled = $rules[0].Enabled.ToString() } }
    else                    { @{ exists = $false; enabled = 'NotPresent' } }
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
