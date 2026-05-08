# wpad_disable.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegTestAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc' `
    -Name 'Start' `
    -Expected 4