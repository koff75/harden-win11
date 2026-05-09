# bing_news.action.ps1
# Desinstalle l'app Store 'Microsoft Bing News' (pattern : *Microsoft.BingNews*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxRemove -Pattern '*Microsoft.BingNews*'