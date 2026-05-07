# Tests Pester 5 pour nis.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'nis.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'nis.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'nis.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'nis.test.ps1' {
    It 'compliant=true when DisableIntrusionPreventionSystem is false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableIntrusionPreventionSystem = $false }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
    }

    It 'compliant=false when DisableIntrusionPreventionSystem is true' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableIntrusionPreventionSystem = $true }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'nis.action.ps1' {
    It 'calls Set-MpPreference -DisableIntrusionPreventionSystem $false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableIntrusionPreventionSystem = $true }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableIntrusionPreventionSystem -eq $false }
    }
}

Describe 'nis.undo.ps1' {
    It 'restores DisableIntrusionPreventionSystem from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"DisableIntrusionPreventionSystem":true}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableIntrusionPreventionSystem -eq $true }
    }
}
