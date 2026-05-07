# cloud_protection.action.ps1
# Configure 3 paramètres de protection cloud Defender :
#   - MAPSReporting       : niveau de partage avec Microsoft Active Protection Service
#                           0=Disabled, 1=Basic, 2=Advanced
#   - CloudBlockLevel     : agressivité du blocage cloud
#                           0=Default, 2=Moderate, 4=High, 6=HighPlus, 8=ZeroTolerance
#   - CloudExtendedTimeout: secondes d'extension du timeout d'analyse cloud (0-50)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$mapsNames  = @{ 0 = 'Disabled'; 1 = 'Basic'; 2 = 'Advanced' }
$blockNames = @{ 0 = 'Default'; 2 = 'Moderate'; 4 = 'High'; 6 = 'HighPlus'; 8 = 'ZeroTolerance' }

$pref = Get-MpPreference
$rawMaps  = [int]$pref.MAPSReporting
$rawBlock = [int]$pref.CloudBlockLevel
$rawTo    = [int]$pref.CloudExtendedTimeout

$before = @{
    MAPSReporting        = if ($mapsNames.ContainsKey($rawMaps))   { $mapsNames[$rawMaps] }   else { "Unknown($rawMaps)" }
    CloudBlockLevel      = if ($blockNames.ContainsKey($rawBlock)) { $blockNames[$rawBlock] } else { "Unknown($rawBlock)" }
    CloudExtendedTimeout = $rawTo
}

Set-MpPreference -MAPSReporting Advanced
Set-MpPreference -CloudBlockLevel High
Set-MpPreference -CloudExtendedTimeout 50

$pref = Get-MpPreference
$rawMaps  = [int]$pref.MAPSReporting
$rawBlock = [int]$pref.CloudBlockLevel
$rawTo    = [int]$pref.CloudExtendedTimeout

$after = @{
    MAPSReporting        = if ($mapsNames.ContainsKey($rawMaps))   { $mapsNames[$rawMaps] }   else { "Unknown($rawMaps)" }
    CloudBlockLevel      = if ($blockNames.ContainsKey($rawBlock)) { $blockNames[$rawBlock] } else { "Unknown($rawBlock)" }
    CloudExtendedTimeout = $rawTo
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
