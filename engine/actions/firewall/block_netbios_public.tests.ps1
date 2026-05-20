# Tests Pester 5 pour block_netbios_public.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'block_netbios_public.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'block_netbios_public.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'block_netbios_public.undo.ps1'

    Import-Module NetSecurity -ErrorAction SilentlyContinue
}

# Note Pester 5 : les params nommes sont accessibles via $PesterBoundParameters
# (pas $PSBoundParameters qui est vide dans le mock body).

Describe 'block_netbios_public.test.ps1' {
    It 'compliant=true when both UDP and TCP rules exist and are Enabled' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            @(
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
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.udp.exists | Should -Be $true
        $output.current.tcp.exists | Should -Be $true
    }

    It 'compliant=false when UDP rule missing' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            @(
                [PSCustomObject]@{
                    Name        = 'guid-tcp'
                    DisplayName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'
                    Enabled     = 'True'
                }
            )
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.udp.exists | Should -Be $false
        $output.current.tcp.exists | Should -Be $true
    }

    It 'compliant=false when both rules missing' {
        Mock -CommandName Get-NetFirewallRule -MockWith { @() }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'block_netbios_public.action.ps1' {
    It 'creates both UDP and TCP rules when neither present' {
        # Get sans args (Where-Object filter) : rien avant New.
        # Get -Name <guid> apres New : retourne la rule fabriquee.
        Mock -CommandName Get-NetFirewallRule -MockWith {
            if ($PesterBoundParameters.ContainsKey('Name')) {
                [PSCustomObject]@{
                    Name        = $PesterBoundParameters['Name']
                    DisplayName = 'whatever'
                    Enabled     = 'True'
                }
            } else {
                @()
            }
        }
        Mock -CommandName Remove-NetFirewallRule -MockWith { }
        Mock -CommandName New-NetFirewallRule -MockWith {
            [PSCustomObject]@{
                Name        = ('new-' + [Guid]::NewGuid().ToString())
                DisplayName = $PesterBoundParameters['DisplayName']
                Enabled     = 'True'
            }
        } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName New-NetFirewallRule -Times 2
        Should -Invoke -CommandName New-NetFirewallRule -Times 1 `
            -ParameterFilter { "$Protocol" -eq 'UDP' -and (@($LocalPort) -contains 137) -and (@($LocalPort) -contains 138) }
        Should -Invoke -CommandName New-NetFirewallRule -Times 1 `
            -ParameterFilter { "$Protocol" -eq 'TCP' -and ($LocalPort -eq 139) }
        Should -Invoke -CommandName Remove-NetFirewallRule -Times 0
    }
}

Describe 'block_netbios_public.undo.ps1' {
    It 'removes both rules when present' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            @(
                [PSCustomObject]@{
                    Name        = 'guid-udp'
                    DisplayName = 'Block NetBIOS UDP Inbound (Public) [Hardening]'
                },
                [PSCustomObject]@{
                    Name        = 'guid-tcp'
                    DisplayName = 'Block NetBIOS TCP Inbound (Public) [Hardening]'
                }
            )
        }
        Mock -CommandName Remove-NetFirewallRule -MockWith { } -Verifiable

        $output = '{}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        # undo.ps1 fait 2 boucles foreach (UDP, TCP). Pour chaque, il Where-
        # Object filtre la liste mockee par DisplayName. UDP match 1 rule, TCP
        # match 1 rule. Donc 2 appels Remove au total.
        Should -Invoke -CommandName Remove-NetFirewallRule -Times 2
    }
}
