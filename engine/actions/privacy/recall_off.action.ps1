# recall_off.action.ps1
# Windows Recall dÃ©sactivÃ© (prÃ©ventif).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsAI' `
    -Name 'DisableAIDataAnalysis' `
    -Value 1 `
    -Type DWord