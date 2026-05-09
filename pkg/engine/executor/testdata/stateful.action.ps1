# Action fixture qui crée le marker file. Combiné avec stateful.test.ps1
# pour simuler une action qui modifie réellement l'état observé par le test.

$ErrorActionPreference = 'Stop'
$marker = $env:HARDEN_TEST_MARKER
if (-not $marker) { $marker = Join-Path $env:TEMP 'harden-stateful-test.marker' }

$beforeExists = Test-Path $marker
New-Item -ItemType File -Path $marker -Force | Out-Null
$afterExists = Test-Path $marker

@{
    ok     = $true
    before = @{ marker = $marker; exists = $beforeExists }
    after  = @{ marker = $marker; exists = $afterExists }
} | ConvertTo-Json -Compress
