# rdp_firewall_disable.action.ps1
# RDP : désactive les firewall rules du groupe "Remote Desktop".
# Belt-and-suspenders avec rdp_disable (registre).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$rules = Get-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue
$before = @()
foreach ($r in $rules) {
    $before += @{ Name = $r.Name; DisplayName = $r.DisplayName; Enabled = $r.Enabled.ToString() }
}

Disable-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue

$rules = Get-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue
$after = @()
foreach ($r in $rules) {
    $after += @{ Name = $r.Name; DisplayName = $r.DisplayName; Enabled = $r.Enabled.ToString() }
}

@{ ok = $true; before = $before; after = $after } | ConvertTo-Json -Compress -Depth 10
