# hibernate_off.test.ps1
# Conforme = hiberfil.sys n'existe pas.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$hiberPath = "$env:SystemDrive\hiberfil.sys"
$exists = Test-Path $hiberPath
$compliant = -not $exists

@{ compliant = $compliant; current = @{ HiberfilExists = $exists } } | ConvertTo-Json -Compress
