# advertising_id_user.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegUndoAction `
    -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo' `
    -Name 'Enabled' `
    -Type DWord