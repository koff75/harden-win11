# block_netbios_public.undo.ps1
# Supprime les 2 rules [Hardening] créées par block_netbios_public.action.ps1.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$names = @(
    'Block NetBIOS UDP Inbound (Public) [Hardening]',
    'Block NetBIOS TCP Inbound (Public) [Hardening]'
)

foreach ($n in $names) {
    if (Get-NetFirewallRule -DisplayName $n -ErrorAction SilentlyContinue) {
        Remove-NetFirewallRule -DisplayName $n
    }
}

@{ ok = $true } | ConvertTo-Json -Compress
