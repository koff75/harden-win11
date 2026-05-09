package snapshot

// scanScript est le PowerShell qui scanne les chemins HKLM ciblés par
// les manifests + services + defender + OS info, et émet du JSON.
//
// Ne dépend pas de _helpers — autonome, copy-paste safe.
const scanScript = `
$ErrorActionPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Liste des (path, name) à snapshot. Synchronisée avec ce que les manifests touchent.
$keys = @(
    # Defender (HKLM)
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection'; Name = 'DisableRealtimeMonitoring' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection'; Name = 'DisableBehaviorMonitoring' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection'; Name = 'DisableIOAVProtection' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection'; Name = 'DisableScriptScanning' },
    # UAC
    @{ Path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'; Name = 'EnableLUA' },
    @{ Path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'; Name = 'ConsentPromptBehaviorAdmin' },
    @{ Path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'; Name = 'PromptOnSecureDesktop' },
    @{ Path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'; Name = 'ConsentPromptBehaviorUser' },
    # RDP
    @{ Path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server'; Name = 'fDenyTSConnections' },
    # Network hardening
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient'; Name = 'EnableMulticast' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient'; Name = 'EnableMDNS' },
    @{ Path = 'HKLM:\SYSTEM\CurrentControlSet\Services\NetBT\Parameters'; Name = 'NodeType' },
    @{ Path = 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc'; Name = 'Start' },
    @{ Path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa'; Name = 'LmCompatibilityLevel' },
    # SMB signing
    @{ Path = 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters'; Name = 'RequireSecuritySignature' },
    @{ Path = 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters'; Name = 'RequireSecuritySignature' },
    @{ Path = 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters'; Name = 'AllowInsecureGuestAuth' },
    # Privacy / Telemetry
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection'; Name = 'AllowTelemetry' },
    @{ Path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AdvertisingInfo'; Name = 'DisabledByGroupPolicy' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\InputPersonalization'; Name = 'AllowInputPersonalization' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System'; Name = 'PublishUserActivities' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Windows Search'; Name = 'AllowCortana' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsAI'; Name = 'DisableAIDataAnalysis' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent'; Name = 'DisableWindowsConsumerFeatures' },
    @{ Path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent'; Name = 'DisableWindowsSpotlightFeatures' }
)

$registry = @()
foreach ($k in $keys) {
    $exists = $false
    $value = $null
    try {
        $item = Get-ItemProperty -Path $k.Path -Name $k.Name -ErrorAction Stop
        $exists = $true
        $value = $item.($k.Name)
    } catch {}
    $registry += @{ path = $k.Path; name = $k.Name; exists = $exists; value = $value }
}

# Defender state (ne nécessite pas admin pour read-only).
$defender = @{}
try {
    $mp = Get-MpPreference -ErrorAction Stop
    $fields = 'DisableRealtimeMonitoring','DisableBehaviorMonitoring','DisableIOAVProtection','DisableScriptScanning','EnableNetworkProtection','EnableControlledFolderAccess','PUAProtection','MAPSReporting','SubmitSamplesConsent'
    foreach ($f in $fields) {
        $defender[$f] = "$($mp.$f)"
    }
    $defender['AsrIds'] = ($mp.AttackSurfaceReductionRules_Ids -join ',')
    $defender['AsrActions'] = ($mp.AttackSurfaceReductionRules_Actions -join ',')
} catch {}

# Services watch-listés (ceux que le hardening peut toucher).
$services = @()
foreach ($svc in @('WinDefend','WinHttpAutoProxySvc','LanmanWorkstation','LanmanServer','TermService','NetBT')) {
    try {
        $s = Get-Service -Name $svc -ErrorAction Stop
        $services += @{ name = $svc; start_type = "$($s.StartType)"; status = "$($s.Status)" }
    } catch {}
}

# OS info
$os = @{}
try {
    $info = Get-CimInstance Win32_OperatingSystem -ErrorAction Stop
    $os['caption'] = "$($info.Caption)"
    $os['version'] = "$($info.Version)"
    $os['build_number'] = "$($info.BuildNumber)"
} catch {}

@{
    registry = $registry
    defender = $defender
    services = $services
    os_info  = $os
    errors   = @()
} | ConvertTo-Json -Compress -Depth 10
`
