# Tests Pester 5 pour block_netbios_public.{action,test,undo}.ps1
#
# Strategie : mock state-based (cf. block_smb_public.tests.ps1 pour le
# rationnel). Necessaire depuis v0.4.0 pour le pattern Where-Object
# DisplayName + capture du Name retourne par New-NetFirewallRule.

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'block_netbios_public.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'block_netbios_public.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'block_netbios_public.undo.ps1'

    Import-Module NetSecurity -ErrorAction SilentlyContinue
}

function script:Init-FwMocks {
    param([array]$initialRules = @())
    $script:fwRules = @() + $initialRules

    Mock -CommandName Get-NetFirewallRule -MockWith {
        if ($PSBoundParameters.ContainsKey('Name')) {
            $n = $PSBoundParameters['Name']
            return @($script:fwRules | Where-Object { $_.Name -eq $n })
        }
        return $script:fwRules
    }

    Mock -CommandName New-NetFirewallRule -MockWith {
        $r = [PSCustomObject]@{
            Name        = ('TEST-' + [Guid]::NewGuid().ToString())
            DisplayName = $PSBoundParameters['DisplayName']
            Enabled     = 'True'
        }
        $script:fwRules += $r
        return $r
    }

    Mock -CommandName Remove-NetFirewallRule -MockWith {
        if ($PSBoundParameters.ContainsKey('Name')) {
            $n = $PSBoundParameters['Name']
            $script:fwRules = @($script:fwRules | Where-Object { $_.Name -ne $n })
        }
    }
}

Describe 'block_netbios_public.test.ps1' {
    It 'compliant=true when both UDP and TCP rules exist and are Enabled' {
        Init-FwMocks @(
            [PSCustomObject]@{
                Name        = 'guid-udp'
                DisplayName = 'Block NetBIOS UDP Inbound (Public) [Hardening]'
                Enabled     = 'True'
            },
            [PSCustomObject]@{
                Name        = 'guid-tcp'
                DisplayName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'
                Enabled     = 'True'
            }
        )
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.udp.exists | Should -Be $true
        $output.current.tcp.exists | Should -Be $true
    }

    It 'compliant=false when UDP rule missing' {
        Init-FwMocks @(
            [PSCustomObject]@{
                Name        = 'guid-tcp'
                DisplayName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'
                Enabled     = 'True'
            }
        )
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.udp.exists | Should -Be $false
        $output.current.tcp.exists | Should -Be $true
    }

    It 'compliant=false when both rules missing' {
        Init-FwMocks @()
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'block_netbios_public.action.ps1' {
    It 'creates both UDP and TCP rules when neither present' {
        Init-FwMocks @()

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName New-NetFirewallRule -Times 2
        Should -Invoke -CommandName New-NetFirewallRule -Times 1 `
            -ParameterFilter { "$Protocol" -eq 'UDP' -and (@($LocalPort) -contains 137) -and (@($LocalPort) -contains 138) }
        Should -Invoke -CommandName New-NetFirewallRule -Times 1 `
            -ParameterFilter { "$Protocol" -eq 'TCP' -and ($LocalPort -eq 139) }
        Should -Invoke -CommandName Remove-NetFirewallRule -Times 0
        $script:fwRules.Count | Should -Be 2
    }
}

Describe 'block_netbios_public.undo.ps1' {
    It 'removes both rules when present' {
        Init-FwMocks @(
            [PSCustomObject]@{
                Name        = 'guid-udp'
                DisplayName = 'Block NetBIOS UDP Inbound (Public) [Hardening]'
            },
            [PSCustomObject]@{
                Name        = 'guid-tcp'
                DisplayName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'
            }
        )

        $output = '{}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 2
        $script:fwRules.Count | Should -Be 0
    }
}
