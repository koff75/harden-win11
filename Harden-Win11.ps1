<#
.SYNOPSIS
    Hardening Windows 11 - Configuration de sécurité complète et reproductible.

.DESCRIPTION
    Applique une baseline de sécurité Windows 11 :
      - Defender : Real-time, Behavior, IOAV, NIS, Tamper Protection, PUA, CFA, Network Protection, ASR rules
      - Firewall : 3 profils ON, blocage SMB entrant sur Public
      - Comptes : Built-in Admin / Guest / WsiAccount désactivés
      - UAC : niveau 5 sur Secure Desktop
      - RDP : désactivé
      - Hibernation : désactivée, Fast Startup : OFF
      - Privacy : AdvertisingID OFF, OnlineSpeech OFF, Telemetry minimum, anti-bloat HKCU
      - Réseau : LLMNR off, NetBIOS off, NTLMv2 only (LmCompatibilityLevel=5)
      - Bloatware : suppression apps Store suspectes / inutiles

    Le script est idempotent : il peut être relancé. Il fait un backup registre avant modifs.

.PARAMETER LogPath
    Chemin du log. Par défaut : C:\ProgramData\Harden-Win11\harden-<date>.log

.PARAMETER SkipBloatware
    Ne pas désinstaller le bloatware Store (utile pour test).

.PARAMETER AsrAuditMode
    Active les règles ASR en mode Audit (loggent sans bloquer). Recommandé au premier passage.

.PARAMETER DryRun
    N'applique rien, affiche seulement ce qui serait fait.

.EXAMPLE
    # Premier passage : audit mode pour ASR, voir si ça casse des apps
    .\Harden-Win11.ps1 -AsrAuditMode

.EXAMPLE
    # Passage final : tout en mode bloquant
    .\Harden-Win11.ps1

.NOTES
    Doit tourner en tant qu'administrateur.
    Section A (HKCU) s'applique à l'utilisateur courant uniquement.
    Pour multi-utilisateurs, relancer la section A sous chaque session.
#>

[CmdletBinding()]
param(
    [string]$LogPath = "C:\ProgramData\Harden-Win11\harden-$(Get-Date -Format 'yyyyMMdd-HHmmss').log",
    [switch]$SkipBloatware,
    [switch]$AsrAuditMode,
    [switch]$DryRun
)

# ============================================================================
# Préambule : vérifications, logging, helpers
# ============================================================================

$ErrorActionPreference = 'Stop'
$script:Errors = 0
$script:Warnings = 0
$script:Applied = 0
$script:Skipped = 0

# Vérif admin
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "ERREUR : ce script doit être lancé en tant qu'administrateur." -ForegroundColor Red
    exit 1
}

# Vérif version Windows
$os = Get-CimInstance Win32_OperatingSystem
if ($os.BuildNumber -lt 22000) {
    Write-Host "ATTENTION : ce script vise Windows 11 (build 22000+). Build détectée : $($os.BuildNumber)" -ForegroundColor Yellow
    $continue = Read-Host "Continuer quand même ? (o/N)"
    if ($continue -ne 'o') { exit 0 }
}

# Préparer le log
$logDir = Split-Path $LogPath -Parent
if (-not (Test-Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
}

function Write-Log {
    param(
        [string]$Message,
        [ValidateSet('INFO','OK','WARN','ERROR','SKIP','DRY')]
        [string]$Level = 'INFO'
    )
    $ts = Get-Date -Format 'HH:mm:ss'
    $line = "[$ts][$Level] $Message"
    $color = switch ($Level) {
        'OK'    { 'Green' }
        'WARN'  { 'Yellow' }
        'ERROR' { 'Red' }
        'SKIP'  { 'DarkGray' }
        'DRY'   { 'Cyan' }
        default { 'White' }
    }
    Write-Host $line -ForegroundColor $color
    Add-Content -Path $LogPath -Value $line
}

function Invoke-Step {
    <#
    Wrapper pour chaque action : attrape les erreurs, gère DryRun, logue le résultat.
    #>
    param(
        [string]$Description,
        [scriptblock]$Action,
        [scriptblock]$Test  # optionnel : retourne $true si déjà appliqué
    )

    if ($Test) {
        try {
            if (& $Test) {
                Write-Log "$Description -> déjà conforme" -Level SKIP
                $script:Skipped++
                return
            }
        } catch {
            # Si le test échoue, on tente quand même l'action
        }
    }

    if ($DryRun) {
        Write-Log "$Description -> serait appliqué" -Level DRY
        return
    }

    try {
        & $Action
        Write-Log "$Description -> appliqué" -Level OK
        $script:Applied++
    } catch {
        Write-Log "$Description -> ÉCHEC : $($_.Exception.Message)" -Level ERROR
        $script:Errors++
    }
}

function Set-RegValue {
    <#
    Crée le chemin si nécessaire, puis set la valeur. Idempotent.
    #>
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)]$Value,
        [ValidateSet('String','ExpandString','Binary','DWord','MultiString','QWord')]
        [string]$Type = 'DWord'
    )
    if (-not (Test-Path $Path)) {
        New-Item -Path $Path -Force | Out-Null
    }
    New-ItemProperty -Path $Path -Name $Name -Value $Value -PropertyType $Type -Force | Out-Null
}

function Test-RegValue {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)]$Expected
    )
    if (-not (Test-Path $Path)) { return $false }
    $actual = (Get-ItemProperty -Path $Path -Name $Name -ErrorAction SilentlyContinue).$Name
    return $actual -eq $Expected
}

# Backup registre
function Backup-Registry {
    $backupDir = Join-Path $logDir "regbackup-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    Write-Log "Backup registre dans $backupDir" -Level INFO

    $hives = @{
        'HKLM-SOFTWARE-Policies' = 'HKLM\SOFTWARE\Policies'
        'HKLM-System-CCS-Lsa'    = 'HKLM\SYSTEM\CurrentControlSet\Control\Lsa'
        'HKCU-CDM'               = 'HKCU\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'
    }
    foreach ($name in $hives.Keys) {
        $file = Join-Path $backupDir "$name.reg"
        & reg.exe export $hives[$name] $file /y 2>&1 | Out-Null
    }
    Write-Log "Backup registre terminé" -Level OK
}

# ============================================================================
# Démarrage
# ============================================================================

Write-Log "=== Hardening Windows 11 - démarrage ===" -Level INFO
Write-Log "Hôte : $env:COMPUTERNAME | Utilisateur : $env:USERNAME | Build : $($os.BuildNumber)"
if ($DryRun)        { Write-Log "MODE DRY-RUN : aucune modification ne sera appliquée" -Level WARN }
if ($AsrAuditMode)  { Write-Log "Règles ASR en mode AUDIT (loggent sans bloquer)" -Level WARN }

if (-not $DryRun) { Backup-Registry }

# ============================================================================
# SECTION 1 : Microsoft Defender
# ============================================================================
Write-Log "--- Section 1/8 : Microsoft Defender ---" -Level INFO

Invoke-Step "Defender : Real-time Protection" {
    Set-MpPreference -DisableRealtimeMonitoring $false
} -Test {
    -not (Get-MpPreference).DisableRealtimeMonitoring
}

Invoke-Step "Defender : Behavior Monitoring" {
    Set-MpPreference -DisableBehaviorMonitoring $false
} -Test {
    -not (Get-MpPreference).DisableBehaviorMonitoring
}

Invoke-Step "Defender : IOAV (analyse pièces jointes / DL)" {
    Set-MpPreference -DisableIOAVProtection $false
} -Test {
    -not (Get-MpPreference).DisableIOAVProtection
}

Invoke-Step "Defender : Network Inspection System (NIS)" {
    Set-MpPreference -DisableIntrusionPreventionSystem $false
} -Test {
    -not (Get-MpPreference).DisableIntrusionPreventionSystem
}

Invoke-Step "Defender : Script scanning" {
    Set-MpPreference -DisableScriptScanning $false
} -Test {
    -not (Get-MpPreference).DisableScriptScanning
}

Invoke-Step "Defender : Cloud-delivered protection (HIGH)" {
    Set-MpPreference -MAPSReporting Advanced
    Set-MpPreference -CloudBlockLevel High
    Set-MpPreference -CloudExtendedTimeout 50
} -Test {
    $p = Get-MpPreference
    $p.MAPSReporting -eq 'Advanced' -and $p.CloudBlockLevel -eq 'High'
}

Invoke-Step "Defender : Sample submission (envoi auto échantillons sûrs)" {
    Set-MpPreference -SubmitSamplesConsent SendSafeSamples
} -Test {
    (Get-MpPreference).SubmitSamplesConsent -eq 'SendSafeSamples'
}

Invoke-Step "Defender : PUA Protection (Potentially Unwanted Apps)" {
    Set-MpPreference -PUAProtection Enabled
} -Test {
    (Get-MpPreference).PUAProtection -eq 'Enabled'
}

Invoke-Step "Defender : Controlled Folder Access (anti-ransomware)" {
    Set-MpPreference -EnableControlledFolderAccess Enabled
} -Test {
    (Get-MpPreference).EnableControlledFolderAccess -eq 'Enabled'
}

Invoke-Step "Defender : Network Protection" {
    Set-MpPreference -EnableNetworkProtection Enabled
} -Test {
    (Get-MpPreference).EnableNetworkProtection -eq 'Enabled'
}

Invoke-Step "Defender : signatures à jour" {
    Update-MpSignature -ErrorAction Stop
}

# Tamper Protection : ne se modifie pas via PowerShell, juste vérification
$tp = (Get-MpComputerStatus).IsTamperProtected
if ($tp) {
    Write-Log "Defender : Tamper Protection active -> OK" -Level OK
} else {
    Write-Log "Defender : Tamper Protection INACTIVE -> activer manuellement dans Windows Security > Virus & threat protection > Manage settings" -Level WARN
    $script:Warnings++
}

# ============================================================================
# SECTION 2 : ASR Rules (Attack Surface Reduction)
# ============================================================================
Write-Log "--- Section 2/8 : ASR rules ---" -Level INFO

# Mode : 1 = Block, 2 = Audit, 6 = Warn
$asrAction = if ($AsrAuditMode) { 2 } else { 1 }

# Règles ASR avec leur GUID. Source : docs Microsoft Defender ASR.
$asrRules = @{
    'Block Office apps from creating child processes'                     = 'D4F940AB-401B-4EFC-AADC-AD5F3C50688A'
    'Block Office apps from creating executable content'                  = '3B576869-A4EC-4529-8536-B80A7769E899'
    'Block Office apps from injecting code into other processes'          = '75668C1F-73B5-4CF0-BB93-3ECF5CB7CC84'
    'Block Office communication apps from creating child processes'       = '26190899-1602-49E8-8B27-EB1D0A1CE869'
    'Block Win32 API calls from Office macros'                            = '92E97FA1-2EDF-4476-BDD6-9DD0B4DDDC7B'
    'Block all Office applications from creating child processes'         = 'D4F940AB-401B-4EFC-AADC-AD5F3C50688A'
    'Block executable content from email client and webmail'              = 'BE9BA2D9-53EA-4CDC-84E5-9B1EEEE46550'
    'Block execution of potentially obfuscated scripts'                   = '5BEB7EFE-FD9A-4556-801D-275E5FFC04CC'
    'Block JS/VBS from launching downloaded executable content'           = 'D3E037E1-3EB8-44C8-A917-57927947596D'
    'Block credential stealing from LSASS'                                = '9E6C4E1F-7D60-472F-BA1A-A39EF669E4B2'
    'Block untrusted/unsigned processes that run from USB'                = 'B2B3F03D-6A65-4F7B-A9C7-1C7EF74A9BA4'
    'Block persistence through WMI event subscription'                    = 'E6DB77E5-3DF2-4CF1-B95A-636979351E5B'
    'Block process creations from PSExec and WMI commands'                = 'D1E49AAC-8F56-4280-B9BA-993A6D77406C'
    'Block executable files unless they meet a prevalence/age criterion'  = '01443614-CD74-433A-B99E-2ECDC07BFC25'
    'Use advanced protection against ransomware'                          = 'C1DB55AB-C21A-4637-BB3F-A12568109D35'
    'Block Adobe Reader from creating child processes'                    = '7674BA52-37EB-4A4F-A9A1-F0F9A1619A2C'
    'Block abuse of exploited vulnerable signed drivers'                  = '56A863A9-875E-4185-98A7-B882C64B5CE5'
    'Block rebooting machine in Safe Mode'                                = '33DDEDF1-C6E0-47CB-833E-DE6133960387'
    'Block use of copied or impersonated system tools'                    = 'C0033C00-D16D-4114-A5A0-DC9B3A7D2CEB'
    'Block Webshell creation for Servers'                                 = 'A8F5898E-1DC8-49A9-9878-85004B8A61E6'
}

# Dédup (certains GUIDs apparaissent deux fois pour clarté)
$uniqueRules = $asrRules.Values | Select-Object -Unique

foreach ($guid in $uniqueRules) {
    $name = ($asrRules.GetEnumerator() | Where-Object { $_.Value -eq $guid } | Select-Object -First 1).Name
    Invoke-Step "ASR : $name" {
        Add-MpPreference -AttackSurfaceReductionRules_Ids $guid -AttackSurfaceReductionRules_Actions $asrAction -ErrorAction Stop
    } -Test {
        $current = (Get-MpPreference).AttackSurfaceReductionRules_Ids
        $actions = (Get-MpPreference).AttackSurfaceReductionRules_Actions
        if (-not $current) { return $false }
        for ($i = 0; $i -lt $current.Count; $i++) {
            if ($current[$i] -eq $guid -and $actions[$i] -eq $asrAction) { return $true }
        }
        return $false
    }
}

# ============================================================================
# SECTION 3 : Firewall
# ============================================================================
Write-Log "--- Section 3/8 : Firewall ---" -Level INFO

Invoke-Step "Firewall : profil Domain activé, blocage entrant par défaut" {
    Set-NetFirewallProfile -Profile Domain -Enabled True -DefaultInboundAction Block -DefaultOutboundAction Allow
}
Invoke-Step "Firewall : profil Private activé, blocage entrant par défaut" {
    Set-NetFirewallProfile -Profile Private -Enabled True -DefaultInboundAction Block -DefaultOutboundAction Allow
}
Invoke-Step "Firewall : profil Public activé, blocage entrant par défaut" {
    Set-NetFirewallProfile -Profile Public -Enabled True -DefaultInboundAction Block -DefaultOutboundAction Allow
}

Invoke-Step "Firewall : bloquer SMB (445) entrant sur profil Public" {
    $existing = Get-NetFirewallRule -DisplayName "Block SMB Inbound (Public) [Hardening]" -ErrorAction SilentlyContinue
    if ($existing) { Remove-NetFirewallRule -DisplayName "Block SMB Inbound (Public) [Hardening]" }
    New-NetFirewallRule -DisplayName "Block SMB Inbound (Public) [Hardening]" `
        -Direction Inbound -Protocol TCP -LocalPort 445 -Action Block -Profile Public | Out-Null
} -Test {
    $r = Get-NetFirewallRule -DisplayName "Block SMB Inbound (Public) [Hardening]" -ErrorAction SilentlyContinue
    $r -and $r.Enabled -eq 'True'
}

Invoke-Step "Firewall : bloquer NetBIOS (137-139) entrant sur profil Public" {
    $existing = Get-NetFirewallRule -DisplayName "Block NetBIOS Inbound (Public) [Hardening]" -ErrorAction SilentlyContinue
    if ($existing) { Remove-NetFirewallRule -DisplayName "Block NetBIOS Inbound (Public) [Hardening]" }
    New-NetFirewallRule -DisplayName "Block NetBIOS Inbound (Public) [Hardening]" `
        -Direction Inbound -Protocol UDP -LocalPort 137,138 -Action Block -Profile Public | Out-Null
    New-NetFirewallRule -DisplayName "Block NetBIOS TCP Inbound (Public) [Hardening]" `
        -Direction Inbound -Protocol TCP -LocalPort 139 -Action Block -Profile Public | Out-Null
}

# ============================================================================
# SECTION 4 : Comptes locaux
# ============================================================================
Write-Log "--- Section 4/8 : Comptes locaux ---" -Level INFO

$accountsToDisable = @('Administrator', 'Guest', 'WsiAccount', 'DefaultAccount')
foreach ($acct in $accountsToDisable) {
    Invoke-Step "Compte : désactiver '$acct'" {
        $u = Get-LocalUser -Name $acct -ErrorAction SilentlyContinue
        if ($u) { Disable-LocalUser -Name $acct }
    } -Test {
        $u = Get-LocalUser -Name $acct -ErrorAction SilentlyContinue
        # Considéré conforme si le compte n'existe pas OU est désactivé
        (-not $u) -or (-not $u.Enabled)
    }
}

# Renommer le compte Administrator built-in (défense en profondeur)
Invoke-Step "Compte : renommer Administrator built-in -> AdminLocal_$env:COMPUTERNAME" {
    $admin = Get-LocalUser | Where-Object { $_.SID -like 'S-1-5-*-500' }
    if ($admin -and $admin.Name -eq 'Administrator') {
        Rename-LocalUser -Name 'Administrator' -NewName "AdminLocal_$env:COMPUTERNAME"
    }
} -Test {
    $admin = Get-LocalUser | Where-Object { $_.SID -like 'S-1-5-*-500' }
    $admin.Name -ne 'Administrator'
}

# ============================================================================
# SECTION 5 : UAC, RDP, Hibernation, Fast Startup
# ============================================================================
Write-Log "--- Section 5/8 : UAC, RDP, Power ---" -Level INFO

Invoke-Step "UAC : EnableLUA = 1" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'EnableLUA' -Value 1
} -Test {
    Test-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'EnableLUA' -Expected 1
}

Invoke-Step "UAC : ConsentPromptBehaviorAdmin = 5 (prompt sur secure desktop)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorAdmin' -Value 5
} -Test {
    Test-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorAdmin' -Expected 5
}

Invoke-Step "UAC : PromptOnSecureDesktop = 1" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'PromptOnSecureDesktop' -Value 1
} -Test {
    Test-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'PromptOnSecureDesktop' -Expected 1
}

Invoke-Step "UAC : refuser élévation pour comptes standard (ConsentPromptBehaviorUser=0)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorUser' -Value 0
} -Test {
    Test-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorUser' -Expected 0
}

Invoke-Step "RDP : désactivé (fDenyTSConnections=1)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server' -Name 'fDenyTSConnections' -Value 1
} -Test {
    Test-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server' -Name 'fDenyTSConnections' -Expected 1
}

Invoke-Step "RDP : règle firewall désactivée" {
    Disable-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue
}

Invoke-Step "Hibernation : désactivée (libère ~8-12 GB)" {
    & powercfg.exe /hibernate off | Out-Null
} -Test {
    -not (Test-Path "$env:SystemDrive\hiberfil.sys")
}

Invoke-Step "Fast Startup : OFF (HiberbootEnabled=0)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Power' -Name 'HiberbootEnabled' -Value 0
} -Test {
    Test-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Power' -Name 'HiberbootEnabled' -Expected 0
}

# ============================================================================
# SECTION 6 : Hardening protocoles réseau (LLMNR, NetBIOS, NTLM)
# ============================================================================
Write-Log "--- Section 6/8 : Hardening protocoles réseau ---" -Level INFO

Invoke-Step "LLMNR : désactivé (EnableMulticast=0)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient' -Name 'EnableMulticast' -Value 0
} -Test {
    Test-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient' -Name 'EnableMulticast' -Expected 0
}

Invoke-Step "mDNS : désactivé (Bonjour-like, vecteur similaire à LLMNR)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters' -Name 'EnableMDNS' -Value 0
} -Test {
    Test-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters' -Name 'EnableMDNS' -Expected 0
}

Invoke-Step "NetBIOS over TCP/IP : désactivé sur tous les adaptateurs" {
    $adapters = Get-CimInstance -ClassName Win32_NetworkAdapterConfiguration -Filter 'IPEnabled=TRUE'
    foreach ($a in $adapters) {
        $null = Invoke-CimMethod -InputObject $a -MethodName SetTcpipNetbios -Arguments @{TcpipNetbiosOptions = [uint32]2}
    }
    # Persistance via registre pour les futurs adaptateurs
    Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Services\NetBT\Parameters\Interfaces' | ForEach-Object {
        Set-ItemProperty -Path $_.PSPath -Name 'NetbiosOptions' -Value 2 -Force -ErrorAction SilentlyContinue
    }
}

Invoke-Step "WPAD : désactivé (anti-poisoning proxy auto-discovery)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc' -Name 'Start' -Value 4
}

Invoke-Step "NTLM : LmCompatibilityLevel = 5 (NTLMv2 only, refuse LM/NTLMv1)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'LmCompatibilityLevel' -Value 5
} -Test {
    Test-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'LmCompatibilityLevel' -Expected 5
}

Invoke-Step "NTLM : signature SMB client requise" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' -Name 'RequireSecuritySignature' -Value 1
}
Invoke-Step "NTLM : signature SMB serveur requise" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters' -Name 'RequireSecuritySignature' -Value 1
}

Invoke-Step "SMB : désactiver SMBv1 (legacy, vulnérable EternalBlue)" {
    Set-SmbServerConfiguration -EnableSMB1Protocol $false -Force -ErrorAction SilentlyContinue
    Disable-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -NoRestart -ErrorAction SilentlyContinue | Out-Null
}

Invoke-Step "SMB : autoriser SMB en lecture seule au niveau client (pas de credentials guest)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' -Name 'AllowInsecureGuestAuth' -Value 0
}

# ============================================================================
# SECTION 7 : Privacy & Telemetry
# ============================================================================
Write-Log "--- Section 7/8 : Privacy & Telemetry ---" -Level INFO

Invoke-Step "Telemetry : niveau 1 (Required only - minimum sur Win11 Home)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection' -Name 'AllowTelemetry' -Value 1
}

Invoke-Step "AdvertisingID : désactivé (machine-wide)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\AdvertisingInfo' -Name 'DisabledByGroupPolicy' -Value 1
}

Invoke-Step "Online Speech : désactivé" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\InputPersonalization' -Name 'AllowInputPersonalization' -Value 0
}

Invoke-Step "Activity History : désactivé (bloque envoi cloud)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' -Name 'EnableActivityFeed' -Value 0
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' -Name 'PublishUserActivities' -Value 0
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' -Name 'UploadUserActivities' -Value 0
}

Invoke-Step "Cortana : désactivée" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Windows Search' -Name 'AllowCortana' -Value 0
}

Invoke-Step "Recall : désactivé (préventif si futur push)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsAI' -Name 'DisableAIDataAnalysis' -Value 1
}

Invoke-Step "Consumer features : désactivées (anti-bloat machine-wide)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent' -Name 'DisableWindowsConsumerFeatures' -Value 1
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent' -Name 'DisableConsumerAccountStateContent' -Value 1
}

# --- HKCU : applique uniquement à l'utilisateur courant ---
Write-Log "Privacy HKCU : modifications appliquées à l'utilisateur courant ($env:USERNAME) uniquement"

Invoke-Step "HKCU : SilentInstalledAppsEnabled = 0 (stop reinstall silencieux)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SilentInstalledAppsEnabled' -Value 0
} -Test {
    Test-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SilentInstalledAppsEnabled' -Expected 0
}

Invoke-Step "HKCU : suggestions Settings désactivées (SubscribedContent-338389)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SubscribedContent-338389Enabled' -Value 0
}

Invoke-Step "HKCU : suggestions menu Démarrer désactivées (SystemPaneSuggestionsEnabled)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SystemPaneSuggestionsEnabled' -Value 0
}

Invoke-Step "HKCU : tips/welcome désactivés (SubscribedContent-338388, 310093, 353698)" {
    foreach ($key in @('SubscribedContent-338388Enabled','SubscribedContent-310093Enabled','SubscribedContent-353698Enabled')) {
        Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name $key -Value 0
    }
}

Invoke-Step "HKCU : RestrictImplicitInkCollection = 1 (anti collecte écriture)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\InputPersonalization' -Name 'RestrictImplicitInkCollection' -Value 1
}

Invoke-Step "HKCU : RestrictImplicitTextCollection = 1 (anti collecte texte)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\InputPersonalization' -Name 'RestrictImplicitTextCollection' -Value 1
}

Invoke-Step "HKCU : AdvertisingID utilisateur = 0" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo' -Name 'Enabled' -Value 0
}

# ============================================================================
# SECTION 8 : Bloatware Microsoft Store
# ============================================================================
if (-not $SkipBloatware) {
    Write-Log "--- Section 8/8 : Désinstallation bloatware Store ---" -Level INFO

    # Patterns à matcher dans le PackageFullName ou Name
    $bloatPatterns = @(
        '*JimmyLin*',           # IPTVFluent
        '*5319275A*',           # publisher GUID louche
        '*Clipchamp*',          # éditeur vidéo MS (préinstall)
        '*AppleMusicWin*',
        '*SpotifyAB*',
        '*Spotify*',
        '*DolbyLaboratories.DolbyAccess*',
        '*Microsoft.BingNews*',
        '*Microsoft.BingWeather*',
        '*Microsoft.GetHelp*',
        '*Microsoft.Getstarted*',
        '*Microsoft.MicrosoftSolitaireCollection*',
        '*Microsoft.MixedReality.Portal*',
        '*Microsoft.People*',
        '*Microsoft.SkypeApp*',
        '*Microsoft.WindowsFeedbackHub*',
        '*Microsoft.YourPhone*',
        '*Microsoft.ZuneMusic*',
        '*Microsoft.ZuneVideo*',
        '*Disney*',
        '*TikTok*',
        '*Facebook*',
        '*Instagram*',
        '*Twitter*',
        '*LinkedInforWindows*',
        '*CandyCrush*',
        '*Netflix*'
    )

    foreach ($pattern in $bloatPatterns) {
        $packages = Get-AppxPackage -Name $pattern -AllUsers -ErrorAction SilentlyContinue
        if (-not $packages) { continue }
        foreach ($pkg in $packages) {
            Invoke-Step "Bloat : désinstaller $($pkg.Name)" {
                Remove-AppxPackage -Package $pkg.PackageFullName -AllUsers -ErrorAction Stop
            }
        }

        # Provisionned packages : empêche réinstallation à la création de nouveaux comptes
        $provisioned = Get-AppxProvisionedPackage -Online | Where-Object { $_.DisplayName -like $pattern.Trim('*') -or $_.PackageName -like $pattern }
        foreach ($prov in $provisioned) {
            Invoke-Step "Bloat : retirer provisioned $($prov.DisplayName)" {
                Remove-AppxProvisionedPackage -Online -PackageName $prov.PackageName -ErrorAction Stop | Out-Null
            }
        }
    }
} else {
    Write-Log "Section 8 : skippée (-SkipBloatware)" -Level SKIP
}

# ============================================================================
# Synthèse
# ============================================================================
Write-Log "" -Level INFO
Write-Log "=== SYNTHÈSE ===" -Level INFO
Write-Log "Appliqué : $script:Applied" -Level OK
Write-Log "Déjà conforme : $script:Skipped" -Level INFO
Write-Log "Avertissements : $script:Warnings" -Level WARN
Write-Log "Erreurs : $script:Errors" -Level $(if ($script:Errors -gt 0) { 'ERROR' } else { 'INFO' })
Write-Log "Log complet : $LogPath"
Write-Log ""
Write-Log "REDÉMARRAGE RECOMMANDÉ pour appliquer toutes les modifs (NetBIOS, Fast Startup, etc.)" -Level WARN

if ($AsrAuditMode) {
    Write-Log ""
    Write-Log "Mode AUDIT actif pour ASR. Vérifie les events après quelques jours :" -Level INFO
    Write-Log "  Get-WinEvent -LogName 'Microsoft-Windows-Windows Defender/Operational' | Where-Object {`$_.Id -in 1121,1122,1125,1126}" -Level INFO
    Write-Log "Quand tout est OK, relance ce script SANS -AsrAuditMode pour basculer en mode bloquant." -Level INFO
}
