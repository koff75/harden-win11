# Smoke test E2E — checklist VM Win11

Procédure manuelle à exécuter avant chaque release sur **2 VM neuves** :
1. Windows 11 Home 24H2 (compte local unique = admin)
2. Windows 11 Pro 24H2 (compte local + AD non joint)

Objectif : détecter les divergences Home vs Pro (UAC strict, AppX -AllUsers,
Defender ASR, GPO inheritence) avant de shipper.

## Prérequis VM

- VM avec snapshot "post-OOBE" (Hyper-V, VMware ou VirtualBox)
- 4 GB RAM, 60 GB disque
- Windows Update appliqué (mais pas Microsoft 365 Apps)
- Compte local admin nommé `User` (mot de passe simple)
- PowerShell 5.1 + 7 installés

## Setup repo

```powershell
# Sur la VM, avec internet
git clone https://github.com/<repo>/harden-win11
cd harden-win11
go build -o dist\harden-engine.exe .\cmd\harden-engine
cd cmd\harden-gui ; wails build ; cd ..\..
```

Ou copier le ZIP de release (cf. `docs/release.md`) et `Expand-Archive`.

## Séquence automatisée

```powershell
# Lance la GUI ou la séquence complète
.\dist\harden-engine.exe version
.\dist\harden-engine.exe validate
.\dist\harden-engine.exe coverage

# OU bien le script de smoke
pwsh -File tools\smoke-test.ps1
```

`smoke-test.ps1` enchaîne : build → validate → coverage → dry-run → apply
(scope `system_settings`) → undo → dry-run final.

## Checklist manuelle

| # | Étape | Win11 Home | Win11 Pro |
|---|-------|------------|-----------|
| 1 | `harden-engine version` retourne JSON valide | ☐ | ☐ |
| 2 | `harden-engine validate` exit 0 | ☐ | ☐ |
| 3 | `harden-engine coverage` affiche CIS / ANSSI / MS | ☐ | ☐ |
| 4 | GUI démarre, console PS NON visible | ☐ | ☐ |
| 5 | Dashboard affiche "X règles à renforcer" | ☐ | ☐ |
| 6 | Coverage bar visible avec % | ☐ | ☐ |
| 7 | Dry-run complet sans erreur | ☐ | ☐ |
| 8 | Apply `defender` réussit, defender.realtime conforme | ☐ | ☐ |
| 9 | Apply `bloatware` (sample : tiktok, candy_crush) | ☐ | ☐ |
| 10 | Vue Historique (si implémentée) liste les runs | ☐ | ☐ |
| 11 | Undo last run, dryrun re-suivant montre l'état initial | ☐ | ☐ |
| 12 | Lancement non-admin : bandeau "Relancer en admin" | ☐ | ☐ |

## Pièges connus à vérifier

- **Bloatware AllUsers** : sur Home, `Get-AppxPackage -AllUsers` peut throw
  "Access is denied" dans certains modes UAC. Vérifier que le fallback
  `Get-AppxPackage` (sans -AllUsers) prend le relais (champ `partial=true`
  remonté dans le journal).
- **ASR rules sur Win11 Home** : Defender ASR fonctionne sur Home depuis
  21H2, mais certaines GUIDs sont silencieuses (rule présente mais pas
  enforced). Vérifier `Get-MpPreference | Select AttackSurfaceReductionRules*`.
- **NetBIOS / SMB** : sur poste joint AD, `network.netbios_off` peut casser
  la résolution NetBIOS-name d'un partage. Vérifier après apply qu'on accède
  toujours aux partages internes (IP/FQDN OK, NetBIOS-name KO = attendu).
- **rdp_disable** : si la VM est managée via RDP depuis l'hôte, l'apply
  coupera la connexion. Toujours tester via console hôte (Hyper-V Connect)
  pas via mstsc.

## Critère de release

- Tous les ☐ cochés sur Home ET Pro.
- `harden-engine coverage` ≥ 60% sur CIS L1.
- Pester suite green (98/98 tests).
- Aucune erreur "panic" ou "denied" inattendue dans le journal NDJSON.
