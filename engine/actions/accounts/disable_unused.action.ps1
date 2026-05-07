# disable_unused.action.ps1
# Désactive les comptes locaux peu utilisés et souvent ciblés (Administrator,
# Guest, WsiAccount, DefaultAccount). N'agit que si le compte existe.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$accounts = @('Administrator', 'Guest', 'WsiAccount', 'DefaultAccount')

$before = @{}
foreach ($name in $accounts) {
    $u = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    $before[$name] = if ($u) { @{ exists = $true; enabled = [bool]$u.Enabled } } else { @{ exists = $false; enabled = $null } }
}

foreach ($name in $accounts) {
    $u = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($u -and $u.Enabled) {
        Disable-LocalUser -Name $name
    }
}

$after = @{}
foreach ($name in $accounts) {
    $u = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    $after[$name] = if ($u) { @{ exists = $true; enabled = [bool]$u.Enabled } } else { @{ exists = $false; enabled = $null } }
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
