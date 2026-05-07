# rdp_disable.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

$path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server'
$name = 'fDenyTSConnections'

if ($state.exists) {
    if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }
    Set-ItemProperty -Path $path -Name $name -Value ([int]$state.value) -Type DWord -Force
} elseif (Get-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue) {
    Remove-ItemProperty -Path $path -Name $name
}

@{ ok = $true } | ConvertTo-Json -Compress
