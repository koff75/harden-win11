# harden-win11

Outil de hardening Windows 11 — applique une baseline de sécurité reproductible, auditée, et **réversible**. 69 règles couvrant Defender, ASR, Firewall, comptes locaux, UAC/RDP, hardening réseau (LLMNR/NTLM/SMB), privacy/telemetry et bloatware.

```powershell
# Vérifier ce qui serait modifié (sans rien toucher, sans admin requis)
.\dist\harden-engine.exe apply --dry-run

# Appliquer (admin requis, demande confirmation interactive)
.\dist\harden-engine.exe apply

# Revenir en arrière sur le dernier run
.\dist\harden-engine.exe undo
```

## Pourquoi v2 ?

Le v1 (`Harden-Win11.ps1`) reste fonctionnel à la racine du repo pour usage simple. Le v2 (binaire Go `harden-engine.exe`) ajoute :

- **Réversibilité** : chaque règle a un `.undo.ps1` qui restaure l'état avant. La commande `undo` rejoue l'inverse depuis le journal sur disque.
- **Audit trail** : chaque run est loggé en NDJSON dans `%ProgramData%\Harden-Win11\runs\<run_id>.ndjson` (machine-wide, crash-safe avec `fsync` après chaque event).
- **Auto-rollback** : si une action plante en cours d'apply, le moteur lance immédiatement le `.undo.ps1` correspondant et stoppe le run. Les règles déjà appliquées restent intactes.
- **Validation stricte** : JSONSchema + détection de collisions de `section.id` / `rule.id` cross-fichiers + types stricts.
- **Granularité** : `--section <id>` cible une catégorie ; `undo --rule-id <id>` revert une règle précise.
- **Manifests YAML** : chaque règle est décrite dans un fichier YAML auditable (sévérité, impact, références, irréversibilité documentée). Pas de logique business cachée dans le PS.

## Quick start

### 1. Build

```powershell
git clone https://github.com/koff75/harden-win11.git
cd harden-win11
go build -o dist/harden-engine.exe ./cmd/harden-engine
```

Prérequis : Go 1.26+, Windows 11, PowerShell 5.1+ (intégré).

### 2. Voir ce qui serait modifié

Pas besoin d'admin pour le dry-run :

```powershell
.\dist\harden-engine.exe apply --dry-run
```

Sortie : un flux NDJSON sur stdout avec un event `action_result` par règle. Le `status` indique :

- `would_skip` — la règle est déjà conforme
- `would_apply` — la règle modifierait l'état (en mode réel)
- `would_fail` — le test a planté (souvent admin requis)

Exemple sur une machine fraîchement installée :

```
defender.realtime                    would_skip   (déjà ON)
defender.cloud_protection            would_apply  (CloudBlockLevel pas High)
firewall.profile_public              would_apply  (DefaultInbound pas Block explicite)
network.llmnr_disable                would_apply  (LLMNR pas désactivé)
privacy.recall_off                   would_apply  (Recall pas bloqué)
asr.block_lsass_credential_theft     would_apply  (ASR rule pas activée)
...
```

### 3. Appliquer (avec confirmation)

Dans une PowerShell **élevée** :

```powershell
.\dist\harden-engine.exe apply --section defender   # une section
.\dist\harden-engine.exe apply                       # toutes les sections
.\dist\harden-engine.exe apply --yes                 # skip confirmation (CI/scripting)
```

Le moteur :
1. Re-valide tous les manifests contre le JSONSchema
2. Demande confirmation interactive (sauf `--yes`)
3. Pour chaque règle, lance `.test.ps1` puis `.action.ps1` si non-conforme
4. Capture `before` dans le journal pour permettre `undo`
5. Si une action plante → auto-rollback via `.undo.ps1` + stop

### 4. Revenir en arrière

```powershell
# Undo le dernier run complet (LIFO sur les rules applied)
.\dist\harden-engine.exe undo

# Undo un run précis
.\dist\harden-engine.exe undo --run-id 2026-05-08T14-23-00

# Undo une seule règle
.\dist\harden-engine.exe undo --rule-id defender.cloud_protection
```

Les règles marquées `irreversible: true` (ex: `defender.signatures` car on ne désinstalle pas une signature antivirus, `defender.tamper_protection_check` car TP n'a pas d'API programmatique) sont skippées avec un message explicite.

## Architecture

```
manifests/                  YAML descriptifs (1 par section)
  01-defender.yaml          12 règles
  02-firewall.yaml          5 règles
  03-accounts.yaml          2 règles
  04-system_settings.yaml   8 règles (UAC, RDP, Power)
  05-network.yaml           9 règles (LLMNR, NTLM, SMB)
  06-privacy.yaml           13 règles (telemetry, AdID, Cortana, Recall)
  07-bloatware.yaml         1 règle agrégée (27 patterns d'apps Store)
  08-asr.yaml               19 règles ASR (Attack Surface Reduction)

engine/actions/             snippets PowerShell par règle
  defender/
    realtime.action.ps1     active la rule
    realtime.test.ps1       lit l'état, retourne {compliant, current}
    realtime.undo.ps1       restaure l'état à partir du 'before' fourni
    realtime.tests.ps1      tests Pester (pour les snippets non triviaux)
  ...
  _helpers/reg.psm1         module PS qui factorise le pattern registre

cmd/harden-engine/          binaire Go CLI (Cobra)
pkg/engine/                 library Go partagée
  manifest/                 types + loader YAML + validator JSONSchema
  runner/                   spawn de PS avec I/O JSON
  executor/                 orchestration dry-run et apply (+ auto-rollback)
  journal/                  lecture/écriture NDJSON sur disque
  ndjson/                   writer thread-safe
  winadmin/                 détection privilèges admin
schemas/manifest.schema.json   JSONSchema 2020-12, validation stricte
```

## Sécurité du moteur

- **Validation systématique** : `apply` re-valide les manifests avant de toucher au système (trust + verify).
- **Détection des collisions** : `section.id` et `rule.id` cross-fichiers refusés à `validate` et `apply`.
- **YAML strict** : `KnownFields(true)`, refus des multi-document YAML, types stricts.
- **Timeout par règle** : 30s par défaut (configurable via `--rule-timeout`). Une règle qui hang ne fait pas hanger le moteur.
- **`fsync` après chaque event** : le journal sur disque survit à un crash.
- **Pas de path traversal** : les paths absolus dans `rule.action`/`test`/`undo` sont supportés volontairement (cas tests E2E) ; les paths relatifs sont résolus contre la racine du repo.
- **Admin requis** pour `apply` réel et `undo`. Détecté via probe `%SystemRoot%\Temp\`.

## Exit codes

| Code | Signification |
|---|---|
| 0 | OK |
| 1 | Erreur générique non catégorisée |
| 2 | Run partiellement échoué (some `failed` ou `aborted` après auto-rollback) |
| 3 | Manifest invalide ou collision détectée |
| 4 | Input invalide (dossier absent, section inconnue, schema invalide) |
| 5 | Privilèges admin requis |
| 6 | Cancelled par l'utilisateur |

## Tester / contribuer

```powershell
# Tests Go (7 packages)
go test ./...

# Tests Pester (~71 tests sur les snippets defender + firewall)
Import-Module .\tools\pester\Pester\5.7.1\Pester.psd1 -Force
Invoke-Pester -Path engine\actions
```

Voir [`docs/DEVELOPING.md`](docs/DEVELOPING.md) pour le détail des conventions, format des events NDJSON, schéma JSON attendu par les snippets, et étapes pour ajouter une nouvelle règle.

## Status & roadmap

- ✅ **SP1 — Walking skeleton** : moteur complet, 69 règles migrées (parité v1)
- ✅ **SP2 — Outil utilisable** : apply réel, journal disque, undo, admin check, auto-rollback
- 🟨 **SP3 (futur)** : profile auto-detection (`profile_when` runtime), GUI/TUI, CI/CD release signée

## v1 (script PowerShell legacy)

Le script original `Harden-Win11.ps1` est toujours à la racine du repo pour usage simple sans build Go.

```powershell
.\Harden-Win11.ps1                # menu interactif
.\Harden-Win11.ps1 -DryRun         # test à blanc
.\Harden-Win11.ps1 -AsrAuditMode   # ASR en mode audit (loggent sans bloquer)
.\Harden-Win11.ps1 -SkipBloatware  # garde les apps Store
.\Harden-Win11.ps1 -Quiet          # pas de menu (CI)
```

## Licence

WTFPL — Do whatever you want with this. Pas de garantie. Tu es responsable de ce que tu lances sur ta machine.
