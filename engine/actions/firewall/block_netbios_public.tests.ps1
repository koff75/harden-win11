# Tests Pester 5 pour block_netbios_public.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'block_netbios_public.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'block_netbios_public.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'block_netbios_public.undo.ps1'

    Import-Module NetSecurity -ErrorAction SilentlyContinue
}

Describe 'block_netbios_public.test.ps1' {
    It 'compliant=true when both UDP and TCP rules exist and are Enabled' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            [PSCustomObject]@{ DisplayName = $DisplayName; Enabled = 'True' }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.udp.exists | Should -Be $true
        $output.current.tcp.exists | Should -Be $true
    }

    It 'compliant=false when UDP rule missing' {
        Mock -CommandName Get-NetFirewallRule -MockWith {
            if ($DisplayName -like '*UDP*') { $null }
            else { [PSCustomObject]@{ DisplayName = $DisplayName; Enabled = 'True' } }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.udp.exists | Should -Be $false
        $output.current.tcp.exists | Should -Be $true
    }

    It 'compliant=false when both rules missing' {
        Mock -CommandName Get-NetFirewallRule -MockWith { $null }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }
}

Describe 'block_netbios_public.action.ps1' {
    It 'creates both UDP and TCP rules when neither present' {
        Mock -CommandName Get-NetFirewallRule -MockWith { $null }
        Mock -CommandName Remove-NetFirewallRule -MockWith { }
        Mock -CommandName New-NetFirewallRule -MockWith { } -Verifiable

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
            [PSCustomObject]@{ DisplayName = $DisplayName }
        }
        Mock -CommandName Remove-NetFirewallRule -MockWith { } -Verifiable

        $output = '{}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Remove-NetFirewallRule -Times 2
    }
}
