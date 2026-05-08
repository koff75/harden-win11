# advertising_id_user.action.ps1
# HKCU : Advertising ID off.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo' `
    -Name 'Enabled' `
    -Value 0 `
    -Type DWord