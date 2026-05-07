# hibernate_off.undo.ps1
# Réactive l'hibernation si avant elle l'était.
# Input : { "HiberfilExists": true|false }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

if ($state.HiberfilExists) {
    & powercfg.exe /hibernate on | Out-Null
}

@{ ok = $true } | ConvertTo-Json -Compress
