# Tests Pester 5 pour signatures.{action,test}.ps1 (pas d'undo : irreversible)

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'signatures.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'signatures.test.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'signatures.test.ps1' {
    It 'compliant=true when signatures are 1 hour old' {
        $recent = (Get-Date).AddHours(-1)
        Mock -CommandName Get-MpComputerStatus -MockWith {
            [PSCustomObject]@{
                AntivirusSignatureLastUpdated = $recent
                AntivirusSignatureVersion     = '1.400.0.0'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.AntivirusSignatureVersion | Should -Be '1.400.0.0'
    }

    It 'compliant=false when signatures are 30 days old' {
        $old = (Get-Date).AddDays(-30)
        Mock -CommandName Get-MpComputerStatus -MockWith {
            [PSCustomObject]@{
                AntivirusSignatureLastUpdated = $old
                AntivirusSignatureVersion     = '1.300.0.0'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'signatures.action.ps1' {
    It 'calls Update-MpSignature' {
        $now = Get-Date
        Mock -CommandName Get-MpComputerStatus -MockWith {
            [PSCustomObject]@{
                AntivirusSignatureLastUpdated = $now
                AntivirusSignatureVersion     = '1.400.0.0'
            }
        }
        Mock -CommandName Update-MpSignature -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Update-MpSignature -Times 1
    }
}
