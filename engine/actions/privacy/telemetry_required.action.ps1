# telemetry_required.action.ps1
# Telemetry niveau 1 (Required only).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection' `
    -Name 'AllowTelemetry' `
    -Value 1 `
    -Type DWord