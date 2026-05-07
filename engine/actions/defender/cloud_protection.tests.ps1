# Tests Pester 5 pour cloud_protection.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'cloud_protection.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'cloud_protection.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'cloud_protection.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'cloud_protection.test.ps1' {
    It 'compliant=true when MAPS=2 (Advanced), Block=4 (High), Timeout=50' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{
                MAPSReporting        = [byte]2
                CloudBlockLevel      = [byte]4
                CloudExtendedTimeout = 50
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.MAPSReporting | Should -Be 'Advanced'
        $output.current.CloudBlockLevel | Should -Be 'High'
        $output.current.CloudExtendedTimeout | Should -Be 50
    }

    It 'compliant=false when CloudBlockLevel is 0 (Default)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{
                MAPSReporting        = [byte]2
                CloudBlockLevel      = [byte]0
                CloudExtendedTimeout = 50
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }

    It 'compliant=false when CloudExtendedTimeout is 0' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{
                MAPSReporting        = [byte]2
                CloudBlockLevel      = [byte]4
                CloudExtendedTimeout = 0
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'cloud_protection.action.ps1' {
    It 'sets MAPSReporting=Advanced, CloudBlockLevel=High, CloudExtendedTimeout=50' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{
                MAPSReporting        = [byte]1
                CloudBlockLevel      = [byte]0
                CloudExtendedTimeout = 0
            }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$MAPSReporting" -eq 'Advanced' }
        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$CloudBlockLevel" -eq 'High' }
        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $CloudExtendedTimeout -eq 50 }
    }
}

Describe 'cloud_protection.undo.ps1' {
    It 'restores all 3 fields from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $json = '{"MAPSReporting":"Basic","CloudBlockLevel":"Moderate","CloudExtendedTimeout":10}'
        $output = $json | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$MAPSReporting" -eq 'Basic' }
        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$CloudBlockLevel" -eq 'Moderate' }
        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $CloudExtendedTimeout -eq 10 }
    }
}
