# silent_apps_off.action.ps1
# HKCU : pas de rÃ©install silencieuse.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' `
    -Name 'SilentInstalledAppsEnabled' `
    -Value 0 `
    -Type DWord