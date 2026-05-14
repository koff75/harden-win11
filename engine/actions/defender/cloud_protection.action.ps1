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

# Preflight Home Edition : CloudBlockLevel High requiert Microsoft Defender
# for Endpoint (MDE), une feature Enterprise/Pro/Server. Sur Win11 Home,
# Set-MpPreference -CloudBlockLevel High est accepte silencieusement sans
# effet. CRITIQUE : on detecte AVANT de toucher MAPSReporting et
# CloudExtendedTimeout — sinon on aurait une demi-modif sans path d'undo
# (le moteur, voyant ok=false, ne lance pas .undo.ps1).
$probe = $null
try {
    $probe = Set-MpPreference -CloudBlockLevel High -ErrorAction Stop
    $afterProbe = (Get-MpPreference).CloudBlockLevel
    if ([int]$afterProbe -ne 4) {
        # Pas pris effet — restaure CloudBlockLevel a la valeur initiale puis bail out.
        Set-MpPreference -CloudBlockLevel ([int]$rawBlock) -ErrorAction SilentlyContinue
        $os = Get-CimInstance Win32_OperatingSystem
        @{
            ok    = $false
            error = "Set-MpPreference -CloudBlockLevel High sans effet (reste a $afterProbe). Cette fonctionnalite necessite Microsoft Defender for Endpoint, disponible sur Windows Pro/Enterprise. Edition detectee : $($os.Caption)."
            before = $before
            after  = $before
        } | ConvertTo-Json -Compress -Depth 10
        exit 0
    }
} catch {
    @{
        ok    = $false
        error = "Set-MpPreference -CloudBlockLevel a echoue : $($_.Exception.Message)"
        before = $before
        after  = $before
    } | ConvertTo-Json -Compress -Depth 10
    exit 0
}

# CloudBlockLevel a bien ete applique → on continue avec les 2 autres prefs.
Set-MpPreference -MAPSReporting Advanced
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
