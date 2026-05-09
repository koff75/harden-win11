# refactor-reg-rules.ps1
# Régénère action/test/undo des règles registry single-value pour qu'elles
# utilisent _helpers/reg.psm1. Un fichier devient ~5 lignes au lieu de 30.
# Skip les règles multi-value (qui touchent plusieurs Names dans un même Path)
# et celles qui ne suivent pas le pattern simple.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Liste des règles single-value à régénérer.
# Format : @{ dir, slug, path, name, expected, type, comment }
$rules = @(
    @{ dir='uac'; slug='consent_admin';            path='HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'; name='ConsentPromptBehaviorAdmin'; expected=5; type='DWord'; comment='UAC : prompt sur secure desktop pour admin (5).' }
    @{ dir='uac'; slug='prompt_secure_desktop';    path='HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'; name='PromptOnSecureDesktop';     expected=1; type='DWord'; comment='UAC : tous les prompts UAC sur secure desktop.' }
    @{ dir='uac'; slug='deny_user_elevation';      path='HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'; name='ConsentPromptBehaviorUser'; expected=0; type='DWord'; comment='UAC : refuser élévation pour comptes standard.' }

    @{ dir='system_settings'; slug='rdp_disable';        path='HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server';                  name='fDenyTSConnections'; expected=1; type='DWord'; comment='RDP : refuse les connexions entrantes.' }
    @{ dir='system_settings'; slug='fast_startup_off';   path='HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Power';            name='HiberbootEnabled';   expected=0; type='DWord'; comment='Power : désactive Fast Startup.' }

    @{ dir='network'; slug='llmnr_disable';        path='HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient';            name='EnableMulticast';            expected=0; type='DWord'; comment='LLMNR désactivé (anti-Responder).' }
    @{ dir='network'; slug='mdns_disable';         path='HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters';       name='EnableMDNS';                 expected=0; type='DWord'; comment='mDNS désactivé.' }
    @{ dir='network'; slug='wpad_disable';         path='HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc';       name='Start';                      expected=4; type='DWord'; comment='WPAD désactivé (anti-poisoning).' }
    @{ dir='network'; slug='ntlm_v2_only';         path='HKLM:\SYSTEM\CurrentControlSet\Control\Lsa';                        name='LmCompatibilityLevel';       expected=5; type='DWord'; comment='LmCompatibilityLevel=5 (NTLMv2 only).' }
    @{ dir='network'; slug='smb_client_signing';   path='HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters'; name='RequireSecuritySignature'; expected=1; type='DWord'; comment='SMB client : signature requise.' }
    @{ dir='network'; slug='smb_server_signing';  path='HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters';      name='RequireSecuritySignature'; expected=1; type='DWord'; comment='SMB server : signature requise.' }
    @{ dir='network'; slug='smb_guest_auth_off';   path='HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters'; name='AllowInsecureGuestAuth';  expected=0; type='DWord'; comment='SMB client : refuse auth guest.' }

    @{ dir='privacy'; slug='telemetry_required';        path='HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection';            name='AllowTelemetry';        expected=1; type='DWord'; comment='Telemetry niveau 1 (Required only).' }
    @{ dir='privacy'; slug='advertising_id_machine';    path='HKLM:\SOFTWARE\Policies\Microsoft\Windows\AdvertisingInfo';           name='DisabledByGroupPolicy'; expected=1; type='DWord'; comment='AdvertisingID désactivé machine-wide.' }
    @{ dir='privacy'; slug='online_speech_off';         path='HKLM:\SOFTWARE\Policies\Microsoft\InputPersonalization';              name='AllowInputPersonalization'; expected=0; type='DWord'; comment='Online Speech désactivé.' }
    @{ dir='privacy'; slug='cortana_off';               path='HKLM:\SOFTWARE\Policies\Microsoft\Windows\Windows Search';            name='AllowCortana';          expected=0; type='DWord'; comment='Cortana désactivée.' }
    @{ dir='privacy'; slug='recall_off';                path='HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsAI';                 name='DisableAIDataAnalysis'; expected=1; type='DWord'; comment='Windows Recall désactivé (préventif).' }
    @{ dir='privacy'; slug='silent_apps_off';           path='HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'; name='SilentInstalledAppsEnabled'; expected=0; type='DWord'; comment='HKCU : pas de réinstall silencieuse.' }
    @{ dir='privacy'; slug='settings_suggestions_off';  path='HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'; name='SubscribedContent-338389Enabled'; expected=0; type='DWord'; comment='HKCU : suggestions Settings off.' }
    @{ dir='privacy'; slug='start_suggestions_off';     path='HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'; name='SystemPaneSuggestionsEnabled';    expected=0; type='DWord'; comment='HKCU : suggestions menu Démarrer off.' }
    @{ dir='privacy'; slug='advertising_id_user';       path='HKCU:\Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo';     name='Enabled';               expected=0; type='DWord'; comment='HKCU : Advertising ID off.' }
)

$root = Split-Path -Parent $PSScriptRoot

$actionTemplate = @'
# {SLUG}.action.ps1
# {COMMENT}

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path '{PATH}' `
    -Name '{NAME}' `
    -Value {EXPECTED} `
    -Type {TYPE}
'@

$testTemplate = @'
# {SLUG}.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegTestAction `
    -Path '{PATH}' `
    -Name '{NAME}' `
    -Expected {EXPECTED}
'@

$undoTemplate = @'
# {SLUG}.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegUndoAction `
    -Path '{PATH}' `
    -Name '{NAME}' `
    -Type {TYPE}
'@

$count = 0
foreach ($r in $rules) {
    $dir = Join-Path $root "engine\actions\$($r.dir)"
    if (-not (Test-Path $dir)) {
        Write-Warning "Dir not found, skipping: $dir"
        continue
    }
    foreach ($kind in @('action', 'test', 'undo')) {
        $template = switch ($kind) {
            'action' { $actionTemplate }
            'test'   { $testTemplate }
            'undo'   { $undoTemplate }
        }
        $content = $template
        $content = $content.Replace('{SLUG}',     $r.slug)
        $content = $content.Replace('{PATH}',     $r.path)
        $content = $content.Replace('{NAME}',     $r.name)
        $content = $content.Replace('{EXPECTED}', $r.expected.ToString())
        $content = $content.Replace('{TYPE}',     $r.type)
        $content = $content.Replace('{COMMENT}',  $r.comment)
        $file = Join-Path $dir "$($r.slug).$kind.ps1"
        [System.IO.File]::WriteAllText($file, $content, [System.Text.UTF8Encoding]::new($false))
        $count++
    }
}

Write-Host "Refactored $($rules.Count) rules ($count files written)" -ForegroundColor Green
