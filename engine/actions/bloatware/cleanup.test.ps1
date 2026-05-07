# cleanup.test.ps1
# Conforme = aucun des patterns ne correspond à une app installée ou provisioned.

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

$installedAppx = @()
$installedProvisioned = @()

foreach ($pattern in $patterns) {
    $pkgs = Get-AppxPackage -Name $pattern -AllUsers -ErrorAction SilentlyContinue
    foreach ($p in $pkgs) {
        $installedAppx += $p.Name
    }
    $provs = Get-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like $pattern.Trim('*') -or $_.PackageName -like $pattern
    }
    foreach ($p in $provs) {
        $installedProvisioned += $p.DisplayName
    }
}

$compliant = ($installedAppx.Count -eq 0) -and ($installedProvisioned.Count -eq 0)

@{
    compliant = $compliant
    current   = @{
        AppxInstalled        = $installedAppx
        ProvisionedInstalled = $installedProvisioned
        TotalToRemove        = $installedAppx.Count + $installedProvisioned.Count
    }
} | ConvertTo-Json -Compress -Depth 10
