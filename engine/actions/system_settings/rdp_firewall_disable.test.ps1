# rdp_firewall_disable.test.ps1
# Conforme = aucune rule du groupe "Remote Desktop" n'est Enabled.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$rules = Get-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue
$enabledRules = @($rules | Where-Object { $_.Enabled.ToString() -eq 'True' })

$compliant = ($enabledRules.Count -eq 0)
$summary = @()
foreach ($r in $rules) {
    $summary += @{ Name = $r.Name; Enabled = $r.Enabled.ToString() }
}

@{
    compliant = $compliant
    current   = @{
        TotalRules   = @($rules).Count
        EnabledRules = $enabledRules.Count
        Rules        = $summary
    }
} | ConvertTo-Json -Compress -Depth 10
