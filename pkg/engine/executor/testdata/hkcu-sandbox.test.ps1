# Sandbox test : conforme = HKCU\Software\HardenWin11Test\TestE2EValue == 42

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$path = 'HKCU:\Software\HardenWin11Test'
$name = 'TestE2EValue'

$current = $null
$exists = $false
try {
    $i = Get-ItemProperty -Path $path -Name $name -ErrorAction Stop
    $current = $i.$name
    $exists = $true
} catch {}

@{
    compliant = ($current -eq 42)
    current = @{ value = $current; exists = $exists }
} | ConvertTo-Json -Compress
