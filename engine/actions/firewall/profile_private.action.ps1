# profile_private.action.ps1
# Active le profil Firewall Private + blocage entrant par défaut + allow sortant.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$pf = Get-NetFirewallProfile -Profile Private
$before = @{
    Enabled              = $pf.Enabled.ToString()
    DefaultInboundAction = $pf.DefaultInboundAction.ToString()
    DefaultOutboundAction= $pf.DefaultOutboundAction.ToString()
}

Set-NetFirewallProfile -Profile Private -Enabled True -DefaultInboundAction Block -DefaultOutboundAction Allow

$pf = Get-NetFirewallProfile -Profile Private
$after = @{
    Enabled              = $pf.Enabled.ToString()
    DefaultInboundAction = $pf.DefaultInboundAction.ToString()
    DefaultOutboundAction= $pf.DefaultOutboundAction.ToString()
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
