# Tests Pester 5 pour sample_submission.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'sample_submission.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'sample_submission.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'sample_submission.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'sample_submission.test.ps1' {
    It 'compliant=true when SubmitSamplesConsent is 1 (SendSafeSamples)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ SubmitSamplesConsent = [byte]1 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.SubmitSamplesConsent | Should -Be 'SendSafeSamples'
    }

    It 'compliant=false when SubmitSamplesConsent is 0 (AlwaysPrompt)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ SubmitSamplesConsent = [byte]0 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }

    It 'compliant=false when SubmitSamplesConsent is 3 (SendAllSamples)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ SubmitSamplesConsent = [byte]3 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.SubmitSamplesConsent | Should -Be 'SendAllSamples'
    }
}

Describe 'sample_submission.action.ps1' {
    It 'calls Set-MpPreference -SubmitSamplesConsent SendSafeSamples' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ SubmitSamplesConsent = [byte]0 }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$SubmitSamplesConsent" -eq 'SendSafeSamples' }
    }
}

Describe 'sample_submission.undo.ps1' {
    It 'restores SubmitSamplesConsent from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"SubmitSamplesConsent":"NeverSend"}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$SubmitSamplesConsent" -eq 'NeverSend' }
    }
}
