# block_wmi_persistence.test.ps1
# Conforme = la règle ASR E6DB77E5-3DF2-4CF1-B95A-636979351E5B est en mode Block (1).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$guid = 'E6DB77E5-3DF2-4CF1-B95A-636979351E5B'
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