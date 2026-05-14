# block_smb_public.action.ps1
# Cree une firewall rule pour bloquer SMB (TCP 445) entrant sur le profil Public.
# Idempotent : supprime la rule existante avant recreation.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ruleName = 'Block SMB Inbound (Public) [Hardening]'

# Note : les crochets [ ] dans le DisplayName sont interpretes comme
# wildcards PowerShell par Get-NetFirewallRule -DisplayName. Donc on
# filtre via Where-Object pour matcher litteralement, et on capture
# le Name (GUID) retourne par New-NetFirewallRule pour les Get suivants
# qui sont stables.
$existing = Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -eq $ruleName }
$before = @{
    existed = [bool]$existing
    enabled = if ($existing) { $existing[0].Enabled.ToString() } else { $null }
}

if ($existing) {
    foreach ($r in $existing) { Remove-NetFirewallRule -Name $r.Name -ErrorAction SilentlyContinue }
}

$new = New-NetFirewallRule -DisplayName $ruleName `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 445 `
    -Action Block `
    -Profile Public

# Get-NetFirewallRule -Name est stable (pas de wildcard) — utilise le GUID
# retourne par PassThru pour eviter le piege des crochets dans DisplayName.
$now = Get-NetFirewallRule -Name $new.Name -ErrorAction SilentlyContinue
if (-not $now) {
    @{
        ok    = $false
        error = "Firewall rule non trouvable apres creation (Name=$($new.Name))."
        before = $before
        after  = @{ existed = $false; enabled = $null }
    } | ConvertTo-Json -Compress -Depth 10
    exit 0
}

$after = @{
    existed = $true
    enabled = $now.Enabled.ToString()
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
