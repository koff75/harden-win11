# block_netbios_public.action.ps1
# Crée 2 firewall rules pour bloquer NetBIOS entrant sur le profil Public :
#   - Block NetBIOS UDP Inbound (Public) [Hardening] : UDP 137-138
#   - Block NetBIOS TCP Inbound (Public) [Hardening] : TCP 139
# Idempotent : supprime les rules existantes avant recréation.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$udpName = 'Block NetBIOS UDP Inbound (Public) [Hardening]'
$tcpName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'

function Test-Rule($name) {
    $r = Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue
    if ($r) { @{ existed = $true; enabled = $r.Enabled.ToString() } } else { @{ existed = $false; enabled = $null } }
}

$before = @{
    udp = Test-Rule $udpName
    tcp = Test-Rule $tcpName
}

foreach ($n in @($udpName, $tcpName)) {
    if (Get-NetFirewallRule -DisplayName $n -ErrorAction SilentlyContinue) {
        Remove-NetFirewallRule -DisplayName $n
    }
}

New-NetFirewallRule -DisplayName $udpName `
    -Direction Inbound -Protocol UDP -LocalPort 137,138 -Action Block -Profile Public | Out-Null

New-NetFirewallRule -DisplayName $tcpName `
    -Direction Inbound -Protocol TCP -LocalPort 139 -Action Block -Profile Public | Out-Null

$after = @{
    udp = Test-Rule $udpName
    tcp = Test-Rule $tcpName
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
