# advertising_id_user.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo'
$name = 'Enabled'
$expected = 0

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
$compliant = $value -eq $expected

@{ compliant = $compliant; current = @{ Enabled = $value } } | ConvertTo-Json -Compress
