# wpad_disable.test.ps1
# Plus : détecte si un proxy auto-discovery (WPAD/PAC) est configuré dans
# Internet Settings. Si oui, désactiver le service WinHttpAutoProxy va
# casser l'accès web → on signale feature_in_use.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

$baseJson = Invoke-RegTestAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc' `
    -Name 'Start' `
    -Expected 4

$base = $baseJson | ConvertFrom-Json

# Detect : auto-detect activé dans Internet Settings ou .pac configuré.
$inUse = $false
$inUseReason = $null
try {
    $is = Get-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -ErrorAction Stop
    $autoConfigURL = $is.AutoConfigURL
    $proxyEnable = if ($null -ne $is.ProxyEnable) { [int]$is.ProxyEnable } else { 0 }
    if (-not [string]::IsNullOrWhiteSpace($autoConfigURL)) {
        $inUse = $true
        $inUseReason = "AutoConfigURL configuré ($autoConfigURL). Désactiver WPAD couperait la résolution du proxy automatique."
    }
} catch {}

@{
    compliant             = [bool]$base.compliant
    current               = $base.current
    feature_in_use        = $inUse
    feature_in_use_reason = $inUseReason
} | ConvertTo-Json -Compress -Depth 10
