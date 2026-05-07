# disable_unused.test.ps1
# Conforme = chacun des 4 comptes est soit absent, soit désactivé.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$accounts = @('Administrator', 'Guest', 'WsiAccount', 'DefaultAccount')

$states = @{}
$compliant = $true
foreach ($name in $accounts) {
    $u = Get-LocalUser -Name $name -ErrorAction SilentlyContinue
    if ($u) {
        $states[$name] = @{ exists = $true; enabled = [bool]$u.Enabled }
        if ($u.Enabled) { $compliant = $false }
    } else {
        $states[$name] = @{ exists = $false; enabled = $null }
    }
}

@{
    compliant = $compliant
    current   = $states
} | ConvertTo-Json -Compress -Depth 10
