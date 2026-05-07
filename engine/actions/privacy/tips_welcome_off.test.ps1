# tips_welcome_off.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path  = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager'
$names = @('SubscribedContent-338388Enabled', 'SubscribedContent-310093Enabled', 'SubscribedContent-353698Enabled')

$current = @{}
$compliant = $true
foreach ($n in $names) {
    $e = Get-ItemProperty -Path $path -Name $n -ErrorAction SilentlyContinue
    $val = if ($e) { $e.$n } else { $null }
    $current[$n] = $val
    if ($val -ne 0) { $compliant = $false }
}

@{ compliant = $compliant; current = $current } | ConvertTo-Json -Compress -Depth 10
