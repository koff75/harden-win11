# Tests Pester 5 pour ioav.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'ioav.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'ioav.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'ioav.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'ioav.test.ps1' {
    It 'compliant=true when DisableIOAVProtection is false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableIOAVProtection = $false }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
    }

    It 'compliant=false when DisableIOAVProtection is true' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableIOAVProtection = $true }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'ioav.action.ps1' {
    It 'calls Set-MpPreference -DisableIOAVProtection $false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableIOAVProtection = $true }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableIOAVProtection -eq $false }
    }
}

Describe 'ioav.undo.ps1' {
    It 'restores DisableIOAVProtection from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"DisableIOAVProtection":true}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableIOAVProtection -eq $true }
    }
}
