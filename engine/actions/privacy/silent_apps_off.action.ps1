# silent_apps_off.action.ps1
# HKCU : SilentInstalledAppsEnabled = 0 (stop reinstalls silencieux d'apps Store).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'
$name = 'SilentInstalledAppsEnabled'
$expected = 0

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$before = @{ exists = [bool]$existing; value = if ($existing) { $existing.$name } else { $null } }

if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }
Set-ItemProperty -Path $path -Name $name -Value $expected -Type DWord -Force

$existing = Get-ItemProperty -Path $path -Name $name
$after = @{ exists = $true; value = $existing.$name }

@{ ok = $true; before = $before; after = $after } | ConvertTo-Json -Compress -Depth 10
