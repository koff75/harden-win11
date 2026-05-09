# pub_5319275a.action.ps1
# Desinstalle l'app Store 'Apps OEM publisher 5319275A' (pattern : *5319275A*).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\harden_appx.psm1') -Force

Invoke-AppxRemove -Pattern '*5319275A*'