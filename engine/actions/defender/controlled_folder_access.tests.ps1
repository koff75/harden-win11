# Tests Pester 5 pour controlled_folder_access.{action,test,undo}.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'controlled_folder_access.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'controlled_folder_access.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'controlled_folder_access.undo.ps1'

    Import-Module Defender -ErrorAction SilentlyContinue
}

Describe 'controlled_folder_access.test.ps1' {
    It 'compliant=true when EnableControlledFolderAccess is 1 (Enabled)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ EnableControlledFolderAccess = [byte]1 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.EnableControlledFolderAccess | Should -Be 'Enabled'
    }

    It 'compliant=false when EnableControlledFolderAccess is 0 (Disabled)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ EnableControlledFolderAccess = [byte]0 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.EnableControlledFolderAccess | Should -Be 'Disabled'
    }

    It 'compliant=false when EnableControlledFolderAccess is 2 (AuditMode)' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ EnableControlledFolderAccess = [byte]2 }
        }
        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.EnableControlledFolderAccess | Should -Be 'AuditMode'
    }
}

Describe 'controlled_folder_access.action.ps1' {
    It 'calls Set-MpPreference -EnableControlledFolderAccess Enabled' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ EnableControlledFolderAccess = [byte]0 }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$EnableControlledFolderAccess" -eq 'Enabled' }
    }
}

Describe 'controlled_folder_access.undo.ps1' {
    It 'restores EnableControlledFolderAccess from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"EnableControlledFolderAccess":"Disabled"}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { "$EnableControlledFolderAccess" -eq 'Disabled' }
    }
}
