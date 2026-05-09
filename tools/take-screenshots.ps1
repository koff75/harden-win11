# take-screenshots.ps1 — Capture la fenêtre GUI à intervalles réguliers,
# tu navigues à la main entre les écrans à capturer. Pratique pour générer
# les screenshots du README sans devoir installer un soft tiers.
#
# Usage :
#   1. Lance la GUI à part (run-as-admin.bat ou .\cmd\harden-gui\build\bin\harden-gui.exe)
#   2. Lance ce script : pwsh -File tools/take-screenshots.ps1
#   3. Tu auras 8 secondes pour mettre la GUI au premier plan + arriver sur
#      l'écran à capturer, puis screenshot auto, puis 6s pour le suivant.
#   4. Les fichiers sortent dans docs/screenshots/01.png, 02.png, etc.
#
# Alternative : utilise Win+Shift+S (Snipping Tool) à la main et dépose
# les fichiers directement dans docs/screenshots/ avec les bons noms :
#   01-dashboard.png         vue d'ensemble + dashboard "Ta machine est OK..."
#   02-action-card.png       carte action user-friendly (clique sur Vérifier)
#   03-maturity-score.png    modal "Score" du dashboard
#   04-coverage.png          modal "Couverture standards"
#   05-watchlist.png         modal Watchlist alerts (si tu en as)
#   06-coach-modal.png       modal du 💡 sur une rule (ex: smbv1_disable)

param(
    [int] $WaitSeconds = 8,
    [int] $BetweenSeconds = 6,
    [string] $OutDir = 'docs\screenshots'
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$shots = @(
    @{ name = '01-dashboard';      desc = "Vue d'ensemble + dashboard ('Ta machine est OK sur X points')" }
    @{ name = '02-action-card';    desc = "Carte action user-friendly (Aujourd'hui / Si tu actives / Pour qui / Risque)" }
    @{ name = '03-maturity-score'; desc = "Modal Score (badge A/B/C/D + composants)" }
    @{ name = '04-coverage';       desc = "Modal Couverture (CIS / ANSSI / MS)" }
    @{ name = '05-watchlist';      desc = "Modal Watchlist (si tu as un apply récent — sinon skip)" }
    @{ name = '06-coach-modal';    desc = "Modal coach (clic sur 💡 d'une rule annotée, ex: smbv1)" }
)

if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

Write-Host "=== take-screenshots.ps1 ===" -ForegroundColor Cyan
Write-Host "Tu as $WaitSeconds secondes pour mettre la GUI au premier plan." -ForegroundColor Yellow
Write-Host "Puis $BetweenSeconds secondes entre chaque screenshot." -ForegroundColor Yellow
Write-Host ""

Start-Sleep -Seconds $WaitSeconds

foreach ($shot in $shots) {
    Write-Host "[$($shot.name)] $($shot.desc)" -ForegroundColor Cyan
    $screen = [System.Windows.Forms.SystemInformation]::VirtualScreen
    $bmp = New-Object System.Drawing.Bitmap $screen.Width, $screen.Height
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($screen.X, $screen.Y, 0, 0, $bmp.Size)
    $path = Join-Path $OutDir "$($shot.name).png"
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose()
    $bmp.Dispose()
    Write-Host "  → $path" -ForegroundColor Green

    if ($shot -ne $shots[-1]) {
        Write-Host "  (next screenshot dans $BetweenSeconds s, prépare l'écran suivant)" -ForegroundColor DarkGray
        Start-Sleep -Seconds $BetweenSeconds
    }
}

Write-Host ""
Write-Host "Done. $($shots.Count) screenshots dans $OutDir." -ForegroundColor Green
Write-Host ""
Write-Host "Vérifie/recadre si besoin (Snipping Tool), puis :"
Write-Host "  git add docs/screenshots/"
Write-Host "  git commit -m 'docs: screenshots GUI'"
