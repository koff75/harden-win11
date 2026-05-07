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
# Tests Go (5 packages)
go test ./...

# Tests Pester (snippets PowerShell)
Invoke-Pester engine/actions/defender/realtime.tests.ps1 -Output Detailed
```

## Lancer le walking skeleton

```powershell
# Affiche les versions
.\dist\harden-engine.exe version

# Validation des manifests contre le JSONSchema
.\dist\harden-engine.exe validate

# Dry-run sur la section defender (n'exécute rien, lit l'état)
.\dist\harden-engine.exe apply --dry-run --section defender
```

Sortie attendue de `apply --dry-run` : 3+ lignes NDJSON sur stdout (`run_start`, un `action_result` par règle, `run_end`).

## Structure

- `cmd/harden-engine/` : binaire CLI
- `pkg/engine/` : library partagée
  - `ndjson/` : writer d'events JSON line-delimited
  - `runner/` : exécution de snippets PowerShell avec I/O JSON
  - `manifest/` : types + loader YAML + validator JSONSchema
  - `dryrun/` : orchestration du dry-run (test des règles, émission d'events)
- `manifests/` : YAML descriptifs des règles (`NN-<section>.yaml`)
- `engine/actions/` : snippets PowerShell par règle (`<rule>.action.ps1`, `.test.ps1`, `.undo.ps1`)
- `schemas/` : JSONSchema de validation des manifests
- `Harden-Win11.ps1` : script v1 (toujours à la racine, pas encore migré dans `legacy/`)

## Conventions

- Tests Go : nom de fichier `*_test.go`, fonctions `TestXxx(t *testing.T)`
- Tests Pester : nom de fichier `*.tests.ps1`, dans le même dossier que le snippet testé
- Snippets PS : doivent lire JSON sur stdin (ou pipeline PS pour les tests Pester via `$MyInvocation.ExpectingInput`), émettre JSON compact sur stdout (1 ligne)
- Encodage : tous les fichiers en UTF-8 (avec BOM pour les `.ps1` pour bon parsing PS 5.1)

## Prochaines étapes

Voir `docs/superpowers/plans/2026-05-07-sp1-walking-skeleton.md` pour le plan en cours, et `docs/superpowers/specs/2026-05-07-v2-sp1-core-engine-design.md` pour le design global de SP1.
