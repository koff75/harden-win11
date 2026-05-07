# script_scanning.undo.ps1
# Restaure DisableScriptScanning à la valeur passée en entrée.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with DisableScriptScanning field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -DisableScriptScanning ([bool]$state.DisableScriptScanning)

@{ ok = $true } | ConvertTo-Json -Compress
