# online_speech_off.action.ps1
# Online Speech dÃ©sactivÃ©.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\InputPersonalization' `
    -Name 'AllowInputPersonalization' `
    -Value 0 `
    -Type DWord