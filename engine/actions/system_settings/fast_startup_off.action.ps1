# fast_startup_off.action.ps1
# Power : dÃ©sactive Fast Startup.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Power' `
    -Name 'HiberbootEnabled' `
    -Value 0 `
    -Type DWord