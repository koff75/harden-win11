# netbios_off.undo.ps1
# Restaure NetbiosOptions sur chaque interface selon le 'before' fourni.
# Input : { "interfaces": { "Tcpip_<GUID>": <value>|null, ... } }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if ($MyInvocation.ExpectingInput) { $inputJson = ($input | Out-String).Trim() } else { $inputJson = [Console]::In.ReadToEnd() }
if (-not $inputJson.Trim()) { Write-Error "undo requires JSON input"; exit 1 }
$state = $inputJson | ConvertFrom-Json

$ifaceRoot = 'HKLM:\SYSTEM\CurrentControlSet\Services\NetBT\Parameters\Interfaces'

if ($state.interfaces) {
    foreach ($prop in $state.interfaces.PSObject.Properties) {
        $name = $prop.Name
        $val  = $prop.Value
        $path = Join-Path $ifaceRoot $name
        if (-not (Test-Path $path)) { continue }
        if ($null -ne $val) {
            Set-ItemProperty -Path $path -Name 'NetbiosOptions' -Value ([int]$val) -Force -ErrorAction SilentlyContinue
        } elseif (Get-ItemProperty -Path $path -Name 'NetbiosOptions' -ErrorAction SilentlyContinue) {
            Remove-ItemProperty -Path $path -Name 'NetbiosOptions' -ErrorAction SilentlyContinue
        }
    }
}

@{ ok = $true } | ConvertTo-Json -Compress
