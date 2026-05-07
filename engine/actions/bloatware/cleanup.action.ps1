# cleanup.action.ps1
# Désinstalle les apps Store "bloatware" listées (Appx + provisioned).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$patterns = @(
    '*JimmyLin*', '*5319275A*', '*Clipchamp*', '*AppleMusicWin*',
    '*SpotifyAB*', '*Spotify*', '*DolbyLaboratories.DolbyAccess*',
    '*Microsoft.BingNews*', '*Microsoft.BingWeather*', '*Microsoft.GetHelp*',
    '*Microsoft.Getstarted*', '*Microsoft.MicrosoftSolitaireCollection*',
    '*Microsoft.MixedReality.Portal*', '*Microsoft.People*',
    '*Microsoft.SkypeApp*', '*Microsoft.WindowsFeedbackHub*',
    '*Microsoft.YourPhone*', '*Microsoft.ZuneMusic*', '*Microsoft.ZuneVideo*',
    '*Disney*', '*TikTok*', '*Facebook*', '*Instagram*', '*Twitter*',
    '*LinkedInforWindows*', '*CandyCrush*', '*Netflix*'
)

$removedAppx = @()
$removedProvisioned = @()
$alreadyHandled = @{}

foreach ($pattern in $patterns) {
    # Appx packages installées
    $packages = Get-AppxPackage -Name $pattern -AllUsers -ErrorAction SilentlyContinue
    foreach ($pkg in $packages) {
        if ($alreadyHandled.ContainsKey($pkg.PackageFullName)) { continue }
        $alreadyHandled[$pkg.PackageFullName] = $true
        try {
            Remove-AppxPackage -Package $pkg.PackageFullName -AllUsers -ErrorAction Stop
            $removedAppx += $pkg.Name
        } catch {
            # Continue malgré l'erreur (certains paquets system ne se laissent pas faire)
        }
    }

    # Provisioned (image OS)
    $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like $pattern.Trim('*') -or $_.PackageName -like $pattern
    }
    foreach ($prov in $provisioned) {
        $key = "prov:$($prov.PackageName)"
        if ($alreadyHandled.ContainsKey($key)) { continue }
        $alreadyHandled[$key] = $true
        try {
            Remove-AppxProvisionedPackage -Online -PackageName $prov.PackageName -ErrorAction Stop | Out-Null
            $removedProvisioned += $prov.DisplayName
        } catch {}
    }
}

@{
    ok                  = $true
    before              = @{ note = 'before-state not stored (irreversible)' }
    after               = @{
        AppxRemoved        = $removedAppx
        ProvisionedRemoved = $removedProvisioned
    }
} | ConvertTo-Json -Compress -Depth 10
