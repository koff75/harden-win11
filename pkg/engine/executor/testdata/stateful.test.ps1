# Test fixture qui dépend d'un marker file. Permet de simuler un re-test
# post-apply : le test pré-action ne voit pas le marker (compliant=false),
# l'action crée le marker, le re-test post-action le voit (compliant=true).
#
# Le path du marker est passé via $env:HARDEN_TEST_MARKER. Cleanup côté Go.

$ErrorActionPreference = 'Stop'
$marker = $env:HARDEN_TEST_MARKER
if (-not $marker) { $marker = Join-Path $env:TEMP 'harden-stateful-test.marker' }

$exists = Test-Path $marker
@{
    compliant = $exists
    current   = @{ marker = $marker; exists = $exists }
} | ConvertTo-Json -Compress
