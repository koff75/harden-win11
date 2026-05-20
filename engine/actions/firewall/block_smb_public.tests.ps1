# Tests Pester 5 pour block_smb_public.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'block_smb_public.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'block_smb_public.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'block_smb_public.undo.ps1'

    Import-Module NetSecurity -ErrorAction SilentlyContinue
}

Describe 'block_smb_public.test.ps1' {
    It 'compliant=true when rule exists and is Enabled' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            [PSCustomObject]@{
                Name        = 'guid-1'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                Enabled     = 'True'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
    }

    It 'compliant=false when rule does not exist' {
        Mock -CommandName Get-NetFirewallRule -MockWith { @() }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.exists | Should -Be $false
    }

    It 'compliant=false when rule exists but is disabled' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            [PSCustomObject]@{
                Name        = 'guid-1'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                Enabled     = 'False'
            }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'block_smb_public.action.ps1' {
    It 'creates rule when not present' {
        # Pester 5 : les params nommes sont accessibles via $PesterBoundParameters
        # (pas $PesterBoundParameters qui est vide dans le body). Source :
        # github.com/pester/Pester/issues/2154
        Mock -CommandName Get-NetFirewallRule -MockWith {
            if ($PesterBoundParameters.ContainsKey('Name')) {
                [PSCustomObject]@{
                    Name        = $PesterBoundParameters['Name']
                    DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                    Enabled     = 'True'
                }
            } else {
                @()
            }
        }
        Mock -CommandName Remove-NetFirewallRule -MockWith { }
        Mock -CommandName New-NetFirewallRule -MockWith {
            [PSCustomObject]@{
                Name        = 'new-guid-abc'
                DisplayName = $PesterBoundParameters['DisplayName']
                Enabled     = 'True'
            }
        } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName New-NetFirewallRule -Times 1 `
            -ParameterFilter {
                $DisplayName -eq 'Block SMB Inbound (Public) [Hardening]' -and
                "$Direction" -eq 'Inbound' -and
                "$Protocol" -eq 'TCP' -and
                ($LocalPort -eq 445) -and
                "$Action" -eq 'Block' -and
                "$Profile" -eq 'Public'
            }
        Should -Invoke -CommandName Remove-NetFirewallRule -Times 0
    }

    It 'replaces rule when already present' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            if ($PesterBoundParameters.ContainsKey('Name')) {
                return [PSCustomObject]@{
                    Name        = $PesterBoundParameters['Name']
                    DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                    Enabled     = 'True'
                }
            }
            # Pre-existante.
            return @([PSCustomObject]@{
                Name        = 'old-guid'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                Enabled     = 'True'
            })
        }
        Mock -CommandName Remove-NetFirewallRule -MockWith { } -Verifiable
        Mock -CommandName New-NetFirewallRule -MockWith {
            return [PSCustomObject]@{
                Name        = 'new-guid'
                DisplayName = $PesterBoundParameters['DisplayName']
                Enabled     = 'True'
            }
        } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 1
        Should -Invoke -CommandName New-NetFirewallRule -Times 1
    }
}

Describe 'block_smb_public.undo.ps1' {
    It 'removes rule when present' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            [PSCustomObject]@{
                Name        = 'guid-1'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
            }
        }
        Mock -CommandName Remove-NetFirewallRule -MockWith { } -Verifiable

        $output = '{}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 1
    }

    It 'does nothing when rule absent' {
        Mock -CommandName Get-NetFirewallRule -MockWith { @() }
        Mock -CommandName Remove-NetFirewallRule -MockWith { }

        $output = '{}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 0
    }
}
