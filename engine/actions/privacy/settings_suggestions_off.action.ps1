# settings_suggestions_off.action.ps1
# HKCU : suggestions Settings off.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager' `
    -Name 'SubscribedContent-338389Enabled' `
    -Value 0 `
    -Type DWord