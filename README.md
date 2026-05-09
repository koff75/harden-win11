# harden-win11

Outil de hardening Windows 11 — applique une baseline de sécurité reproductible, auditée et **réversible**. **95 règles** couvrant Defender, ASR, Firewall, comptes locaux, UAC/RDP/Power, hardening réseau (LLMNR/NTLM/SMB), privacy/telemetry, et 27 bloatware Microsoft Store individuellement sélectionnables.

Disponible en **GUI Wails** (clic-clic, dashboard, filtres, exclusion par règle) et en **CLI Go** (scripting, CI, batch).

## Quick start

### GUI (recommandé pour usage perso)

Lance `harden-gui.exe` (clic droit → Exécuter en tant qu'administrateur). La GUI :

- Détecte le contexte (machine AD-joined ? imprimante réseau ?) et **suggère un profil de risque** (personal / business / maximal)
- Charge les 95 règles, te montre l'état actuel et ce qui changerait
- Coverage bar : `CIS X% · ANSSI Y% · MS Z%` vs les référentiels publics
- Décoche les règles que tu ne veux pas (ex: garder Spotify mais virer TikTok)
- Apply / Annuler avec auto-rollback si ça plante

Si tu lances la GUI **sans admin**, un bandeau rouge apparaît avec un bouton "Relancer en admin".

### CLI

```powershell
# Voir ce qui serait modifié (pas d'admin requis pour le dry-run)
.\dist\harden-engine.exe apply --dry-run

# Vite (dryrun parallèle, ~36s au lieu de 116s)
.\dist\harden-engine.exe apply --dry-run --parallel 4

# Couverture vs CIS / ANSSI / MS Security Baseline
.\dist\harden-engine.exe coverage

# Apply ciblé sur le profil "personal" (skip les règles trop strictes pour usage perso)
.\dist\harden-engine.exe apply --profile personal

# ASR en mode audit (log les events Defender sans bloquer)
.\dist\harden-engine.exe apply --audit --section asr

# Annuler le dernier run
.\dist\harden-engine.exe undo
```

## Pourquoi v2 ?

Le v1 (`Harden-Win11.ps1`) reste à la racine pour usage simple. Le v2 ajoute :

- **Réversibilité** : chaque règle a un `.undo.ps1` qui restaure l'état avant. La commande `undo` rejoue l'inverse depuis le journal sur disque.
- **Audit trail** : chaque run est loggé en NDJSON dans `%ProgramData%\Harden-Win11\runs\<run_id>.ndjson` (machine-wide, crash-safe avec `fsync` après chaque event).
- **Auto-rollback** : si une action plante en cours d'apply, le moteur lance immédiatement le `.undo.ps1` correspondant et stoppe le run. Les règles déjà appliquées restent intactes.
- **Profils de risque** : `personal` (usage maison), `business` (PME), `maximal` (paranoid). Chaque règle déclare les profils où elle s'applique + un champ `breaks` qui liste user-facing ce qu'elle peut casser (ex: "RDP entrant", "imprimantes Bonjour").
- **Validation stricte** : JSONSchema 2020-12 + détection collisions cross-fichiers + types stricts.
- **Mapping baseline** : 66 règles sur 95 mappées vers CIS Win11 / ANSSI / MS Security Baseline. Affichable via `harden-engine coverage`.
- **GUI Wails** : dashboard, filtres, exclusion par règle, coverage panel, bandeau admin, détection contexte.

## Build

Prérequis : Go 1.26+, Windows 11, PowerShell 5.1+.

```powershell
git clone https://github.com/koff75/harden-win11.git
cd harden-win11

# Engine CLI
go build -o dist/harden-engine.exe ./cmd/harden-engine

# GUI (nécessite wails CLI)
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
cd cmd/harden-gui
wails build
```

Pour un release portable (ZIP avec engine + GUI + manifests + run-as-admin.bat) :

```powershell
pwsh -File tools/build-release.ps1 -Version 0.2.0
# → build/Harden-Win11-0.2.0.zip + .sha256
```

## Architecture

```
manifests/                  YAML descriptifs (1 par section)
  01-defender.yaml          12 règles
  02-firewall.yaml          5 règles
  03-accounts.yaml          2 règles
  04-system_settings.yaml   8 règles (UAC, RDP, Power)
  05-network.yaml           9 règles (LLMNR, NTLM, SMB)
  06-privacy.yaml           13 règles (telemetry, AdID, Cortana, Recall)
  08-asr.yaml               19 règles ASR (Attack Surface Reduction)
  09-bloatware.yaml         27 règles (TikTok, Spotify, Candy Crush, …)

mappings/baselines.yaml     mapping rule → CIS / ANSSI / MS controls

engine/actions/             snippets PowerShell par règle
  defender/
    realtime.action.ps1
    realtime.test.ps1
    realtime.undo.ps1
    realtime.tests.ps1      tests Pester
  _helpers/
    reg.psm1                helper registre (factorisation)
    harden_appx.psm1        helper Appx (bloatware)

cmd/harden-engine/          binaire Go CLI (Cobra)
cmd/harden-gui/             binaire Wails (GUI)
pkg/engine/
  manifest/                 types + loader YAML + validator JSONSchema
  runner/                   spawn de PS avec I/O JSON
  executor/                 orchestration dry-run / apply / parallel pool
  journal/                  lecture/écriture NDJSON
  ndjson/                   writer thread-safe
  winadmin/                 détection admin via TokenElevation API
  baseline/                 calcul couverture vs CIS/ANSSI/MS

schemas/manifest.schema.json   JSONSchema 2020-12, validation stricte
tools/                          smoke-test, audit-coherence, sign, build-release
docs/                           smoke-test.md, test-report-2026-05-09.md
```

## Sécurité du moteur

- **Validation systématique** : `apply` re-valide les manifests avant de toucher au système (trust + verify).
- **Détection des collisions** : `section.id` et `rule.id` cross-fichiers refusés à `validate` et `apply`.
- **YAML strict** : `KnownFields(true)`, refus des multi-documents, types stricts.
- **Timeout par règle** : 30s par défaut, configurable via `--rule-timeout`.
- **`fsync` après chaque event** : le journal survit à un crash brutal.
- **Détection admin via API officielle** : `OpenProcessToken` + `GetTokenInformation(TokenElevation)`. Plus d'heuristique fragile.
- **Admin requis** pour `apply` réel (sans `--dry-run`) et `undo`.

## Exit codes

| Code | Signification |
|------|---------------|
| 0 | OK |
| 1 | Erreur générique non catégorisée |
| 2 | Run partiellement échoué (`failed` ou `aborted` après auto-rollback) |
| 3 | Manifest invalide ou collision détectée |
| 4 | Input invalide (dossier absent, section inconnue, schema invalide) |
| 5 | Privilèges admin requis |
| 6 | Cancelled par l'utilisateur |

## Tests

```powershell
# Tests Go (8 packages)
go test ./...

# Tests Pester (98 tests sur les snippets PS)
Import-Module .\tools\pester\Pester\5.7.1\Pester.psd1 -Force
Invoke-Pester -Path engine\actions

# Audit cohérence statique (manifests ↔ scripts)
.\tools\audit-coherence.ps1

# Smoke E2E (build + validate + coverage + dryrun)
pwsh -File tools/smoke-test.ps1
```

CI GitHub Actions exécute les 4 sur chaque push : `.github/workflows/ci.yml`.

Procédure de smoke test sur VM Win11 propre (Home + Pro) : voir [`docs/smoke-test.md`](docs/smoke-test.md).

## Status & roadmap

- ✅ **SP1** — Walking skeleton (engine + 95 règles)
- ✅ **SP2** — Apply réel + journal + undo + auto-rollback + admin check
- ✅ **SP3** — Profils de risque + détection contexte + mode audit + bloatware split (27 règles)
- ✅ **GUI** — Dashboard + filtres + coverage + admin banner + per-rule exclusion
- ✅ **Baseline** — Mapping CIS / ANSSI / MS Security Baseline (66/95 règles)
- ✅ **Tooling** — Smoke E2E, audit-coherence, build-release ZIP, sign-release
- 🟨 **À venir** : Restore Point Windows automatique avant apply, re-test post-apply, vue Historique GUI, persistance préférences

## v1 (script PowerShell legacy)

Toujours à la racine pour usage simple sans build Go.

```powershell
.\Harden-Win11.ps1                # menu interactif
.\Harden-Win11.ps1 -DryRun         # test à blanc
.\Harden-Win11.ps1 -AsrAuditMode   # ASR en mode audit
.\Harden-Win11.ps1 -SkipBloatware  # garde les apps Store
.\Harden-Win11.ps1 -Quiet          # pas de menu (CI)
```

## Licence

WTFPL — Do whatever you want with this. Pas de garantie. Tu es responsable de ce que tu lances sur ta machine.
