# sign-release.ps1 — Signe harden-engine.exe + harden-gui.exe avec un
# certificat self-signed local. Pour un usage personnel : ça calme
# SmartScreen/AV qui sinon flaggent un .exe non signé à chaque lancement.
#
# Pour un release publique, remplacer par un vrai cert EV (DigiCert, GlobalSign).
# Cf. docs/release.md.
#
# Usage :
#   pwsh -File tools/sign-release.ps1                 # auto-génère un cert si absent
#   pwsh -File tools/sign-release.ps1 -CertSubject 'CN=MyOwnCert'
#   pwsh -File tools/sign-release.ps1 -Force          # régénère le cert même s'il existe

[CmdletBinding()]
param(
    [string] $CertSubject = "CN=harden-win11-selfsigned",
    [string] $CertStore = "Cert:\CurrentUser\My",
    [switch] $Force
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$repoRoot = Split-Path -Parent $PSScriptRoot
$binaries = @(
    (Join-Path $repoRoot 'dist\harden-engine.exe'),
    (Join-Path $repoRoot 'cmd\harden-gui\build\bin\harden-gui.exe')
)

# --- 1. Trouver ou créer le cert ---
$cert = Get-ChildItem $CertStore -CodeSigningCert -ErrorAction SilentlyContinue |
    Where-Object { $_.Subject -eq $CertSubject } |
    Sort-Object NotAfter -Descending |
    Select-Object -First 1

if (-not $cert -or $Force) {
    if ($Force -and $cert) { Remove-Item $cert.PSPath -Force }

    Write-Host "Génération d'un cert self-signed code-signing : $CertSubject" -ForegroundColor Cyan
    $cert = New-SelfSignedCertificate `
        -Subject $CertSubject `
        -Type CodeSigningCert `
        -KeyUsage DigitalSignature `
        -KeySpec Signature `
        -KeyAlgorithm RSA `
        -KeyLength 4096 `
        -CertStoreLocation $CertStore `
        -NotAfter (Get-Date).AddYears(5)

    Write-Host "Cert généré (thumbprint : $($cert.Thumbprint))" -ForegroundColor Green
    Write-Host ""
    Write-Host "POUR FAIRE CONFIANCE A CE CERT (sinon SmartScreen continue de gueuler) :" -ForegroundColor Yellow
    Write-Host "  Export-Certificate -Cert '$($cert.PSPath)' -FilePath '$repoRoot\tools\harden-cert.cer'"
    Write-Host "  Import-Certificate -FilePath '$repoRoot\tools\harden-cert.cer' -CertStoreLocation Cert:\LocalMachine\Root"
    Write-Host "  (la dernière étape requiert admin et installe le cert dans Trusted Root)"
    Write-Host ""
} else {
    Write-Host "Cert existant trouvé : $($cert.Thumbprint) (expire $($cert.NotAfter))" -ForegroundColor Cyan
}

# --- 2. Signer les binaires ---
foreach ($exe in $binaries) {
    if (-not (Test-Path $exe)) {
        Write-Warning "Binaire absent (skip) : $exe"
        continue
    }
    Write-Host "Signing : $exe" -ForegroundColor Cyan
    $sig = Set-AuthenticodeSignature `
        -FilePath $exe `
        -Certificate $cert `
        -TimestampServer 'http://timestamp.digicert.com' `
        -HashAlgorithm SHA256

    # Status possibles :
    # - Valid : cert dans Trusted Root → tout vert
    # - UnknownError + msg "root certificate which is not trusted" :
    #   signature appliquée correctement, juste pas validable tant que le cert
    #   n'est pas importé dans LocalMachine\Root (cf. instructions plus haut).
    if ($sig.Status -eq 'Valid') {
        Write-Host "  Status: Valid (cert trusted)" -ForegroundColor Green
    } elseif ($sig.Status -eq 'UnknownError' -and $sig.StatusMessage -match 'not trusted') {
        Write-Host "  Status: signed (self-signed, cert pas encore en Trusted Root)" -ForegroundColor Yellow
    } else {
        Write-Error "Échec signature : $($sig.Status) — $($sig.StatusMessage)"
    }
}

Write-Host ""
Write-Host "[OK] Signature terminée. Vérifie avec :"
Write-Host "  Get-AuthenticodeSignature dist\harden-engine.exe"
Write-Host "  signtool verify /pa /v dist\harden-engine.exe       # si Windows SDK installé"
