# block_smb_public.undo.ps1
# Supprime la rule [Hardening] créée par block_smb_public.action.ps1.
# (On ne tente pas de restaurer une rule pré-existante : la rule [Hardening]
#  est gérée exclusivement par cet outil.)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ruleName = 'Block SMB Inbound (Public) [Hardening]'

# Les crochets [ ] dans le DisplayName sont interpretes comme wildcards par
# Get/Remove-NetFirewallRule -DisplayName. On filtre via Where-Object pour
# matcher litteralement, et on supprime par Name (GUID) qui est stable.
$existing = @(Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -eq $ruleName })
foreach ($r in $existing) { Remove-NetFirewallRule -Name $r.Name -ErrorAction SilentlyContinue }

@{ ok = $true } | ConvertTo-Json -Compress
