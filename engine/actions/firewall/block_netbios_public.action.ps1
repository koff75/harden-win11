# block_netbios_public.action.ps1
# Cree 2 firewall rules pour bloquer NetBIOS entrant sur le profil Public :
#   - Block NetBIOS UDP Inbound (Public) [Hardening] : UDP 137-138
#   - Block NetBIOS TCP Inbound (Public) [Hardening] : TCP 139
# Idempotent : supprime les rules existantes avant recreation.
#
# Note : les crochets [ ] dans le DisplayName sont interpretes comme
# wildcards par Get-NetFirewallRule -DisplayName. On filtre via Where-Object
# pour matcher litteralement, et on capture le Name (GUID) via -PassThru.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$udpName = 'Block NetBIOS UDP Inbound (Public) [Hardening]'
$tcpName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'

function Find-RulesByDisplay($name) {
    return @(Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -eq $name })
}

function State-Of($rule) {
    if ($rule) { return @{ existed = $true; enabled = $rule[0].Enabled.ToString() } }
    return @{ existed = $false; enabled = $null }
}

$before = @{
    udp = State-Of (Find-RulesByDisplay $udpName)
    tcp = State-Of (Find-RulesByDisplay $tcpName)
}

foreach ($n in @($udpName, $tcpName)) {
    $rs = Find-RulesByDisplay $n
    foreach ($r in $rs) { Remove-NetFirewallRule -Name $r.Name -ErrorAction SilentlyContinue }
}

$newUdp = New-NetFirewallRule -DisplayName $udpName `
    -Direction Inbound -Protocol UDP -LocalPort 137,138 -Action Block -Profile Public

$newTcp = New-NetFirewallRule -DisplayName $tcpName `
    -Direction Inbound -Protocol TCP -LocalPort 139 -Action Block -Profile Public

# Check via Name (GUID) — stable, pas de probleme de wildcard.
$verifyUdp = Get-NetFirewallRule -Name $newUdp.Name -ErrorAction SilentlyContinue
$verifyTcp = Get-NetFirewallRule -Name $newTcp.Name -ErrorAction SilentlyContinue

$after = @{
    udp = if ($verifyUdp) { @{ existed = $true; enabled = $verifyUdp.Enabled.ToString() } } else { @{ existed = $false; enabled = $null } }
    tcp = if ($verifyTcp) { @{ existed = $true; enabled = $verifyTcp.Enabled.ToString() } } else { @{ existed = $false; enabled = $null } }
}

if (-not $after.udp.existed -or -not $after.tcp.existed) {
    @{
        ok    = $false
        error = "Une ou plusieurs firewall rules non trouvables apres creation."
        before = $before
        after  = $after
    } | ConvertTo-Json -Compress -Depth 10
    exit 0
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
