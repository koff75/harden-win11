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

# Tamper Protection ne peut PAS etre activee par script (design Microsoft :
# Set-MpPreference -DisableTamperProtection n'existe pas, seul l'UI ou MDM
# peut le faire). Si OFF, on retourne ok=false avec un message clair : c'est
# une action manuelle utilisateur, pas une defaillance technique.
if (-not $tp) {
    @{
        ok    = $false
        error = "Tamper Protection est desactivee. Reactivation manuelle requise : Windows Security > Virus & threat protection > Manage settings > Tamper Protection ON. (Microsoft ne permet pas d'activer Tamper par script.)"
        before = $before
        after  = $after
    } | ConvertTo-Json -Compress -Depth 10
    exit 0
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
