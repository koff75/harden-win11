# wpad_disable.action.ps1
# WPAD dÃ©sactivÃ© (anti-poisoning).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc' `
    -Name 'Start' `
    -Value 4 `
    -Type DWord