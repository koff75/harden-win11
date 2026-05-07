# Tests Pester 5 pour profile_public.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'profile_public.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'profile_public.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'profile_public.undo.ps1'

    Import-Module NetSecurity -ErrorAction SilentlyContinue
}

Describe 'profile_public.test.ps1' {
    It 'compliant=true when Enabled=True, Block inbound, Allow outbound' {
        Mock -CommandName Get-NetFirewallProfile -MockWith {
            [PSCustomObject]@{
                Enabled               = 'True'
                DefaultInboundAction  = 'Block'
                DefaultOutboundAction = 'Allow'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.Enabled | Should -Be 'True'
        $output.current.DefaultInboundAction | Should -Be 'Block'
    }

    It 'compliant=false when DefaultInboundAction is NotConfigured' {
        Mock -CommandName Get-NetFirewallProfile -MockWith {
            [PSCustomObject]@{
                Enabled               = 'True'
                DefaultInboundAction  = 'NotConfigured'
                DefaultOutboundAction = 'NotConfigured'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.DefaultInboundAction | Should -Be 'NotConfigured'
    }

    It 'compliant=false when Enabled is False' {
        Mock -CommandName Get-NetFirewallProfile -MockWith {
            [PSCustomObject]@{
                Enabled               = 'False'
                DefaultInboundAction  = 'Block'
                DefaultOutboundAction = 'Allow'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'profile_public.action.ps1' {
    It 'sets Public profile to Enabled=True, Block inbound, Allow outbound' {
        Mock -CommandName Get-NetFirewallProfile -MockWith {
            [PSCustomObject]@{
                Enabled               = 'False'
                DefaultInboundAction  = 'Allow'
                DefaultOutboundAction = 'Allow'
            }
        }
        Mock -CommandName Set-NetFirewallProfile -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-NetFirewallProfile -Times 1 `
            -ParameterFilter {
                $Profile -eq 'Public' -and
                "$Enabled" -eq 'True' -and
                "$DefaultInboundAction" -eq 'Block' -and
                "$DefaultOutboundAction" -eq 'Allow'
            }
    }
}

Describe 'profile_public.undo.ps1' {
    It 'restores Public profile from input' {
        Mock -CommandName Set-NetFirewallProfile -MockWith { } -Verifiable

        $json = '{"Enabled":"False","DefaultInboundAction":"Allow","DefaultOutboundAction":"Allow"}'
        $output = $json | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-NetFirewallProfile -Times 1 `
            -ParameterFilter {
                $Profile -eq 'Public' -and
                "$Enabled" -eq 'False' -and
                "$DefaultInboundAction" -eq 'Allow'
            }
    }
}
