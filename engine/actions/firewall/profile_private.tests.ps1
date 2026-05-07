# Tests Pester 5 pour profile_private.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'profile_private.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'profile_private.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'profile_private.undo.ps1'

    Import-Module NetSecurity -ErrorAction SilentlyContinue
}

Describe 'profile_private.test.ps1' {
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
    }

    It 'compliant=false when DefaultInboundAction is Allow' {
        Mock -CommandName Get-NetFirewallProfile -MockWith {
            [PSCustomObject]@{
                Enabled               = 'True'
                DefaultInboundAction  = 'Allow'
                DefaultOutboundAction = 'Allow'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'profile_private.action.ps1' {
    It 'sets Private profile to Enabled=True, Block inbound, Allow outbound' {
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
                $Profile -eq 'Private' -and
                "$Enabled" -eq 'True' -and
                "$DefaultInboundAction" -eq 'Block'
            }
    }
}

Describe 'profile_private.undo.ps1' {
    It 'restores Private profile from input' {
        Mock -CommandName Set-NetFirewallProfile -MockWith { } -Verifiable

        $json = '{"Enabled":"False","DefaultInboundAction":"Allow","DefaultOutboundAction":"Allow"}'
        $output = $json | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-NetFirewallProfile -Times 1 `
            -ParameterFilter { $Profile -eq 'Private' -and "$Enabled" -eq 'False' }
    }
}
