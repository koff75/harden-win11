# Tests Pester 5 pour realtime.action.ps1, realtime.test.ps1, realtime.undo.ps1
# Lancer : Invoke-Pester engine/actions/defender/realtime.tests.ps1 -Output Detailed

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'realtime.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'realtime.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'realtime.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'realtime.test.ps1' {
    It 'returns compliant=true when DisableRealtimeMonitoring is false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableRealtimeMonitoring = $false }
        }

        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.DisableRealtimeMonitoring | Should -Be $false
    }

    It 'returns compliant=false when DisableRealtimeMonitoring is true' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableRealtimeMonitoring = $true }
        }

        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.DisableRealtimeMonitoring | Should -Be $true
    }
}

Describe 'realtime.action.ps1' {
    It 'calls Set-MpPreference -DisableRealtimeMonitoring $false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableRealtimeMonitoring = $true }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableRealtimeMonitoring -eq $false }
    }
}

Describe 'realtime.undo.ps1' {
    It 'restores DisableRealtimeMonitoring from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"DisableRealtimeMonitoring":true}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableRealtimeMonitoring -eq $true }
    }
}
