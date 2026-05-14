# block_netbios_public.undo.ps1
# Supprime les 2 rules [Hardening] créées par block_netbios_public.action.ps1.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$names = @(
    'Block NetBIOS UDP Inbound (Public) [Hardening]',
    'Block NetBIOS TCP Inbound (Public) [Hardening]'
)

# Les crochets [ ] dans le DisplayName sont interpretes comme wildcards par
# Get/Remove-NetFirewallRule -DisplayName. On filtre via Where-Object pour
# matcher litteralement, et on supprime par Name (GUID) qui est stable.
foreach ($n in $names) {
    $rules = @(Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -eq $n })
    foreach ($r in $rules) { Remove-NetFirewallRule -Name $r.Name -ErrorAction SilentlyContinue }
}

@{ ok = $true } | ConvertTo-Json -Compress
