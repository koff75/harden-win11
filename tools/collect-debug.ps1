# collect-debug.ps1 — bundle anonymise pour analyse post-mortem d'une session.
#
# Usage :
#   .\tools\collect-debug.ps1              # bundle dans .\debug-bundles\
#   .\tools\collect-debug.ps1 -Open        # ouvre l'explorateur sur le ZIP
#   .\tools\collect-debug.ps1 -Runs 5      # garde les 5 derniers runs NDJSON (default 3)
#   .\tools\collect-debug.ps1 -NoAnonymize # garde les paths/usernames bruts (deconseille)
#
# Le bundle contient :
#   - gui.log (le dernier, tronque aux 5000 dernieres lignes)
#   - les N derniers runs NDJSON (N=3 par defaut)
#   - sortie de `harden-engine version|validate|coverage`
#   - sysinfo.json : edition Win11, build, RAM, CPU, admin oui/non
#   - bundle-info.txt : version git, repo, date
#
# Anonymisation par defaut : remplace $env:USERNAME par <user>, le nom du PC
# par <pc>, et les paths C:\Users\* par C:\Users\<user>\.

[CmdletBinding()]
param(
    [int]$Runs = 3,
    [switch]$Open,
    [switch]$NoAnonymize
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot

# --- Resolution de l'exe ----------------------------------------------------
# On cherche dans cet ordre : dist/ du repo (dev), build/bin/ (Wails build),
# puis PATH (release installee).
function Find-HardenEngine {
    $candidates = @(
        (Join-Path $repoRoot 'dist\harden-engine.exe'),
        (Join-Path $repoRoot 'cmd\harden-gui\build\bin\harden-engine.exe')
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) { return $c }
    }
    $onPath = (Get-Command harden-engine.exe -ErrorAction SilentlyContinue).Source
    if ($onPath) { return $onPath }
    return $null
}

# --- Setup destination ------------------------------------------------------
$stamp = Get-Date -Format 'yyyy-MM-dd-HHmm'
$bundleName = "debug-bundle-$stamp"
$outDir = Join-Path $repoRoot 'debug-bundles'
$stagingDir = Join-Path $outDir $bundleName

New-Item -ItemType Directory -Path $stagingDir -Force | Out-Null
Write-Host "[collect-debug] staging : $stagingDir" -ForegroundColor Cyan

# --- Anonymisation helper ---------------------------------------------------
$username = $env:USERNAME
$hostname = $env:COMPUTERNAME
function Sanitize-Text {
    param([string]$text)
    if ($NoAnonymize -or [string]::IsNullOrEmpty($text)) { return $text }
    # Ordre important : USERNAME peut apparaitre dans des paths, donc on le
    # remplace en premier (les paths Users\X seront deja substitues).
    $out = $text -replace [regex]::Escape($username), '<user>'
    $out = $out -replace [regex]::Escape($hostname), '<pc>'
    return $out
}

function Copy-Anonymized {
    param([string]$src, [string]$dst, [int]$tailLines = 0)
    if (-not (Test-Path $src)) { return $false }
    if ($tailLines -gt 0) {
        $content = Get-Content $src -Tail $tailLines -ErrorAction SilentlyContinue
    } else {
        $content = Get-Content $src -ErrorAction SilentlyContinue
    }
    if ($null -eq $content) { return $false }
    $joined = ($content -join "`r`n")
    $sanitized = Sanitize-Text $joined
    Set-Content -Path $dst -Value $sanitized -Encoding UTF8
    return $true
}

# --- 1. gui.log -------------------------------------------------------------
$guiLogPath = Join-Path $env:LOCALAPPDATA 'Harden-Win11\gui.log'
$guiLogOut = Join-Path $stagingDir 'gui.log'
if (Copy-Anonymized -src $guiLogPath -dst $guiLogOut -tailLines 5000) {
    Write-Host "  + gui.log (tail 5000 lignes)" -ForegroundColor Green
} else {
    Write-Host "  - gui.log absent ($guiLogPath)" -ForegroundColor Yellow
}

# --- 2. Runs NDJSON ---------------------------------------------------------
$runsDir = Join-Path $env:ProgramData 'Harden-Win11\runs'
$runsOutDir = Join-Path $stagingDir 'runs'
New-Item -ItemType Directory -Path $runsOutDir -Force | Out-Null
if (Test-Path $runsDir) {
    $latestRuns = Get-ChildItem $runsDir -Filter *.ndjson |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First $Runs
    foreach ($r in $latestRuns) {
        $dst = Join-Path $runsOutDir $r.Name
        Copy-Anonymized -src $r.FullName -dst $dst | Out-Null
        Write-Host "  + run $($r.Name) ($([math]::Round($r.Length / 1KB, 1)) KB)" -ForegroundColor Green
    }
    if ($latestRuns.Count -eq 0) {
        Write-Host "  - aucun NDJSON dans $runsDir" -ForegroundColor Yellow
    }
} else {
    Write-Host "  - dossier runs absent ($runsDir)" -ForegroundColor Yellow
}

# --- 3. Engine output -------------------------------------------------------
$engine = Find-HardenEngine
$engineOutDir = Join-Path $stagingDir 'engine'
New-Item -ItemType Directory -Path $engineOutDir -Force | Out-Null
if ($engine) {
    Write-Host "  + harden-engine : $engine" -ForegroundColor Green
    # PS 5.1 wrap chaque ligne stderr d'un native exe en NativeCommandError.
    # Avec $ErrorActionPreference='Stop' (top du script), ca leve une vraie
    # exception meme avec 2>$null. Scope local en 'Continue' + 2>&1 pour
    # capturer stderr dans le fichier output.
    foreach ($cmd in @('version', 'validate', 'coverage')) {
        $outFile = Join-Path $engineOutDir "$cmd.txt"
        $prevEAP = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            $output = & $engine $cmd 2>&1 | Out-String
            $exit = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $prevEAP
        }
        $sanitized = Sanitize-Text $output
        $header = "# harden-engine $cmd (exit=$exit)`r`n`r`n"
        Set-Content -Path $outFile -Value ($header + $sanitized) -Encoding UTF8
        if ($exit -eq 0) {
            Write-Host "    -> $cmd OK" -ForegroundColor DarkGray
        } else {
            Write-Host "    -> $cmd exit $exit" -ForegroundColor Yellow
        }
    }
} else {
    Write-Host "  - harden-engine.exe introuvable (lance 'go build ...' d'abord)" -ForegroundColor Yellow
}

# --- 4. System info (anonymise) --------------------------------------------
# Pas d'info user-identifiable : OS, build, RAM, CPU, admin oui/non. Pas de
# nom de machine, pas de domaine, pas d'IP, pas de timezone offset (qui peut
# leak la geo grossierement).
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)

$os = Get-CimInstance Win32_OperatingSystem
$cs = Get-CimInstance Win32_ComputerSystem
$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1

$sysinfo = @{
    timestamp_utc  = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    os_caption     = $os.Caption
    os_build       = $os.BuildNumber
    os_version     = $os.Version
    os_arch        = $os.OSArchitecture
    ram_gb         = [math]::Round($cs.TotalPhysicalMemory / 1GB, 1)
    cpu_name       = $cpu.Name
    cpu_cores      = $cpu.NumberOfCores
    is_admin       = $isAdmin
    ps_version     = $PSVersionTable.PSVersion.ToString()
    powershell_edition = $PSVersionTable.PSEdition
}
$sysinfo | ConvertTo-Json -Depth 3 | Out-File (Join-Path $stagingDir 'sysinfo.json') -Encoding UTF8
Write-Host "  + sysinfo.json (admin=$isAdmin, $($os.Caption) build $($os.BuildNumber))" -ForegroundColor Green

# --- 5. Bundle info (repo state) --------------------------------------------
$gitSha = ''
$gitBranch = ''
try {
    Push-Location $repoRoot
    $gitSha = (git rev-parse HEAD 2>$null)
    $gitBranch = (git rev-parse --abbrev-ref HEAD 2>$null)
} catch {} finally { Pop-Location }

$info = @"
Harden-Win11 debug bundle
=========================
Generated  : $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss zzz')
Git branch : $gitBranch
Git SHA    : $gitSha
Anonymized : $(-not $NoAnonymize)
Runs kept  : $Runs

Contents :
  gui.log         tail 5000 lignes du log GUI
  runs/*.ndjson   les $Runs derniers runs (journal events)
  engine/*.txt    sortie de harden-engine version|validate|coverage
  sysinfo.json    OS, RAM, CPU, admin yes/no (rien d'identifiable)

Pour analyser : envoie ce ZIP a Claude / partage en chat.
"@
$info | Out-File (Join-Path $stagingDir 'bundle-info.txt') -Encoding UTF8

# --- 6. Zip + cleanup --------------------------------------------------------
$zipPath = Join-Path $outDir "$bundleName.zip"
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
Compress-Archive -Path "$stagingDir\*" -DestinationPath $zipPath -CompressionLevel Optimal
Remove-Item $stagingDir -Recurse -Force

$zipSizeKB = [math]::Round((Get-Item $zipPath).Length / 1KB, 1)
Write-Host ""
Write-Host "[collect-debug] OK -> $zipPath ($zipSizeKB KB)" -ForegroundColor Cyan

if ($Open) {
    Start-Process explorer.exe "/select,`"$zipPath`""
}
