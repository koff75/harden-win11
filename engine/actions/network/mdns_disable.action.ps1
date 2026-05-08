# mdns_disable.action.ps1
# mDNS dÃ©sactivÃ©.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters' `
    -Name 'EnableMDNS' `
    -Value 0 `
    -Type DWord