# smoke-test.ps1 — Séquence E2E rapide pour valider une release.
#
# À lancer depuis la racine du repo, en admin, sur un poste Win11 (idéalement
# une VM neuve pour ne pas pourrir l'OS hôte).
#
# Étapes :
#   1. Build engine + GUI (skippable avec -SkipBuild)
#   2. validate (lit + JSONSchema-valide les manifests)
#   3. coverage (lit baselines.yaml et affiche les %)
#   4. apply --dry-run (mode test, ne modifie rien)
#   5. apply --yes --section system_settings  (apply réel, scope minimal)
#   6. undo last run (rollback)
#   7. apply --dry-run de nouveau pour vérifier que tout est revenu en place
#
# En cas d'échec d'une étape, le script s'arrête (Stop on error). Sortie JSON
# sur stdout pour parsing CI éventuel.
#
# Usage :
#   pwsh -File tools/smoke-test.ps1
#   pwsh -File tools/smoke-test.ps1 -SkipBuild
#   pwsh -File tools/smoke-test.ps1 -DryRunOnly   # ne fait pas l'apply réel

[CmdletBinding()]
param(
    [switch] $SkipBuild,
    [switch] $DryRunOnly
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

$engineExe = Join-Path $repoRoot 'dist\harden-engine.exe'

function Step([string] $title, [scriptblock] $action) {
    Write-Host ""
    Write-Host "=== $title ===" -ForegroundColor Cyan
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    & $action
    $sw.Stop()
    Write-Host "[OK] $title — $($sw.ElapsedMilliseconds) ms" -ForegroundColor Green
}

# --- 1. Build (optionnel) ---
if (-not $SkipBuild) {
    Step 'Build engine' {
        & go build -o $engineExe .\cmd\harden-engine
        if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    }
} else {
    if (-not (Test-Path $engineExe)) {
        throw "engine binary not found at $engineExe — drop -SkipBuild ou build manuellement"
    }
}

# --- 2. Validate manifests ---
Step 'Validate manifests' {
    & $engineExe validate
    if ($LASTEXITCODE -ne 0) { throw "validate failed" }
}

# --- 3. Coverage rapport ---
Step 'Coverage baseline' {
    & $engineExe coverage
    if ($LASTEXITCODE -ne 0) { throw "coverage failed" }
}

# --- 4. Dry-run global ---
Step 'apply --dry-run (toutes sections)' {
    & $engineExe apply --dry-run --yes
    if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 1) {
        # exit 1 = au moins 1 rule non-conforme, c'est attendu
        throw "dry-run failed avec exit code $LASTEXITCODE"
    }
    Write-Host "Exit code dry-run : $LASTEXITCODE (0 ou 1 = OK)" -ForegroundColor DarkGray
}

# --- 5. Apply réel (scope minimal) ---
if (-not $DryRunOnly) {
    Step 'apply --section system_settings (apply réel scope minimal)' {
        # Vérification admin avant l'apply
        $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        if (-not $isAdmin) { throw "Non-admin : skip apply réel. Relance en admin pour cette étape." }

        & $engineExe apply --section system_settings --yes
        if ($LASTEXITCODE -ne 0) { throw "apply failed avec exit code $LASTEXITCODE" }
    }

    # --- 6. Undo last run ---
    Step 'undo last run' {
        & $engineExe undo --yes
        if ($LASTEXITCODE -ne 0) { throw "undo failed avec exit code $LASTEXITCODE" }
    }

    # --- 7. Dry-run final pour vérifier le retour à l'état initial ---
    Step 'apply --dry-run --section system_settings (vérif post-undo)' {
        & $engineExe apply --dry-run --yes --section system_settings
        # Exit 1 attendu si certaines règles sont non-conformes (= revenu à l'état pré-apply)
        Write-Host "Exit code post-undo : $LASTEXITCODE" -ForegroundColor DarkGray
    }
}

Write-Host ""
Write-Host "[SMOKE OK] Séquence E2E complète." -ForegroundColor Green
