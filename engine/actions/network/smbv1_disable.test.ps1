# smbv1_disable.test.ps1
# Conforme = SmbServerConfiguration.EnableSMB1Protocol=false ET feature Disabled.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$cfg = Get-SmbServerConfiguration -ErrorAction SilentlyContinue
$feat = Get-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -ErrorAction SilentlyContinue

$smbServerOff   = ($null -eq $cfg) -or (-not $cfg.EnableSMB1Protocol)
$featureOff     = ($null -eq $feat) -or ($feat.State.ToString() -in @('Disabled', 'DisabledWithPayloadRemoved'))
$compliant      = $smbServerOff -and $featureOff

@{
    compliant = $compliant
    current   = @{
        SmbServerEnableSMB1  = if ($cfg) { [bool]$cfg.EnableSMB1Protocol } else { $null }
        OptionalFeatureState = if ($feat) { $feat.State.ToString() } else { 'Unknown' }
    }
} | ConvertTo-Json -Compress -Depth 10
