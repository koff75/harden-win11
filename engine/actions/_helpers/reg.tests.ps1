# Pester tests pour _helpers/reg.psm1.
#
# Couvre indirectement les 21 règles registry single-value refactorisées qui
# dépendent de Get-RegState / Set-RegValue / Invoke-Reg{Set,Test,Undo}Action.
#
# Note Pester 5 + module : les mocks de Get-ItemProperty / Set-ItemProperty
# doivent être appliqués DANS le scope du module avec -ModuleName 'reg', sinon
# les fonctions du module appellent les vrais cmdlets et le test plante.

BeforeAll {
    Import-Module (Join-Path $PSScriptRoot 'reg.psm1') -Force
}

Describe 'Get-RegState' {
    It 'returns exists=true and value when the key exists' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith {
            [PSCustomObject]@{ MyName = 42 }
        }
        $state = Get-RegState -Path 'HKLM:\Foo' -Name 'MyName'
        $state.exists | Should -Be $true
        $state.value | Should -Be 42
    }

    It 'returns exists=false and value=null when the key is missing' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith { $null }
        $state = Get-RegState -Path 'HKLM:\Foo' -Name 'Missing'
        $state.exists | Should -Be $false
        $state.value | Should -Be $null
    }
}

Describe 'Set-RegValue' {
    It 'creates the registry path if missing then sets the value' {
        Mock -ModuleName reg -CommandName Test-Path -MockWith { $false }
        Mock -ModuleName reg -CommandName New-Item -MockWith { } -Verifiable
        Mock -ModuleName reg -CommandName Set-ItemProperty -MockWith { } -Verifiable

        Set-RegValue -Path 'HKLM:\NewPath' -Name 'X' -Value 1

        Should -Invoke -ModuleName reg -CommandName New-Item -Times 1 -ParameterFilter {
            $Path -eq 'HKLM:\NewPath' -and $Force
        }
        Should -Invoke -ModuleName reg -CommandName Set-ItemProperty -Times 1 -ParameterFilter {
            $Path -eq 'HKLM:\NewPath' -and $Name -eq 'X' -and $Value -eq 1
        }
    }

    It 'skips New-Item if the path already exists' {
        Mock -ModuleName reg -CommandName Test-Path -MockWith { $true }
        Mock -ModuleName reg -CommandName New-Item -MockWith { }
        Mock -ModuleName reg -CommandName Set-ItemProperty -MockWith { } -Verifiable

        Set-RegValue -Path 'HKLM:\Existing' -Name 'X' -Value 1

        Should -Invoke -ModuleName reg -CommandName New-Item -Times 0
        Should -Invoke -ModuleName reg -CommandName Set-ItemProperty -Times 1
    }
}

Describe 'Invoke-RegTestAction' {
    It 'returns compliant=true when current value matches Expected' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith {
            [PSCustomObject]@{ EnableLUA = 1 }
        }
        $output = Invoke-RegTestAction -Path 'HKLM:\Foo' -Name 'EnableLUA' -Expected 1 | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.EnableLUA | Should -Be 1
    }

    It 'returns compliant=false when value differs' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith {
            [PSCustomObject]@{ EnableLUA = 0 }
        }
        $output = Invoke-RegTestAction -Path 'HKLM:\Foo' -Name 'EnableLUA' -Expected 1 | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.EnableLUA | Should -Be 0
    }

    It 'returns compliant=false when key is missing' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith { $null }
        $output = Invoke-RegTestAction -Path 'HKLM:\Foo' -Name 'EnableLUA' -Expected 1 | ConvertFrom-Json
        $output.compliant | Should -Be $false
    }

    It 'uses CurrentLabel for the current_state key when provided' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith {
            [PSCustomObject]@{ Some = 5 }
        }
        $output = Invoke-RegTestAction -Path 'HKLM:\Foo' -Name 'Some' -Expected 5 -CurrentLabel 'CustomLabel' | ConvertFrom-Json
        $output.current.CustomLabel | Should -Be 5
    }
}

Describe 'Invoke-RegSetAction' {
    It 'captures before, sets value, captures after, emits ok=true' {
        $callCount = [ref]0
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith {
            $callCount.Value++
            if ($callCount.Value -eq 1) {
                [PSCustomObject]@{ EnableLUA = 0 }   # before
            } else {
                [PSCustomObject]@{ EnableLUA = 1 }   # after
            }
        }
        Mock -ModuleName reg -CommandName Test-Path -MockWith { $true }
        Mock -ModuleName reg -CommandName Set-ItemProperty -MockWith { }

        $output = Invoke-RegSetAction -Path 'HKLM:\Foo' -Name 'EnableLUA' -Value 1 | ConvertFrom-Json
        $output.ok | Should -Be $true
        $output.before.exists | Should -Be $true
        $output.before.value | Should -Be 0
        $output.after.value | Should -Be 1
    }
}

Describe 'Invoke-RegUndoAction' {
    It 'restores the previous value when state.exists=true (via -State param)' {
        Mock -ModuleName reg -CommandName Test-Path -MockWith { $true }
        Mock -ModuleName reg -CommandName Set-ItemProperty -MockWith { } -Verifiable

        $state = @{ exists = $true; value = 42 }
        $output = Invoke-RegUndoAction -Path 'HKLM:\Foo' -Name 'X' -State $state | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -ModuleName reg -CommandName Set-ItemProperty -Times 1 -ParameterFilter {
            $Path -eq 'HKLM:\Foo' -and $Name -eq 'X' -and $Value -eq 42
        }
    }

    It 'removes the value when state.exists=false and the value is currently present' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith {
            [PSCustomObject]@{ X = 1 }
        }
        Mock -ModuleName reg -CommandName Remove-ItemProperty -MockWith { } -Verifiable

        $state = @{ exists = $false; value = $null }
        $output = Invoke-RegUndoAction -Path 'HKLM:\Foo' -Name 'X' -State $state | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -ModuleName reg -CommandName Remove-ItemProperty -Times 1 -ParameterFilter {
            $Path -eq 'HKLM:\Foo' -and $Name -eq 'X'
        }
    }

    It 'is a no-op when state.exists=false and the value is already absent' {
        Mock -ModuleName reg -CommandName Get-ItemProperty -MockWith { $null }
        Mock -ModuleName reg -CommandName Remove-ItemProperty -MockWith { }

        $state = @{ exists = $false; value = $null }
        $output = Invoke-RegUndoAction -Path 'HKLM:\Foo' -Name 'X' -State $state | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -ModuleName reg -CommandName Remove-ItemProperty -Times 0
    }
}
