# block_email_executable_content.action.ps1
# ASR : Block executable content from email/webmail
# GUID : BE9BA2D9-53EA-4CDC-84E5-9B1EEEE46550
# Action choisie :
#   - 1 (Block) par defaut
#   - 2 (Audit) si l'env var HARDEN_ASR_MODE=audit est positionnee (le
#     runner Go la passe quand l'utilisateur active le mode audit GUI).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$guid = 'BE9BA2D9-53EA-4CDC-84E5-9B1EEEE46550'
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