# signatures.test.ps1
# Conforme = signatures mises à jour il y a moins de 7 jours.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$status = Get-MpComputerStatus
$last = $status.AntivirusSignatureLastUpdated
$threshold = (Get-Date).AddDays(-7)
$compliant = $last -gt $threshold

@{
    compliant = $compliant
    current   = @{
        AntivirusSignatureLastUpdated = $last.ToString('o')
        AntivirusSignatureVersion     = $status.AntivirusSignatureVersion
        AgeInHours                    = [math]::Round(((Get-Date) - $last).TotalHours, 1)
    }
} | ConvertTo-Json -Compress -Depth 10
