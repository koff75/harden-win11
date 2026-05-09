# Test report E2E — round 2 — 2026-05-09

Suite E2E complète après les changements UX/i18n. **2 vrais bugs trouvés, 2 fixés.**

| Phase | Statut | Détail |
|-------|--------|--------|
| P1. Régression Go+Pester+JS+audit | ✅ | 11/11 + 98/98 + 14/14 + 0 issue + 8/8 manifests |
| P2. Smoke CLI exhaustif | ✅ | 30/30 commandes (sub-cmds + sections + profils + sévérités + parallélisme + combinaisons) |
| P3. Cohérence Wails front↔back | ✅ | 12/12 méthodes Go + 5/5 events Wails |
| P4. Audit i18n | 🔥 → ✅ | **264 bugs** trouvés (rules sans EN), **fixés** |
| P5. GUI boot smoke | ✅ | Boot propre, 0 panic, toutes les méthodes répondent |
| P6. Qualité code | ⚠ → ✅ | 11 fichiers gofmt corrigés, 0 issue gosec, 0 go vet |
| P7. Bug hunt | ✅ | Synthèse ci-dessous |

## Bugs trouvés

### 🔥 Bug #1 (CRITICAL) — 264 textes EN manquants pour 66 rules

**Phase** : P4 audit i18n
**Symptôme** : sur les 66 rules annotées dans la 2ème vague (`tools/annotate-user-text-rest`), les champs `user_today_en`, `user_after_en`, `user_for_who_en`, `user_risk_en` étaient absents. Conséquence : en mode anglais, ces 66 rules tombaient sur le fallback FR via `userTodayEn || userToday`. L'utilisateur EN voyait du français dans le tooltip de 70% des rules.

**Détecté par** : nouveau outil `tools/audit-i18n` qui :
- Parcourt tous les `t('key')` et `data-i18n="key"` du frontend
- Vérifie qu'ils existent dans le dictionnaire FR ET EN de `i18n.js`
- Vérifie la parité des clés (toute clé FR doit avoir un EN et vice-versa)
- Vérifie que les 95 rules YAML ont les 8 champs user_*

**Fix** : nouvel outil `tools/annotate-user-text-rest-en` qui annote les 66 rules en anglais. 264 textes EN ajoutés. Re-audit : 0 bug.

### ⚠ Bug #2 (mineur) — 11 fichiers Go non gofmt-formatés

**Phase** : P6 qualité code
**Symptôme** : `gofmt -l` retournait 11 fichiers nécessitant un reformatage. Pas de problème fonctionnel mais déclencherait un échec sur un CI strict.

**Fix** : `gofmt -w` sur les 11 fichiers. Re-test après reformat : tous les tests verts (98/98 Pester + 11/11 Go).

## Observations / non-bugs

### 8 dead keys dans i18n.js

L'audit a remonté 8 clés définies en FR et EN mais jamais utilisées (`cell.hoverhint`, `dashboard.improveCrit`, `cell.canbreakprefix`, etc.). C'est cosmétique — pas un bug mais peut être nettoyé. Je laisse pour l'instant car certaines pourraient être réutilisées.

### Coverage Go

Les packages business logic sont à 70%+ (baseline 98%, maturity 94%, ndjson 89%, manifest 72%, runner 73%, winadmin 75%). Les packages PS-spawning (snapshot, watchlist, restorepoint) sont à 0-30% car ils sont testables uniquement via PS réel (couverts par le test E2E HKCU `TestE2E_HKCU_*`). Les tools (annotate-*, fix-mojibake) sont à 0% — ce sont des scripts one-shot.

## Workflow global validé

- ✅ Lancer `harden-engine.exe` en CLI : tous les flags marchent (`apply`, `undo`, `validate`, `coverage`, `snapshot`, `watchlist`, `watch-events`)
- ✅ Lancer la GUI : boote sans crash, charge profils + sections + watchlist alerts
- ✅ Switch FR ↔ EN : tout se traduit (statiques + dynamiques + tooltip)
- ✅ Tooltip riche au survol : suit la souris, limité Niveau+Règle, affiche les 4 lignes user-friendly
- ✅ Le binaire et les manifests sont packageables via `tools/build-release.ps1`

## Verdict

**0 bug bloquant restant.** Le projet est dans un état "production-ready" pour usage perso et démo publique. Les 2 vraies issues trouvées sont fixées.

Prochain step recommandé : tester manuellement la GUI sur tes propres machines (laptop perso, peut-être une VM Win11 propre via [`docs/manual-e2e-checklist.md`](manual-e2e-checklist.md)).
