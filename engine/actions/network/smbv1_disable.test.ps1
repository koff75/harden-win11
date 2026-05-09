# smbv1_disable.test.ps1
# Conforme = SmbServerConfiguration.EnableSMB1Protocol=false ET feature Disabled.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$cfg = $null
try { $cfg = Get-SmbServerConfiguration -ErrorAction Stop } catch {}

# Get-WindowsOptionalFeature -Online requiert admin (DISM), même en read-only.
# En non-admin on capture la COMException et on continue sans ce check.
$feat = $null
$featPartial = $false
try {
    $feat = Get-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -ErrorAction Stop
} catch {
    $featPartial = $true
}

$smbServerOff   = ($null -eq $cfg) -or (-not $cfg.EnableSMB1Protocol)
$featureOff     = ($null -eq $feat) -or ($feat.State.ToString() -in @('Disabled', 'DisabledWithPayloadRemoved'))
$compliant      = $smbServerOff -and $featureOff

# Détection partage SMB legacy en cours d'utilisation.
$inUse = $false
$inUseReason = $null
try {
    $smbConn = @(Get-SmbConnection -ErrorAction Stop)
    $legacy = @($smbConn | Where-Object { $_.Dialect -like '1.*' -or $_.Dialect -like 'SMB1*' })
    if ($legacy.Count -gt 0) {
        $inUse = $true
        $inUseReason = "$($legacy.Count) connexion(s) SMB1 active(s) (dialect=$($legacy.Dialect -join ',')). Désactiver SMBv1 maintenant coupera ces partages."
    }
} catch {}

@{
    compliant             = $compliant
    current               = @{
        SmbServerEnableSMB1  = if ($cfg) { [bool]$cfg.EnableSMB1Protocol } else { $null }
        OptionalFeatureState = if ($feat) { $feat.State.ToString() } else { 'Unknown' }
        PartialScan          = $featPartial
    }
    feature_in_use        = $inUse
    feature_in_use_reason = $inUseReason
} | ConvertTo-Json -Compress -Depth 10
