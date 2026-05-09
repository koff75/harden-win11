# get_help.action.ps1
# Desinstalle l'app Store 'Get Help (assistant Microsoft)' (pattern : *Microsoft.GetHelp*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxRemove -Pattern '*Microsoft.GetHelp*'