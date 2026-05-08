# ntlm_v2_only.action.ps1
# LmCompatibilityLevel=5 (NTLMv2 only).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\reg.psm1') -Force

Invoke-RegSetAction `
    -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' `
    -Name 'LmCompatibilityLevel' `
    -Value 5 `
    -Type DWord