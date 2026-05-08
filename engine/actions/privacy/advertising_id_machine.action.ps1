# advertising_id_machine.action.ps1
# AdvertisingID dÃ©sactivÃ© machine-wide.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\AdvertisingInfo' `
    -Name 'DisabledByGroupPolicy' `
    -Value 1 `
    -Type DWord