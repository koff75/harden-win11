# Tests de cohérence structurelle pour les 27 règles bloatware.
#
# Chaque règle est un thin-wrapper de 3 lignes sur Invoke-AppxRemove /
# Invoke-AppxTest (cf. _helpers/harden_appx.psm1, déjà couvert par
# harden_appx.tests.ps1). Plutôt que de dupliquer 81 tests fonctionnels
# identiques, on vérifie ici :
#   - tous les fichiers .action.ps1 ont leur paire .test.ps1
#   - chaque action utilise un pattern non vide via Invoke-AppxRemove
#   - chaque test utilise un pattern non vide via Invoke-AppxTest
#   - 1 test end-to-end "sample" sur bing_news (action+test sur mocks)
#
# Note : pas d'undo.ps1 attendu — les bloatware sont 'irreversible: true'
# (Apps Store désinstallées ne peuvent être réinstallées que via Microsoft
# Store + compte Microsoft).

BeforeAll {
    $script:bloatDir = $PSScriptRoot
    $script:actions = Get-ChildItem $bloatDir -Filter '*.action.ps1' -File

    if (-not (Get-Command Get-AppxPackage -ErrorAction SilentlyContinue)) {
        function Get-AppxPackage { param($Name, [switch]$AllUsers) }
    }
    if (-not (Get-Command Get-AppxProvisionedPackage -ErrorAction SilentlyContinue)) {
        function Get-AppxProvisionedPackage { param([switch]$Online) }
    }
}

Describe 'Bloatware rule files' {
    It 'has 27 action.ps1 files (matches manifests/09-bloatware.yaml)' {
        $script:actions.Count | Should -Be 27
    }

    It 'each action.ps1 has a matching test.ps1' {
        foreach ($a in $script:actions) {
            $base = $a.BaseName -replace '\.action$', ''
            $testPath = Join-Path $bloatDir "$base.test.ps1"
            (Test-Path $testPath) | Should -Be $true -Because "$base.test.ps1 expected"
        }
    }

    It 'each action calls Invoke-AppxRemove with a non-empty Pattern' {
        foreach ($a in $script:actions) {
            $content = Get-Content $a.FullName -Raw
            $content | Should -Match "Invoke-AppxRemove\s+-Pattern\s+'[^']+'"
        }
    }

    It 'each test calls Invoke-AppxTest with a non-empty Pattern' {
        foreach ($a in $script:actions) {
            $base = $a.BaseName -replace '\.action$', ''
            $testFile = Join-Path $bloatDir "$base.test.ps1"
            $content = Get-Content $testFile -Raw
            $content | Should -Match "Invoke-AppxTest\s+-Pattern\s+'[^']+'"
        }
    }
}

# Note : pas de test end-to-end ici (lancer & bing_news.test.ps1 dans Pester
# crée un nouveau runspace où les Mock -ModuleName ne sont pas vus). La
# couverture fonctionnelle est faite par harden_appx.tests.ps1 sur les
# helpers Invoke-AppxTest / Invoke-AppxRemove directement.
