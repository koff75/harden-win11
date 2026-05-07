# telemetry_required.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection'
$name = 'AllowTelemetry'
$expected = 1

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
# Compliant si <= expected (les niveaux plus bas sont OK aussi)
$compliant = ($null -ne $value) -and ($value -le $expected)

@{ compliant = $compliant; current = @{ AllowTelemetry = $value } } | ConvertTo-Json -Compress
