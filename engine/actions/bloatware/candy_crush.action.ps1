# candy_crush.action.ps1
# Desinstalle l'app Store 'Candy Crush' (pattern : *CandyCrush*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxRemove -Pattern '*CandyCrush*'