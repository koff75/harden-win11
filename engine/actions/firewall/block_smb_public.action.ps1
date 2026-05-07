# block_smb_public.action.ps1
# Crée une firewall rule pour bloquer SMB (TCP 445) entrant sur le profil Public.
# Idempotent : supprime la rule existante avant recréation.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ruleName = 'Block SMB Inbound (Public) [Hardening]'

$existing = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
$before = @{
    existed = [bool]$existing
    enabled = if ($existing) { $existing.Enabled.ToString() } else { $null }
}

if ($existing) {
    Remove-NetFirewallRule -DisplayName $ruleName
}

New-NetFirewallRule -DisplayName $ruleName `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 445 `
    -Action Block `
    -Profile Public | Out-Null

$now = Get-NetFirewallRule -DisplayName $ruleName
$after = @{
    existed = $true
    enabled = $now.Enabled.ToString()
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
