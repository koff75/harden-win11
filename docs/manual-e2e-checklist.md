# Checklist E2E manuelle (admin requis)

À faire UNE FOIS sur ta propre machine pour valider que les safety features
fonctionnent réellement avant un release. ~10 minutes.

## Prérequis

1. PowerShell **élevée** (clic droit → Exécuter en tant qu'administrateur)
2. `cd C:\Users\koff7\projet_hardening`

## 1. Test GUI avec admin

```powershell
.\cmd\harden-gui\build\bin\harden-gui.exe
```

**Vérifie** :
- [ ] Pas de bandeau rouge "Tu n'es pas administrateur"
- [ ] Header dit "admin ✓"
- [ ] La barre **Couverture standards** affiche `CIS 59/95 · ANSSI 40/95 · MS 62/95`
- [ ] Sous le profil suggéré, un cadre orange liste 1 règle auto-décochée :
      `system_settings.hibernate_off — Laptop détecté…`
- [ ] Clique **Vérifier** → 95 lignes apparaissent (sauf si profil filtre)

## 2. Test Apply réel sur 1 rule bénigne

Dans la GUI :
1. Décoche TOUTES les sections sauf **Privacy & Telemetry**
2. Dans le tableau, **décoche tout sauf** `privacy.tips_welcome_off`
   (juste 1 ligne cochée)
3. Clique **Appliquer** → confirme

**Observe les events successifs dans le panneau status** :
- [ ] "Création d'un Restore Point Windows…" (30-60s)
- [ ] "Restore Point créé en X secondes."
- [ ] La règle passe en `Appliquée ✓`
- [ ] Pas de message d'erreur

## 3. Vérifier le journal NDJSON

```powershell
Get-Content (Get-ChildItem $env:ProgramData\Harden-Win11\runs\*.ndjson | Sort LastWriteTime -Desc | Select -First 1) | ConvertFrom-Json | Select type, status, recheck, reason
```

**Vérifie** :
- [ ] Un event `type=restore_point` avec `created=true`
- [ ] Un event `type=action_result` avec `status=applied` ET `recheck=compliant`

## 4. Test Undo

Dans la GUI : clique **Annuler le dernier run**.

**Vérifie** :
- [ ] La règle revient à son état initial (vérifiable via clic droit sur Bureau →
      Personnaliser → Bouton **« Afficher des suggestions… »** est ré-activé si
      on l'avait désactivé via la rule)
- [ ] Re-clique **Vérifier** dans la GUI → la rule revient en "À renforcer"

## 5. Test feature_in_use (RDP)

Si ton RDP est désactivé : skip ce test.
Si ton RDP est actif :
1. Active le serveur RDP (Settings → System → Remote Desktop)
2. Ouvre une session RDP depuis un autre PC
3. Dans la GUI, sélectionne uniquement `system_settings.rdp_disable`
4. Clique Appliquer

**Vérifie** :
- [ ] La rule passe en `Skipped` avec reason="feature_in_use"
- [ ] Le message mentionne "1 session(s) RDP active(s)"

## 6. Cleanup

```powershell
# Vérifier les Restore Points créés
Get-ComputerRestorePoint | Where-Object { $_.Description -like '*harden-win11*' }
```

Tu peux les laisser (utile en cas de problème futur) ou les supprimer via
**Paramètres → Système → Récupération → Restore Points**.

---

**Critère de validation E2E** : tous les ☐ cochés. Si quelque chose foire,
copie le contenu de `%LOCALAPPDATA%\Harden-Win11\gui.log` et le dernier
NDJSON du journal.
