<#
.SYNOPSIS
    Hardening Windows 11 - Configuration de securite complete et reproductible.

.DESCRIPTION
    Applique une baseline de securite Windows 11 :
      - Defender : Real-time, Behavior, IOAV, NIS, Tamper Protection, PUA, CFA, Network Protection, ASR rules
      - Firewall : 3 profils ON, blocage SMB entrant sur Public
      - Comptes : Built-in Admin / Guest / WsiAccount desactives
      - UAC : niveau 5 sur Secure Desktop
      - RDP : desactive
      - Hibernation : desactivee, Fast Startup : OFF
      - Privacy : AdvertisingID OFF, OnlineSpeech OFF, Telemetry minimum, anti-bloat HKCU
      - Reseau : LLMNR off, NetBIOS off, NTLMv2 only, SMBv1 off
      - Bloatware : suppression apps Store inutiles

    Le script est idempotent : il peut etre relance. Backup registre automatique.

.PARAMETER LogPath
    Chemin du log. Defaut : C:\ProgramData\Harden-Win11\harden-<date>.log

.PARAMETER SkipBloatware
    Ne pas desinstaller le bloatware Store.

.PARAMETER AsrAuditMode
    Active les regles ASR en mode Audit (loggent sans bloquer).

.PARAMETER DryRun
    N'applique rien, affiche seulement ce qui serait fait.

.PARAMETER Quiet
    Mode silencieux : pas de menu interactif, pas de pause finale.

.EXAMPLE
    .\Harden-Win11.ps1
    # Lance avec menu interactif

.EXAMPLE
    .\Harden-Win11.ps1 -DryRun
    # Test a blanc, aucune modification

.EXAMPLE
    .\Harden-Win11.ps1 -AsrAuditMode
    # 1er passage recommande : tout applique sauf ASR en mode log

.EXAMPLE
    .\Harden-Win11.ps1 -Quiet
    # Passage final automatise

.NOTES
    Doit tourner en admin. Section HKCU : utilisateur courant uniquement.
#>

[CmdletBinding()]
param(
    [string]$LogPath = "C:\ProgramData\Harden-Win11\harden-$(Get-Date -Format 'yyyyMMdd-HHmmss').log",
    [switch]$SkipBloatware,
    [switch]$AsrAuditMode,
    [switch]$DryRun,
    [switch]$Quiet
)

# ============================================================================
# Encodage : force UTF-8 pour les accents en console
# ============================================================================
try {
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    $OutputEncoding = [System.Text.Encoding]::UTF8
    chcp 65001 | Out-Null
} catch { }

# ============================================================================
# Verifications, etat global
# ============================================================================
$ErrorActionPreference = 'Stop'
$script:Errors = 0
$script:Warnings = 0
$script:Applied = 0
$script:Skipped = 0
$script:DryActions = 0
$script:StartTime = Get-Date

# Verif admin avec message d'aide explicite
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host ""
    Write-Host "  [X] ERREUR : ce script doit etre lance en tant qu'administrateur." -ForegroundColor Red
    Write-Host ""
    Write-Host "  Comment faire :" -ForegroundColor Yellow
    Write-Host "    1. Ferme cette fenetre"
    Write-Host "    2. Tape 'PowerShell' dans le menu Demarrer"
    Write-Host "    3. Clic droit sur 'Windows PowerShell' -> 'Executer en tant qu'administrateur'"
    Write-Host "    4. Accepte le prompt UAC"
    Write-Host "    5. cd vers le dossier du script et relance"
    Write-Host ""
    if (-not $Quiet) { Read-Host "Appuie sur Entree pour fermer" }
    exit 1
}

# Verif version Windows
$os = Get-CimInstance Win32_OperatingSystem
if ($os.BuildNumber -lt 22000) {
    Write-Host "ATTENTION : ce script vise Windows 11 (build 22000+). Build detectee : $($os.BuildNumber)" -ForegroundColor Yellow
    if (-not $Quiet) {
        $continue = Read-Host "Continuer quand meme ? (o/N)"
        if ($continue -ne 'o') { exit 0 }
    }
}

# Preparer le repertoire de log
$logDir = Split-Path $LogPath -Parent
if (-not (Test-Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
}

# ============================================================================
# Affichage : header, menu, log helper
# ============================================================================
function Show-Banner {
    Write-Host ""
    Write-Host "  ====================================================================" -ForegroundColor Cyan
    Write-Host "                                                                      " -ForegroundColor Cyan
    Write-Host "      HARDEN-WIN11   |   Baseline de securite Windows 11              " -ForegroundColor Cyan
    Write-Host "                                                                      " -ForegroundColor Cyan
    Write-Host "  ====================================================================" -ForegroundColor Cyan
    Write-Host ""
}

function Show-Context {
    $modeLabel = if     ($DryRun)        { "DRY-RUN (aucune modification)" }
                 elseif ($AsrAuditMode)  { "AUDIT ASR (regles ASR loggees, pas bloquees)" }
                 else                    { "APPLICATION REELLE" }
    $modeColor = if     ($DryRun -or $AsrAuditMode) { 'Yellow' } else { 'Green' }

    Write-Host "  Hote      : " -NoNewline -ForegroundColor Gray; Write-Host $env:COMPUTERNAME -ForegroundColor White
    Write-Host "  User      : " -NoNewline -ForegroundColor Gray; Write-Host $env:USERNAME     -ForegroundColor White
    Write-Host "  Build     : " -NoNewline -ForegroundColor Gray; Write-Host $os.BuildNumber   -ForegroundColor White
    Write-Host "  Mode      : " -NoNewline -ForegroundColor Gray; Write-Host $modeLabel        -ForegroundColor $modeColor
    Write-Host "  Log       : " -NoNewline -ForegroundColor Gray; Write-Host $LogPath          -ForegroundColor White
    Write-Host ""
}

function Show-Menu {
    Write-Host "  Aucun mode specifie. Que veux-tu faire ?" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "    [1] " -NoNewline -ForegroundColor White
    Write-Host "Test a blanc (DryRun)" -NoNewline -ForegroundColor Cyan
    Write-Host "         - aucune modification, juste un rapport" -ForegroundColor Gray

    Write-Host "    [2] " -NoNewline -ForegroundColor White
    Write-Host "Mode AUDIT ASR" -NoNewline -ForegroundColor Yellow
    Write-Host " (recommande 1er) - applique tout, ASR en mode log" -ForegroundColor Gray

    Write-Host "    [3] " -NoNewline -ForegroundColor White
    Write-Host "Application complete" -NoNewline -ForegroundColor Green
    Write-Host " (FINAL)    - tout applique, ASR bloquant" -ForegroundColor Gray

    Write-Host "    [Q] " -NoNewline -ForegroundColor White
    Write-Host "Quitter" -ForegroundColor Gray
    Write-Host ""

    $choice = Read-Host "  Ton choix"
    switch ($choice.ToUpper()) {
        '1' { $script:DryRun = $true;          return $true }
        '2' { $script:AsrAuditMode = $true;    return $true }
        '3' {
            Write-Host ""
            Write-Host "  [!] ATTENTION : tu vas appliquer les regles ASR en mode BLOQUANT." -ForegroundColor Yellow
            Write-Host "  Si tu n'as jamais lance ce script, le mode AUDIT (option 2) est recommande." -ForegroundColor Yellow
            $confirm = Read-Host "  Continuer quand meme ? (o/N)"
            if ($confirm -eq 'o') { return $true } else { return $false }
        }
        'Q' { return $false }
        ''  { Write-Host "  Choix vide." -ForegroundColor Red; return Show-Menu }
        default {
            Write-Host "  Choix invalide." -ForegroundColor Red
            return Show-Menu
        }
    }
}

# Si interactif et aucun mode choisi -> afficher banner + menu
$noModeSpecified = -not $DryRun -and -not $AsrAuditMode
if (-not $Quiet -and $noModeSpecified) {
    Clear-Host
    Show-Banner
    if (-not (Show-Menu)) { exit 0 }
    Show-Banner
} else {
    Show-Banner
}

Show-Context

function Write-Log {
    param(
        [string]$Message,
        [ValidateSet('INFO','OK','WARN','ERROR','SKIP','DRY')]
        [string]$Level = 'INFO'
    )
    $ts = Get-Date -Format 'HH:mm:ss'
    $line = "[$ts][$Level] $Message"

    Add-Content -Path $LogPath -Value $line -Encoding UTF8

    $icon = switch ($Level) {
        'OK'    { '[+]' }
        'WARN'  { '[!]' }
        'ERROR' { '[X]' }
        'SKIP'  { '[=]' }
        'DRY'   { '[?]' }
        default { '[i]' }
    }
    $color = switch ($Level) {
        'OK'    { 'Green' }
        'WARN'  { 'Yellow' }
        'ERROR' { 'Red' }
        'SKIP'  { 'DarkGray' }
        'DRY'   { 'Cyan' }
        default { 'White' }
    }
    Write-Host "  $icon $Message" -ForegroundColor $color
}

function Write-Section {
    param([string]$Number, [string]$Title)
    Write-Host ""
    Write-Host "  --- Section $Number : $Title ---" -ForegroundColor Magenta
    Add-Content -Path $LogPath -Value "" -Encoding UTF8
    Add-Content -Path $LogPath -Value "===== Section $Number : $Title =====" -Encoding UTF8
}

function Invoke-Step {
    param(
        [string]$Description,
        [scriptblock]$Action,
        [scriptblock]$Test
    )

    if ($Test) {
        try {
            if (& $Test) {
                Write-Log "$Description -> deja conforme" -Level SKIP
                $script:Skipped++
                return
            }
        } catch { }
    }

    if ($DryRun) {
        Write-Log "$Description -> serait applique" -Level DRY
        $script:DryActions++
        return
    }

    try {
        & $Action
        Write-Log "$Description -> applique" -Level OK
        $script:Applied++
    } catch {
        Write-Log "$Description -> ECHEC : $($_.Exception.Message)" -Level ERROR
        $script:Errors++
    }
}

function Set-RegValue {
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

function Backup-Registry {
    $backupDir = Join-Path $logDir "regbackup-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    Write-Log "Backup registre : $backupDir" -Level INFO

    $hives = @{
        'HKLM-SOFTWARE-Policies' = 'HKLM\SOFTWARE\Policies'
        'HKLM-System-CCS-Lsa'    = 'HKLM\SYSTEM\CurrentControlSet\Control\Lsa'
        'HKCU-CDM'               = 'HKCU\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'
    }
    foreach ($name in $hives.Keys) {
        $file = Join-Path $backupDir "$name.reg"
        & reg.exe export $hives[$name] $file /y 2>&1 | Out-Null
    }
    Write-Log "Backup registre termine" -Level OK
}

# ============================================================================
# Demarrage
# ============================================================================
if (-not $DryRun) { Backup-Registry }

# ============================================================================
# SECTION 1 : Microsoft Defender
# ============================================================================
Write-Section "1/8" "Microsoft Defender"

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

Invoke-Step "Defender : IOAV (analyse pieces jointes / DL)" {
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

Invoke-Step "Defender : Sample submission (envoi auto echantillons surs)" {
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

Invoke-Step "Defender : signatures a jour" {
    Update-MpSignature -ErrorAction Stop
}

# Tamper Protection : non modifiable par script (par design Microsoft)
$tp = (Get-MpComputerStatus).IsTamperProtected
if ($tp) {
    Write-Log "Defender : Tamper Protection active -> OK" -Level OK
} else {
    Write-Log "Defender : Tamper Protection INACTIVE -> activer manuellement dans Windows Security > Virus & threat protection > Manage settings" -Level WARN
    $script:Warnings++
}

# ============================================================================
# SECTION 2 : ASR Rules
# ============================================================================
Write-Section "2/8" "ASR rules (Attack Surface Reduction)"

# 1 = Block, 2 = Audit, 6 = Warn
$asrAction = if ($AsrAuditMode) { 2 } else { 1 }
$asrModeLabel = if ($AsrAuditMode) { "AUDIT (logge sans bloquer)" } else { "BLOCK (bloque effectivement)" }
Write-Log "Mode ASR : $asrModeLabel" -Level INFO

$asrRules = [ordered]@{
    'D4F940AB-401B-4EFC-AADC-AD5F3C50688A' = 'Block all Office apps from creating child processes'
    '3B576869-A4EC-4529-8536-B80A7769E899' = 'Block Office apps from creating executable content'
    '75668C1F-73B5-4CF0-BB93-3ECF5CB7CC84' = 'Block Office apps from injecting code into other processes'
    '26190899-1602-49E8-8B27-EB1D0A1CE869' = 'Block Office communication apps from creating child processes'
    '92E97FA1-2EDF-4476-BDD6-9DD0B4DDDC7B' = 'Block Win32 API calls from Office macros'
    'BE9BA2D9-53EA-4CDC-84E5-9B1EEEE46550' = 'Block executable content from email/webmail'
    '5BEB7EFE-FD9A-4556-801D-275E5FFC04CC' = 'Block execution of potentially obfuscated scripts'
    'D3E037E1-3EB8-44C8-A917-57927947596D' = 'Block JS/VBS from launching downloaded executable'
    '9E6C4E1F-7D60-472F-BA1A-A39EF669E4B2' = 'Block credential stealing from LSASS'
    'B2B3F03D-6A65-4F7B-A9C7-1C7EF74A9BA4' = 'Block untrusted/unsigned processes from USB'
    'E6DB77E5-3DF2-4CF1-B95A-636979351E5B' = 'Block persistence through WMI event subscription'
    'D1E49AAC-8F56-4280-B9BA-993A6D77406C' = 'Block process creations from PSExec/WMI commands'
    '01443614-CD74-433A-B99E-2ECDC07BFC25' = 'Block executables unless prevalent/aged/trusted'
    'C1DB55AB-C21A-4637-BB3F-A12568109D35' = 'Use advanced protection against ransomware'
    '7674BA52-37EB-4A4F-A9A1-F0F9A1619A2C' = 'Block Adobe Reader from creating child processes'
    '56A863A9-875E-4185-98A7-B882C64B5CE5' = 'Block abuse of exploited vulnerable signed drivers'
    '33DDEDF1-C6E0-47CB-833E-DE6133960387' = 'Block rebooting machine in Safe Mode'
    'C0033C00-D16D-4114-A5A0-DC9B3A7D2CEB' = 'Block use of copied/impersonated system tools'
    'A8F5898E-1DC8-49A9-9878-85004B8A61E6' = 'Block Webshell creation for Servers'
}

foreach ($guid in $asrRules.Keys) {
    $name = $asrRules[$guid]
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
Write-Section "3/8" "Firewall"

Invoke-Step "Firewall : profil Domain active, blocage entrant par defaut" {
    Set-NetFirewallProfile -Profile Domain -Enabled True -DefaultInboundAction Block -DefaultOutboundAction Allow
}
Invoke-Step "Firewall : profil Private active, blocage entrant par defaut" {
    Set-NetFirewallProfile -Profile Private -Enabled True -DefaultInboundAction Block -DefaultOutboundAction Allow
}
Invoke-Step "Firewall : profil Public active, blocage entrant par defaut" {
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
    foreach ($name in @("Block NetBIOS UDP Inbound (Public) [Hardening]", "Block NetBIOS TCP Inbound (Public) [Hardening]")) {
        $existing = Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue
        if ($existing) { Remove-NetFirewallRule -DisplayName $name }
    }
    New-NetFirewallRule -DisplayName "Block NetBIOS UDP Inbound (Public) [Hardening]" `
        -Direction Inbound -Protocol UDP -LocalPort 137,138 -Action Block -Profile Public | Out-Null
    New-NetFirewallRule -DisplayName "Block NetBIOS TCP Inbound (Public) [Hardening]" `
        -Direction Inbound -Protocol TCP -LocalPort 139 -Action Block -Profile Public | Out-Null
} -Test {
    $u = Get-NetFirewallRule -DisplayName "Block NetBIOS UDP Inbound (Public) [Hardening]" -ErrorAction SilentlyContinue
    $t = Get-NetFirewallRule -DisplayName "Block NetBIOS TCP Inbound (Public) [Hardening]" -ErrorAction SilentlyContinue
    $u -and $t
}

# ============================================================================
# SECTION 4 : Comptes locaux
# ============================================================================
Write-Section "4/8" "Comptes locaux"

$accountsToDisable = @('Administrator', 'Guest', 'WsiAccount', 'DefaultAccount')
foreach ($acct in $accountsToDisable) {
    Invoke-Step "Compte : desactiver '$acct'" {
        $u = Get-LocalUser -Name $acct -ErrorAction SilentlyContinue
        if ($u) { Disable-LocalUser -Name $acct }
    } -Test {
        $u = Get-LocalUser -Name $acct -ErrorAction SilentlyContinue
        (-not $u) -or (-not $u.Enabled)
    }
}

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
# SECTION 5 : UAC, RDP, Power
# ============================================================================
Write-Section "5/8" "UAC, RDP, Power"

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

Invoke-Step "UAC : refuser elevation pour comptes standard (ConsentPromptBehaviorUser=0)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorUser' -Value 0
} -Test {
    Test-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorUser' -Expected 0
}

Invoke-Step "RDP : desactive (fDenyTSConnections=1)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server' -Name 'fDenyTSConnections' -Value 1
} -Test {
    Test-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server' -Name 'fDenyTSConnections' -Expected 1
}

Invoke-Step "RDP : regle firewall desactivee" {
    Disable-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue
}

Invoke-Step "Hibernation : desactivee (libere ~8-12 GB)" {
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
# SECTION 6 : Hardening protocoles reseau
# ============================================================================
Write-Section "6/8" "Hardening protocoles reseau (LLMNR, NetBIOS, NTLM, SMB)"

Invoke-Step "LLMNR : desactive (EnableMulticast=0)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient' -Name 'EnableMulticast' -Value 0
} -Test {
    Test-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient' -Name 'EnableMulticast' -Expected 0
}

Invoke-Step "mDNS : desactive (Bonjour-like, vecteur similaire a LLMNR)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters' -Name 'EnableMDNS' -Value 0
} -Test {
    Test-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters' -Name 'EnableMDNS' -Expected 0
}

Invoke-Step "NetBIOS over TCP/IP : desactive sur tous les adaptateurs" {
    $adapters = Get-CimInstance -ClassName Win32_NetworkAdapterConfiguration -Filter 'IPEnabled=TRUE'
    foreach ($a in $adapters) {
        $null = Invoke-CimMethod -InputObject $a -MethodName SetTcpipNetbios -Arguments @{TcpipNetbiosOptions = [uint32]2}
    }
    Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Services\NetBT\Parameters\Interfaces' | ForEach-Object {
        Set-ItemProperty -Path $_.PSPath -Name 'NetbiosOptions' -Value 2 -Force -ErrorAction SilentlyContinue
    }
}

Invoke-Step "WPAD : desactive (anti-poisoning proxy auto-discovery)" {
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

Invoke-Step "SMB : desactiver SMBv1 (legacy, vulnerable EternalBlue)" {
    Set-SmbServerConfiguration -EnableSMB1Protocol $false -Force -ErrorAction SilentlyContinue
    Disable-WindowsOptionalFeature -Online -FeatureName SMB1Protocol -NoRestart -ErrorAction SilentlyContinue | Out-Null
}

Invoke-Step "SMB : refuser auth guest cote client (AllowInsecureGuestAuth=0)" {
    Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' -Name 'AllowInsecureGuestAuth' -Value 0
}

# ============================================================================
# SECTION 7 : Privacy & Telemetry
# ============================================================================
Write-Section "7/8" "Privacy & Telemetry"

Invoke-Step "Telemetry : niveau 1 (Required only - minimum sur Win11 Home)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection' -Name 'AllowTelemetry' -Value 1
}

Invoke-Step "AdvertisingID : desactive (machine-wide)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\AdvertisingInfo' -Name 'DisabledByGroupPolicy' -Value 1
}

Invoke-Step "Online Speech : desactive" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\InputPersonalization' -Name 'AllowInputPersonalization' -Value 0
}

Invoke-Step "Activity History : desactive (bloque envoi cloud)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' -Name 'EnableActivityFeed' -Value 0
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' -Name 'PublishUserActivities' -Value 0
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' -Name 'UploadUserActivities' -Value 0
}

Invoke-Step "Cortana : desactivee" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Windows Search' -Name 'AllowCortana' -Value 0
}

Invoke-Step "Recall : desactive (preventif si futur push)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsAI' -Name 'DisableAIDataAnalysis' -Value 1
}

Invoke-Step "Consumer features : desactivees (anti-bloat machine-wide)" {
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent' -Name 'DisableWindowsConsumerFeatures' -Value 1
    Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent' -Name 'DisableConsumerAccountStateContent' -Value 1
}

# --- HKCU : utilisateur courant uniquement ---
Write-Log "Privacy HKCU : modifications appliquees a l'utilisateur courant ($env:USERNAME) uniquement" -Level INFO

Invoke-Step "HKCU : SilentInstalledAppsEnabled = 0 (stop reinstall silencieux)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SilentInstalledAppsEnabled' -Value 0
} -Test {
    Test-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SilentInstalledAppsEnabled' -Expected 0
}

Invoke-Step "HKCU : suggestions Settings desactivees (SubscribedContent-338389)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SubscribedContent-338389Enabled' -Value 0
}

Invoke-Step "HKCU : suggestions menu Demarrer desactivees (SystemPaneSuggestionsEnabled)" {
    Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name 'SystemPaneSuggestionsEnabled' -Value 0
}

Invoke-Step "HKCU : tips/welcome desactives (SubscribedContent-338388, 310093, 353698)" {
    foreach ($key in @('SubscribedContent-338388Enabled','SubscribedContent-310093Enabled','SubscribedContent-353698Enabled')) {
        Set-RegValue -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' -Name $key -Value 0
    }
}

Invoke-Step "HKCU : RestrictImplicitInkCollection = 1 (anti collecte ecriture)" {
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
    Write-Section "8/8" "Desinstallation bloatware Store"

    $bloatPatterns = @(
        '*JimmyLin*',
        '*5319275A*',
        '*Clipchamp*',
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

    # Dedup
    $alreadyHandled = @{}

    foreach ($pattern in $bloatPatterns) {
        $packages = Get-AppxPackage -Name $pattern -AllUsers -ErrorAction SilentlyContinue
        if ($packages) {
            foreach ($pkg in $packages) {
                if ($alreadyHandled.ContainsKey($pkg.PackageFullName)) { continue }
                $alreadyHandled[$pkg.PackageFullName] = $true
                Invoke-Step "Bloat : desinstaller $($pkg.Name)" {
                    Remove-AppxPackage -Package $pkg.PackageFullName -AllUsers -ErrorAction Stop
                }
            }
        }

        $provisioned = Get-AppxProvisionedPackage -Online | Where-Object {
            $_.DisplayName -like $pattern.Trim('*') -or $_.PackageName -like $pattern
        }
        foreach ($prov in $provisioned) {
            $key = "prov:$($prov.PackageName)"
            if ($alreadyHandled.ContainsKey($key)) { continue }
            $alreadyHandled[$key] = $true
            Invoke-Step "Bloat : retirer provisioned $($prov.DisplayName)" {
                Remove-AppxProvisionedPackage -Online -PackageName $prov.PackageName -ErrorAction Stop | Out-Null
            }
        }
    }
} else {
    Write-Section "8/8" "Bloatware Store"
    Write-Log "Section 8 : skippee (-SkipBloatware)" -Level SKIP
}

# ============================================================================
# Synthese finale
# ============================================================================
$elapsed = (Get-Date) - $script:StartTime
$total = $script:Applied + $script:Skipped + $script:DryActions + $script:Errors

Write-Host ""
Write-Host ""
Write-Host "  ====================================================================" -ForegroundColor Cyan
Write-Host "                              SYNTHESE                                " -ForegroundColor Cyan
Write-Host "  ====================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Duree d'execution    : $([math]::Round($elapsed.TotalSeconds, 1)) secondes" -ForegroundColor Gray
Write-Host "  Total operations     : $total" -ForegroundColor Gray
Write-Host ""

if ($DryRun) {
    Write-Host "  [?] $script:DryActions actions seraient appliquees" -ForegroundColor Cyan
    Write-Host "  [=] $script:Skipped items deja conformes" -ForegroundColor DarkGray
    if ($script:Warnings -gt 0) { Write-Host "  [!] $script:Warnings avertissement(s)" -ForegroundColor Yellow }
} else {
    Write-Host "  [+] $script:Applied applique(s)" -ForegroundColor Green
    Write-Host "  [=] $script:Skipped deja conforme(s)" -ForegroundColor DarkGray
    if ($script:Warnings -gt 0) { Write-Host "  [!] $script:Warnings avertissement(s)" -ForegroundColor Yellow }
    if ($script:Errors -gt 0)   { Write-Host "  [X] $script:Errors erreur(s)" -ForegroundColor Red }
}

# Score de conformite
if ($total -gt 0) {
    $compliantBefore = $script:Skipped
    $compliantAfter = if ($DryRun) {
        $script:Skipped + $script:DryActions
    } else {
        $script:Skipped + $script:Applied
    }
    $pctBefore = [math]::Round(($compliantBefore / $total) * 100, 1)
    $pctAfter = [math]::Round(($compliantAfter / $total) * 100, 1)

    Write-Host ""
    $colorBefore = if ($pctBefore -ge 80) { 'Green' } elseif ($pctBefore -ge 50) { 'Yellow' } else { 'Red' }
    Write-Host "  Score de conformite avant : $pctBefore %" -ForegroundColor $colorBefore
    if ($DryRun) {
        Write-Host "  Score de conformite apres : $pctAfter % (si applique)" -ForegroundColor Cyan
    } else {
        $colorAfter = if ($pctAfter -ge 95) { 'Green' } elseif ($pctAfter -ge 80) { 'Yellow' } else { 'Red' }
        Write-Host "  Score de conformite apres : $pctAfter %" -ForegroundColor $colorAfter
    }
}

Write-Host ""
Write-Host "  Log complet : $LogPath" -ForegroundColor Gray
Write-Host ""

if ($DryRun) {
    Write-Host "  >>> Rien n'a ete modifie. Pour appliquer en mode audit ASR :" -ForegroundColor Cyan
    Write-Host "      .\Harden-Win11.ps1 -AsrAuditMode" -ForegroundColor White
} elseif ($AsrAuditMode) {
    Write-Host "  >>> Mode AUDIT ASR : verifie les events dans 2-3 jours :" -ForegroundColor Yellow
    Write-Host "      Get-WinEvent -LogName 'Microsoft-Windows-Windows Defender/Operational' |" -ForegroundColor Gray
    Write-Host "        Where-Object {`$_.Id -in 1121,1122,1125,1126} |" -ForegroundColor Gray
    Write-Host "        Select TimeCreated, Id, Message -First 50" -ForegroundColor Gray
    Write-Host ""
    Write-Host "      Si rien de legitime n'est bloque, relance sans -AsrAuditMode" -ForegroundColor Yellow
} else {
    Write-Host "  >>> REDEMARRAGE RECOMMANDE pour appliquer toutes les modifs" -ForegroundColor Yellow
    Write-Host "      (NetBIOS, Fast Startup, services reseau)" -ForegroundColor Yellow
}
Write-Host ""

if (-not $Quiet) {
    Read-Host "Appuie sur Entree pour fermer"
}
