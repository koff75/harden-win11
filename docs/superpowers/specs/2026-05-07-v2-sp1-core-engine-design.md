# Harden-Win11 v2 — SP1 Core Engine — Design Spec

**Date** : 2026-05-07
**Auteur** : koff75 (brainstorming avec Claude)
**Statut** : Draft — en attente de revue utilisateur
**Sous-projet** : SP1 (Core Engine + Knowledge Base) — premier de 3 sous-projets de la v2

---

## 1. Contexte & problème

Le projet `harden-win11` actuel est un script PowerShell mono-fichier (~820 lignes) qui applique une baseline de sécurité Windows 11 de manière idempotente. Il est utilisable en CLI par des techs.

**La v2 vise un produit grand public** avec GUI Wails (Go + WebView), à destination d'un jeune actif tech-curious (cible primaire) et d'une TPE/artisan/profession libérale (cible secondaire). L'utilisateur double-clique sur un .exe, l'app scanne son système, lui pose 3-5 questions ciblées, lui propose un plan personnalisé, exécute, et lui donne un rapport avec undo granulaire.

Pour rendre ça possible, le script PowerShell mono-fichier doit être refactoré en **rule engine piloté par manifest** : une couche d'orchestration en Go qui lit un manifest YAML décrivant les règles, exécute des snippets PowerShell atomiques, écrit un journal d'actions structuré, et expose une API consommable à la fois par un CLI standalone et par l'app Wails.

**Ce spec couvre uniquement SP1 (Core Engine + KB)**. SP2 (App Wails GUI) et SP3 (Distribution) feront l'objet de specs ultérieurs.

---

## 2. Goals & non-goals

### Goals (MVP v1.0)

- Refactor du script existant en **rule engine** piloté par un manifest YAML extensible
- Conserver les **8 sections existantes** (Defender, ASR, Firewall, Comptes locaux, UAC/RDP/Power, Réseau, Privacy, Bloatware) avec leurs règles déjà éprouvées
- Garantir l'**idempotence** par règle (test-then-act)
- Produire un **journal d'actions** structuré (NDJSON) qui permet l'**undo par action / par run / total**
- Intégrer un **System Restore Point** automatique avant chaque run modifiant l'état
- Supporter le **dry-run** par règle
- Exposer une **API CLI** stable (sous-commandes `scan`/`plan`/`apply`/`undo`/`journal`/`validate`/`version`) consommée à la fois par un binaire standalone `harden-engine.exe` et par l'app Wails
- Inclure 11 **détecteurs contextuels** (printer count, RDP, BitLocker, etc.) qui alimentent le profilage utilisateur
- Cible budget : 3-4 semaines réelles de dev (côté SP1)

### Non-goals (out of MVP, backlog v1.1+)

- BitLocker, DNS over HTTPS, Edge SmartScreen, Windows Update auto (scope B initial mais reportés au backlog)
- Alignement CIS Benchmark Win11 complet (v2.x)
- Code signing du binaire (cert payant — après le MVP)
- Versionning multi-engine du journal (un seul engine version embedé via `go:embed`)
- Internationalisation (FR uniquement au MVP)
- Cache cross-session des détecteurs
- Parallélisme des tests de règles
- Retention auto du journal

### Anti-goals (à ne pas faire)

- Réécrire la logique en Go pur (Win32 API + COM) → 3-6 mois de réécriture, hors-budget MVP
- Wrapper aveugle du script existant via shellout (perd journal granulaire + progress live)
- Engine d'expressions complexe (`and`/`or`/parenthèses) → si besoin, décomposer en plusieurs règles

---

## 3. Architecture overview

```
┌─────────────────────┐         ┌──────────────────────┐
│ harden-engine.exe   │         │ Harden-Win11.exe     │
│ (CLI standalone)    │         │ (Wails app — SP2)    │
└──────────┬──────────┘         └──────────┬───────────┘
           │                                │
           └────────────┬───────────────────┘
                        ▼
            ┌────────────────────────┐
            │  pkg/engine (Go)       │  ← lib partagée
            │  • parser manifest     │
            │  • runner              │
            │  • journal             │
            │  • restore point       │
            └────────────┬───────────┘
                         ▼
                ┌──────────────────┐
                │ powershell.exe   │  ← exécute les snippets atomiques
                │  -Command "..."  │
                └────────┬─────────┘
                         ▼
              [API Windows : registre, firewall,
               Defender, AppX, services, etc.]
```

### Décision clé : Go orchestre, PowerShell exécute

- **Go** : parsing YAML, gestion du journal, sequencing, état, expressions de profilage, écriture NDJSON, gestion des erreurs
- **PowerShell** : exécution des actions atomiques (1-15 lignes par snippet), parce que c'est lui qui sait parler aux API Windows (`Set-MpPreference`, `Set-NetFirewallProfile`, `Disable-WindowsOptionalFeature`, etc.)
- **Communication Go → PS** : `os/exec` (PowerShell.exe -Command) avec entrée JSON sur stdin, sortie JSON sur stdout, erreurs sur stderr
- **Pas de PowerShell long-running** : un nouveau process par snippet (overhead ~200-300ms par action, acceptable pour un total de 50 règles en ~30s)

### Séparation CLI / GUI

`harden-engine.exe` et `Harden-Win11.exe` (Wails) **importent tous les deux** `pkg/engine`. Pas de shellout entre eux. Le CLI est utile pour :
- Tests automatisés en CI (Pester + Go test)
- Power users / sysadmins en headless
- Forcer une API propre (la GUI ne peut pas prendre de raccourci sale)

---

## 4. Format du manifest & organisation

### Layout de fichiers

```
harden-win11/
├── manifests/                          ← knowledge base (YAML)
│   ├── 01-defender.yaml
│   ├── 02-asr.yaml
│   ├── 03-firewall.yaml
│   ├── 04-accounts.yaml
│   ├── 05-uac-rdp-power.yaml
│   ├── 06-network.yaml
│   ├── 07-privacy.yaml
│   ├── 08-bloatware.yaml
│   └── profiling.yaml                  ← détecteurs + questions
├── engine/
│   ├── actions/                        ← snippets PS par règle
│   │   ├── defender/
│   │   │   ├── realtime.action.ps1
│   │   │   ├── realtime.test.ps1
│   │   │   └── realtime.undo.ps1
│   │   └── ... (par section)
│   └── detectors/                      ← scripts de détection
│       ├── printer-count.ps1
│       ├── smb-shares-local.ps1
│       └── ... (11 au MVP)
├── pkg/engine/                         ← lib Go partagée
├── cmd/
│   ├── harden-engine/                  ← CLI standalone
│   └── harden-win11/                   ← app Wails (SP2)
└── schemas/
    └── manifest.schema.json            ← JSONSchema de validation
```

### Choix : YAML par section + snippets PS dans fichiers `.ps1` séparés

Justification :
- **YAML** : lisibilité humaine + commentaires + multi-line strings (vs JSON sans commentaires)
- **Un fichier par section** : alignement 1:1 avec les sections actuelles, fichiers tractables (~150-250 lignes), diff propre en code review
- **Snippets PS séparés** : syntax highlighting natif, lint via `PSScriptAnalyzer`, tests Pester standalone, pas d'escaping pénible

### Schéma manifest section : exemple `01-defender.yaml`

```yaml
version: "1.0"

section:
  id: defender
  order: 1
  title: "Microsoft Defender"
  description: "Antivirus intégré à Windows. On active toutes ses protections."

rules:
  - id: defender.realtime
    title: "Protection temps réel"
    description: "Scanner les fichiers à chaque ouverture / téléchargement."
    explanation: |
      Defender scanne en arrière-plan chaque fichier que tu ouvres.
      Sans ça, un malware peut s'exécuter sans alerte.

    severity: critical                  # critical | important | nice-to-have
    impact: "Aucun. Activé par défaut sur Win11."
    requires_reboot: false

    profile_when: always                # voir §5.3
    depends_on: []
    irreversible: false                 # voir §7.4

    references:
      - "https://learn.microsoft.com/.../configure-real-time-protection"
    tags: [malware, defender]
    added_in: "1.0"

    action: ./engine/actions/defender/realtime.action.ps1
    test:   ./engine/actions/defender/realtime.test.ps1
    undo:   ./engine/actions/defender/realtime.undo.ps1
```

### Champs règle (référence)

| Champ | Type | Obligatoire | Description |
|---|---|---|---|
| `id` | string | oui | Identifiant unique `<section>.<rule>` |
| `title` | string | oui | Titre court (1 ligne, GUI) |
| `description` | string | oui | Description (1-2 lignes, GUI) |
| `explanation` | string (multi-line) | oui | Paragraphe pédagogique (GUI "explique-moi vite") |
| `severity` | enum | oui | `critical` / `important` / `nice-to-have` |
| `impact` | string | oui | Conséquences pour l'utilisateur — *critique pour la cible* |
| `requires_reboot` | bool | oui | Pour grouper et prévenir |
| `profile_when` | string (expr) | oui | Expression de profilage (cf. §5.3) |
| `depends_on` | array<string> | oui | IDs requises avant. Peut être `[]` |
| `irreversible` | bool | non (default false) | Si true, pas de `.undo.ps1` requis (cf. §7.4) |
| `irreversible_reason` | string | si irreversible | Affiché par la GUI lors de la confirmation |
| `references` | array<string> | non | URLs / docs pour audit trail |
| `tags` | array<string> | non | Filtres GUI |
| `added_in` | string | non | Version de KB d'introduction |
| `action` | path | oui | Chemin vers `.action.ps1` |
| `test` | path | oui | Chemin vers `.test.ps1` |
| `undo` | path | non si irreversible | Chemin vers `.undo.ps1` |

### Validation

JSONSchema strict (`schemas/manifest.schema.json`) validé côté Go via `github.com/santhosh-tekuri/jsonschema` au démarrage. Si validation KO → engine refuse de démarrer (code retour `3`) avec un message explicite (fichier + ligne + champ fautif).

---

## 5. Profilage : détecteurs et questions

### 5.1 Distinction conceptuelle

| | `rule.test` (par règle) | détecteur (système) |
|---|---|---|
| Question | « Cette règle est-elle déjà conforme ? » | « Quel est le contexte ? » |
| Sortie | bool | int / bool / string / array |
| Quand | Avant chaque action | Une fois au démarrage du `scan` |
| Pilote | Idempotence | Questions à poser + règles applicables |

### 5.2 Liste des 11 détecteurs MVP

| ID | Output | Utilité |
|---|---|---|
| `os_edition` | string (Home/Pro/Enterprise) | Skipper règles non-applicables |
| `os_build` | int | Compatibilité min |
| `domain_joined` | string (workgroup/ad/aad) | Adapter conseils comptes |
| `account_type` | string (local/microsoft) | Idem |
| `printer_count` | int | Conditionne `q_old_printer` |
| `smb_shares_local` | int | Conditionne `q_uses_smb_share` |
| `smb_shares_remote` | int | Idem |
| `rdp_active` | bool | Skip `q_use_rdp` si déjà inactif |
| `bitlocker_status` | object (par drive) | Pour module BitLocker v1.1 |
| `defender_tamper` | bool | Signaler à l'utilisateur de l'activer |
| `appx_installed` | array<string> | Pré-remplir liste bloatware réelle |

### 5.3 Schéma `profiling.yaml`

```yaml
version: "1.0"

detectors:
  - id: printer_count
    description: "Compte les imprimantes installées"
    script: ./engine/detectors/printer-count.ps1
    output: int                         # int | bool | string | object | array
    timeout_ms: 5000

questions:
  - id: q_old_printer
    title: "As-tu une imprimante de plus de 5 ans ?"
    explanation: |
      Les vieilles imprimantes utilisent parfois SMBv1, qui sera désactivé.
      Si oui, on garde SMBv1 activé pour ne pas casser l'impression.
    type: bool                          # bool | choice | text
    asked_when: "facts.printer_count > 0"
    default: false

  - id: q_use_rdp
    title: "Utilises-tu le Bureau à distance (RDP) ?"
    type: bool
    asked_when: always
    default: false

  - id: q_keep_apps
    title: "Apps Microsoft Store à conserver ?"
    type: choice                        # multi-select
    options_from: facts.appx_installed  # liste dynamique
    asked_when: "facts.appx_installed.length > 0"
    default: []
```

### 5.4 Moteur d'expressions (volontairement minimal)

Formes acceptées au MVP :

| Forme | Exemple |
|---|---|
| `always` | toujours vrai |
| `never` | toujours faux |
| `<id>` | vrai si fact/answer truthy |
| `not <id>` | inverse |
| `<id> == <valeur>` | comparaison stricte (nombre/string/bool) |
| `<id> > <n>` / `<id> < <n>` | comparaisons numériques |
| `<id>.length > <n>` | sur arrays |

**Pas** d'opérateurs `and`/`or`/parenthèses. Si un cas l'exige, on dégrade en plusieurs règles. Si vraiment nécessaire en v1.1+, adoption de [`expr-lang/expr`](https://github.com/expr-lang/expr).

### 5.5 Fallback détecteur en échec

Si un détecteur échoue (timeout, erreur, service mort) :
- Sortie JSON inclut `{ status: "failed" | "timeout", error: "..." }`
- Le fact est marqué comme **inconnu**
- L'expression `asked_when` qui le référence est traitée comme **vraie** (on pose la question à l'utilisateur en fallback gracieux)
- Le scan **continue** avec les autres détecteurs
- Timeout par défaut : 5s par détecteur

### 5.6 Performance

- 11 détecteurs en parallèle (goroutine pool, max 4 concurrents)
- Cible total scan : **< 3 secondes**
- Si > 5s, GUI affiche progress par détecteur

### 5.7 Frontière SP1 / SP2

| | SP1 (engine) | SP2 (GUI) |
|---|---|---|
| Définition des détecteurs | ✓ | – |
| Exécution des détecteurs | ✓ | – |
| Évaluation `profile_when` / `asked_when` | ✓ | – |
| Décision des questions à poser | ✓ (dans `plan`) | – |
| Affichage des questions | – | ✓ |
| Collecte des réponses | – | ✓ |
| Affichage du plan / progress / rapport | – | ✓ |

→ La GUI est pure UX. Aucune logique métier côté frontend.

---

## 6. Contrat CLI

### 6.1 Sous-commandes

```
harden-engine scan
    Lance détecteurs + tests des règles.
    Sortie : JSON unique sur stdout.

harden-engine plan --answers answers.json
    Calcule la liste d'actions à exécuter.
    Sortie : plan JSON.

harden-engine apply --plan plan.json
    Exécute le plan.
    Crée un Restore Point (sauf --no-restore-point).
    Sortie : NDJSON streaming.

harden-engine undo (--action <id> | --run <id> | --last-run | --all) [--force]
    Annule des actions du journal.
    Sortie : NDJSON streaming.

harden-engine journal [--last-n 50] [--from <date>] [--run <id>]
    Lit le journal.
    Sortie : JSON.

harden-engine validate
    Valide le manifest contre JSONSchema.
    Code retour ≠ 0 si invalide.

harden-engine version
    Sortie JSON : engine + manifest version + OS info.
```

### 6.2 Flags globaux

| Flag | Effet |
|---|---|
| `--manifest-dir <path>` | Override (défaut : `./manifests` à côté du binaire) |
| `--journal-dir <path>` | Override (défaut : `%ProgramData%\Harden-Win11\`) |
| `--dry-run` | Pas d'exécution réelle, juste les actions qui seraient faites |
| `--quiet` | Pas de logs sur stderr |
| `--debug` | Logs verbeux sur stderr |
| `--no-restore-point` | Skip Restore Point (avec avertissement GUI) |
| `--force` | Override de l'idempotence (et drift detection sur undo) |

### 6.3 Discipline stdout / stderr

- **stdout** : *uniquement* JSON / NDJSON structuré (consommé par GUI ou `jq`)
- **stderr** : logs humains / progression / debug (jamais consommé pour parsing)

### 6.4 Codes retour

| Code | Sens |
|---|---|
| 0 | Succès complet |
| 1 | Erreur générique |
| 2 | Succès partiel (≥ 1 règle échouée) |
| 3 | Manifest invalide |
| 64 | Invocation invalide (mauvais flags) |
| 65 | Pré-condition manquée (pas admin, pas Win11, RP impossible…) |

### 6.5 Format I/O : NDJSON streaming pour `apply` et `undo`

Une ligne JSON par évènement, émise dès qu'elle est connue. Permet à la GUI d'afficher progress + logs live.

---

## 7. Journal d'actions et undo

### 7.1 Layout sur disque

```
%ProgramData%\Harden-Win11\
├── journal\
│   ├── index.json                            ← métadonnées des runs (réels)
│   ├── 2026-05-07T14-23-00.jsonl
│   ├── 2026-05-04T09-15-00.jsonl
│   └── dry-runs\                              ← journals dry-run séparés
│       └── 2026-05-07T14-20-00.jsonl
├── logs\
│   └── 2026-05-07T14-23-00.log               ← log humain compagnon
├── manifest-snapshots\
│   └── 1.0.0\                                 ← copie du manifest au moment du run
│       └── ...
└── engine.lock                                 ← lockfile (un run à la fois)
```

ACL : Admin/SYSTEM = full, Users = read-only.

### 7.2 Schéma `index.json`

```json
{
  "version": "1.0",
  "runs": [
    {
      "run_id": "2026-05-07T14-23-00",
      "started_at": "2026-05-07T14:23:00Z",
      "completed_at": "2026-05-07T14:23:34Z",
      "engine_version": "1.0.0",
      "manifest_version": "1.0.0",
      "manifest_snapshot": "manifest-snapshots/1.0.0/",
      "restore_point_id": "RP_42",
      "summary": { "applied": 32, "skipped": 5, "failed": 0, "undone": 0 },
      "status": "completed"
    }
  ]
}
```

### 7.3 Schéma des évènements NDJSON

```jsonc
// run_start
{ "type": "run_start", "run_id": "...", "engine_version": "...",
  "manifest_version": "...", "user": "...", "computer": "...",
  "os_build": 26200, "dry_run": false }

// restore_point
{ "type": "restore_point", "run_id": "...", "timestamp": "...",
  "status": "created" | "skipped" | "failed" | "would_create",
  "restore_point_id": "RP_42", "throttle_overridden": true,
  "reason": "..." }

// action_result (le cœur — un par règle exécutée)
{ "type": "action_result",
  "act_id": "act_<run_id>_<rule_id>",
  "run_id": "...",
  "timestamp": "...",
  "rule_id": "defender.realtime",
  "rule_version": "1.0",
  "status": "applied" | "skipped" | "failed" | "would_apply" | "would_skip" | "undone",
  "before": { "DisableRealtimeMonitoring": true },
  "after":  { "DisableRealtimeMonitoring": false },
  "duration_ms": 234,
  "stderr": "",
  "test_status": "ok" | "failed",
  "forced": false,
  "undo_ref": {
    "script": "engine/actions/defender/realtime.undo.ps1",
    "input": { "DisableRealtimeMonitoring": true }
  }
}

// undo_result (lors d'un undo ultérieur)
{ "type": "undo_result", "act_id": "act_...", "timestamp": "...",
  "status": "ok" | "failed" | "drift_detected",
  "expected": {...}, "actual": {...},
  "stderr": "..." }

// run_end
{ "type": "run_end", "run_id": "...", "timestamp": "...",
  "summary": { "applied": 32, "skipped": 5, "failed": 0 } }
```

### 7.4 Mécanique d'undo

**3 modes d'undo** :

| Mode | CLI |
|---|---|
| Action unique | `engine undo --action <act_id>` |
| Tout un run | `engine undo --run <run_id>` ou `--last-run` |
| Tout depuis le début | `engine undo --all` |

**Politique** :
- **Ordre LIFO** (dernière action faite = première annulée)
- **Continue-on-error** : un undo qui échoue n'arrête pas le batch ; en fin, code retour 2 + résumé
- **Drift detection strict par défaut** : avant chaque undo, lit l'état courant via `.test.ps1`. Si état courant ≠ `after` du journal → refuse (`status: drift_detected`). `--force` override.
- **Idempotence** : si état courant = `before` du journal → skip (déjà annulé).
- **Règles `irreversible: true`** (ex: `bloatware.spotify`) : pas de `.undo.ps1`. Au plan, GUI affiche badge + confirmation explicite. À l'undo, refus avec suggestion de Restore Point.

**Compatibilité multi-version** :
- Le journal référence `manifest_snapshot: manifest-snapshots/1.0.0/`
- Les `.undo.ps1` sont **embedés dans le binaire** via `go:embed` (taggés par version d'engine)
- Au MVP : pas de versionning multi-engine. Si après upgrade à v1.5, undo d'un run v1.0 dont la règle a été supprimée → l'engine dégrade vers Restore Point + prévient l'utilisateur.

### 7.5 Concurrence

- **Lockfile** `engine.lock` (PID + timestamp). Si présent → engine refuse de démarrer avec message explicite (« Une instance est déjà active, PID 1234, démarrée à 14:23. Si tu es sûr qu'elle est morte, supprime engine.lock »).
- **Append atomique** sur `<run_id>.jsonl` (`O_APPEND | O_CREATE`). Writes < 4 KB sont atomiques sur Windows.
- **Crash mid-write** : dernière ligne potentiellement partielle. Parser tolérant (skip + warning).

### 7.6 Retention

MVP : pas de retention auto. ~12-24 fichiers de 5-50 KB par an = ~500 KB. Acceptable.
v1.1+ : flag `--retention 30d` ou `--max-runs 50`.

---

## 8. Intégration System Restore

### 8.1 Quand créer un Restore Point

| Cas | RP créé ? |
|---|---|
| `apply` avec ≥ 1 action à faire | **Oui** |
| `apply` avec 0 action (tout conforme) | Non |
| `dry-run` | Non |
| `undo` | Non |

### 8.2 Comment

```powershell
Checkpoint-Computer `
    -Description "Harden-Win11 - run <run_id>" `
    -RestorePointType MODIFY_SETTINGS
```

`MODIFY_SETTINGS` (~10-15s) suffit pour nos changements (registre + services + features). Pas besoin d'`APPLICATION_INSTALL` (plus lourd).

### 8.3 Throttle Windows : override temporaire

Windows refuse > 1 RP par 24h par défaut (`SystemRestorePointCreationFrequency` en minutes, défaut 1440).

**Mitigation** : avant la création, l'engine set la clé à 0 (allow always), crée le RP, restore la valeur précédente (ou supprime la clé si elle était absente).

```powershell
$prev = Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\SystemRestore" `
    -Name "SystemRestorePointCreationFrequency" -ErrorAction SilentlyContinue
Set-ItemProperty -Path "..." -Name "SystemRestorePointCreationFrequency" -Value 0
try {
    Checkpoint-Computer -Description "..." -RestorePointType MODIFY_SETTINGS
} finally {
    if ($null -eq $prev) {
        Remove-ItemProperty -Path "..." -Name "SystemRestorePointCreationFrequency"
    } else {
        Set-ItemProperty -Path "..." -Name "SystemRestorePointCreationFrequency" `
            -Value $prev.SystemRestorePointCreationFrequency
    }
}
```

Loggé : `{ throttle_overridden: true }`.

### 8.4 Pré-conditions

| Pré-condition | Si KO |
|---|---|
| System Restore activé sur C: | Tente `Enable-ComputerRestore -Drive C:\`. Échec → abort 65 |
| Service VSS démarré | Tente `Start-Service VSS`. Échec → abort 65 |
| Espace shadow > 0 | Tente `vssadmin resize shadowstorage /On=C: /For=C: /MaxSize=5%`. Échec → abort 65 |

Avec `--no-restore-point` : skip toutes les pré-conditions. La GUI doit afficher un avertissement explicite avant de proposer ce flag.

### 8.5 Mapping RP ↔ run

Après création, l'engine récupère l'ID via `Get-ComputerRestorePoint | Select -Last 1`, le stocke dans `index.json` et dans l'évènement `restore_point` du NDJSON. La GUI peut afficher : « Pour annuler ce run en mode panique, lance la Restauration système et choisis le point n°42 du 7 mai 14:23. »

### 8.6 Timing UX

- Création ≈ 10-15s
- Pendant : GUI affiche progress determinate + message clair
- > 30s : message « Création plus longue que prévu, c'est normal sur certains systèmes »
- Timeout dur 120s : abort + message

### 8.7 Cleanup

**Non géré par l'engine.** Windows gère la rotation via le quota VSS.

---

## 9. Idempotence & dry-run

### 9.1 Contrat d'idempotence

Pour chaque règle :

```
1. Lance .test.ps1
2. Si test → true (déjà conforme) : log "skipped", n'exécute PAS .action.ps1
3. Si test → false : log "applying", exécute .action.ps1, log "applied"
```

→ Lancer `apply` 100 fois sur un système conforme = 100 × log « skipped », **0 modification**, ~3-5s par run (tests seulement).

### 9.2 Si le test lui-même échoue

**Politique : test fail → on tente l'action.**

Justification : si on ne sait pas, on tente — plus sûr que de skipper et laisser un système non durci. L'échec du test est loggé (`test_status: "failed"`) ; si l'action échoue aussi, c'est visible.

Anti-cas considéré : test fail systématique → action retentée à chaque run → toujours échouée → utilisateur voit l'échec et investigue. C'est OK.

### 9.3 `--force`

Pour un user qui veut forcer l'exécution même si test → true :
- `engine apply --force --rule defender.realtime`
- Test ignoré, action exécutée
- Loggé : `forced: true`
- Pas exposé dans la GUI MVP (cas avancé)

### 9.4 Dry-run

`harden-engine apply --dry-run` :
- Statuts `would_apply` / `would_skip` / `would_create` au lieu de `applied` / `skipped` / `created`
- Aucune modification réelle (action.ps1 NON exécuté, RP NON créé)
- Journal écrit dans `journal/dry-runs/<run_id>.jsonl` (séparé des runs réels, **pas** dans `index.json` principal)
- Utile pour debug et audit

### 9.5 Parallélisme

**Tests séquentiels** au MVP (ordre du manifest). Justifications :
- Certains tests dépendent d'autres (`network.smbv1` peut dépendre de `network.smb.signature`)
- Gains perf faibles (la plupart des tests < 100ms)
- Complexité de gestion des dépendances pas justifiée au MVP

À étudier en v1.1+ si perf devient un problème.

### 9.6 Robustesse : continue après échec

Si action.ps1 plante pour règle X :
- Logue `{ status: "failed", stderr: "..." }`
- **Continue** avec les règles suivantes
- En fin de run : code retour 2 (succès partiel) si ≥ 1 échec
- GUI affiche un panel rouge avec les règles failed + leurs erreurs

Justification : système 95% durci > système 0% durci.

---

## 10. Plan de migration depuis v1.x

L'objectif est de **réutiliser le maximum** du script existant, pas de tout réécrire.

### 10.1 Étapes

1. **Extraction des snippets** : pour chaque `Invoke-Step` du `Harden-Win11.ps1` actuel, extraire le scriptblock `Action` et `Test` dans `engine/actions/<section>/<rule>.action.ps1` et `.test.ps1`. Mécanique. Aucun changement logique.
2. **Création des `.undo.ps1`** : pour chaque règle, écrire l'inverse. La plupart sont triviales (set DWord à l'ancienne valeur). Quelques règles sont `irreversible: true` (bloatware AppX).
3. **Écriture des manifests YAML** : 8 fichiers, ~50 règles total. Métadonnées (`title`, `description`, `explanation`, `impact`) à rédiger en mode pédagogique.
4. **Implémentation Go** : `pkg/engine` (parser, runner, journal, restore point, expressions). ~2-3 semaines.
5. **CLI `harden-engine.exe`** : commandes `scan`/`plan`/`apply`/`undo`/`journal`/`validate`/`version`. ~3-5 jours.
6. **Tests** : Pester sur les snippets PS, Go test sur `pkg/engine`, integration test E2E sur Windows Sandbox.
7. **Le script `Harden-Win11.ps1` v1 reste disponible en parallèle** dans `legacy/` pendant la transition.

### 10.2 Backwards compatibility

- Le `Harden-Win11.ps1` actuel est conservé dans `legacy/Harden-Win11.ps1` pour qu'un user existant puisse continuer à l'utiliser
- Pas de migration auto du backup `.reg` actuel vers le nouveau journal (formats incompatibles, peu de valeur à migrer)

---

## 11. Risques et mitigations

| Risque | Mitigation |
|---|---|
| Refactor du script en ~50 règles atomiques plus long que prévu | Étapes 1-2 (extraction) sont mécaniques, faisables 5-10 règles/heure. Garder la v1 en parallèle |
| `go:embed` PS scripts → issues d'encoding (BOM UTF-8) | Tests d'intégration dès le 1er sprint. PowerShell exigeant sur encoding accents |
| Détecteurs lents en pratique (Get-Printer hangs si Spooler dead) | Timeout dur 5s par détecteur + fallback "ask human" |
| Throttle Restore Point cassant à cause de l'override | Test E2E dans Sandbox avec multiple runs dans la journée |
| Schéma YAML qui évolue après début du dev | JSONSchema versionné (`version: "1.0"` en tête de chaque fichier) |
| Concurrent runs (utilisateur double-clique 2x) | Lockfile robuste, message clair |

---

## 12. Open questions / décisions reportées

- **Cert de signing** : décidé hors-MVP. À reprendre quand le projet existe et a un peu de traction. Microsoft Trusted Signing à $120/an semble la bonne option.
- **Internationalisation** : MVP en français uniquement. Manifest `title`/`description`/`explanation` un jour multilingues (`title.fr`, `title.en`).
- **Télémétrie de l'app** : aucune au MVP. Si v1.1+ veut analytics opt-in (crash reports anonymes), à brainstormer séparément.
- **Tests Pester sur les snippets** : workflow CI à définir dans SP3 (Distribution).
- **Schéma `manifest.schema.json` exhaustif** : ébauche dans ce spec, version finale à figer pendant l'implémentation.

---

## 13. Liens

- Spec actuel : ce fichier
- Sous-projets suivants (à brainstormer après) :
  - SP2 : App Wails GUI
  - SP3 : Build, packaging, release GitHub
- Code source : `Harden-Win11.ps1` (v1, à conserver dans `legacy/`)
- README v1 : `README.md`

---

## Appendice A — Exemple complet de manifest section

`manifests/01-defender.yaml` (extrait, 2 règles sur ~10) :

```yaml
version: "1.0"

section:
  id: defender
  order: 1
  title: "Microsoft Defender"
  description: "Antivirus intégré à Windows. On active toutes ses protections."

rules:
  - id: defender.realtime
    title: "Protection temps réel"
    description: "Scanner les fichiers à chaque ouverture / téléchargement."
    explanation: |
      Defender scanne en arrière-plan chaque fichier que tu ouvres.
      Sans ça, un malware peut s'exécuter sans alerte.
    severity: critical
    impact: "Aucun. Activé par défaut sur Win11."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    references:
      - "https://learn.microsoft.com/.../configure-real-time-protection"
    tags: [malware, defender]
    added_in: "1.0"
    action: ./engine/actions/defender/realtime.action.ps1
    test:   ./engine/actions/defender/realtime.test.ps1
    undo:   ./engine/actions/defender/realtime.undo.ps1

  - id: defender.cloud_high
    title: "Protection cloud niveau HIGH"
    description: "Active la protection cloud Defender en mode agressif."
    explanation: |
      Defender envoie un hash des fichiers suspects à Microsoft pour analyse.
      Le mode HIGH bloque plus agressivement les fichiers inconnus.
    severity: important
    impact: "Très rares faux positifs sur des outils de dev peu courants."
    requires_reboot: false
    profile_when: always
    depends_on: [defender.realtime]
    irreversible: false
    tags: [malware, defender, cloud]
    added_in: "1.0"
    action: ./engine/actions/defender/cloud-high.action.ps1
    test:   ./engine/actions/defender/cloud-high.test.ps1
    undo:   ./engine/actions/defender/cloud-high.undo.ps1
```

## Appendice B — Exemple de session NDJSON

```jsonc
$ harden-engine apply --plan plan.json

{ "type": "run_start", "run_id": "2026-05-07T14-23-00", "engine_version": "1.0.0",
  "manifest_version": "1.0.0", "user": "koff75", "computer": "DESKTOP-XYZ",
  "os_build": 26200, "dry_run": false }

{ "type": "restore_point", "run_id": "2026-05-07T14-23-00", "timestamp": "2026-05-07T14:23:01Z",
  "status": "created", "restore_point_id": "RP_42", "throttle_overridden": true }

{ "type": "action_result",
  "act_id": "act_2026-05-07T14-23-00_defender.realtime",
  "run_id": "2026-05-07T14-23-00",
  "timestamp": "2026-05-07T14:23:14Z",
  "rule_id": "defender.realtime",
  "rule_version": "1.0",
  "status": "skipped",
  "before": { "DisableRealtimeMonitoring": false },
  "after":  { "DisableRealtimeMonitoring": false },
  "duration_ms": 89,
  "test_status": "ok" }

{ "type": "action_result",
  "act_id": "act_2026-05-07T14-23-00_asr.office.child",
  "run_id": "2026-05-07T14-23-00",
  "timestamp": "2026-05-07T14:23:15Z",
  "rule_id": "asr.office.child",
  "rule_version": "1.0",
  "status": "applied",
  "before": { "asr_action": null },
  "after":  { "asr_action": 1 },
  "duration_ms": 312,
  "test_status": "ok",
  "undo_ref": {
    "script": "engine/actions/asr/office-child.undo.ps1",
    "input": { "asr_action": null }
  }
}

{ "type": "run_end", "run_id": "2026-05-07T14-23-00", "timestamp": "2026-05-07T14:23:34Z",
  "summary": { "applied": 32, "skipped": 5, "failed": 0 } }
```

---

**Fin du spec SP1.**
