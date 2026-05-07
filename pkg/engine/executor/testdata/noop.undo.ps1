# Undo qui ne touche à rien — juste reporte ok pour les tests.

if ($MyInvocation.ExpectingInput) {
    $null = ($input | Out-String)
} else {
    $null = [Console]::In.ReadToEnd()
}

@{ ok = $true } | ConvertTo-Json -Compress
