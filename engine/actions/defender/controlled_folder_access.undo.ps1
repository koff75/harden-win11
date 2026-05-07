# controlled_folder_access.undo.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with EnableControlledFolderAccess field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

Set-MpPreference -EnableControlledFolderAccess ([string]$state.EnableControlledFolderAccess)

@{ ok = $true } | ConvertTo-Json -Compress
