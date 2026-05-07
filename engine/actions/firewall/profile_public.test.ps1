# profile_public.test.ps1
# Conforme = Enabled=True ET DefaultInboundAction=Block ET DefaultOutboundAction=Allow

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$pf = Get-NetFirewallProfile -Profile Public
$enabled  = $pf.Enabled.ToString()
$inAction = $pf.DefaultInboundAction.ToString()
$outAction= $pf.DefaultOutboundAction.ToString()

$compliant = ($enabled -eq 'True') -and ($inAction -eq 'Block') -and ($outAction -eq 'Allow')

@{
    compliant = $compliant
    current   = @{
        Enabled              = $enabled
        DefaultInboundAction = $inAction
        DefaultOutboundAction= $outAction
    }
} | ConvertTo-Json -Compress -Depth 10
