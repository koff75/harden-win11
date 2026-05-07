# block_smb_public.undo.ps1
# Supprime la rule [Hardening] créée par block_smb_public.action.ps1.
# (On ne tente pas de restaurer une rule pré-existante : la rule [Hardening]
#  est gérée exclusivement par cet outil.)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ruleName = 'Block SMB Inbound (Public) [Hardening]'

$existing = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
if ($existing) {
    Remove-NetFirewallRule -DisplayName $ruleName
}

@{ ok = $true } | ConvertTo-Json -Compress
