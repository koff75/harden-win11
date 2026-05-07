# mdns_disable.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters'
$name = 'EnableMDNS'
$expected = 0

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
$compliant = $value -eq $expected

@{ compliant = $compliant; current = @{ EnableMDNS = $value } } | ConvertTo-Json -Compress
