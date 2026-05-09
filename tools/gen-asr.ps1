# gen-asr.ps1
# Génère les 19 paires action/test/undo + 1 manifest pour les règles ASR Defender.
# Usage : powershell -File tools/gen-asr.ps1
# Ce script est utilisé une fois pour bootstrap puis les fichiers générés sont
# édités à la main. Le script lui-même n'est pas committé (tools/ gitignored).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$asrRules = @(
    @{ guid = 'D4F940AB-401B-4EFC-AADC-AD5F3C50688A'; slug = 'block_office_child_processes';     title = "Block Office apps from creating child processes" }
    @{ guid = '3B576869-A4EC-4529-8536-B80A7769E899'; slug = 'block_office_executable_content';  title = "Block Office apps from creating executable content" }
    @{ guid = '75668C1F-73B5-4CF0-BB93-3ECF5CB7CC84'; slug = 'block_office_code_injection';      title = "Block Office apps from injecting code into other processes" }
    @{ guid = '26190899-1602-49E8-8B27-EB1D0A1CE869'; slug = 'block_office_comm_child_processes';title = "Block Office communication apps from creating child processes" }
    @{ guid = '92E97FA1-2EDF-4476-BDD6-9DD0B4DDDC7B'; slug = 'block_win32_api_office_macros';    title = "Block Win32 API calls from Office macros" }
    @{ guid = 'BE9BA2D9-53EA-4CDC-84E5-9B1EEEE46550'; slug = 'block_email_executable_content';   title = "Block executable content from email/webmail" }
    @{ guid = '5BEB7EFE-FD9A-4556-801D-275E5FFC04CC'; slug = 'block_obfuscated_scripts';         title = "Block execution of potentially obfuscated scripts" }
    @{ guid = 'D3E037E1-3EB8-44C8-A917-57927947596D'; slug = 'block_js_vbs_launch';              title = "Block JS/VBS from launching downloaded executable" }
    @{ guid = '9E6C4E1F-7D60-472F-BA1A-A39EF669E4B2'; slug = 'block_lsass_credential_theft';     title = "Block credential stealing from LSASS" }
    @{ guid = 'B2B3F03D-6A65-4F7B-A9C7-1C7EF74A9BA4'; slug = 'block_unsigned_usb';               title = "Block untrusted/unsigned processes from USB" }
    @{ guid = 'E6DB77E5-3DF2-4CF1-B95A-636979351E5B'; slug = 'block_wmi_persistence';            title = "Block persistence through WMI event subscription" }
    @{ guid = 'D1E49AAC-8F56-4280-B9BA-993A6D77406C'; slug = 'block_psexec_wmi';                 title = "Block process creations from PSExec/WMI commands" }
    @{ guid = '01443614-CD74-433A-B99E-2ECDC07BFC25'; slug = 'block_unprevalent_executables';    title = "Block executables unless prevalent/aged/trusted" }
    @{ guid = 'C1DB55AB-C21A-4637-BB3F-A12568109D35'; slug = 'advanced_ransomware_protection';   title = "Use advanced protection against ransomware" }
    @{ guid = '7674BA52-37EB-4A4F-A9A1-F0F9A1619A2C'; slug = 'block_adobe_reader_child';         title = "Block Adobe Reader from creating child processes" }
    @{ guid = '56A863A9-875E-4185-98A7-B882C64B5CE5'; slug = 'block_vulnerable_drivers';         title = "Block abuse of exploited vulnerable signed drivers" }
    @{ guid = '33DDEDF1-C6E0-47CB-833E-DE6133960387'; slug = 'block_safe_mode_reboot';           title = "Block rebooting machine in Safe Mode" }
    @{ guid = 'C0033C00-D16D-4114-A5A0-DC9B3A7D2CEB'; slug = 'block_impersonated_tools';         title = "Block use of copied/impersonated system tools" }
    @{ guid = 'A8F5898E-1DC8-49A9-9878-85004B8A61E6'; slug = 'block_webshell_servers';           title = "Block Webshell creation for Servers" }
)

$root = Split-Path -Parent $PSScriptRoot
$asrDir = Join-Path $root 'engine\actions\asr'
$manifestPath = Join-Path $root 'manifests\08-asr.yaml'
New-Item -ItemType Directory -Force -Path $asrDir | Out-Null

# ---------- Templates ----------

$actionTemplate = @'
# {SLUG}.action.ps1
# ASR : {TITLE}
# GUID : {GUID}
# Action choisie :
#   - 1 (Block) par defaut
#   - 2 (Audit) si l'env var HARDEN_ASR_MODE=audit est positionnee (le
#     runner Go la passe quand l'utilisateur active le mode audit GUI).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$guid = '{GUID}'
$action = if ($env:HARDEN_ASR_MODE -eq 'audit') { 2 } else { 1 }

function Get-AsrAction([string]$g) {
    $pref = Get-MpPreference
    $ids = @($pref.AttackSurfaceReductionRules_Ids)
    $acts = @($pref.AttackSurfaceReductionRules_Actions)
    for ($i = 0; $i -lt $ids.Count; $i++) {
        if ($ids[$i] -ieq $g) { return [int]$acts[$i] }
    }
    return $null
}

$beforeAction = Get-AsrAction $guid
$before = @{ AsrAction = $beforeAction }

Add-MpPreference -AttackSurfaceReductionRules_Ids $guid -AttackSurfaceReductionRules_Actions $action -ErrorAction Stop

$afterAction = Get-AsrAction $guid
$after = @{ AsrAction = $afterAction }

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
'@

$testTemplate = @'
# {SLUG}.test.ps1
# Conforme = la règle ASR {GUID} est en mode Block (1).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$guid = '{GUID}'
$expected = 1

$pref = Get-MpPreference
$ids = @($pref.AttackSurfaceReductionRules_Ids)
$acts = @($pref.AttackSurfaceReductionRules_Actions)

$current = $null
for ($i = 0; $i -lt $ids.Count; $i++) {
    if ($ids[$i] -ieq $guid) { $current = [int]$acts[$i]; break }
}

$compliant = ($current -eq $expected)
$names = @{ 0 = 'NotConfigured'; 1 = 'Block'; 2 = 'Audit'; 6 = 'Warn' }
$mode = if ($null -ne $current -and $names.ContainsKey($current)) { $names[$current] } elseif ($null -ne $current) { "Unknown($current)" } else { 'NotPresent' }

@{
    compliant = $compliant
    current   = @{
        AsrRule    = $guid
        AsrAction  = $current
        AsrMode    = $mode
    }
} | ConvertTo-Json -Compress -Depth 10
'@

$undoTemplate = @'
# {SLUG}.undo.ps1
# Restaure l'état AsrAction de la règle {GUID} selon 'before'.
# Input : { "AsrAction": <int|null> }
# Si AsrAction était null (règle absente), on Remove-MpPreference.
# Sinon on ré-Add-MpPreference avec la valeur précédente.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

$guid = '{GUID}'

# Toujours retirer d'abord (pour ne pas accumuler de doublons côté Defender)
Remove-MpPreference -AttackSurfaceReductionRules_Ids $guid -ErrorAction SilentlyContinue

if ($null -ne $state.AsrAction) {
    Add-MpPreference -AttackSurfaceReductionRules_Ids $guid -AttackSurfaceReductionRules_Actions ([int]$state.AsrAction) -ErrorAction Stop
}

@{ ok = $true } | ConvertTo-Json -Compress
'@

# ---------- Génération des PS files ----------

foreach ($rule in $asrRules) {
    $slug = $rule.slug
    $guid = $rule.guid
    $title = $rule.title

    foreach ($kind in @('action', 'test', 'undo')) {
        $template = switch ($kind) {
            'action' { $actionTemplate }
            'test'   { $testTemplate }
            'undo'   { $undoTemplate }
        }
        $content = $template.Replace('{SLUG}', $slug).Replace('{GUID}', $guid).Replace('{TITLE}', $title)
        $file = Join-Path $asrDir "$slug.$kind.ps1"
        [System.IO.File]::WriteAllText($file, $content, [System.Text.UTF8Encoding]::new($false))
    }
}

# ---------- Génération du manifest ----------

$manifestHeader = @'
version: "1.0"

section:
  id: asr
  order: 8
  title: "Attack Surface Reduction (ASR)"
  description: "Règles Defender qui bloquent des comportements offensifs courants (Office macros, LSASS dump, scripts obfusqués, USB, WMI, ransomware)."

rules:
'@

$ruleTemplate = @'
  - id: asr.{SLUG}
    title: "{TITLE}"
    description: "ASR rule {GUID} en mode Block."
    explanation: |
      ASR rule '{TITLE}'.
      GUID : {GUID}.
      Mode : Block (1). Si tu rencontres des faux positifs, tu peux passer
      cette règle en Audit (2) en éditant manuellement {SLUG}.action.ps1.
    severity: important
    impact: "Bloque le comportement décrit. Risque de faux positif sur certaines apps légitimes — auditer les events Defender (event log Microsoft-Windows-Windows Defender/Operational, event ID 1121/1122) en cas de souci."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    references:
      - "https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/attack-surface-reduction-rules-reference"
    tags: [defender, asr, hardening]
    added_in: "1.0"
    action: ./engine/actions/asr/{SLUG}.action.ps1
    test: ./engine/actions/asr/{SLUG}.test.ps1
    undo: ./engine/actions/asr/{SLUG}.undo.ps1
'@

$manifestBody = $manifestHeader + "`n"
foreach ($rule in $asrRules) {
    $entry = $ruleTemplate.Replace('{SLUG}', $rule.slug).Replace('{GUID}', $rule.guid).Replace('{TITLE}', $rule.title)
    $manifestBody += $entry + "`n"
}

[System.IO.File]::WriteAllText($manifestPath, $manifestBody, [System.Text.UTF8Encoding]::new($false))

Write-Host "Generated $($asrRules.Count) rules :" -ForegroundColor Green
Write-Host "  - $($asrRules.Count * 3) PS scripts in $asrDir"
Write-Host "  - 1 manifest : $manifestPath"
