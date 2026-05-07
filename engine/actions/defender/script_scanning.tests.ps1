# Tests Pester 5 pour script_scanning.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'script_scanning.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'script_scanning.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'script_scanning.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'script_scanning.test.ps1' {
    It 'compliant=true when DisableScriptScanning is false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableScriptScanning = $false }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
    }

    It 'compliant=false when DisableScriptScanning is true' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableScriptScanning = $true }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'script_scanning.action.ps1' {
    It 'calls Set-MpPreference -DisableScriptScanning $false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableScriptScanning = $true }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableScriptScanning -eq $false }
    }
}

Describe 'script_scanning.undo.ps1' {
    It 'restores DisableScriptScanning from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"DisableScriptScanning":true}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableScriptScanning -eq $true }
    }
}
