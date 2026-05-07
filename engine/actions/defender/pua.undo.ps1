# pua.undo.ps1
# Restaure PUAProtection à la valeur passée en entrée (Disabled, Enabled, AuditMode).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with PUAProtection field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -PUAProtection ([string]$state.PUAProtection)

@{ ok = $true } | ConvertTo-Json -Compress
