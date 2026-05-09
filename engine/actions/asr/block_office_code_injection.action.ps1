# block_office_code_injection.action.ps1
# ASR : Block Office apps from injecting code into other processes
# GUID : 75668C1F-73B5-4CF0-BB93-3ECF5CB7CC84
# Action choisie :
#   - 1 (Block) par defaut
#   - 2 (Audit) si l'env var HARDEN_ASR_MODE=audit est positionnee (le
#     runner Go la passe quand l'utilisateur active le mode audit GUI).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$guid = '75668C1F-73B5-4CF0-BB93-3ECF5CB7CC84'
$action = if ($env:HARDEN_ASR_MODE -eq 'audit') { 2 } else { 1 }

function Get-AsrAction([string]$g) {
    $pref = Get-MpPreference
    $ids = @($pref.AttackSurfaceReductionRules_Ids)
    $acts = @($pref.AttackSurfaceReductionRules_Actions)
    for ($i = 0; $i -lt $ids.Count; $i++) {
        if ($ids[$i] -ieq $g) { return [int]$acts[$i] }
    }
    return $null
}

$beforeAction = Get-AsrAction $guid
$before = @{ AsrAction = $beforeAction }

Add-MpPreference -AttackSurfaceReductionRules_Ids $guid -AttackSurfaceReductionRules_Actions $action -ErrorAction Stop

$afterAction = Get-AsrAction $guid
$after = @{ AsrAction = $afterAction }

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10