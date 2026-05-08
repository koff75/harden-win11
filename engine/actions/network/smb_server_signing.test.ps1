# smb_server_signing.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegTestAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters' `
    -Name 'RequireSecuritySignature' `
    -Expected 1