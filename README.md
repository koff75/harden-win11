<div align="center">

<img src="docs/logo.svg" alt="Win11 Hardening" width="540"/>

### Secure your Windows 11 in 3 clicks. No command line. Nothing breaks.

**An open-source desktop app that explains every security setting in plain English, applies them with one click, and undoes everything if you change your mind.**

Out of the box, Windows 11 is **not hardened against modern attacks**. This project closes the gap — bringing your home PC to the level of a properly-managed enterprise machine.

[![Release](https://img.shields.io/github/v/release/koff75/harden-win11?style=for-the-badge&color=2ea44f&logo=github)](https://github.com/koff75/harden-win11/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/koff75/harden-win11/total?style=for-the-badge&color=blue&logo=download)](https://github.com/koff75/harden-win11/releases/latest)
[![Stars](https://img.shields.io/github/stars/koff75/harden-win11?style=for-the-badge&color=yellow&logo=star)](https://github.com/koff75/harden-win11/stargazers)
[![License](https://img.shields.io/badge/license-WTFPL-lightgrey?style=for-the-badge)](LICENSE)

[![CI](https://img.shields.io/github/actions/workflow/status/koff75/harden-win11/ci.yml?branch=main&style=flat-square&label=CI&logo=github-actions)](https://github.com/koff75/harden-win11/actions)
[![Tests](https://img.shields.io/badge/tests-98%20Pester%20%2B%20100%2B%20Go-green?style=flat-square)](https://github.com/koff75/harden-win11)
[![CIS](https://img.shields.io/badge/CIS%20Win11-62%25-blue?style=flat-square)](mappings/baselines.yaml) [![MS](https://img.shields.io/badge/MS%20Baseline-65%25-blue?style=flat-square)](mappings/baselines.yaml) [![ANSSI](https://img.shields.io/badge/ANSSI-42%25-blue?style=flat-square)](mappings/baselines.yaml)
[![EN/FR](https://img.shields.io/badge/EN%20%2F%20FR-purple?style=flat-square)](#)

### [⬇️ **Download for Windows 11**](https://github.com/koff75/harden-win11/releases/latest) · [🇫🇷 **Version française**](README.fr.md)

![Hero — main GUI](docs/screenshots/01-dashboard.png)

</div>

---

## ⚡ How it works — 3 steps

```
1. Download Win11Hardening.zip
2. Right-click → Extract → Double-click run-as-admin.bat
3. The app opens. Click "Check" → see what to fix → click "Apply".
```

**Zero installation.** No service, no registry until you click. Delete the folder = uninstalled. **No command line knowledge needed.**

---

## 🎯 Why Windows 11 needs hardening

A fresh Windows 11 install in 2026 still ships with :

- **SMBv1**, **NTLMv1**, **WPAD** — protocols whose names appear in every "how WannaCry / NotPetya spread" post-mortem.
- **Telemetry, ads, advertising-ID** baked into Settings, fed to Microsoft and Bing daily.
- **Office macros** that auto-run from anywhere on the internet.
- **Credentials caching** that lets a single phishing email leak your domain password.
- **Microsoft Defender ASR rules** that exist but are off by default.

This is not a configuration question — these are decisions Microsoft makes for backwards compatibility. **Win11 Hardening flips them**, while explaining each one in plain language and letting you undo any rule individually.

### How it compares to other Windows hardening tools

Honest take : there are great tools out there. Most cover **part of the surface** — privacy *or* enterprise hardening, GUI *or* CLI, popular *or* exhaustive. Win11 Hardening tries to combine plain-English UX with enterprise-grade coverage.

| Feature | [O&O ShutUp10++](https://www.oo-software.com/en/shutup10) | [Privatezilla](https://github.com/builtbybel/privatezilla) | [Chris Titus WinUtil](https://github.com/ChrisTitusTech/winutil) | [MS Security Baseline](https://www.microsoft.com/en-us/download/details.aspx?id=55319) | **Win11 Hardening** |
|---|:---:|:---:|:---:|:---:|:---:|
| Consumer-friendly GUI | ✅ | ✅ | ✅ | ❌ (GPO/expert) | ✅ |
| Plain-English per-rule explanation | ⚠️ (1 line) | ⚠️ (1 line) | ❌ | ❌ (.docx) | ✅ (4 lines + hover) |
| Privacy / bloatware coverage | ✅ | ✅ | ✅ | ❌ | ✅ |
| Defender + Firewall + ASR coverage | ❌ | ❌ | ⚠️ (basic) | ✅ | ✅ |
| Reversible per individual rule | ⚠️ (presets) | ⚠️ (toggle) | ⚠️ (re-run) | ❌ | ✅ |
| Auto Restore Point before apply | ✅ | ❌ | ⚠️ (manual) | ❌ | ✅ |
| Post-apply re-test + auto rollback | ❌ | ❌ | ❌ | ❌ | ✅ |
| Detects Windows Update drift | ❌ | ❌ | ❌ | ❌ | ✅ |
| Mapped to CIS / ANSSI / MS baselines | ❌ | ❌ | ❌ | MS only | ✅ |
| Open source | ❌ (freeware) | ✅ | ✅ | ⚠️ (partial) | ✅ |
| Locale | EN/DE | EN/DE/RU | EN | EN | EN/FR |

**Where Win11 Hardening is uniquely useful** : the combination of (1) detailed plain-English UX of a consumer privacy tool, (2) Defender/Firewall/ASR coverage of an enterprise baseline, and (3) original features no other tool has — post-apply re-test with auto-rollback, and Windows Update drift detection.

**Where other tools are better** : O&O ShutUp10++ if you only care about privacy and want a polished freeware experience · Microsoft Security Baseline if you manage a fleet via GPO/Intune · Chris Titus WinUtil if you want a popular all-in-one tweaker including non-security features (debloat installer presets, dark mode, etc.).

---

## 📸 What you'll see in the app

<table>
<tr>
<td width="50%" align="center">
<b>1. The dashboard tells you what matters</b><br/>
<img src="docs/screenshots/01-dashboard.png" alt="Dashboard"/>
<sub>Your machine is OK on X points · N improvements possible. The app auto-detects if you're on a laptop / domain / etc.</sub>
</td>
<td width="50%" align="center">
<b>2. Hover any rule, get the why in plain English</b><br/>
<img src="docs/screenshots/02-tooltip.png" alt="Tooltip"/>
<sub>Today / If you activate / For whom / What might bother you. No registry keys, no jargon.</sub>
</td>
</tr>
<tr>
<td align="center">
<b>3. Click "Apply", see real progress</b><br/>
<img src="docs/screenshots/03-apply.png" alt="Apply progress"/>
<sub>Restore Point first. Then each rule with its result. Cancel anytime.</sub>
</td>
<td align="center">
<b>4. Get a maturity score A/B/C/D</b><br/>
<img src="docs/screenshots/04-maturity-score.png" alt="Maturity score"/>
<sub>How hardened your machine is, with concrete next steps to gain points.</sub>
</td>
</tr>
<tr>
<td align="center">
<b>5. Windows Update reset something? You'll know.</b><br/>
<img src="docs/screenshots/05-drift-banner.png" alt="Drift detection"/>
<sub>The app re-checks after every Windows Update and warns you if Microsoft silently undid a setting.</sub>
</td>
<td align="center">
<b>6. Switch language with one click</b><br/>
<img src="docs/screenshots/06-language-toggle.png" alt="EN/FR toggle"/>
<sub>FR/EN button top-right. The whole UI flips instantly — including the tooltips.</sub>
</td>
</tr>
</table>

---

## 🔥 What makes it different

### 🧠 Zero jargon. Built for humans.

Every rule has a 4-line plain-English explanation. Hover any rule in the table to read it.

> **Today** — A 30-year-old file sharing system is still on your PC. WannaCry and NotPetya used it to spread.
> **If you activate** — That old system is off. Your PC only speaks modern (encrypted) versions.
> **For whom** — Everyone, unless you have a NAS from before 2012.
> **What might bother you** — If a network share stops working, it was on SMBv1. Update the NAS or whitelist.

### 🛡️ 6 safety layers before any rule changes your system

1. **Context auto-skip** — Laptop? hibernation kept on. Corporate domain? we don't rename Administrator.
2. **In-use detection** — Active RDP session? we refuse to turn off RDP. Active SMB1 share? we refuse to kill SMBv1.
3. **Windows Restore Point** — Created automatically before any apply.
4. **Pre/post snapshot** — 25+ critical settings captured. You can see *exactly* what changed.
5. **Post-apply re-test** — If a setting didn't actually take effect, automatic rollback.
6. **24h monitoring** — The app watches Event Viewer for 24h after apply. Banner if SMB / Defender / printers start crying.

### 🔄 Detects when Windows Update breaks your hardening

Microsoft Cumulative Updates regularly reset registry settings. **No other Windows 11 hardening tool catches this.**

The app registers a system task that fires after every successful Windows Update install. It re-runs all your enabled rules and compares them against the last known-good baseline. If something drifted, you see a banner on next boot with a one-click "Re-apply".

### ↩️ Everything reversible from the GUI

The **History sidebar** lists every run. Click ↶ next to any run to roll it back. No regrets.

### 📊 Mapped to public standards

Coverage against three public Windows 11 hardening baselines :

- **CIS Microsoft Windows 11 Enterprise Benchmark** v3.0.0 — 62% coverage
- **Microsoft Security Baseline** Win11 24H2 — 65% coverage
- **ANSSI** Recommandations Windows — 42% coverage

You're not running someone's homemade ideas. You're running rules that are already in industry baselines.

---

## 🤔 FAQ

**I'm not a tech person — is this for me?** Yes. The app explains every setting in plain language. You hover, you read, you decide. If you change your mind, click ↶ to undo.

**Will it break my apps?** Unlikely. Restore Point first, in-use detection, automatic rollback if a rule misbehaves. And every rule warns you if it might break something specific (like "this could break your old NAS").

**Windows 11 Home compatible?** Yes. Most rules work on Home and Pro. Rules that need a corporate domain are auto-unchecked.

**100% local?** Yes. Zero network call. All your data stays on your machine.

**Free / Open source?** Yes. WTFPL — do whatever you want.

**The binary isn't signed by Microsoft, is that risky?** SHA256 published next to the ZIP. Reproducible build from source : you can compile it yourself with `go build`.

---

## 🛠 For developers / IT admins

A CLI binary (`harden-engine.exe`) ships alongside the GUI in the same ZIP. Run it from PowerShell to script applies in CI/CD or batch deploy across machines :

```powershell
.\harden-engine.exe apply --dry-run --profile personal --severity critical
.\harden-engine.exe coverage
.\harden-engine.exe undo --since 168h
```

Full CLI reference : `harden-engine.exe --help`. Manifests format and how to add a rule : [`docs/`](docs/).

**Stack** : Go 1.26 · Wails 2 · PowerShell 5.1 · YAML manifests · NDJSON crash-safe journal · 12 Go packages · 98 Pester + 100+ Go tests + property-based + fuzz + gosec.

---

## 📖 More

[`docs/smoke-test.md`](docs/smoke-test.md) · [`docs/manual-e2e-checklist.md`](docs/manual-e2e-checklist.md) · [`mappings/baselines.yaml`](mappings/baselines.yaml) · [`README.fr.md`](README.fr.md)

