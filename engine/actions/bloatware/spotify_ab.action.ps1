# spotify_ab.action.ps1
# Desinstalle l'app Store 'Spotify (variante OEM)' (pattern : *SpotifyAB*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxRemove -Pattern '*SpotifyAB*'