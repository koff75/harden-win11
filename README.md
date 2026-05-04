# harden-win11

Script PowerShell de hardening Windows 11 — baseline de sécurité reproductible.

## Prérequis

- Windows 11 (build 22000+)
- PowerShell 5.1+ ou 7.x
- Droits administrateur

## Usage

```powershell
# Autoriser l'exécution dans la session courante
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass

# Test à blanc (n'applique rien)
.\Harden-Win11.ps1 -DryRun

# Premier passage : ASR en mode audit (loggent sans bloquer)
.\Harden-Win11.ps1 -AsrAuditMode

# Passage final : tout en mode bloquant
.\Harden-Win11.ps1
```

## Options

| Paramètre | Effet |
|-----------|-------|
| `-DryRun` | Affiche les actions sans les appliquer |
| `-AsrAuditMode` | Règles ASR en mode audit (recommandé au 1er passage) |
| `-SkipBloatware` | Ne désinstalle pas les apps Store |
| `-LogPath <path>` | Chemin du log (défaut : `C:\ProgramData\Harden-Win11\`) |

## Ce que fait le script

1. Microsoft Defender (Real-time, Behavior, IOAV, NIS, PUA, CFA, Network Protection)
2. Règles ASR (Attack Surface Reduction)
3. Firewall (3 profils + blocage SMB/NetBIOS sur Public)
4. Désactivation des comptes Administrator/Guest/WsiAccount
5. UAC niveau 5 + Secure Desktop, RDP off, Hibernation off, Fast Startup off
6. LLMNR/mDNS/NetBIOS off, NTLMv2 only, SMBv1 off, signatures SMB
7. Privacy : telemetry minimum, AdvertisingID off, anti-bloat HKCU
8. Désinstallation bloatware Microsoft Store

Backup du registre automatique avant modifications.
Redémarrage requis après exécution.
