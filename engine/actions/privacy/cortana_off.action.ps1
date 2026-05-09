# cortana_off.action.ps1
# Cortana désactivée.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Windows Search' `
    -Name 'AllowCortana' `
    -Value 0 `
    -Type DWord