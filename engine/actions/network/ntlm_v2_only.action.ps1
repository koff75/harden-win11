# ntlm_v2_only.action.ps1
# LmCompatibilityLevel = 5 : NTLMv2 only, refuse LM/NTLMv1.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa'
$name = 'LmCompatibilityLevel'
$expected = 5

$existing = Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
$before = @{ exists = [bool]$existing; value = if ($existing) { $existing.$name } else { $null } }

if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }
Set-ItemProperty -Path $path -Name $name -Value $expected -Type DWord -Force

$existing = Get-ItemProperty -Path $path -Name $name
$after = @{ exists = $true; value = $existing.$name }

@{ ok = $true; before = $before; after = $after } | ConvertTo-Json -Compress -Depth 10
