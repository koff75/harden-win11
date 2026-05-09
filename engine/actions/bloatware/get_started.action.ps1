# get_started.action.ps1
# Desinstalle l'app Store 'Get Started (tour Win11)' (pattern : *Microsoft.Getstarted*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxRemove -Pattern '*Microsoft.Getstarted*'