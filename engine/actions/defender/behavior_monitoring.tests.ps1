# Tests Pester 5 pour behavior_monitoring.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'behavior_monitoring.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'behavior_monitoring.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'behavior_monitoring.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'behavior_monitoring.test.ps1' {
    It 'compliant=true when DisableBehaviorMonitoring is false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableBehaviorMonitoring = $false }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
    }

    It 'compliant=false when DisableBehaviorMonitoring is true' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableBehaviorMonitoring = $true }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'behavior_monitoring.action.ps1' {
    It 'calls Set-MpPreference -DisableBehaviorMonitoring $false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableBehaviorMonitoring = $true }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableBehaviorMonitoring -eq $false }
    }
}

Describe 'behavior_monitoring.undo.ps1' {
    It 'restores DisableBehaviorMonitoring from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"DisableBehaviorMonitoring":true}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableBehaviorMonitoring -eq $true }
    }
}
