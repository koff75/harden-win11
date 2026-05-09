# jimmylin.action.ps1
# Desinstalle l'app Store 'Apps OEM publisher JimmyLin (LiveOS games)' (pattern : *JimmyLin*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxRemove -Pattern '*JimmyLin*'