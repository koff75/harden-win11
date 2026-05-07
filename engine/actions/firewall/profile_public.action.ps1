# profile_public.action.ps1
# Active le profil Firewall Public + blocage entrant par défaut + allow sortant.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$pf = Get-NetFirewallProfile -Profile Public
$before = @{
    Enabled              = $pf.Enabled.ToString()
    DefaultInboundAction = $pf.DefaultInboundAction.ToString()
    DefaultOutboundAction= $pf.DefaultOutboundAction.ToString()
}

Set-NetFirewallProfile -Profile Public -Enabled True -DefaultInboundAction Block -DefaultOutboundAction Allow

$pf = Get-NetFirewallProfile -Profile Public
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
