# rdp_disable.test.ps1
# Test = registre fDenyTSConnections == 1.
# Plus : détecte si une session RDP entrante est active (port 3389 Established).
# Si oui, on signale feature_in_use=true → l'executor skip avec un warning
# au lieu de couper la session de l'utilisateur à chaud.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

$baseJson = Invoke-RegTestAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server' `
    -Name 'fDenyTSConnections' `
    -Expected 1

$base = $baseJson | ConvertFrom-Json

# Détection session RDP active (best-effort).
$inUse = $false
$inUseReason = $null
try {
    $sessions = @(Get-NetTCPConnection -LocalPort 3389 -State Established -ErrorAction Stop)
    if ($sessions.Count -gt 0) {
        $inUse = $true
        $inUseReason = "$($sessions.Count) session(s) RDP active(s) sur port 3389. Ferme-les avant d'appliquer cette règle pour éviter une coupure de connexion."
    }
} catch {}

@{
    compliant             = [bool]$base.compliant
    current               = $base.current
    feature_in_use        = $inUse
    feature_in_use_reason = $inUseReason
} | ConvertTo-Json -Compress -Depth 10
