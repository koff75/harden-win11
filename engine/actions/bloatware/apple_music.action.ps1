# apple_music.action.ps1
# Desinstalle l'app Store 'Apple Music' (pattern : *AppleMusicWin*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxRemove -Pattern '*AppleMusicWin*'