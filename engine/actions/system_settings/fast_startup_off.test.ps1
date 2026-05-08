# fast_startup_off.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegTestAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Power' `
    -Name 'HiberbootEnabled' `
    -Expected 0