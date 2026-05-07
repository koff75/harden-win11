# signatures.action.ps1
# Met à jour les signatures antivirus de Defender.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$beforeStatus = Get-MpComputerStatus
$before = @{
    AntivirusSignatureLastUpdated = $beforeStatus.AntivirusSignatureLastUpdated.ToString('o')
    AntivirusSignatureVersion     = $beforeStatus.AntivirusSignatureVersion
}

Update-MpSignature -ErrorAction Stop

$afterStatus = Get-MpComputerStatus
$after = @{
    AntivirusSignatureLastUpdated = $afterStatus.AntivirusSignatureLastUpdated.ToString('o')
    AntivirusSignatureVersion     = $afterStatus.AntivirusSignatureVersion
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
