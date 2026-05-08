# wpad_disable.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegUndoAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\WinHttpAutoProxySvc' `
    -Name 'Start' `
    -Type DWord