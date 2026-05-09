# zune_video.action.ps1
# Desinstalle l'app Store 'Films & TV (Zune Video)' (pattern : *Microsoft.ZuneVideo*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxRemove -Pattern '*Microsoft.ZuneVideo*'