# gen-bloatware-rules.ps1
# Genere 27 regles bloatware individuelles (1 par pattern d'app) + le
# manifest 07-bloatware.yaml. Permet a l'utilisateur de decocher chaque
# app individuellement dans la GUI au lieu d'un cleanup all-or-nothing.

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Liste des apps Store : pattern + slug (rule_id) + titre humain + categorie + profils.
# Categories servent a regrouper visuellement et a justifier le profil.
$apps = @(
    # Microsoft preinstall apps - presents partout, peu utiles
    @{ pattern = '*Clipchamp*';                                slug = 'clipchamp';        title = 'Clipchamp (editeur video Microsoft)';            category = 'microsoft'; profiles = @('personal', 'maximal') }
    @{ pattern = '*Microsoft.BingNews*';                       slug = 'bing_news';        title = 'Microsoft Bing News';                            category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.BingWeather*';                    slug = 'bing_weather';     title = 'Microsoft Bing Weather';                         category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.GetHelp*';                        slug = 'get_help';         title = 'Get Help (assistant Microsoft)';                 category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.Getstarted*';                     slug = 'get_started';      title = 'Get Started (tour Win11)';                       category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.MicrosoftSolitaireCollection*';  slug = 'solitaire';        title = 'Microsoft Solitaire Collection';                 category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.MixedReality.Portal*';            slug = 'mixed_reality';    title = 'Mixed Reality Portal (VR)';                      category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.People*';                         slug = 'people';           title = 'People (carnet d adresses)';                     category = 'microsoft'; profiles = @('personal', 'maximal') }
    @{ pattern = '*Microsoft.SkypeApp*';                       slug = 'skype';            title = 'Skype';                                          category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.WindowsFeedbackHub*';             slug = 'feedback_hub';     title = 'Feedback Hub';                                   category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.YourPhone*';                      slug = 'your_phone';       title = 'Your Phone (Phone Link)';                        category = 'microsoft'; profiles = @('personal', 'maximal') }
    @{ pattern = '*Microsoft.ZuneMusic*';                      slug = 'zune_music';       title = 'Groove Music (Zune)';                            category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Microsoft.ZuneVideo*';                      slug = 'zune_video';       title = 'Films & TV (Zune Video)';                        category = 'microsoft'; profiles = @('personal', 'business', 'maximal') }

    # Reseaux sociaux et streaming - pre-installes en partenariat OEM
    @{ pattern = '*Disney*';                                   slug = 'disney';           title = 'Disney+ (preinstalle OEM)';                       category = 'streaming';   profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*TikTok*';                                   slug = 'tiktok';           title = 'TikTok';                                         category = 'social';      profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Facebook*';                                 slug = 'facebook';         title = 'Facebook';                                       category = 'social';      profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Instagram*';                                slug = 'instagram';        title = 'Instagram';                                      category = 'social';      profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*Twitter*';                                  slug = 'twitter';          title = 'Twitter / X';                                    category = 'social';      profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*LinkedInforWindows*';                       slug = 'linkedin';         title = 'LinkedIn';                                       category = 'social';      profiles = @('personal', 'maximal') }
    @{ pattern = '*Netflix*';                                  slug = 'netflix';          title = 'Netflix';                                        category = 'streaming';   profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*CandyCrush*';                               slug = 'candy_crush';      title = 'Candy Crush';                                    category = 'games';       profiles = @('personal', 'business', 'maximal') }

    # Music / Media tiers
    @{ pattern = '*SpotifyAB*';                                slug = 'spotify_ab';       title = 'Spotify (variante OEM)';                         category = 'media';       profiles = @('personal', 'maximal') }
    @{ pattern = '*Spotify*';                                  slug = 'spotify';          title = 'Spotify';                                        category = 'media';       profiles = @('personal', 'maximal') }
    @{ pattern = '*AppleMusicWin*';                            slug = 'apple_music';      title = 'Apple Music';                                    category = 'media';       profiles = @('personal', 'maximal') }
    @{ pattern = '*DolbyLaboratories.DolbyAccess*';            slug = 'dolby_access';     title = 'Dolby Access';                                   category = 'media';       profiles = @('personal', 'maximal') }

    # Patterns OEM divers (publishers identifies par GUID)
    @{ pattern = '*JimmyLin*';                                 slug = 'jimmylin';         title = 'Apps OEM publisher JimmyLin (LiveOS games)';      category = 'oem';         profiles = @('personal', 'business', 'maximal') }
    @{ pattern = '*5319275A*';                                 slug = 'pub_5319275a';     title = 'Apps OEM publisher 5319275A';                     category = 'oem';         profiles = @('personal', 'business', 'maximal') }
)

$root = Split-Path -Parent $PSScriptRoot
$bloatDir = Join-Path $root 'engine\actions\bloatware'
$manifestPath = Join-Path $root 'manifests\09-bloatware.yaml'

# Backup l'ancien cleanup avant suppression (si on doit revert).
if (Test-Path "$bloatDir\cleanup.action.ps1") {
    Remove-Item "$bloatDir\cleanup.action.ps1" -Force
    Remove-Item "$bloatDir\cleanup.test.ps1" -Force
}

# Templates
$actionTemplate = @'
# {SLUG}.action.ps1
# Desinstalle l'app Store '{TITLE}' (pattern : {PATTERN}).

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxRemove -Pattern '{PATTERN}'
'@

$testTemplate = @'
# {SLUG}.test.ps1

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Import-Module (Join-Path $PSScriptRoot '..\_helpers\appx.psm1') -Force

Invoke-AppxTest -Pattern '{PATTERN}'
'@

# Genere les fichiers PS
$count = 0
foreach ($app in $apps) {
    $slug = $app.slug
    $pattern = $app.pattern
    $title = $app.title

    foreach ($kind in @('action', 'test')) {
        $template = if ($kind -eq 'action') { $actionTemplate } else { $testTemplate }
        $content = $template
        $content = $content.Replace('{SLUG}', $slug)
        $content = $content.Replace('{PATTERN}', $pattern)
        $content = $content.Replace('{TITLE}', $title)
        $file = Join-Path $bloatDir "$slug.$kind.ps1"
        [System.IO.File]::WriteAllText($file, $content, [System.Text.UTF8Encoding]::new($false))
        $count++
    }
}

# Genere le manifest
$header = @'
version: "1.0"

section:
  id: bloatware
  order: 9
  title: "Bloatware Microsoft Store"
  description: "Desinstallation des apps Store consommateurs. Tu peux selectionner individuellement celles que tu veux supprimer."

rules:
'@

$ruleTemplate = @'

  - id: bloatware.{SLUG}
    title: "{TITLE}"
    description: "Desinstalle '{TITLE}' (Appx + provisioned)."
    explanation: |
      App Store {CATEGORY} preinstallee sur Win11 (pattern Appx : {PATTERN}).
      Cette regle desinstalle l'app pour TOUS les comptes utilisateur de la
      machine (Get-AppxPackage -AllUsers + Get-AppxProvisionedPackage).
    severity: nice-to-have
    impact: "L'app disparait. Reinstallation manuelle via Microsoft Store."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: true
    irreversible_reason: "Reinstaller une app Store desinstallee requiert le Microsoft Store + un compte Microsoft connecte. Pas une operation scriptable de facon fiable."
    references:
      - "https://learn.microsoft.com/en-us/powershell/module/appx/remove-appxpackage"
    tags: [bloatware, store, {CATEGORY}]
    added_in: "1.0"
    action: ./engine/actions/bloatware/{SLUG}.action.ps1
    test: ./engine/actions/bloatware/{SLUG}.test.ps1
    profiles:
{PROFILE_LINES}
'@

$body = $header + "`n"
foreach ($app in $apps) {
    $entry = $ruleTemplate
    $entry = $entry.Replace('{SLUG}', $app.slug)
    $entry = $entry.Replace('{TITLE}', $app.title)
    $entry = $entry.Replace('{PATTERN}', $app.pattern)
    $entry = $entry.Replace('{CATEGORY}', $app.category)
    $profileLines = ($app.profiles | ForEach-Object { "      - $_" }) -join "`n"
    $entry = $entry.Replace('{PROFILE_LINES}', $profileLines)
    $body += $entry
}

[System.IO.File]::WriteAllText($manifestPath, $body, [System.Text.UTF8Encoding]::new($false))

Write-Host ("Generated {0} bloatware rules ({1} files)" -f $apps.Count, $count) -ForegroundColor Green
