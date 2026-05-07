# advertising_id_machine.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\AdvertisingInfo'
$name = 'DisabledByGroupPolicy'
$expected = 1

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
$compliant = $value -eq $expected

@{ compliant = $compliant; current = @{ DisabledByGroupPolicy = $value } } | ConvertTo-Json -Compress
