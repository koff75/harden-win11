# Pester tests pour _helpers/appx.psm1.
#
# Couvre indirectement les 27 règles bloatware qui consomment Invoke-AppxTest
# et Invoke-AppxRemove via leur pattern respectif (ex: 'Microsoft.BingNews*').
#
# Note Pester 5 + module : les mocks Get-AppxPackage / Get-AppxProvisionedPackage
# sont appliqués DANS le scope du module avec -ModuleName 'appx', sinon les
# fonctions du module appellent les vrais cmdlets et le test plante.

BeforeAll {
    Import-Module (Join-Path $PSScriptRoot 'harden_appx.psm1') -Force

    # Stubs : Get-AppxPackage et Get-AppxProvisionedPackage n'existent pas
    # forcément dans la session de test (PowerShell 7 sans modules Appx).
    # On définit des fonctions stub si absentes pour pouvoir les Mock.
    if (-not (Get-Command Get-AppxPackage -ErrorAction SilentlyContinue)) {
        function Get-AppxPackage { param($Name, [switch]$AllUsers) }
    }
    if (-not (Get-Command Get-AppxProvisionedPackage -ErrorAction SilentlyContinue)) {
        function Get-AppxProvisionedPackage { param([switch]$Online) }
    }
    if (-not (Get-Command Remove-AppxPackage -ErrorAction SilentlyContinue)) {
        function Remove-AppxPackage { param($Package, [switch]$AllUsers) }
    }
    if (-not (Get-Command Remove-AppxProvisionedPackage -ErrorAction SilentlyContinue)) {
        function Remove-AppxProvisionedPackage { param([switch]$Online, $PackageName) }
    }
}

Describe 'Get-AppxByPattern' {
    It 'returns installed list when -AllUsers succeeds' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith {
            @(
                [PSCustomObject]@{ Name = 'Microsoft.BingNews'; PackageFullName = 'Microsoft.BingNews_4.55.0.0_x64' }
            )
        }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }

        $r = Get-AppxByPattern -Pattern 'Microsoft.BingNews*'
        $r.Installed.Count | Should -Be 1
        $r.Provisioned.Count | Should -Be 0
        $r.Total | Should -Be 1
        $r.Partial | Should -Be $false
    }

    It 'fallbacks to non-AllUsers when -AllUsers throws Access Denied' {
        $callCount = [ref]0
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith {
            $callCount.Value++
            if ($AllUsers) { throw 'Access is denied' }
            @([PSCustomObject]@{ Name = 'Microsoft.BingNews'; PackageFullName = 'Microsoft.BingNews_x' })
        }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }

        $r = Get-AppxByPattern -Pattern 'Microsoft.BingNews*'
        $r.Installed.Count | Should -Be 1
        $r.Partial | Should -Be $true
        $callCount.Value | Should -BeGreaterThan 1
    }

    It 'returns empty list and Partial=true when both Get-AppxPackage calls throw' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith { throw 'denied' }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }

        $r = Get-AppxByPattern -Pattern 'Microsoft.X*'
        $r.Installed.Count | Should -Be 0
        $r.Total | Should -Be 0
        $r.Partial | Should -Be $true
    }

    It 'sets Partial=true when Get-AppxProvisionedPackage throws' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith { @() }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { throw 'denied' }

        $r = Get-AppxByPattern -Pattern 'Microsoft.X*'
        $r.Partial | Should -Be $true
    }
}

Describe 'Invoke-AppxTest' {
    It 'returns compliant=true when nothing is installed' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith { @() }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }

        $out = Invoke-AppxTest -Pattern 'Microsoft.NotInstalled*' | ConvertFrom-Json
        $out.compliant | Should -Be $true
        $out.current.AppxInstalled | Should -Be 0
        $out.current.ProvisionedInstalled | Should -Be 0
    }

    It 'returns compliant=false when an Appx package is installed' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith {
            @([PSCustomObject]@{ Name = 'Microsoft.BingNews'; PackageFullName = 'Microsoft.BingNews_x' })
        }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }

        $out = Invoke-AppxTest -Pattern 'Microsoft.BingNews*' | ConvertFrom-Json
        $out.compliant | Should -Be $false
        $out.current.AppxInstalled | Should -Be 1
        $out.current.InstalledNames | Should -Contain 'Microsoft.BingNews'
    }

    It 'returns compliant=false when only Provisioned is present' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith { @() }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith {
            @([PSCustomObject]@{ DisplayName = 'Microsoft.BingNews'; PackageName = 'Microsoft.BingNews_4.55_neutral' })
        }

        $out = Invoke-AppxTest -Pattern 'Microsoft.BingNews*' | ConvertFrom-Json
        $out.compliant | Should -Be $false
        $out.current.ProvisionedInstalled | Should -Be 1
    }

    It 'reflects PartialScan flag when Get-AppxPackage degrades to non-AllUsers' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith {
            if ($AllUsers) { throw 'Access denied' }
            @()
        }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }

        $out = Invoke-AppxTest -Pattern 'Microsoft.X*' | ConvertFrom-Json
        $out.current.PartialScan | Should -Be $true
    }
}

Describe 'Invoke-AppxRemove' {
    It 'removes installed Appx and lists them in after.Removed' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith {
            @([PSCustomObject]@{ Name = 'Microsoft.BingNews'; PackageFullName = 'Microsoft.BingNews_x' })
        }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }
        Mock -ModuleName harden_appx Remove-AppxPackage -MockWith { } -Verifiable
        Mock -ModuleName harden_appx Remove-AppxProvisionedPackage -MockWith { }

        $out = Invoke-AppxRemove -Pattern 'Microsoft.BingNews*' | ConvertFrom-Json
        $out.ok | Should -Be $true
        $out.before.AppxInstalled | Should -Be 1
        $out.after.Removed | Should -Contain 'Microsoft.BingNews'

        Should -Invoke -ModuleName harden_appx Remove-AppxPackage -Times 1
    }

    It 'is best-effort : continues when one Remove fails' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith {
            @(
                [PSCustomObject]@{ Name = 'A'; PackageFullName = 'A_x' },
                [PSCustomObject]@{ Name = 'B'; PackageFullName = 'B_x' }
            )
        }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith { @() }
        Mock -ModuleName harden_appx Remove-AppxPackage -MockWith {
            if ($Package -eq 'A_x') { throw 'fail on A' }
        }
        Mock -ModuleName harden_appx Remove-AppxProvisionedPackage -MockWith { }

        $out = Invoke-AppxRemove -Pattern '*' | ConvertFrom-Json
        $out.ok | Should -Be $true
        $out.after.Removed | Should -Contain 'B'
        $out.after.Removed | Should -Not -Contain 'A'
    }

    It 'removes Provisioned packages when present' {
        Mock -ModuleName harden_appx Get-AppxPackage -MockWith { @() }
        Mock -ModuleName harden_appx Get-AppxProvisionedPackage -MockWith {
            @([PSCustomObject]@{ DisplayName = 'Microsoft.BingNews'; PackageName = 'Microsoft.BingNews_4.55_neutral' })
        }
        Mock -ModuleName harden_appx Remove-AppxPackage -MockWith { }
        Mock -ModuleName harden_appx Remove-AppxProvisionedPackage -MockWith { } -Verifiable

        $out = Invoke-AppxRemove -Pattern 'Microsoft.BingNews*' | ConvertFrom-Json
        $out.ok | Should -Be $true
        $out.after.Removed | Should -Contain 'prov:Microsoft.BingNews'

        Should -Invoke -ModuleName harden_appx Remove-AppxProvisionedPackage -Times 1
    }
}
