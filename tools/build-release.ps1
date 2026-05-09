# build-release.ps1 — Construit un ZIP portable de release.
#
# Le ZIP est self-contained : double-clic sur run-as-admin.bat → la GUI
# démarre avec UAC. Aucune installation, aucune entrée registre, l'utilisateur
# supprime simplement le dossier pour désinstaller.
#
# Usage :
#   pwsh -File tools/build-release.ps1                      # version auto = git describe
#   pwsh -File tools/build-release.ps1 -Version 0.2.0      # version explicite
#   pwsh -File tools/build-release.ps1 -Sign               # signe avant zip

[CmdletBinding()]
param(
    [string] $Version,
    [switch] $Sign
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if (-not $Version) {
    try {
        $Version = (& git describe --tags --always 2>$null).Trim()
        if (-not $Version) { $Version = '0.1.0-dev' }
    } catch {
        $Version = '0.1.0-dev'
    }
}

$releaseName = "Harden-Win11-$Version"
$stagingDir = Join-Path $repoRoot "build\$releaseName"
$zipPath = Join-Path $repoRoot "build\$releaseName.zip"
$shaPath = "$zipPath.sha256"

Write-Host "Release : $releaseName" -ForegroundColor Cyan
Write-Host "Staging : $stagingDir"
Write-Host ""

# --- 1. Build engine + GUI ---
Write-Host "[1/5] Building harden-engine..." -ForegroundColor Cyan
& go build -ldflags "-X main.Version=$Version" -o "$repoRoot\dist\harden-engine.exe" .\cmd\harden-engine
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

Write-Host "[2/5] Building harden-gui..." -ForegroundColor Cyan
Push-Location (Join-Path $repoRoot 'cmd\harden-gui')
try {
    & wails build
    if ($LASTEXITCODE -ne 0) { throw "wails build failed" }
} finally { Pop-Location }

# --- 2. Sign si demandé ---
if ($Sign) {
    Write-Host "[3/5] Signing binaries..." -ForegroundColor Cyan
    & "$repoRoot\tools\sign-release.ps1"
} else {
    Write-Host "[3/5] Skipping signature (-Sign not passed)" -ForegroundColor DarkGray
}

# --- 3. Staging ---
Write-Host "[4/5] Assembling staging dir..." -ForegroundColor Cyan
if (Test-Path $stagingDir) { Remove-Item $stagingDir -Recurse -Force }
New-Item -ItemType Directory -Path $stagingDir -Force | Out-Null

# Binaries
Copy-Item "$repoRoot\dist\harden-engine.exe" $stagingDir
Copy-Item "$repoRoot\cmd\harden-gui\build\bin\harden-gui.exe" $stagingDir

# Manifests + actions + schemas + mappings
Copy-Item "$repoRoot\manifests" $stagingDir -Recurse
Copy-Item "$repoRoot\engine" $stagingDir -Recurse -Exclude '*.tests.ps1'
Copy-Item "$repoRoot\schemas" $stagingDir -Recurse
Copy-Item "$repoRoot\mappings" $stagingDir -Recurse

# Launcher batch (auto-élève si non-admin)
$launcher = @'
@echo off
:: Lance harden-gui.exe avec UAC. Si déjà admin, lance direct.
NET SESSION >nul 2>&1
IF %errorlevel% EQU 0 (
    start "" "%~dp0harden-gui.exe"
) ELSE (
    powershell -Command "Start-Process -FilePath '%~dp0harden-gui.exe' -Verb RunAs"
)
'@
Set-Content -Path (Join-Path $stagingDir 'run-as-admin.bat') -Value $launcher -Encoding ASCII

# README
$readme = @"
# Harden-Win11 $Version (portable)

## Lancer
- Double-clic sur **run-as-admin.bat** (la GUI demande UAC).
- Ou, en CLI : ouvre un PowerShell admin, ``cd`` dans ce dossier,
  puis ``.\harden-engine.exe apply --dry-run``.

## Pas d'installation
Aucune entrée registre, aucun service. Pour désinstaller : supprime ce dossier.
Le journal des runs est écrit dans ``%ProgramData%\Harden-Win11\runs\`` —
nettoyer ce dossier si tu veux effacer toute trace.

## Documentation
- Vue d'ensemble : ``README.md`` du dépôt source.
- Référentiels couverts (CIS / ANSSI / MS) : ``mappings/baselines.yaml``
  ou ``harden-engine.exe coverage``.

## Sécurité
Vérifie le checksum :
``Get-FileHash -Algorithm SHA256 Harden-Win11-$Version.zip``
et compare avec le fichier ``.sha256`` à côté.

Version : $Version
Build date : $(Get-Date -Format 'yyyy-MM-dd HH:mm')
"@
Set-Content -Path (Join-Path $stagingDir 'README.txt') -Value $readme -Encoding UTF8

# --- 4. ZIP ---
Write-Host "[5/5] Creating ZIP + SHA256..." -ForegroundColor Cyan
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
Compress-Archive -Path "$stagingDir\*" -DestinationPath $zipPath -CompressionLevel Optimal

$hash = (Get-FileHash -Algorithm SHA256 $zipPath).Hash
"$hash  $releaseName.zip" | Set-Content $shaPath -Encoding ASCII

$size = [math]::Round((Get-Item $zipPath).Length / 1MB, 2)
Write-Host ""
Write-Host "[OK] Release ZIP : $zipPath ($size MB)" -ForegroundColor Green
Write-Host "    SHA256 : $hash"
Write-Host "    SHA256 file : $shaPath"
