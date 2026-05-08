# online_speech_off.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegTestAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\InputPersonalization' `
    -Name 'AllowInputPersonalization' `
    -Expected 0