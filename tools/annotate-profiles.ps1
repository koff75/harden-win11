# annotate-profiles.ps1
# Annote chaque rule des manifests avec : profiles + breaks.
# Mapping base sur l'analyse de risque par regle.
#
# IMPORTANT : ce fichier est en ASCII pur (sans accents) pour eviter les
# soucis d'encoding PS 5.1 sans BOM. Les YAML cibles sont en UTF-8 et le
# moteur les lit en UTF-8 strict.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Table de mapping rule_id -> @{ profiles, breaks }
$mapping = @{
    'defender.controlled_folder_access' = @{
        profiles = @('maximal')
        breaks   = @('Office (Word, Excel, PowerPoint), Photoshop, Visual Studio Code, OBS, Firefox profiles, certains jeux. Liste blanche manuelle requise apres activation.')
    }
    'defender.network_protection' = @{
        profiles = @('business', 'maximal')
        breaks   = @('Apps qui se connectent a des hotes flagges faux positifs (occasionnel). A auditer dans Event Viewer si une app a un comportement reseau anormal.')
    }

    'firewall.profile_domain' = @{
        profiles = @('business', 'maximal')
        breaks   = @()
    }

    'accounts.rename_admin' = @{
        profiles = @('personal', 'maximal')
        breaks   = @('Scripts qui hardcodent le nom Administrator (rare en perso, possible en entreprise). Preferer SID-500 ou (point)Administrator par alias.')
    }

    'system_settings.uac_deny_user_elevation' = @{
        profiles = @('maximal')
        breaks   = @('Le mode Run as administrator depuis un compte standard ne fonctionne plus. Sur PC perso (compte unique = admin), aucun impact.')
    }
    'system_settings.rdp_disable' = @{
        profiles = @('personal', 'maximal')
        breaks   = @('Connexion RDP entrante (impossible de se connecter via Bureau a distance). Si tu utilises RDP pour le support distant, laisser active.')
    }
    'system_settings.rdp_firewall_disable' = @{
        profiles = @('personal', 'maximal')
        breaks   = @('Idem rdp_disable : ports firewall RDP fermes.')
    }
    'system_settings.hibernate_off' = @{
        profiles = @('business', 'maximal')
        breaks   = @('Plus d hibernation possible (sleep et shutdown OK). Sur laptop, verifier que ton flow ne depend pas d hibernation longue.')
    }

    'network.llmnr_disable' = @{
        profiles = @('personal', 'maximal')
        breaks   = @('Decouverte locale par nom NetBIOS-like (rare en perso). Imprimantes anciennes peuvent ne plus etre decouvertes automatiquement en entreprise.')
    }
    'network.mdns_disable' = @{
        profiles = @('maximal')
        breaks   = @('Chromecast, AirPrint, imprimantes via Bonjour, devices IoT en mDNS. Casse les setups entreprise avec imprimante reseau Apple/multi-fonctions.')
    }
    'network.netbios_off' = @{
        profiles = @('personal', 'maximal')
        breaks   = @('Partages SMB qui passent par NetBIOS-name (NAS legacy, anciens NAS Synology/QNAP). Si tes partages utilisent FQDN ou IP, aucun impact.')
    }
    'network.wpad_disable' = @{
        profiles = @('personal', 'maximal')
        breaks   = @('Configuration de proxy automatique via WPAD/PAC (souvent utilise en entreprise). Si tes settings Internet utilisent un proxy auto-discovery, casse l acces web.')
    }

    'bloatware.cleanup' = @{
        profiles = @('personal', 'maximal')
        breaks   = @('Apps Store desinstallees ne peuvent etre reinstallees que via Microsoft Store + compte Microsoft. Liste : TikTok, Spotify, Disney, CandyCrush, Bing News, Microsoft Solitaire, Skype, etc.')
    }

    # ASR a risque eleve
    'asr.block_psexec_wmi' = @{
        profiles = @('maximal')
        breaks   = @('PsExec utilise pour administration distante (Sysadmin / IT). Casse les outils Sysinternals en usage admin legitime.')
    }
    'asr.block_unprevalent_executables' = @{
        profiles = @('maximal')
        breaks   = @('Executables peu communs (apps internes, builds dev, scripts metier). Faux positifs frequents en environnement developpement.')
    }
    'asr.block_unsigned_usb' = @{
        profiles = @('business', 'maximal')
        breaks   = @('Outils portables non signes sur cle USB (utilitaires admin, scripts custom).')
    }
    'asr.block_safe_mode_reboot' = @{
        profiles = @('business', 'maximal')
        breaks   = @('Reboot en Safe Mode bloque (par ex. via msconfig). Affecte les troubleshooting workflows.')
    }
}

$root = Split-Path -Parent $PSScriptRoot
$manifestsDir = Join-Path $root 'manifests'

$count = @{ updated = 0; default = 0 }

Get-ChildItem $manifestsDir -Filter '*.yaml' | ForEach-Object {
    $path = $_.FullName
    $content = Get-Content $path -Raw -Encoding UTF8

    $lines = $content -split "`r?`n"
    $output = New-Object System.Collections.Generic.List[string]
    $i = 0
    while ($i -lt $lines.Count) {
        $line = $lines[$i]
        $output.Add($line)

        if ($line -match '^\s*-\s+id:\s+(\S+)\s*$') {
            $ruleID = $matches[1]
            $blockStart = $i + 1
            $blockEnd = $blockStart
            while ($blockEnd -lt $lines.Count) {
                $bl = $lines[$blockEnd]
                if ($bl -match '^\s*-\s+id:\s+' -or $bl -match '^[a-z]') {
                    break
                }
                $blockEnd++
            }
            $insertAt = -1
            for ($j = $blockStart; $j -lt $blockEnd; $j++) {
                if ($lines[$j] -match '^\s+(undo|test):\s+') {
                    $insertAt = $j
                }
            }
            $linesToSkip = New-Object System.Collections.Generic.HashSet[int]
            for ($j = $blockStart; $j -lt $blockEnd; $j++) {
                if ($lines[$j] -match '^\s+(profiles|breaks):') {
                    $linesToSkip.Add($j) | Out-Null
                    $k = $j + 1
                    while ($k -lt $blockEnd -and $lines[$k] -match '^\s+-\s+') {
                        $linesToSkip.Add($k) | Out-Null
                        $k++
                    }
                }
            }

            for ($j = $blockStart; $j -lt $blockEnd; $j++) {
                if ($linesToSkip.Contains($j)) { continue }
                $output.Add($lines[$j])
                if ($j -eq $insertAt) {
                    $entry = $mapping[$ruleID]
                    $profiles = $null
                    $breaks = @()
                    if ($entry) {
                        $profiles = $entry.profiles
                        $breaks = $entry.breaks
                        $count.updated++
                    } else {
                        $profiles = @('personal', 'business', 'maximal')
                        $breaks = @()
                        $count.default++
                    }
                    $indent = '    '
                    $output.Add("${indent}profiles:")
                    foreach ($p in $profiles) {
                        $output.Add("${indent}  - $p")
                    }
                    if ($breaks.Count -gt 0) {
                        $output.Add("${indent}breaks:")
                        foreach ($b in $breaks) {
                            $escaped = $b -replace '"', '\"'
                            $output.Add("${indent}  - `"$escaped`"")
                        }
                    }
                }
            }
            $i = $blockEnd
            continue
        }
        $i++
    }

    $newContent = $output -join "`r`n"
    [System.IO.File]::WriteAllText($path, $newContent, [System.Text.UTF8Encoding]::new($false))
}

Write-Host ("Annotated rules: {0} explicit + {1} default = {2} total" -f $count.updated, $count.default, ($count.updated + $count.default)) -ForegroundColor Green
