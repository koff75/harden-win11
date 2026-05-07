# Developing harden-win11 v2

## Prérequis

- Windows 11
- Go 1.26+ (`go version`)
- PowerShell 5.1+ (intégré Win11)
- Pester 5.7.1+

### Installer Pester 5

Méthode standard :

```powershell
Install-Module Pester -RequiredVersion 5.7.1 -Force -SkipPublisherCheck -Scope CurrentUser
```

**Si Controlled Folder Access bloque** (erreur du genre `Could not find a part of the path 'C:\Users\<you>\Documents\WindowsPowerShell\Modules\Pester\5.7.1'`), installer en local dans le repo :

```powershell
New-Item -ItemType Directory -Force -Path tools\pester | Out-Null
Save-Module -Name Pester -RequiredVersion 5.7.1 -Path tools\pester -Force
```

Le dossier `tools/` est gitignored. Pour charger Pester depuis cet emplacement :

```powershell
Import-Module .\tools\pester\Pester\5.7.1\Pester.psd1 -Force
```

## Build

```powershell
go build -o dist/harden-engine.exe ./cmd/harden-engine
```

## Tests

```powershell
# Tests Go (7 packages)
go test ./...

# Tests Pester (snippets PowerShell, ~71 tests sur defender + firewall)
Import-Module .\tools\pester\Pester\5.7.1\Pester.psd1 -Force
Invoke-Pester -Path engine\actions
```

## Commandes CLI

### `version`

```powershell
.\dist\harden-engine.exe version
```

JSON `{version, manifest_version, go, os, arch}` sur stdout.

### `validate`

Vérifie tous les `manifests/*.yaml` contre le JSONSchema, détecte les
collisions de `section.id` et de `rule.id` dans une section.

```powershell
.\dist\harden-engine.exe validate
```

Exit codes : 0 OK, 3 manifest invalid ou collision détectée, 4 input
invalide (dossier absent, schéma invalide).

### `apply --dry-run`

Lit l'état système via les `.test.ps1` sans rien modifier. Pas besoin
d'admin. Pas de journal sur disque (output stdout uniquement).

```powershell
# Toutes les sections
.\dist\harden-engine.exe apply --dry-run

# Une section
.\dist\harden-engine.exe apply --dry-run --section defender
```

Statuses dans `action_result` : `would_skip` (déjà conforme) /
`would_apply` (à appliquer) / `would_fail` (test.ps1 a planté, souvent
admin requis).

### `apply` (réel)

Exécute les `.action.ps1` pour les règles non-conformes, en gardant le
`before` state pour permettre `undo`. **Nécessite admin.** Demande
confirmation interactive (skip avec `--yes`).

```powershell
# Dans une PowerShell élevée
.\dist\harden-engine.exe apply
.\dist\harden-engine.exe apply --section defender
.\dist\harden-engine.exe apply --yes        # CI / scripting
```

Statuses dans `action_result` : `skipped` (déjà conforme) /
`applied` (action OK) / `failed` (action plantée, pas de rollback) /
`rolled_back` (action plantée + .undo.ps1 exécuté avec succès).

**Auto-rollback** : si une `.action.ps1` plante après que `.test.ps1`
a capturé l'état, le moteur lance immédiatement `.undo.ps1` avec ce
`before` state, émet un `rollback_result` event, et **stoppe le run**
(les règles déjà appliquées restent OK).

### `undo`

Restaure l'état avant un run en rejouant les `.undo.ps1` dans l'ordre
inverse, à partir du journal NDJSON sur disque.

```powershell
# Undo le dernier run complet
.\dist\harden-engine.exe undo

# Undo un run précis
.\dist\harden-engine.exe undo --run-id 2026-05-08T14-23-00

# Undo une seule règle
.\dist\harden-engine.exe undo --rule-id defender.cloud_protection

# Skip la confirmation interactive
.\dist\harden-engine.exe undo --yes
```

Émet des events `undo_result` dans un journal séparé `undo-<id>.ndjson`.
Les règles `irreversible: true` ou sans `.undo.ps1` sont skippées
silencieusement.

## Journal NDJSON

Path par défaut : `%ProgramData%\Harden-Win11\runs\<run_id>.ndjson`
(machine-wide, écrit par apply/undo, requiert admin).

Override : flag `--journal-dir` (utile pour tests, CI, dev local).

Le journal est crash-safe : `file.Sync()` après chaque event, donc un
crash du process préserve l'audit trail.

### Format des events

| `type` | Émis quand | Champs principaux |
|---|---|---|
| `run_start` | Début du run global | `run_id`, `mode`, `engine_version`, `sections`, `journal_path` |
| `section_start` | Début d'une section | `section_id`, `section_order`, `rule_count` |
| `action_result` | Pour chaque règle | `rule_id`, `status`, `current_state` ou `before`/`after`, `duration_ms` |
| `rollback_result` | Auto-rollback déclenché | `rule_id`, `status: rollback_ok`/`rollback_failed`, `trigger_err` |
| `section_end` | Fin de section | `section_id`, optionnel `aborted: true` |
| `run_end` | Fin du run | `skipped`, `applied`, `failed`, `rolled_back` |
| `undo_result` | Pour chaque undo | `rule_id`, `status: ok`/`failed`/`skipped` |

## Exit codes

| Code | Signification |
|---|---|
| 0 | OK |
| 1 | Erreur générique non catégorisée |
| 2 | Run partiellement échoué (some `failed` ou `aborted` après auto-rollback) |
| 3 | Manifest invalide ou collision détectée par validate |
| 4 | Input invalide (dossier absent, section inconnue, schema invalide) |
| 5 | Privilèges admin requis (apply réel ou undo) |
| 6 | Cancelled by user (refus de la confirmation interactive) |

## Structure du repo

- `cmd/harden-engine/` : binaire CLI (Cobra)
- `pkg/engine/` :
  - `ndjson/` : writer JSON line-delimited (thread-safe)
  - `runner/` : exécution de snippets PS avec I/O JSON
  - `manifest/` : types + loader YAML + validator JSONSchema
  - `executor/` : orchestration dry-run et apply (avec auto-rollback)
  - `journal/` : lecture/écriture NDJSON sur disque
  - `winadmin/` : détection privilèges admin (Windows-only avec stub Linux)
- `manifests/` : YAML descriptifs des règles (`NN-<section>.yaml`)
- `engine/actions/` : snippets PowerShell par règle (action/test/undo + tests Pester)
- `schemas/` : JSONSchema de validation des manifests
- `tools/` : helpers locaux (gitignored — Pester install, générateurs)
- `Harden-Win11.ps1` : script v1 (à la racine, à déplacer dans `legacy/` plus tard)

## Conventions snippets PowerShell

- Tests Pester : `<rule>.tests.ps1` dans le même dossier que le snippet
- Snippets PS doivent :
  - Lire JSON sur stdin via `[Console]::In.ReadToEnd()` (runner Go) **ou**
    `$MyInvocation.ExpectingInput` + `$input | Out-String` (pipeline Pester)
  - Émettre JSON compact sur stdout (1 ligne, `ConvertTo-Json -Compress`)
  - Set `[Console]::OutputEncoding = [System.Text.Encoding]::UTF8`
- Encodage : UTF-8 (avec BOM pour `.ps1` pour bon parsing PS 5.1 avec accents)

## Schéma JSON de sortie attendu par snippet

### `<rule>.test.ps1`

```json
{ "compliant": true|false, "current": { ... } }
```

`compliant` est obligatoire et doit être un bool strict (le moteur fail
si manquant ou typé `string` etc.).

### `<rule>.action.ps1`

```json
{ "ok": true, "before": { ... }, "after": { ... } }
```

`before` doit contenir tout ce qu'il faut à `.undo.ps1` pour restaurer
l'état. C'est le contrat entre action et undo.

### `<rule>.undo.ps1`

Reçoit `before` sur stdin, doit produire :

```json
{ "ok": true }
```

## Prochaines étapes

- Pester sur les ~33 règles registry skipped (Network/UAC/Privacy)
- GUI/TUI
- Profile auto-detection (`profile_when` évalué runtime)
- CI/CD + release binaire signé

Voir `docs/superpowers/plans/` pour les plans détaillés.
