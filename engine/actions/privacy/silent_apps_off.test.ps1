# silent_apps_off.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'
$name = 'SilentInstalledAppsEnabled'
$expected = 0

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
$compliant = $value -eq $expected

@{ compliant = $compliant; current = @{ SilentInstalledAppsEnabled = $value } } | ConvertTo-Json -Compress
