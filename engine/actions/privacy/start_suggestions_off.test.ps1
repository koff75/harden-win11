# start_suggestions_off.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'
$name = 'SystemPaneSuggestionsEnabled'
$expected = 0

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$value = if ($existing) { $existing.$name } else { $null }
$compliant = $value -eq $expected

@{ compliant = $compliant; current = @{ SystemPaneSuggestionsEnabled = $value } } | ConvertTo-Json -Compress
