# online_speech_off.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegUndoAction `
    -Path 'HKLM:\SOFTWARE\Policies\Microsoft\InputPersonalization' `
    -Name 'AllowInputPersonalization' `
    -Type DWord