# block_smb_public.test.ps1
# Conforme = la rule existe ET est Enabled.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ruleName = 'Block SMB Inbound (Public) [Hardening]'

$rule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
$exists  = [bool]$rule
$enabled = if ($rule) { $rule.Enabled.ToString() } else { 'NotPresent' }
$compliant = $exists -and ($enabled -eq 'True')

@{
    compliant = $compliant
    current   = @{
        ruleName = $ruleName
        exists   = $exists
        enabled  = $enabled
    }
} | ConvertTo-Json -Compress -Depth 10
