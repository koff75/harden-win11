# your_phone.action.ps1
# Desinstalle l'app Store 'Your Phone (Phone Link)' (pattern : *Microsoft.YourPhone*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxRemove -Pattern '*Microsoft.YourPhone*'