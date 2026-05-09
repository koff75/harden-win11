# Test report E2E — 2026-05-09

Tests automatisés bout-en-bout exécutés en autonomie sur poste hôte (Win11,
session non-élevée). Aucun apply réel — tout en `--dry-run`.

## Résumé exécutif

- **Phase 1** : Go 8/8 packages ✓ — Pester 98/98 tests ✓
- **Phase 2** : CLI complet OK — 2 bugs trouvés et fixés
- **Phase 3** : 0 problème de cohérence manifest ↔ scripts
- **Phase 4** : profils filtent correctement (personal=85, business=84, maximal=95)
- **Phase 5** : GUI boot sans panic, détecte contexte, charge 95 règles
- **Phase 6** : 4 bugs identifiés, **4 fixés** (3 critiques + 1 majeur)

## Phase 1 — Tests automatisés existants

| Suite | Result |
|-------|--------|
| Go (`go test ./...`) — 8 packages | **8/8 ok** |
| Pester (19 fichiers `*.tests.ps1`) | **98/98 passed** |

Aucun test rouge ou skip.

## Phase 2 — Smoke CLI complet

| Commande | Result |
|----------|--------|
| `harden-engine version` | OK — JSON {arch, go, manifest_version, os, version} |
| `harden-engine validate` | OK — exit 0, 8 manifests validés |
| `harden-engine coverage` | OK — CIS 62%, ANSSI 42%, MS 65% |
| `harden-engine apply --dry-run --yes` (global) | **1 fail trouvé → bug #1** puis OK après fix |
| `harden-engine apply --dry-run --yes --section <X>` | OK pour les 8 sections |
| `harden-engine apply --rule X` (n'existe pas) | Limitation OK : `--rule` est sur `undo`, pas `apply` |
| `harden-engine apply --profile X` | **flag absent → bug #2** puis OK après ajout |
| `harden-engine undo --yes` (sans run précédent) | Erreur claire : "no applied rules found in run …" exit 4 |
| `harden-engine validate --manifest-dir bad` | Erreur claire exit 1 |
| `harden-engine validate --schema bad.json` | Erreur claire exit 4 |

## Phase 3 — Audit cohérence manifests ↔ scripts

Script créé : `tools/audit-coherence.ps1`. Vérifie :

- 95 rules toutes ID-uniques au format `[a-z_]+\.[a-z0-9_]+`
- Tous les `action`/`test`/`undo` référencés existent sur disque
- Cohérence `irreversible: true` ⇔ pas d'undo déclaré
- `profiles` : valeurs ∈ {personal, business, maximal}
- Aucun script orphelin (présent mais pas dans un manifest)
- Aucun import obsolète vers `_helpers/appx.psm1` (renommé en `harden_appx.psm1`)

**Résultat : 0 problème.**

## Phase 4 — Tests par profil

| Profile | Règles applicables | Exit | Fails |
|---------|--------------------|------|-------|
| personal | 85 / 95 | 0 | 0 |
| business | 84 / 95 | 0 | 0 |
| maximal | 95 / 95 | 0 | 0 |

Le filtrage par profil fonctionne.

## Phase 5 — GUI boot smoke

Lancement headless 5 secondes puis kill. Log `%LOCALAPPDATA%\Harden-Win11\gui.log` :

- Startup propre, pas de panic
- `DetectContext` répond en ~1.8s : ADJoined=false, suggère "personal"
- `GetSections` charge les 8 sections en ~50ms
- `ListRuns` retrouve 1 run précédent
- `GetCoverage` retourne total=95, mapped=66

## Phase 6 — Bugs identifiés et fixés

| # | Sévérité | Fichier | Description | Statut |
|---|----------|---------|-------------|--------|
| 1 | high | `engine/actions/network/smbv1_disable.test.ps1` | `Get-WindowsOptionalFeature -Online` requiert admin (DISM), même en read-only. La COMException remontait malgré `-ErrorAction SilentlyContinue` (incompatible avec `$ErrorActionPreference = 'Stop'`). Sur session non-élevée, status=would_fail. | ✅ fixé : try/catch + flag `PartialScan` honnête |
| 2 | high | `cmd/harden-engine/main.go` | La CLI ne supportait pas `--profile` ni `--audit`, alors que la GUI les utilisait. Incohérence Front/Back : impossible de tester un profil sans la GUI. | ✅ fixé : flags ajoutés à la commande `apply` |
| 3 | **CRITICAL** | `pkg/engine/winadmin/winadmin_windows.go` | L'heuristique "puis-je écrire dans `C:\Windows\Temp`" donne un faux-positif **admin = true** sur Win11 (le dossier accepte les writes BUILTIN\Users). La GUI activait Apply/Undo pour des sessions non-élevées qui plantent ensuite à la 1ère écriture HKLM. | ✅ fixé : `OpenProcessToken` + `GetTokenInformation(TokenElevation)` (vraie API UAC). Vérifié : session non-élevée → IsAdmin=false → bandeau rouge GUI + apply CLI bloqué |
| 4 | medium | `engine/actions/asr/*.test.ps1` (×19) | Les tests ASR comparaient en dur à mode Block (1). En mode `--audit`, l'action met en mode Audit (2) mais le test continue à attendre Block → would_apply en boucle. | ✅ fixé : `$expected` lit `$env:HARDEN_ASR_MODE` (1=Block, 2=Audit) |

### Notes / observations (pas des bugs, mais à connaître)

- **Lenteur dry-run global** : ~90s pour 95 règles (1.4s/rule). Chaque règle = 1 spawn `powershell.exe`. Optimisation possible : pool de workers parallèle dans `executor.Run`. Pas critique pour un usage one-shot, mais à considérer si on étend la knowledge base.
- **`Get-AppxPackage -AllUsers` denied même en admin** : sur certaines configs Win11, l'opération `-AllUsers` échoue avec "Access denied" même avec un token élevé. Le helper `harden_appx.psm1` fait déjà un fallback gracieux vers `Get-AppxPackage` (sans `-AllUsers`) et signale `PartialScan: true`. Comportement OK.
- **Cobra affiche `--help` en cas d'erreur** : sortie polluée mais comportement standard. Pas un bug.
- **Encoding mojibake dans les commentaires PS** : certains commentaires ont des `Ã©` au lieu de `é` (artefact de Set-Content sans `-Encoding UTF8`). N'affecte pas la fonctionnalité, à nettoyer dans une passe de cosmétique.

## État de la suite après fixes

```
go test ./...     → 8/8 ok
Pester suite      → 98/98 passed
audit-coherence   → 0 problèmes (critical/high/medium/low)
dry-run global    → 0 fails (42 skip + 53 apply = 95)
GUI boot          → IsAdmin détecté correctement, plus de faux-positif
```

## Prochaines étapes recommandées

1. Smoke test sur VM Win11 propre (Home + Pro) — la procédure manuelle est dans `docs/smoke-test.md`
2. Optimisation perf : paralléliser `executor.Run` (~3-4× plus rapide attendu)
3. Nettoyage des accents mojibake dans les commentaires PS

## Conclusion

**4 bugs trouvés, 4 fixés.** Le bug #3 (faux-positif admin) était critique et serait passé en production sans cette session de test : un user non-admin aurait cliqué Apply, vu plein d'erreurs HKLM, et perdu confiance dans l'outil.
