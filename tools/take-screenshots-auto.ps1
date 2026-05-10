# take-screenshots-auto.ps1 — Capture automatisée des screenshots du README.
#
# Lance la GUI Win11 Hardening, pilote l'UI via UIAutomation pour produire
# les états souhaités (vérification, hover tooltip, modal score, switch EN),
# et capture la fenêtre dans docs/screenshots/.
#
# Usage : powershell.exe -ExecutionPolicy Bypass -File tools\take-screenshots-auto.ps1

param(
    [string] $OutDir = 'docs\screenshots',
    [string] $Exe    = '.\cmd\harden-gui\build\bin\harden-gui.exe',
    [int]    $WindowW = 1280,
    [int]    $WindowH = 820
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes

Add-Type @"
using System;
using System.Runtime.InteropServices;
[StructLayout(LayoutKind.Sequential)]
public struct RECT { public int Left, Top, Right, Bottom; }
public class W32 {
    [DllImport("user32.dll")] public static extern IntPtr FindWindow(string c, string n);
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int n);
    [DllImport("user32.dll")] public static extern bool MoveWindow(IntPtr h, int x, int y, int w, int hh, bool r);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
    [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
    [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int cx, int cy, uint flags);
    [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h, IntPtr hdc, uint nFlags);
    [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, IntPtr pid);
    [DllImport("user32.dll")] public static extern bool AttachThreadInput(uint idAttach, uint idAttachTo, bool fAttach);
    [DllImport("kernel32.dll")] public static extern uint GetCurrentThreadId();
    [DllImport("user32.dll")] public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, IntPtr dwExtraInfo);
    [DllImport("user32.dll")] public static extern bool BringWindowToTop(IntPtr h);
    [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, int dx, int dy, int dwData, IntPtr dwExtraInfo);
    public static readonly IntPtr HWND_TOPMOST    = new IntPtr(-1);
    public static readonly IntPtr HWND_NOTOPMOST  = new IntPtr(-2);
    public const uint SWP_SHOWWINDOW = 0x0040;
    public const uint PW_RENDERFULLCONTENT = 0x00000002;
}
"@

if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

if (-not (Test-Path $Exe)) {
    throw "Binary not found: $Exe`nBuild first: cd cmd/harden-gui && wails build"
}

Write-Host "=== Win11 Hardening — auto screenshots ===" -ForegroundColor Cyan

# Tue toute instance résiduelle
Get-Process harden-gui -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

$proc = Start-Process $Exe -PassThru
Write-Host "GUI launched (PID $($proc.Id)). Waiting for window..."

# Attend que MainWindowHandle soit disponible via Get-Process (plus fiable que FindWindow)
$hWnd = [IntPtr]::Zero
$tries = 0
while ($hWnd -eq [IntPtr]::Zero -and $tries -lt 150) {
    Start-Sleep -Milliseconds 200
    $p = Get-Process -Id $proc.Id -ErrorAction SilentlyContinue
    if ($p -and $p.MainWindowHandle -ne [IntPtr]::Zero) {
        $hWnd = $p.MainWindowHandle
    }
    $tries++
}
if ($hWnd -eq [IntPtr]::Zero) {
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    throw "MainWindowHandle not available after 30s"
}
Write-Host "Window hWnd=$hWnd"

function Bring-ToFront {
    param([IntPtr]$h)
    [W32]::ShowWindow($h, 9) | Out-Null  # SW_RESTORE
    [W32]::SetWindowPos($h, [W32]::HWND_TOPMOST, 50, 50, $WindowW, $WindowH, [W32]::SWP_SHOWWINDOW) | Out-Null
    # Trick : tap ALT to reset focus stealing protection
    [W32]::keybd_event(0x12, 0, 0, [IntPtr]::Zero) | Out-Null
    [W32]::keybd_event(0x12, 0, 2, [IntPtr]::Zero) | Out-Null
    Start-Sleep -Milliseconds 50
    # Bypass focus stealing protection : attach our thread input to the foreground window's thread
    $fg     = [W32]::GetForegroundWindow()
    $fgTid  = [W32]::GetWindowThreadProcessId($fg, [IntPtr]::Zero)
    $myTid  = [W32]::GetCurrentThreadId()
    [W32]::AttachThreadInput($myTid, $fgTid, $true)  | Out-Null
    [W32]::BringWindowToTop($h)                       | Out-Null
    [W32]::SetForegroundWindow($h)                    | Out-Null
    [W32]::AttachThreadInput($myTid, $fgTid, $false) | Out-Null
}

Bring-ToFront $hWnd
# Laisser WebView2 + JS finir d'initialiser le DOM avant tout UIA
Start-Sleep -Seconds 6
Bring-ToFront $hWnd
Start-Sleep -Milliseconds 800

# Attend que la fenetre ait sa vraie taille (Wails peut animer apres MoveWindow)
$stableTries = 0
while ($stableTries -lt 30) {
    $rect = New-Object RECT
    [W32]::GetWindowRect($hWnd, [ref]$rect) | Out-Null
    $w = $rect.Right - $rect.Left
    $h = $rect.Bottom - $rect.Top
    if ($w -ge 1000 -and $h -ge 600) { Write-Host "  window stable: ${w}x${h}"; break }
    Bring-ToFront $hWnd
    Start-Sleep -Milliseconds 500
    $stableTries++
}
# Wait additionnel pour que le WebView2 finisse de rendre tout le DOM correctement
Start-Sleep -Seconds 4
Bring-ToFront $hWnd
Start-Sleep -Milliseconds 500

function Capture-Window {
    param([string]$path)
    $rect = New-Object RECT
    [W32]::GetWindowRect($hWnd, [ref]$rect) | Out-Null
    $w = $rect.Right - $rect.Left
    $h = $rect.Bottom - $rect.Top
    if ($w -le 0 -or $h -le 0) { Write-Warning "Invalid rect $w x $h"; return }

    # WebView2 + DirectComposition : PrintWindow renvoie du noir.
    # On force la fenetre au foreground et on fait CopyFromScreen.
    Bring-ToFront $hWnd
    Start-Sleep -Milliseconds 350

    $bmp = New-Object System.Drawing.Bitmap $w, $h
    $g   = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($rect.Left, $rect.Top, 0, 0, $bmp.Size)
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose(); $bmp.Dispose()
    Write-Host "  -> $path" -ForegroundColor Green
}

function Refresh-UIA {
    try { return [System.Windows.Automation.AutomationElement]::FromHandle($hWnd) } catch { return $null }
}

# UIA root scoped to the window
$uiaWin = $null
$uiaTries = 0
while ($null -eq $uiaWin -and $uiaTries -lt 20) {
    try { $uiaWin = [System.Windows.Automation.AutomationElement]::FromHandle($hWnd) } catch { Start-Sleep -Milliseconds 200 }
    $uiaTries++
}
if ($null -eq $uiaWin) { Write-Warning "UIA element not available — falling back to keyboard nav only" }

function Click-ById {
    param([string]$id)
    # Refresh UIA tree à chaque appel : le DOM Wails se met à jour au fil des actions
    $win = Refresh-UIA
    if ($null -eq $win) { Write-Warning "UIA not available"; return $false }
    $cond = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::AutomationIdProperty, $id)
    $btn  = $win.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $cond)
    if ($null -eq $btn) { Write-Warning "Element with AutomationId '$id' not found"; return $false }
    try {
        $pat = $btn.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
        $pat.Invoke()
        Write-Host "  click: id='$id' name='$($btn.Current.Name)'" -ForegroundColor DarkGray
        return $true
    } catch {
        Write-Warning "Failed to invoke '$id': $_"
        return $false
    }
}

function Dump-AllButtons {
    $win = Refresh-UIA
    if ($null -eq $win) { return }
    $cond = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ControlTypeProperty, [System.Windows.Automation.ControlType]::Button)
    $all  = $win.FindAll([System.Windows.Automation.TreeScope]::Descendants, $cond)
    Write-Host "  UIA buttons visible : $($all.Count)" -ForegroundColor DarkCyan
    foreach ($b in $all) {
        try {
            Write-Host "    id='$($b.Current.AutomationId)' name='$($b.Current.Name)'" -ForegroundColor DarkGray
        } catch {}
    }
}

# Dump UIA pour debug initial
Write-Host "[debug] Listing UIA buttons:" -ForegroundColor DarkCyan
Dump-AllButtons

# === 1. Etat initial : dashboard ===
Write-Host "[1] Capture initial state -> 01-dashboard.png"
Capture-Window (Join-Path $OutDir '01-dashboard.png')

# === 2. Click Verifier (dry-run) ===
Write-Host "[2] Click btn-dryrun"
$clicked = Click-ById -id "btn-dryrun"

# Pendant le dry-run, le loader est visible. Capture rapide pour 03-apply.
if ($clicked) {
    Start-Sleep -Milliseconds 2500
    Write-Host "[3] Capture during dry-run (loader) -> 03-apply.png"
    Capture-Window (Join-Path $OutDir '03-apply.png')
    # Sans admin le dry-run prend 90-120s pour 85 regles. On attend ~10s qu'il y ait
    # quelques regles dans le tableau, puis on Cancel pour avoir un dashboard partiel.
    Write-Host "  let dry-run run for 10s, then cancel..."
    Start-Sleep -Seconds 10
    Write-Host "  click btn-cancel"
    $null = Click-ById -id "btn-cancel"
    Start-Sleep -Seconds 2
} else {
    Copy-Item (Join-Path $OutDir '01-dashboard.png') (Join-Path $OutDir '03-apply.png') -Force
}

# === Re-capture 01-dashboard avec le tableau rempli ===
Write-Host "[3b] Re-capture dashboard with results -> 01-dashboard.png"
Capture-Window (Join-Path $OutDir '01-dashboard.png')

# === 3. Hover row pour tooltip ===
$rect = New-Object RECT
[W32]::GetWindowRect($hWnd, [ref]$rect) | Out-Null

# Scroll dans la results-pane pour ramener le tableau dans le viewport visible
$scrollX = $rect.Left + 800
$scrollY = $rect.Top + 400
[W32]::SetCursorPos($scrollX, $scrollY) | Out-Null
Start-Sleep -Milliseconds 200
for ($i = 0; $i -lt 6; $i++) {
    [W32]::mouse_event(0x0800, 0, 0, -120, [IntPtr]::Zero) | Out-Null   # MOUSEEVENTF_WHEEL down
    Start-Sleep -Milliseconds 80
}
Start-Sleep -Milliseconds 600
Bring-ToFront $hWnd
Start-Sleep -Milliseconds 300

$rowFound = $false
$winUIA = Refresh-UIA
if ($null -ne $winUIA) {
    foreach ($ctrlType in @([System.Windows.Automation.ControlType]::DataItem, [System.Windows.Automation.ControlType]::TreeItem, [System.Windows.Automation.ControlType]::ListItem, [System.Windows.Automation.ControlType]::Custom)) {
        $cond = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ControlTypeProperty, $ctrlType)
        $rows = $winUIA.FindAll([System.Windows.Automation.TreeScope]::Descendants, $cond)
        Write-Host "  found $($rows.Count) elements of type $ctrlType"
        foreach ($r in $rows) {
            try {
                $br = $r.Current.BoundingRectangle
                # Cherche une row visible : largeur >= 200, hauteur 20-80, Y dans viewport
                if ($br.Width -ge 200 -and $br.Height -ge 20 -and $br.Height -le 80 `
                    -and $br.Y -gt ($rect.Top + 250) -and $br.Y -lt ($rect.Bottom - 100)) {
                    $hoverX = [int]($br.X + $br.Width / 3)
                    $hoverY = [int]($br.Y + $br.Height / 2)
                    Write-Host "  hover row at $hoverX,$hoverY (W=$($br.Width) H=$($br.Height))"
                    $rowFound = $true
                    break
                }
            } catch {}
        }
        if ($rowFound) { break }
    }
}
if (-not $rowFound) {
    # Fallback : tableau typiquement a y=420-700 dans la fenetre
    $hoverX = $rect.Left + 600
    $hoverY = $rect.Top + 480
    Write-Host "  fallback hover at $hoverX,$hoverY"
}
[W32]::SetCursorPos($hoverX, $hoverY) | Out-Null
Start-Sleep -Milliseconds 200
[W32]::mouse_event(0x0001, 2, 1, 0, [IntPtr]::Zero) | Out-Null
Start-Sleep -Milliseconds 200
[W32]::SetCursorPos(($hoverX + 8), ($hoverY + 3)) | Out-Null
[W32]::mouse_event(0x0001, -2, -1, 0, [IntPtr]::Zero) | Out-Null
Start-Sleep -Milliseconds 1800
Write-Host "[4] Capture hover tooltip -> 02-tooltip.png"
Capture-Window (Join-Path $OutDir '02-tooltip.png')
# Bouge la souris hors du tableau pour cacher le tooltip
[W32]::SetCursorPos(($rect.Left + 100), ($rect.Top + 100)) | Out-Null
[W32]::mouse_event(0x0001, 0, 0, 0, [IntPtr]::Zero) | Out-Null
Start-Sleep -Milliseconds 500

# === 4. Click bouton lang-toggle (bascule la langue) ===
# Fait avant le modal Score pour eviter interference
Write-Host "[5] Click btn-lang-toggle"
$clicked = Click-ById -id "btn-lang-toggle"
Start-Sleep -Milliseconds 1500
Write-Host "[6] Capture other-language UI -> 06-language-toggle.png"
Capture-Window (Join-Path $OutDir '06-language-toggle.png')
# Re-toggle pour revenir a la langue d'origine
$null = Click-ById -id "btn-lang-toggle"
Start-Sleep -Milliseconds 1000

# === 5. Click Score pour ouvrir le modal maturity ===
Write-Host "[7] Click btn-maturity"
$clicked = Click-ById -id "btn-maturity"
if ($clicked) {
    Start-Sleep -Milliseconds 1500
    Write-Host "[8] Capture score modal -> 04-maturity-score.png"
    Capture-Window (Join-Path $OutDir '04-maturity-score.png')
} else {
    Copy-Item (Join-Path $OutDir '01-dashboard.png') (Join-Path $OutDir '04-maturity-score.png') -Force
}

# === 6. Drift banner — non reproductible en live ===
# (besoin d'un drift detecte par une scheduled task post-WU)
# Strategie : on ne peut pas le forcer ici. Placeholder = dashboard.
# Sera ecrase par le mock Edge headless dans un second script.
if (-not (Test-Path (Join-Path $OutDir '05-drift-banner.png'))) {
    Copy-Item (Join-Path $OutDir '01-dashboard.png') (Join-Path $OutDir '05-drift-banner.png') -Force
    Write-Host "  placeholder 05-drift-banner.png (drift banner not reproducible live)" -ForegroundColor Yellow
}

# === Cleanup ===
Write-Host "Stopping GUI..."
Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
Get-Process harden-gui -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "=== Done ===" -ForegroundColor Green
Get-ChildItem $OutDir -Filter *.png | Format-Table Name, Length
