# llmnr_disable.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegTestAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient' `
    -Name 'EnableMulticast' `
    -Expected 0