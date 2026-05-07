# ioav.undo.ps1
# Restaure DisableIOAVProtection à la valeur passée en entrée.
# Input : { "DisableIOAVProtection": <bool> }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with DisableIOAVProtection field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -DisableIOAVProtection ([bool]$state.DisableIOAVProtection)

@{ ok = $true } | ConvertTo-Json -Compress
