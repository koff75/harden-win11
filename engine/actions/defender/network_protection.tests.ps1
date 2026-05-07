# Tests Pester 5 pour network_protection.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'network_protection.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'network_protection.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'network_protection.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'network_protection.test.ps1' {
    It 'compliant=true when EnableNetworkProtection is 1 (Enabled)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ EnableNetworkProtection = [byte]1 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.EnableNetworkProtection | Should -Be 'Enabled'
    }

    It 'compliant=false when EnableNetworkProtection is 0 (Disabled)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ EnableNetworkProtection = [byte]0 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.EnableNetworkProtection | Should -Be 'Disabled'
    }
}

Describe 'network_protection.action.ps1' {
    It 'calls Set-MpPreference -EnableNetworkProtection Enabled' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ EnableNetworkProtection = [byte]0 }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$EnableNetworkProtection" -eq 'Enabled' }
    }
}

Describe 'network_protection.undo.ps1' {
    It 'restores EnableNetworkProtection from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"EnableNetworkProtection":"Disabled"}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$EnableNetworkProtection" -eq 'Disabled' }
    }
}
