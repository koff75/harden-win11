# cloud_protection.test.ps1
# Conforme = MAPSReporting=Advanced (2) ET CloudBlockLevel=High (4) ET CloudExtendedTimeout=50

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$mapsNames  = @{ 0 = 'Disabled'; 1 = 'Basic'; 2 = 'Advanced' }
$blockNames = @{ 0 = 'Default'; 2 = 'Moderate'; 4 = 'High'; 6 = 'HighPlus'; 8 = 'ZeroTolerance' }

$pref = Get-MpPreference
$rawMaps  = [int]$pref.MAPSReporting
$rawBlock = [int]$pref.CloudBlockLevel
$rawTo    = [int]$pref.CloudExtendedTimeout

$compliant = ($rawMaps -eq 2) -and ($rawBlock -eq 4) -and ($rawTo -eq 50)

@{
    compliant = $compliant
    current   = @{
        MAPSReporting        = if ($mapsNames.ContainsKey($rawMaps))   { $mapsNames[$rawMaps] }   else { "Unknown($rawMaps)" }
        CloudBlockLevel      = if ($blockNames.ContainsKey($rawBlock)) { $blockNames[$rawBlock] } else { "Unknown($rawBlock)" }
        CloudExtendedTimeout = $rawTo
    }
} | ConvertTo-Json -Compress -Depth 10
