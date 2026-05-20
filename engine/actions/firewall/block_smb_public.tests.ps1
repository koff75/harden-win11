# Tests Pester 5 pour block_smb_public.{action,test,undo}.ps1
#
# Strategie : mock state-based. $script:fwRules est un array de rules.
# Get-NetFirewallRule (sans args) retourne tout l'array. Get-NetFirewallRule
# -Name <guid> filtre par GUID. New cree une rule + ajoute. Remove enleve
# par Name. Necessaire depuis v0.4.0 pour le pattern Where-Object DisplayName
# + capture du Name via -PassThru.

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'block_smb_public.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'block_smb_public.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'block_smb_public.undo.ps1'

    Import-Module NetSecurity -ErrorAction SilentlyContinue
}

# Helper : monte un set de mocks state-based pour Get/New/Remove-NetFirewallRule.
# Les rules vivent dans $script:fwRules.
function script:Init-FwMocks {
    param([array]$initialRules = @())
    $script:fwRules = @() + $initialRules

    Mock -CommandName Get-NetFirewallRule -MockWith {
        if ($PSBoundParameters.ContainsKey('Name')) {
            $n = $PSBoundParameters['Name']
            return @($script:fwRules | Where-Object { $_.Name -eq $n })
        }
        return $script:fwRules
    } -ModuleName $null

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

Describe 'block_smb_public.test.ps1' {
    It 'compliant=true when rule exists and is Enabled' {
        Init-FwMocks @(
            [PSCustomObject]@{
                Name        = 'guid-1'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                Enabled     = 'True'
            }
        )
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.exists | Should -Be $true
        $output.current.enabled | Should -Be 'True'
    }

    It 'compliant=false when rule does not exist' {
        Init-FwMocks @()
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.exists | Should -Be $false
    }

    It 'compliant=false when rule exists but is disabled' {
        Init-FwMocks @(
            [PSCustomObject]@{
                Name        = 'guid-1'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                Enabled     = 'False'
            }
        )
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'block_smb_public.action.ps1' {
    It 'creates rule when not present' {
        Init-FwMocks @()

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
        $script:fwRules.Count | Should -Be 1
    }

    It 'replaces rule when already present' {
        Init-FwMocks @(
            [PSCustomObject]@{
                Name        = 'old-guid'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
                Enabled     = 'True'
            }
        )

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 1
        Should -Invoke -CommandName New-NetFirewallRule -Times 1
        # Apres apply : 1 rule (la nouvelle, l'ancienne supprimee).
        $script:fwRules.Count | Should -Be 1
    }
}

Describe 'block_smb_public.undo.ps1' {
    It 'removes rule when present' {
        Init-FwMocks @(
            [PSCustomObject]@{
                Name        = 'guid-1'
                DisplayName = 'Block SMB Inbound (Public) [Hardening]'
            }
        )

        $output = '{}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 1
        $script:fwRules.Count | Should -Be 0
    }

    It 'does nothing when rule absent' {
        Init-FwMocks @()

        $output = '{}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 0
    }
}
