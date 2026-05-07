# smb_client_signing.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters'
$name = 'RequireSecuritySignature'
$expected = 1

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
$compliant = $value -eq $expected

@{ compliant = $compliant; current = @{ RequireSecuritySignature = $value } } | ConvertTo-Json -Compress
