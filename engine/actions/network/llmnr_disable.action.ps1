# llmnr_disable.action.ps1
# Désactive LLMNR (Link-Local Multicast Name Resolution).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient'
$name = 'EnableMulticast'
$expected = 0

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$before = @{ exists = [bool]$existing; value = if ($existing) { $existing.$name } else { $null } }

if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }
Set-ItemProperty -Path $path -Name $name -Value $expected -Type DWord -Force

$existing = Get-ItemProperty -Path $path -Name $name
$after = @{ exists = $true; value = $existing.$name }

@{ ok = $true; before = $before; after = $after } | ConvertTo-Json -Compress -Depth 10
