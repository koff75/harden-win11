# block_office_comm_child_processes.action.ps1
# ASR : Block Office communication apps from creating child processes
# GUID : 26190899-1602-49E8-8B27-EB1D0A1CE869
# Action choisie :
#   - 1 (Block) par defaut
#   - 2 (Audit) si l'env var HARDEN_ASR_MODE=audit est positionnee (le
#     runner Go la passe quand l'utilisateur active le mode audit GUI).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$guid = '26190899-1602-49E8-8B27-EB1D0A1CE869'
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