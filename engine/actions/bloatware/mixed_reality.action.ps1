# mixed_reality.action.ps1
# Desinstalle l'app Store 'Mixed Reality Portal (VR)' (pattern : *Microsoft.MixedReality.Portal*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxRemove -Pattern '*Microsoft.MixedReality.Portal*'