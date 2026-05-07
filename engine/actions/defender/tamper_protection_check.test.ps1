# tamper_protection_check.test.ps1
# Conforme = Tamper Protection est ACTIVÉ.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$status = Get-MpComputerStatus
$tp = [bool]$status.IsTamperProtected

@{
    compliant = $tp
    current   = @{ IsTamperProtected = $tp }
} | ConvertTo-Json -Compress
