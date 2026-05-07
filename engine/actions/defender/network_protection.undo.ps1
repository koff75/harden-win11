# network_protection.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with EnableNetworkProtection field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -EnableNetworkProtection ([string]$state.EnableNetworkProtection)

@{ ok = $true } | ConvertTo-Json -Compress
