# hibernate_off.action.ps1
# Désactive l'hibernation. powercfg.exe /hibernate off supprime hiberfil.sys
# (gain disque ~RAM size).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$hiberPath = "$env:SystemDrive\hiberfil.sys"
$before = @{ HiberfilExists = (Test-Path $hiberPath) }

& powercfg.exe /hibernate off | Out-Null

$after = @{ HiberfilExists = (Test-Path $hiberPath) }

@{ ok = $true; before = $before; after = $after } | ConvertTo-Json -Compress -Depth 10
