# Tests Pester 5 pour cloud_protection.{action,test,undo}.ps1
#
# Strategie : mocks state-based. Set-MpPreference modifie un $script:mpState
# que Get-MpPreference lit. Necessaire depuis v0.4.0 ou l'action probe
# Get-MpPreference apres Set pour detecter Windows Home (CloudBlockLevel
# accepte silencieusement sans effet).

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'cloud_protection.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'cloud_protection.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'cloud_protection.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue

    $script:mapsToByte  = @{ 'Disabled' = 0; 'Basic' = 1; 'Advanced' = 2 }
    $script:blockToByte = @{ 'Default' = 0; 'Moderate' = 2; 'High' = 4; 'HighPlus' = 6; 'ZeroTolerance' = 8 }
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

Describe 'cloud_protection.action.ps1 (Enterprise/Pro path)' {
    BeforeEach {
        # Etat initial mutable : valeurs avant apply.
        $script:mpState = [PSCustomObject]@{
            MAPSReporting        = [byte]1   # Basic
            CloudBlockLevel      = [byte]0   # Default
            CloudExtendedTimeout = 0
        }

        Mock -CommandName Get-MpPreference -MockWith {
            # Retourne un nouveau snapshot a chaque appel pour eviter les surprises.
            [PSCustomObject]@{
                MAPSReporting        = $script:mpState.MAPSReporting
                CloudBlockLevel      = $script:mpState.CloudBlockLevel
                CloudExtendedTimeout = $script:mpState.CloudExtendedTimeout
            }
        }

        Mock -CommandName Set-MpPreference -MockWith {
            if ($PSBoundParameters.ContainsKey('MAPSReporting')) {
                $v = $PSBoundParameters['MAPSReporting']
                $byte = if ($v -is [string]) { [byte]$script:mapsToByte[$v] } else { [byte]$v }
                $script:mpState.MAPSReporting = $byte
            }
            if ($PSBoundParameters.ContainsKey('CloudBlockLevel')) {
                $v = $PSBoundParameters['CloudBlockLevel']
                $byte = if ($v -is [string]) { [byte]$script:blockToByte[$v] } else { [byte]$v }
                $script:mpState.CloudBlockLevel = $byte
            }
            if ($PSBoundParameters.ContainsKey('CloudExtendedTimeout')) {
                $script:mpState.CloudExtendedTimeout = [int]$PSBoundParameters['CloudExtendedTimeout']
            }
        }
    }

    It 'returns ok=true and sets MAPS=Advanced, Block=High, Timeout=50' {
        $output = & $ActionScript | ConvertFrom-Json

        $output.ok | Should -Be $true
        $script:mpState.MAPSReporting | Should -Be 2
        $script:mpState.CloudBlockLevel | Should -Be 4
        $script:mpState.CloudExtendedTimeout | Should -Be 50

        Should -Invoke -CommandName Set-MpPreference -Times 1 -ParameterFilter { "$MAPSReporting" -eq 'Advanced' }
        Should -Invoke -CommandName Set-MpPreference -Times 1 -ParameterFilter { "$CloudBlockLevel" -eq 'High' }
        Should -Invoke -CommandName Set-MpPreference -Times 1 -ParameterFilter { $CloudExtendedTimeout -eq 50 }
    }
}

Describe 'cloud_protection.action.ps1 (Windows Home path)' {
    BeforeEach {
        # Sur Home, CloudBlockLevel reste a 0 quoi qu'il arrive (accept silencieux).
        $script:mpStateHome = [PSCustomObject]@{
            MAPSReporting        = [byte]1
            CloudBlockLevel      = [byte]0
            CloudExtendedTimeout = 0
        }

        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{
                MAPSReporting        = $script:mpStateHome.MAPSReporting
                CloudBlockLevel      = $script:mpStateHome.CloudBlockLevel
                CloudExtendedTimeout = $script:mpStateHome.CloudExtendedTimeout
            }
        }

        # Set-MpPreference -CloudBlockLevel = no-op (simulate Home behavior).
        # Les autres prefs marchent normalement, mais on n'arrive jamais a les
        # mettre car l'action bail out apres le probe Block qui echoue.
        Mock -CommandName Set-MpPreference -MockWith {
            # CloudBlockLevel ignored
            if ($PSBoundParameters.ContainsKey('MAPSReporting')) {
                $v = $PSBoundParameters['MAPSReporting']
                $script:mpStateHome.MAPSReporting = if ($v -is [string]) { [byte]$script:mapsToByte[$v] } else { [byte]$v }
            }
            if ($PSBoundParameters.ContainsKey('CloudExtendedTimeout')) {
                $script:mpStateHome.CloudExtendedTimeout = [int]$PSBoundParameters['CloudExtendedTimeout']
            }
        }

        Mock -CommandName Get-CimInstance -MockWith {
            [PSCustomObject]@{ Caption = 'Microsoft Windows 11 Home' }
        }
    }

    It 'returns ok=false with clear error mentioning MDE / Pro requirement' {
        $output = & $ActionScript | ConvertFrom-Json

        $output.ok | Should -Be $false
        $output.error | Should -Match 'CloudBlockLevel'
        $output.error | Should -Match 'Pro|MDE|Enterprise'

        # Should NOT have touched MAPSReporting nor CloudExtendedTimeout
        # (bail out before they're set).
        $script:mpStateHome.MAPSReporting | Should -Be 1
        $script:mpStateHome.CloudExtendedTimeout | Should -Be 0
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
