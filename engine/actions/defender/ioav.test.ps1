# ioav.test.ps1
# Vérifie si la règle defender.ioav est déjà conforme.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$current = (Get-MpPreference).DisableIOAVProtection
$compliant = -not $current

@{
    compliant = $compliant
    current   = @{ DisableIOAVProtection = $current }
} | ConvertTo-Json -Compress -Depth 10
