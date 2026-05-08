# prompt_secure_desktop.action.ps1
# UAC : tous les prompts UAC sur secure desktop.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'PromptOnSecureDesktop' `
    -Value 1 `
    -Type DWord