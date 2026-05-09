# tiktok.action.ps1
# Desinstalle l'app Store 'TikTok' (pattern : *TikTok*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxRemove -Pattern '*TikTok*'