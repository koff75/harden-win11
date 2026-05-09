# Sandbox undo : restaure HKCU\Software\HardenWin11Test\TestE2EValue à son état before.
# Si le before disait 'existed=false', on remove la value.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
$state = $inputJson | ConvertFrom-Json

$path = 'HKCU:\Software\HardenWin11Test'
$name = 'TestE2EValue'

if (-not (Test-Path $path)) {
    @{ ok = $true; note = 'key already absent' } | ConvertTo-Json -Compress
    return
}

if ($state.existed -eq $false -or $null -eq $state.existed) {
    Remove-ItemProperty -Path $path -Name $name -ErrorAction SilentlyContinue
} else {
    Set-ItemProperty -Path $path -Name $name -Value $state.value
}

@{ ok = $true } | ConvertTo-Json -Compress
