# smbv1_disable.action.ps1
# Désactive le protocole SMBv1 (legacy, vulnérable EternalBlue).
# 2 actions : Set-SmbServerConfiguration + Disable-WindowsOptionalFeature.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$cfg = Get-SmbServerConfiguration -ErrorAction SilentlyContinue
$feat = Get-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -ErrorAction SilentlyContinue

$before = @{
    SmbServerEnableSMB1 = if ($cfg) { [bool]$cfg.EnableSMB1Protocol } else { $null }
    OptionalFeatureState = if ($feat) { $feat.State.ToString() } else { 'Unknown' }
}

if ($cfg -and $cfg.EnableSMB1Protocol) {
    Set-SmbServerConfiguration -EnableSMB1Protocol $false -Force -ErrorAction SilentlyContinue
}
if ($feat -and $feat.State -ne 'Disabled' -and $feat.State -ne 'DisabledWithPayloadRemoved') {
    Disable-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -NoRestart -ErrorAction SilentlyContinue | Out-Null
}

$cfg = Get-SmbServerConfiguration -ErrorAction SilentlyContinue
$feat = Get-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -ErrorAction SilentlyContinue
$after = @{
    SmbServerEnableSMB1 = if ($cfg) { [bool]$cfg.EnableSMB1Protocol } else { $null }
    OptionalFeatureState = if ($feat) { $feat.State.ToString() } else { 'Unknown' }
}

@{ ok = $true; before = $before; after = $after } | ConvertTo-Json -Compress -Depth 10
