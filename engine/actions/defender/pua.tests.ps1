# Tests Pester 5 pour pua.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'pua.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'pua.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'pua.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'pua.test.ps1' {
    It 'compliant=true when PUAProtection is 1 (Enabled)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ PUAProtection = [byte]1 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.PUAProtection | Should -Be 'Enabled'
    }

    It 'compliant=false when PUAProtection is 0 (Disabled)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ PUAProtection = [byte]0 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.PUAProtection | Should -Be 'Disabled'
    }
}

Describe 'pua.action.ps1' {
    It 'calls Set-MpPreference -PUAProtection Enabled' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ PUAProtection = [byte]0 }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$PUAProtection" -eq 'Enabled' }
    }
}

Describe 'pua.undo.ps1' {
    It 'restores PUAProtection from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"PUAProtection":"Disabled"}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$PUAProtection" -eq 'Disabled' }
    }
}
