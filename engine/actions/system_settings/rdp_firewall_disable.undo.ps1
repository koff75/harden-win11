# rdp_firewall_disable.undo.ps1
# Restaure l'état Enabled des rules "Remote Desktop" selon le 'before' fourni.
# Input : [{ "Name":"...", "Enabled":"True"|"False" }, ...]

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON array of {Name, Enabled} objects"; exit 1 }
$state = $inputJson | ConvertFrom-Json

foreach ($item in $state) {
    $r = Get-NetFirewallRule -Name $item.Name -ErrorAction SilentlyContinue
    if (-not $r) { continue }
    if ($item.Enabled -eq 'True') {
        Enable-NetFirewallRule -Name $item.Name
    } else {
        Disable-NetFirewallRule -Name $item.Name
    }
}

@{ ok = $true } | ConvertTo-Json -Compress
