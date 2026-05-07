# ntlm_v2_only.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa'
$name = 'LmCompatibilityLevel'
$expected = 5

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
$compliant = $value -eq $expected

@{ compliant = $compliant; current = @{ LmCompatibilityLevel = $value } } | ConvertTo-Json -Compress
