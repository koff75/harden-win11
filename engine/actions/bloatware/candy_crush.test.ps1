# candy_crush.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxTest -Pattern '*CandyCrush*'