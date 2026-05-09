# Sandbox action : crée une valeur dans HKCU\Software\HardenWin11Test\.
# Réversible, ne nécessite pas admin (HKCU = current user). Idéal pour tests E2E.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKCU:\Software\HardenWin11Test'
$name = 'TestE2EValue'

if (-not (Test-Path $path)) {
    New-Item -Path $path -Force | Out-Null
}

$before = $null
try {
    $b = Get-ItemProperty -Path $path -Name $name -ErrorAction Stop
    $before = $b.$name
} catch {}

Set-ItemProperty -Path $path -Name $name -Value 42 -Type DWord

$after = (Get-ItemProperty -Path $path -Name $name).$name

@{
    ok = $true
    before = @{ value = $before; existed = ($null -ne $before) }
    after  = @{ value = $after }
} | ConvertTo-Json -Compress
