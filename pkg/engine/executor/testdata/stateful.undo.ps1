# Undo qui restaure l'état exists de avant. Si le before disait
# exists=false, on supprime le marker.

$ErrorActionPreference = 'Stop'
if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
$state = $inputJson | ConvertFrom-Json

$marker = $state.marker
if (-not $marker) { $marker = Join-Path $env:TEMP 'harden-stateful-test.marker' }

if ($state.exists -eq $false -and (Test-Path $marker)) {
    Remove-Item $marker -Force
}

@{ ok = $true } | ConvertTo-Json -Compress
