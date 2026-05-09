<div align="center">

# 🛡️ Harden-Win11

### Harden Windows 11 in 1 click. Without breaking anything.

**95 security rules · plain-English explanations · everything reversible · zero installer**

[![Release](https://img.shields.io/github/v/release/koff75/harden-win11?style=for-the-badge&color=2ea44f&logo=github)](https://github.com/koff75/harden-win11/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/koff75/harden-win11/total?style=for-the-badge&color=blue&logo=download)](https://github.com/koff75/harden-win11/releases/latest)
[![Stars](https://img.shields.io/github/stars/koff75/harden-win11?style=for-the-badge&color=yellow&logo=star)](https://github.com/koff75/harden-win11/stargazers)
[![License](https://img.shields.io/badge/license-WTFPL-lightgrey?style=for-the-badge)](LICENSE)

[![CI](https://img.shields.io/github/actions/workflow/status/koff75/harden-win11/ci.yml?branch=main&style=flat-square&label=CI&logo=github-actions)](https://github.com/koff75/harden-win11/actions)
[![Tests](https://img.shields.io/badge/tests-98%20Pester%20%2B%20100%2B%20Go-green?style=flat-square)](https://github.com/koff75/harden-win11)
[![Coverage](https://img.shields.io/badge/CIS%20Win11-62%25-blue?style=flat-square)](mappings/baselines.yaml) [![Coverage](https://img.shields.io/badge/MS%20Baseline-65%25-blue?style=flat-square)](mappings/baselines.yaml) [![Coverage](https://img.shields.io/badge/ANSSI-42%25-blue?style=flat-square)](mappings/baselines.yaml)
[![EN/FR](https://img.shields.io/badge/lang-EN%20%2F%20FR-purple?style=flat-square)](#)

### [⬇️ **Download Latest**](https://github.com/koff75/harden-win11/releases/latest) · [🇫🇷 **Version française**](README.fr.md)

![Hero — main GUI](docs/screenshots/00-hero.gif)

</div>

---

## ⚡ 30 seconds to start

```powershell
# 1. Download Harden-Win11-X.Y.Z.zip from Releases
# 2. Extract anywhere
# 3. Double-click run-as-admin.bat
```

**Zero installation.** No service, no registry until you click. Delete the folder = uninstalled.

---

## 🎯 Why?

|                              | Manual GPO / Registry | O&O ShutUp10++ | PowerShell scripts | **Harden-Win11** |
| ---------------------------- | :-------------------: | :------------: | :----------------: | :--------------: |
| Plain-English explanations   |           ❌           |       ⚠️        |          ❌         |         ✅        |
| Reversible per rule          |           ❌           |       ⚠️        |          ❌         |         ✅        |
| Auto Restore Point           |           ❌           |       ❌        |          ❌         |         ✅        |
| Re-test after apply          |           ❌           |       ❌        |          ❌         |         ✅        |
| 24h Event Viewer monitoring  |           ❌           |       ❌        |          ❌         |         ✅        |
| Detects Windows-Update drift |           ❌           |       ❌        |          ❌         |         ✅        |
| Mapped to CIS / ANSSI / MS   |           ❌           |       ❌        |          ⚠️         |         ✅        |
| Open source                  |           ✅           |       ❌        |          ⚠️         |         ✅        |

---

## 📸 What it looks like

<table>
<tr>
<td width="50%" align="center">
<b>Dashboard — see the impact in one line</b><br/>
<img src="docs/screenshots/01-dashboard.png" alt="Dashboard"/>
<sub>Score, severity counts, profile auto-detected.</sub>
</td>
<td width="50%" align="center">
<b>Cursor-following tooltip — zero jargon</b><br/>
<img src="docs/screenshots/02-tooltip.png" alt="Tooltip"/>
<sub>Today / If you activate / For whom / What might bother you.</sub>
</td>
</tr>
<tr>
<td align="center">
<b>Maturity score A/B/C/D</b><br/>
<img src="docs/screenshots/03-maturity-score.png" alt="Maturity score"/>
<sub>Weighted score with concrete next actions.</sub>
</td>
<td align="center">
<b>Post-Windows-Update drift detection</b><br/>
<img src="docs/screenshots/04-drift-banner.png" alt="Drift detection"/>
<sub>Auto-detects rules silently reset by Microsoft updates.</sub>
</td>
</tr>
</table>

---

## 🔥 What makes it different

### 🧠 No jargon. Ever.

Every rule is explained in 4 plain-English lines, not registry keys :

> **Today** — A 30-year-old file sharing system is still on your PC. WannaCry and NotPetya used it to spread.
> **If you activate** — That old system is off. Your PC only speaks modern (encrypted) versions.
> **For whom** — Everyone, unless you have a NAS from before 2012.
> **What might bother you** — If a network share stops working, it was on SMBv1. Update the NAS or whitelist.

### 🛡️ 6 safety layers before any rule changes your system

1. **Context-aware auto-skip** — Laptop? we don't disable hibernation. AD-joined? we don't rename Administrator.
2. **In-use detection** — Active RDP session? we refuse to disable RDP. Active SMB1 share? we refuse to kill SMBv1.
3. **Windows Restore Point** — Created automatically before every real apply.
4. **Pre/post snapshot** — 25+ registry keys + Defender + Services. `harden-engine snapshot diff <runID>` shows exactly what changed.
5. **Post-apply re-test** — If an action returns OK but the test reports non-compliant (GPO override, lying action), automatic rollback.
6. **24h watchlist** — Scheduled task watches Event Viewer with adaptive thresholds (learns your normal baseline).

### 🔄 Detects when Windows Update breaks your hardening

Microsoft Cumulative Updates regularly reset registry settings. **No other tool catches this.** A scheduled task fires on every successful KB install and re-checks everything against the last known-good baseline. Drift detected → banner in the GUI on next boot. Click "Re-apply" to fix.

### 📊 Mapped to public standards, not home-grown

```
$ harden-engine coverage
[CIS Win11 Enterprise v3.0.0]   59 / 95 rules covered (62%)
[ANSSI Windows]                  40 / 95 rules covered (42%)
[MS Security Baseline 24H2]      62 / 95 rules covered (65%)
```

### ↩️ Everything is undoable

```powershell
harden-engine undo                                # last run
harden-engine undo --rule-id defender.cloud_protection  # one rule
harden-engine undo --since 168h                   # 7 days, LIFO across runs
```

---

## 🚀 Quick Start

### GUI (recommended)

```
1. Download Harden-Win11-X.Y.Z.zip
2. Extract
3. Double-click run-as-admin.bat
```

The GUI auto-detects your context (laptop, AD-joined…), suggests a profile, and lets you decide rule by rule with hover tooltips. Apply / Undo with one click.

### CLI (scripting / CI)

```powershell
# See what would change (no admin, no modification)
.\harden-engine.exe apply --dry-run --parallel 4

# Coverage vs CIS / ANSSI / MS
.\harden-engine.exe coverage

# Real apply, profile + severity wave
.\harden-engine.exe apply --profile personal --severity critical

# Undo last 7 days
.\harden-engine.exe undo --since 168h
```

---

## 🏗️ Architecture

```
manifests/         95 rules in YAML, one file per section (Defender, Firewall, ASR…)
engine/actions/    PowerShell snippets per rule (action / test / undo)
pkg/engine/        Go library : manifest, executor, snapshot, watchlist, drift, …
cmd/harden-engine/ CLI binary (Cobra)
cmd/harden-gui/    GUI binary (Wails 2 + WebView2)
mappings/          CIS / ANSSI / MS Security Baseline mapping
```

**Stack** : Go 1.26 · Wails 2 · PowerShell 5.1 · YAML · NDJSON crash-safe journal · 12 Go packages · 98 Pester tests + ~100 Go tests + property-based + fuzz + gosec.

---

## 🤔 FAQ

**Will I break my PC?** Restore Point before apply, crash-safe NDJSON journal, per-rule undo. If an action lies, post-apply re-test triggers auto-rollback.

**Windows 11 Home compatible?** Yes. Most rules work on Home and Pro. Rules that need a domain are auto-unchecked.

**100% local?** Yes. Zero network call (except Microsoft's standard antivirus signatures). All data stays on your machine.

**Free / Open source?** Yes. WTFPL — do whatever you want.

**Self-signed binary, is that risky?** Verify the SHA256 published next to the ZIP. Reproducible build : `go build` from this repo + public GitHub Actions logs.

**How to contribute?** Run `go test ./... && Invoke-Pester engine/actions`, open a PR. Manifest format documented in [`docs/`](docs/).

---

## 📖 Docs

[`docs/smoke-test.md`](docs/smoke-test.md) · [`docs/manual-e2e-checklist.md`](docs/manual-e2e-checklist.md) · [`mappings/baselines.yaml`](mappings/baselines.yaml) · [`README.fr.md`](README.fr.md)

---

<div align="center">

### Useful? **A ⭐ helps spread the word.**

[![Star History Chart](https://api.star-history.com/svg?repos=koff75/harden-win11&type=Date)](https://star-history.com/#koff75/harden-win11&Date)

</div>
