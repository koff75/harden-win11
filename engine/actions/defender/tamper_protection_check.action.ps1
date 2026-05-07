# tamper_protection_check.action.ps1
# Tamper Protection ne peut PAS être activé par script (design Microsoft).
# Cette "action" se contente de ré-évaluer l'état et signaler à l'user
# qu'il faut activer TP manuellement dans Windows Security.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$status = Get-MpComputerStatus
$tp = [bool]$status.IsTamperProtected

$before = @{ IsTamperProtected = $tp }
$after  = $before  # immuable par script

@{
    ok                    = $true
    manual_step_required  = if ($tp) { $null } else { "Activer manuellement dans : Windows Security > Virus & threat protection > Manage settings > Tamper Protection ON" }
    before                = $before
    after                 = $after
} | ConvertTo-Json -Compress -Depth 10
