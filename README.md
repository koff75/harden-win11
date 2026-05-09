<div align="center">

# 🛡️ Harden-Win11

**Durcis ton Windows 11 en 1 clic. Sans casser ton PC.**

95 règles de sécurité, expliquées en français normal. Tu vois exactement ce qui change avant de cliquer. Tu peux tout annuler.

[![Release](https://img.shields.io/github/v/release/koff75/harden-win11?style=flat-square&color=blue)](https://github.com/koff75/harden-win11/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/koff75/harden-win11/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/koff75/harden-win11/actions)
[![Tests](https://img.shields.io/badge/tests-98%20Pester%20%2B%2011%20Go-green?style=flat-square)](https://github.com/koff75/harden-win11)
[![Coverage](https://img.shields.io/badge/coverage-CIS%2062%25%20%C2%B7%20ANSSI%2042%25%20%C2%B7%20MS%2065%25-blue?style=flat-square)](https://github.com/koff75/harden-win11)
[![License](https://img.shields.io/badge/license-WTFPL-lightgrey?style=flat-square)](LICENSE)

[**📥 Télécharger**](https://github.com/koff75/harden-win11/releases/latest) · [**📖 Documentation**](#-documentation) · [**🎯 Pourquoi**](#-pourquoi-cet-outil)

![hero screenshot](docs/screenshots/01-dashboard.png)

</div>

---

## ⚡ En 30 secondes

```powershell
# 1. Télécharge le ZIP depuis Releases
# 2. Décompresse
# 3. Double-clic sur run-as-admin.bat
```

C'est tout. Pas d'installation, pas de service, pas de registre touché tant que tu ne cliques pas.

---

## 🎯 Pourquoi cet outil

La plupart des guides "hardening Windows 11" :
- ❌ Te demandent d'éditer la base de registre à la main
- ❌ Liste de 200 paramètres en jargon technique
- ❌ Tu ne sais pas ce qui va casser tes apps
- ❌ Pas de retour en arrière propre

**Harden-Win11** :
- ✅ Une GUI qui t'explique chaque règle en **français normal** ("Aujourd'hui : ..." / "Si tu actives : ..." / "Ce qui peut t'embêter : ...")
- ✅ **Annule tout** en un clic (journal NDJSON + .undo.ps1 par règle + Restore Point Windows)
- ✅ Détecte si tu utilises RDP / SMB legacy / etc. et **refuse d'appliquer** une règle qui couperait ton usage en cours
- ✅ Surveille **24h après** ton apply pour repérer si quelque chose se met à pleurer dans Event Viewer
- ✅ **Score A/B/C/D** pour mesurer la maturité de ton durcissement
- ✅ Mappé contre **CIS Win11**, **ANSSI**, **MS Security Baseline** (pas du fait-maison)

---

## 📸 Aperçu

<table>
<tr>
<td width="50%">

**Carte action user-friendly** — chaque règle explique en français
ce qu'elle change concrètement.

![carte action](docs/screenshots/02-action-card.png)

</td>
<td width="50%">

**Score de maturité** — A/B/C/D pondéré, avec actions concrètes
pour gagner des points.

![score](docs/screenshots/03-maturity-score.png)

</td>
</tr>
<tr>
<td>

**Coverage standards** — combien de règles sont mappées à
CIS / ANSSI / MS Security Baseline.

![coverage](docs/screenshots/04-coverage.png)

</td>
<td>

**Watchlist 24h** — surveille Event Viewer après ton apply pour
détecter une casse fonctionnelle.

![watchlist](docs/screenshots/05-watchlist.png)

</td>
</tr>
</table>

---

## 🚀 Quick Start

### Option 1 : GUI (recommandé)

1. Télécharge `Harden-Win11-X.Y.Z.zip` depuis la [page Release](https://github.com/koff75/harden-win11/releases/latest)
2. Décompresse où tu veux
3. Double-clic sur `run-as-admin.bat`

La GUI détecte ton contexte (laptop / fixe / AD-joined), te suggère un profil, et tu décides règle par règle.

### Option 2 : CLI

```powershell
# Voir ce qui serait modifié — sans rien toucher
.\harden-engine.exe apply --dry-run --parallel 4

# Score de maturité
.\harden-engine.exe coverage

# Apply ciblé
.\harden-engine.exe apply --profile personal --severity critical

# Annuler tout ce qui a été fait dans les 7 derniers jours
.\harden-engine.exe undo --since 168h
```

---

## 🔥 Features qui font la différence

### 🛡️ 6 couches de sécurité avant qu'une règle change ton système

1. **Auto-skip contextuel** : laptop ⇒ on ne désactive pas l'hibernation. AD-joined ⇒ on ne renomme pas Administrator.
2. **Detection feature in-use** : si tu as une session RDP active ⇒ refus de désactiver RDP.
3. **Restore Point Windows** créé automatiquement avant l'apply.
4. **Snapshot pre/post** : 25+ clés registre + Defender + Services capturés. `harden-engine snapshot diff <runID>` te montre exactement ce qui a changé.
5. **Re-test post-apply** : si l'action a réussi mais le test post dit non-conforme (GPO override, action menteuse), rollback automatique.
6. **Watchlist 24h** : tâche planifiée surveille Event Viewer (SMB / Defender / NetBIOS / Schannel / PrintService) avec **seuils adaptatifs** (apprend ta baseline normale).

### 🧠 Vulgarisation maximale

Chaque règle clé a 4 phrases :
- **Aujourd'hui** : situation présente
- **Si tu actives** : ce qui change concrètement
- **Pour qui** : profil cible
- **Ce qui peut t'embêter** : impact concret

Aucun jargon, aucun nom de regkey, aucun chiffre brut. Tu décides en 5 secondes.

### 📊 Mesurable

```
$ harden-engine coverage
Total règles harden-win11 : 95
Règles avec ≥1 mapping    : 66 (69%)

[CIS Win11 Enterprise v3.0.0]
  Règles couvertes : 59 / 95 (62%)
  Contrôles uniques cités : 65

[ANSSI Windows]
  Règles couvertes : 40 / 95 (42%)

[MS Security Baseline 24H2]
  Règles couvertes : 62 / 95 (65%)
```

### ↩️ Tout annulable

```powershell
# Annuler le dernier run
harden-engine undo

# Annuler une règle précise
harden-engine undo --rule-id defender.cloud_protection

# Annuler 7 jours de modifs (LIFO multi-runs)
harden-engine undo --since 168h
```

---

## 🏗️ Architecture

```
manifests/         95 règles en YAML (1 fichier par section)
engine/actions/    Snippets PowerShell par règle (action / test / undo)
pkg/engine/        Library Go : manifest, executor, snapshot, watchlist, ...
cmd/harden-engine/ Binaire CLI (Cobra)
cmd/harden-gui/    Binaire Wails (GUI)
mappings/          Mapping CIS / ANSSI / MS Security Baseline
```

11 packages Go, **98 tests Pester + ~50 tests Go** + property-based + fuzz + benchmarks + gosec scan.

---

## 🤔 FAQ

**Est-ce que je risque de casser mon PC ?**
Faible. Restore Point créé avant l'apply, journal NDJSON crash-safe, undo par règle. Et si l'action a "menti", re-test post-apply déclenche un rollback auto.

**Est-ce que ça marche sur Windows 11 Home ?**
Oui. La plupart des règles fonctionnent sur Home et Pro. Quelques unes (rename Administrator) ont moins de sens en environnement AD-joined → auto-décochées.

**C'est gratuit / opensource ?**
Oui. Licence WTFPL — fais ce que tu veux. Mais aucune garantie, tu es responsable de ce que tu lances.

**Le binaire n'est pas signé par un éditeur connu, c'est risqué ?**
Le binaire est self-signé pour identifier la version. Tu peux vérifier le SHA256 publié à côté du ZIP. Build reproductible via `go build` depuis les sources de ce repo + GitHub Actions publique.

**Comment contribuer ?**
Lance les tests (`go test ./... && Invoke-Pester engine/actions`), ouvre une PR. Le format des manifests est documenté dans [`docs/`](docs/).

---

## 📖 Documentation

- [`docs/smoke-test.md`](docs/smoke-test.md) — checklist VM Win11 Home/Pro avant release
- [`docs/manual-e2e-checklist.md`](docs/manual-e2e-checklist.md) — vérif manuelle GUI en admin
- [`docs/test-report-2026-05-09.md`](docs/test-report-2026-05-09.md) — bug hunt session report
- [`mappings/baselines.yaml`](mappings/baselines.yaml) — mapping détaillé CIS / ANSSI / MS

---

## 🙋 v1 (script PowerShell legacy)

Le script original `Harden-Win11.ps1` reste dans le repo pour usage simple sans build Go. Moins safe que v2 (pas de undo, pas de journal, pas de profils) mais one-file.

---

<div align="center">

**Tu trouves ça utile ? Une ⭐ sur GitHub aide à le faire connaître.**

</div>
