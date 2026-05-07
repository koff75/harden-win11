# consumer_features_off.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

$path  = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent'
$names = @('DisableWindowsConsumerFeatures', 'DisableConsumerAccountStateContent')

foreach ($n in $names) {
    $info = $state.$n
    if (-not $info) { continue }
    if ($info.exists) {
        if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }
        Set-ItemProperty -Path $path -Name $n -Value ([int]$info.value) -Type DWord -Force
    } elseif (Get-ItemProperty -Path $path -Name $n -ErrorAction SilentlyContinue) {
        Remove-ItemProperty -Path $path -Name $n -ErrorAction SilentlyContinue
    }
}

@{ ok = $true } | ConvertTo-Json -Compress
