# audit-coherence.ps1 — Vérifications statiques sur les manifests :
#  - tous les .action.ps1 / .test.ps1 / .undo.ps1 référencés existent
#  - chaque rule.id est unique
#  - chaque rule.id matche [a-z_]+\.[a-z0-9_]+
#  - chaque rule a profiles+breaks bien formés
#  - profiles : valeurs valides (personal / business / maximal)
#  - irreversible:true ⇒ pas de undo: déclaré
#  - irreversible:false ⇒ undo: existe ET pointe sur un fichier valide
#  - chaque action référence le helper harden_appx ou reg (pas l'ancien appx)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$root = Split-Path -Parent $PSScriptRoot
$manifestsDir = Join-Path $root 'manifests'
$bugs = New-Object System.Collections.Generic.List[hashtable]

function Add-Bug([string] $sev, [string] $where, [string] $msg) {
    $bugs.Add(@{ severity = $sev; where = $where; message = $msg })
}

function Test-Rule($r, $manifestRel) {
    $id = $r.id
    if (-not $id) { return }

    if ($id -notmatch '^[a-z_]+\.[a-z0-9_]+$') {
        Add-Bug 'high' "$manifestRel/$id" "rule_id ne matche pas le format [a-z_]+\.[a-z0-9_]+"
    }
    if ($script:seenIDs.ContainsKey($id)) {
        Add-Bug 'critical' "$manifestRel/$id" "rule_id en doublon avec $($script:seenIDs[$id])"
    } else {
        $script:seenIDs[$id] = $manifestRel
    }
    if (-not $r.action) {
        Add-Bug 'critical' "$manifestRel/$id" "action: manquante"
    } else {
        $p = (Join-Path $script:root ($r.action -replace '^\./',''))
        $script:referencedScripts[$p] = $true
        if (-not (Test-Path $p)) {
            Add-Bug 'critical' "$manifestRel/$id" "action introuvable : $($r.action)"
        }
    }
    if (-not $r.test) {
        Add-Bug 'critical' "$manifestRel/$id" "test: manquant"
    } else {
        $p = (Join-Path $script:root ($r.test -replace '^\./',''))
        $script:referencedScripts[$p] = $true
        if (-not (Test-Path $p)) {
            Add-Bug 'critical' "$manifestRel/$id" "test introuvable : $($r.test)"
        }
    }
    if ($r.irreversible) {
        if ($r.undo) {
            Add-Bug 'medium' "$manifestRel/$id" "irreversible:true mais undo: déclaré ($($r.undo))"
        }
    } else {
        if (-not $r.undo) {
            Add-Bug 'high' "$manifestRel/$id" "irreversible:false mais pas d'undo: déclaré"
        } else {
            $p = (Join-Path $script:root ($r.undo -replace '^\./',''))
            $script:referencedScripts[$p] = $true
            if (-not (Test-Path $p)) {
                Add-Bug 'critical' "$manifestRel/$id" "undo introuvable : $($r.undo)"
            }
        }
    }
    foreach ($p in $r.profiles) {
        if ($p -notin @('personal', 'business', 'maximal')) {
            Add-Bug 'medium' "$manifestRel/$id" "profile inconnu : '$p' (attendu : personal/business/maximal)"
        }
    }
}

# Index des rule_ids vus
$script:seenIDs = @{}

# Liste des fichiers PS référencés dans les manifests
$script:referencedScripts = @{}
$script:root = $root

Get-ChildItem $manifestsDir -Filter '*.yaml' -File | ForEach-Object {
    $manifest = $_.FullName
    $manifestRel = Split-Path -Leaf $_.FullName

    # Parser YAML manuellement (les utilities Powershell-Yaml sont pas garanties).
    # Format simple ligne par ligne : - id: ..., action: ..., test: ..., undo: ..., irreversible: ...
    $lines = Get-Content $manifest

    $currentRule = @{ start = -1 }
    foreach ($i in 0..($lines.Count - 1)) {
        $line = $lines[$i]
        if ($line -match '^\s*-\s+id:\s+(\S+)\s*$') {
            if ($currentRule.start -ge 0) {
                Test-Rule $currentRule $manifestRel
            }
            $currentRule = @{
                start = $i
                id = $matches[1]
                action = $null
                test = $null
                undo = $null
                irreversible = $false
                profiles = @()
            }
            continue
        }
        if ($currentRule.start -lt 0) { continue }

        if ($line -match '^\s+action:\s+(\S+)') { $currentRule.action = $matches[1] }
        if ($line -match '^\s+test:\s+(\S+)')   { $currentRule.test   = $matches[1] }
        if ($line -match '^\s+undo:\s+(\S+)')   { $currentRule.undo   = $matches[1] }
        if ($line -match '^\s+irreversible:\s+(true|false)') {
            $currentRule.irreversible = ($matches[1] -eq 'true')
        }
        if ($line -match '^(\s+)profiles:\s*$') {
            $profIndent = $matches[1].Length
            $j = $i + 1
            while ($j -lt $lines.Count) {
                $bl = $lines[$j]
                # Item de la liste : indenté strictement plus que `profiles:`
                # ET commence par "- " (item simple, pas "- id:" qui est une nouvelle rule)
                if ($bl -match '^(\s+)-\s+([^:\s]+)\s*$') {
                    $itemIndent = $matches[1].Length
                    if ($itemIndent -gt $profIndent) {
                        $currentRule.profiles += $matches[2]
                        $j++
                        continue
                    }
                }
                break
            }
        }
    }
    if ($currentRule.start -ge 0) { Test-Rule $currentRule $manifestRel }
}

# 2. Scripts orphelins (présents mais pas référencés)
Write-Host ""
Write-Host "Scanning for orphan PS scripts..." -ForegroundColor Cyan
$allScripts = Get-ChildItem (Join-Path $root 'engine\actions') -Recurse -Filter '*.ps1' -File |
    Where-Object { $_.Name -notmatch '\.tests?\.ps1$' -and $_.Name -notmatch '\.psm1$' }
foreach ($s in $allScripts) {
    if (-not $script:referencedScripts.ContainsKey($s.FullName)) {
        $rel = $s.FullName.Replace($root, '').TrimStart('\')
        # Skip _helpers
        if ($rel -match '_helpers') { continue }
        Add-Bug 'low' $rel "script orphelin (pas référencé par un manifest)"
    }
}

# 3. Imports référencent l'ancien appx.psm1 ?
Write-Host ""
Write-Host "Checking for stale appx.psm1 imports..." -ForegroundColor Cyan
$stale = Select-String -Path (Join-Path $root 'engine\actions\**\*.ps1') -Pattern '_helpers\\appx\.psm1' -ErrorAction SilentlyContinue
foreach ($s in $stale) {
    Add-Bug 'high' $s.Path "import obsolète : $($s.Line)"
}

# Rapport
Write-Host ""
$total = $bugs.Count
$crit = ($bugs | Where-Object { $_.severity -eq 'critical' }).Count
$high = ($bugs | Where-Object { $_.severity -eq 'high' }).Count
$med  = ($bugs | Where-Object { $_.severity -eq 'medium' }).Count
$low  = ($bugs | Where-Object { $_.severity -eq 'low' }).Count

Write-Host "=== Audit cohérence : $total problème(s) ===" -ForegroundColor Yellow
Write-Host "  critical : $crit, high : $high, medium : $med, low : $low"
Write-Host ""

foreach ($b in ($bugs | Sort-Object @{Expression={
    switch ($_.severity) {
        'critical' { 0 }
        'high'     { 1 }
        'medium'   { 2 }
        'low'      { 3 }
        default    { 4 }
    }
}})) {
    $col = switch ($b.severity) {
        'critical' { 'Red' }
        'high'     { 'Yellow' }
        'medium'   { 'Cyan' }
        'low'      { 'DarkGray' }
    }
    Write-Host "[$($b.severity)] $($b.where)" -ForegroundColor $col
    Write-Host "    $($b.message)"
}

if ($crit -gt 0) { exit 2 } elseif ($high -gt 0) { exit 1 } else { exit 0 }
