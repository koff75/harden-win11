# Fixture de test compatible PS 5.1
$inputJson = [Console]::In.ReadToEnd()
if ($inputJson.Trim()) {
    $obj = $inputJson | ConvertFrom-Json
    # Convertir PSCustomObject en hashtable (5.1 friendly)
    $hash = @{}
    $obj.PSObject.Properties | ForEach-Object { $hash[$_.Name] = $_.Value }
} else {
    $hash = @{}
}
$hash.echoed = $true
$hash | ConvertTo-Json -Compress -Depth 10
