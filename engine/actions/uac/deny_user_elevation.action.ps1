# deny_user_elevation.action.ps1
# UAC : refuser élévation pour comptes standard.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'ConsentPromptBehaviorUser' `
    -Value 0 `
    -Type DWord