# harden-win11

Script PowerShell de hardening Windows 11 — baseline de sécurité reproductible et idempotente.

## Démarrage rapide

1. **Télécharge** le script `Harden-Win11.ps1`
2. **Ouvre PowerShell en administrateur** : menu Démarrer → tape `PowerShell` → clic droit → *Exécuter en tant qu'administrateur*
3. **Débloque le fichier** (Windows marque les fichiers téléchargés depuis Internet) :
   ```powershell
   cd "C:\chemin\vers\le\dossier"
   Unblock-File -Path .\Harden-Win11.ps1
   Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
   ```
4. **Lance le script** :
   ```powershell
   .\Harden-Win11.ps1
   ```
   Un menu s'affiche avec 3 options.

## Workflow recommandé

```
┌─────────────────────────────┐
│ 1. Test à blanc (DryRun)    │ → Vérifie ce qui serait modifié
└─────────────┬───────────────┘
              ↓
┌─────────────────────────────┐
│ 2. Mode AUDIT ASR           │ → Applique tout, ASR loggent sans bloquer
│    (laisse tourner 2-3 j.)  │   Vérifie ensuite les events ASR
└─────────────┬───────────────┘
              ↓
┌─────────────────────────────┐
│ 3. Application complète     │ → Tout en mode bloquant
└─────────────────────────────┘
```

Les règles ASR (Attack Surface Reduction) peuvent bloquer des outils légitimes peu courants. Le mode AUDIT permet de voir ce qui *aurait* été bloqué sans casser ton workflow.

## Options de la ligne de commande

| Paramètre | Effet |
|-----------|-------|
| *(aucun)* | Menu interactif |
| `-DryRun` | Test à blanc, n'applique rien |
| `-AsrAuditMode` | Tout applique, mais ASR en mode audit |
| `-SkipBloatware` | Ne désinstalle pas les apps Store |
| `-Quiet` | Pas de menu, pas de pause finale (idéal pour automatisation) |
| `-LogPath <path>` | Chemin du log custom |

### Exemples

```powershell
# Test à blanc
.\Harden-Win11.ps1 -DryRun

# Premier passage : audit ASR, automatisé
.\Harden-Win11.ps1 -AsrAuditMode -Quiet

# Passage final : tout bloquant, automatisé
.\Harden-Win11.ps1 -Quiet

# Sans bloatware (machines pro où Clipchamp etc. peuvent être nécessaires)
.\Harden-Win11.ps1 -SkipBloatware
```

## Ce que fait le script (8 sections)

1. **Microsoft Defender** : Real-time, Behavior, IOAV, NIS, PUA, Controlled Folder Access, Network Protection, cloud HIGH
2. **ASR rules** : 19 règles (Office, LSASS, scripts obfusqués, USB, WMI, ransomware…)
3. **Firewall** : 3 profils ON, blocage SMB et NetBIOS sur profil Public
4. **Comptes locaux** : désactivation Administrator/Guest/WsiAccount/DefaultAccount, renommage admin built-in
5. **UAC** niveau 5 + Secure Desktop, **RDP** off, **Hibernation** off, **Fast Startup** off
6. **Réseau** : LLMNR/mDNS/NetBIOS/WPAD off, NTLMv2 only, signatures SMB requises, SMBv1 off
7. **Privacy** : Telemetry minimum, AdvertisingID off, Activity History off, Cortana/Recall off, anti-bloat HKCU
8. **Bloatware** : désinstallation apps Store inutiles (Bing Weather, Solitaire, Skype, etc.)

## Sortie type

```
  ====================================================================
      HARDEN-WIN11   |   Baseline de securite Windows 11
  ====================================================================

  Hote      : DESKTOP-XYZ
  User      : moi
  Build     : 26200
  Mode      : DRY-RUN (aucune modification)
  Log       : C:\ProgramData\Harden-Win11\harden-20260504-232341.log

  --- Section 1/8 : Microsoft Defender ---
  [=] Defender : Real-time Protection -> deja conforme
  [=] Defender : Behavior Monitoring -> deja conforme
  [?] Defender : Network Protection -> serait applique
  ...

  ====================================================================
                              SYNTHESE
  ====================================================================

  Duree d'execution    : 14.3 secondes
  Total operations     : 89

  [?] 32 actions seraient appliquees
  [=] 57 items deja conformes

  Score de conformite avant : 64.0 %
  Score de conformite apres : 100.0 % (si applique)
```

Légende des icônes : `[+]` appliqué, `[=]` déjà conforme, `[?]` dry-run, `[!]` warning, `[X]` erreur

## Backup et restauration

Le script fait **automatiquement un backup du registre** avant toute modification dans :
```
C:\ProgramData\Harden-Win11\regbackup-<timestamp>\
```

Trois fichiers `.reg` y sont stockés (Policies, Lsa, ContentDeliveryManager). Pour restaurer :
```cmd
reg import "C:\ProgramData\Harden-Win11\regbackup-XXX\HKLM-SOFTWARE-Policies.reg"
```

## Vérifier les events ASR après mode audit

Après quelques jours d'utilisation en `-AsrAuditMode`, vérifie ce qui aurait été bloqué :

```powershell
Get-WinEvent -LogName 'Microsoft-Windows-Windows Defender/Operational' |
  Where-Object {$_.Id -in 1121,1122,1125,1126} |
  Select-Object TimeCreated, Id, Message -First 50
```

- **1121** = règle ASR a bloqué (mode block)
- **1122** = règle ASR aurait bloqué (mode audit)
- **1125, 1126** = règles ASR sur dossiers protégés

Si rien de légitime n'est bloqué, relance le script sans `-AsrAuditMode`.

## Troubleshooting

**`script cannot be loaded ... not digitally signed`**
→ Le script n'est pas signé. Solution :
```powershell
Unblock-File -Path .\Harden-Win11.ps1
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
```

**`must be lance en tant qu'administrateur`**
→ Tu n'as pas lancé PowerShell en admin. Clic droit sur PowerShell → *Exécuter en tant qu'administrateur*.

**Accents cassés dans la console (`dÃ©jÃ ` au lieu de `déjà`)**
→ Résolu dans la version actuelle du script (BOM UTF-8 + `chcp 65001`). Si tu vois encore des accents cassés, c'est que tu utilises une vieille version — re-télécharge.

**Une appli légitime cassée après application**
→ Probablement une règle ASR. Vérifie l'event log (voir section précédente) et désactive la règle concernée :
```powershell
Add-MpPreference -AttackSurfaceReductionRules_Ids <GUID> -AttackSurfaceReductionRules_Actions Disabled
```

**Partage SMB local cassé**
→ Si tu as un NAS/imprimante très ancien qui ne parle que SMBv1, commente le bloc `Set-SmbServerConfiguration -EnableSMB1Protocol $false` dans le script avant de relancer.

## Déploiement multi-machines

Pour déployer sur plusieurs PC, copie le script via partage réseau ou USB et lance-le en mode automatisé :

```powershell
# Workflow type pour 5+ machines
\\serveur\partage\Harden-Win11.ps1 -AsrAuditMode -Quiet
# Attendre 2-3 jours, vérifier les events
\\serveur\partage\Harden-Win11.ps1 -Quiet
```

Le log généré (`C:\ProgramData\Harden-Win11\harden-*.log`) permet de comparer l'état des machines.

## Compatibilité

- **Cible** : Windows 11 (build 22000+, testé sur 26200/24H2/25H2)
- **PowerShell** : 5.1 (intégré) ou 7.x
- **Windows 10** : la plupart des paramètres fonctionnent mais non garanti
- **Windows Home / Pro / Enterprise** : oui, avec quelques différences sur la télémétrie (niveau 1 minimum sur Home)

## Sécurité du script

Le script est **idempotent** : tu peux le relancer 100 fois, il vérifie l'état avant chaque action et ne fait que ce qui est nécessaire. Backup automatique avant modification. Pas de connexion Internet sortante (sauf `Update-MpSignature` qui télécharge les signatures Defender).
