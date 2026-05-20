# wpad_disable.test.ps1
# Conforme = Start=3 (Manual) ou Start=4 (Disabled).
# On accepte les deux pour ne pas re-declencher un apply sur les machines
# durcies avant le fix v0.4.1 (qui mettait Start=4) — toutes les deux
# protegent contre le poisoning WPAD.
#
# Plus : detecte si un proxy auto-discovery (WPAD/PAC) est configure dans
# Internet Settings. Si oui, desactiver le service WinHttpAutoProxy va
# casser l'acces web → on signale feature_in_use.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

# Le check brut : lire la valeur courante.
$current = $null
try {
    $current = (Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc' -Name 'Start' -ErrorAction Stop).Start
} catch {}

# Conforme si Manual (3) ou Disabled (4). Voir commentaire en tete.
$compliant = ($current -eq 3) -or ($current -eq 4)

# Detect : auto-detect active dans Internet Settings ou .pac configure.
$inUse = $false
$inUseReason = $null
try {
    $is = Get-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -ErrorAction Stop
    $autoConfigURL = $is.AutoConfigURL
    if (-not [string]::IsNullOrWhiteSpace($autoConfigURL)) {
        $inUse = $true
        $inUseReason = "AutoConfigURL configure ($autoConfigURL). Desactiver WPAD couperait la resolution du proxy automatique."
    }
} catch {}

@{
    compliant             = $compliant
    current               = @{ 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc\Start' = $current }
    feature_in_use        = $inUse
    feature_in_use_reason = $inUseReason
} | ConvertTo-Json -Compress -Depth 10
