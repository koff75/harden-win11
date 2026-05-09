# appx.psm1 — Helpers pour les regles de hardening basees sur la
# desinstallation d'apps Microsoft Store (bloatware).
#
# Convention de retour :
#   Invoke-AppxTest    -> { compliant, current: {Pattern, AppxInstalled, ProvisionedInstalled} }
#   Invoke-AppxRemove  -> { ok, before, after: {Removed: [...]} }
#
# Note : ces operations necessitent admin (Get-AppxPackage -AllUsers).
# Sans admin, le test ressort un comportement degrade (Get-AppxPackage
# sans -AllUsers retourne juste les packages du user courant).

Set-StrictMode -Version Latest

function Get-AppxByPattern {
    [CmdletBinding()]
    param([Parameter(Mandatory)] [string] $Pattern)

    $installed = @(Get-AppxPackage -Name $Pattern -AllUsers -ErrorAction SilentlyContinue)
    $provisioned = @(Get-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue |
        Where-Object {
            $_.DisplayName -like $Pattern.Trim('*') -or $_.PackageName -like $Pattern
        })

    @{
        Installed    = $installed
        Provisioned  = $provisioned
        Total        = $installed.Count + $provisioned.Count
    }
}

# Invoke-AppxTest : .test.ps1 pour une regle bloatware single-pattern.
# Conforme = aucun match Appx ni provisioned.
function Invoke-AppxTest {
    [CmdletBinding()]
    param([Parameter(Mandatory)] [string] $Pattern)

    $state = Get-AppxByPattern -Pattern $Pattern

    @{
        compliant = ($state.Total -eq 0)
        current = @{
            Pattern              = $Pattern
            AppxInstalled        = $state.Installed.Count
            ProvisionedInstalled = $state.Provisioned.Count
            InstalledNames       = @($state.Installed | ForEach-Object { $_.Name })
        }
    } | ConvertTo-Json -Compress -Depth 10
}

# Invoke-AppxRemove : .action.ps1 qui desinstalle tous les Appx + provisioned
# matchant le pattern. Best-effort : ignore les erreurs sur un package
# pour permettre la suppression des suivants.
function Invoke-AppxRemove {
    [CmdletBinding()]
    param([Parameter(Mandatory)] [string] $Pattern)

    $state = Get-AppxByPattern -Pattern $Pattern
    $before = @{
        Pattern              = $Pattern
        AppxInstalled        = $state.Installed.Count
        ProvisionedInstalled = $state.Provisioned.Count
    }

    $removed = @()
    foreach ($pkg in $state.Installed) {
        try {
            Remove-AppxPackage -Package $pkg.PackageFullName -AllUsers -ErrorAction Stop
            $removed += $pkg.Name
        } catch {}
    }
    foreach ($prov in $state.Provisioned) {
        try {
            Remove-AppxProvisionedPackage -Online -PackageName $prov.PackageName -ErrorAction Stop | Out-Null
            $removed += "prov:$($prov.DisplayName)"
        } catch {}
    }

    @{
        ok     = $true
        before = $before
        after  = @{ Removed = $removed }
    } | ConvertTo-Json -Compress -Depth 10
}

Export-ModuleMember -Function Get-AppxByPattern, Invoke-AppxTest, Invoke-AppxRemove
