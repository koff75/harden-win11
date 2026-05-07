# consumer_features_off.action.ps1
# Multi-value : DisableWindowsConsumerFeatures, DisableConsumerAccountStateContent.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path  = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent'
$names = @('DisableWindowsConsumerFeatures', 'DisableConsumerAccountStateContent')

$before = @{}
foreach ($n in $names) {
    $e = Get-ItemProperty -Path $path -Name $n -ErrorAction SilentlyContinue
    $before[$n] = if ($e) { @{ exists = $true; value = $e.$n } } else { @{ exists = $false; value = $null } }
}

if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }
foreach ($n in $names) { Set-ItemProperty -Path $path -Name $n -Value 1 -Type DWord -Force }

$after = @{}
foreach ($n in $names) {
    $e = Get-ItemProperty -Path $path -Name $n -ErrorAction SilentlyContinue
    $after[$n] = if ($e) { @{ exists = $true; value = $e.$n } } else { @{ exists = $false; value = $null } }
}

@{ ok = $true; before = $before; after = $after } | ConvertTo-Json -Compress -Depth 10
