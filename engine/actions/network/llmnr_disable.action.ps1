# llmnr_disable.action.ps1
# LLMNR désactivé (anti-Responder).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient' `
    -Name 'EnableMulticast' `
    -Value 0 `
    -Type DWord