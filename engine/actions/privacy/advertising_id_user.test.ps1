# advertising_id_user.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegTestAction `
    -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo' `
    -Name 'Enabled' `
    -Expected 0