# rename_admin.undo.ps1
# Restaure le nom du compte SID-500 à sa valeur d'avant (généralement 'Administrator').
# Input : { "AdminCurrentName": "<original-name>", "SID500Found": true }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) {
    $inputJson = ($input | Out-String).Trim()
} else {
    $inputJson = [Console]::In.ReadToEnd()
}

if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input with AdminCurrentName field"
    exit 1
}
$state = $inputJson | ConvertFrom-Json

if (-not $state.SID500Found -or -not $state.AdminCurrentName) {
    @{ ok = $true; skipped = "no SID-500 account in 'before' state" } | ConvertTo-Json -Compress
    exit 0
}

$admin = Get-LocalUser | Where-Object { $_.SID.Value -like 'S-1-5-*-500' }
if ($admin -and $admin.Name -ne $state.AdminCurrentName) {
    Rename-LocalUser -Name $admin.Name -NewName $state.AdminCurrentName
}

@{ ok = $true } | ConvertTo-Json -Compress
