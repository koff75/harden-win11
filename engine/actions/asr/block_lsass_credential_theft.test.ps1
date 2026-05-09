# block_lsass_credential_theft.test.ps1
# Conforme = la règle ASR 9E6C4E1F-7D60-472F-BA1A-A39EF669E4B2 est en mode Block (1).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$guid = '9E6C4E1F-7D60-472F-BA1A-A39EF669E4B2'
$expected = if ($env:HARDEN_ASR_MODE -eq 'audit') { 2 } else { 1 }

$pref = Get-MpPreference
$ids = @($pref.AttackSurfaceReductionRules_Ids)
$acts = @($pref.AttackSurfaceReductionRules_Actions)

$current = $null
for ($i = 0; $i -lt $ids.Count; $i++) {
    if ($ids[$i] -ieq $guid) { $current = [int]$acts[$i]; break }
}

$compliant = ($current -eq $expected)
$names = @{ 0 = 'NotConfigured'; 1 = 'Block'; 2 = 'Audit'; 6 = 'Warn' }
$mode = if ($null -ne $current -and $names.ContainsKey($current)) { $names[$current] } elseif ($null -ne $current) { "Unknown($current)" } else { 'NotPresent' }

@{
    compliant = $compliant
    current   = @{
        AsrRule    = $guid
        AsrAction  = $current
        AsrMode    = $mode
    }
} | ConvertTo-Json -Compress -Depth 10