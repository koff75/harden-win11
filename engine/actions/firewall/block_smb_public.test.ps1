# block_smb_public.test.ps1
# Conforme = la rule existe ET est Enabled.
#
# Note : les crochets [ ] dans le DisplayName sont interpretes comme
# wildcards par Get-NetFirewallRule -DisplayName. On filtre via Where-Object
# pour matcher litteralement.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ruleName = 'Block SMB Inbound (Public) [Hardening]'

$rules = @(Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -eq $ruleName })
$exists  = $rules.Count -gt 0
$enabled = if ($exists) { $rules[0].Enabled.ToString() } else { 'NotPresent' }
$compliant = $exists -and ($enabled -eq 'True')

@{
    compliant = $compliant
    current   = @{
        ruleName = $ruleName
        exists   = $exists
        enabled  = $enabled
    }
} | ConvertTo-Json -Compress -Depth 10
