# dolby_access.action.ps1
# Desinstalle l'app Store 'Dolby Access' (pattern : *DolbyLaboratories.DolbyAccess*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxRemove -Pattern '*DolbyLaboratories.DolbyAccess*'