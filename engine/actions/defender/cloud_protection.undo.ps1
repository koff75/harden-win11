# cloud_protection.undo.ps1
# Restaure les 3 paramètres cloud à leurs valeurs avant.
# Input : { "MAPSReporting":"...", "CloudBlockLevel":"...", "CloudExtendedTimeout":<int> }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with MAPSReporting/CloudBlockLevel/CloudExtendedTimeout fields"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -MAPSReporting        ([string]$state.MAPSReporting)
Set-MpPreference -CloudBlockLevel      ([string]$state.CloudBlockLevel)
Set-MpPreference -CloudExtendedTimeout ([int]$state.CloudExtendedTimeout)

@{ ok = $true } | ConvertTo-Json -Compress
