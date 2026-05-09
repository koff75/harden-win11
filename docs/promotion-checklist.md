# Checklist promotion / référencement GitHub

À faire pour rendre le repo **trouvable** et **engageant** quand quelqu'un
arrive dessus pour la première fois.

## 🔧 1. Captures d'écran (à faire toi)

Le README a 6 emplacements pour des screenshots :
`docs/screenshots/{01-dashboard, 02-action-card, 03-maturity-score, 04-coverage, 05-watchlist, 06-coach-modal}.png`

**Méthode automatique** :

```powershell
# Terminal 1 : lance la GUI en admin
.\cmd\harden-gui\build\bin\harden-gui.exe

# Terminal 2 : lance le screenshot script (tu as 8s pour mettre la GUI au premier plan)
pwsh -File tools/take-screenshots.ps1
```

**Méthode manuelle** (si tu préfères contrôler) :

Win+Shift+S sur chaque écran, sauve dans `docs/screenshots/` avec les noms
exacts attendus par le README. Recadre pour que la GUI prenne tout le frame.

**Astuce qualité** : redimensionne ta fenêtre GUI à ~1280×800 avant de
capturer. C'est la bonne taille pour GitHub (qui downscale au-delà).

## 🎬 2. GIF animé en haut du README (vraiment impactant)

**Outil recommandé** : [ScreenToGif](https://www.screentogif.com/) — gratuit, Windows, génère du GIF optimisé.

1. Installe ScreenToGif
2. Sélectionne la zone autour de la GUI
3. Enregistre 10-15 secondes max où tu :
   - Lances la GUI
   - Cliques "Vérifier"
   - Une carte action s'affiche
   - Cliques sur 💡 pour montrer le coach
   - Cliques sur "📊 Score" pour montrer le score A/B/C/D
4. Export en GIF, **<2 MB** (fps=8, palette 64 couleurs suffit)
5. Sauve dans `docs/screenshots/00-hero.gif`
6. Update le README pour pointer vers le GIF au lieu de l'image

Plus le GIF est court et fluide, plus il convertit. **15 secondes max**.

## 🎥 3. Vidéo YouTube (optionnel mais SEO++)

Si tu veux pousser la promotion plus loin :

- **OBS Studio** (gratuit) pour record une démo de 1-2 min
- Upload sur YouTube avec titre clair "Harden Windows 11 in 1 click — open source"
- Lien dans le README en haut

C'est ce qui fait que les outils opensource percent : démo vidéo + screenshots.

## 🏷️ 4. Topics GitHub (SEO interne)

Configure les topics qui font remonter le repo dans la search GitHub :

```bash
# Si tu as gh CLI installée :
gh repo edit --add-topic windows-hardening
gh repo edit --add-topic windows-11
gh repo edit --add-topic security
gh repo edit --add-topic cybersecurity
gh repo edit --add-topic open-source
gh repo edit --add-topic hardening-tool
gh repo edit --add-topic privacy
gh repo edit --add-topic defender
gh repo edit --add-topic powershell
gh repo edit --add-topic golang
gh repo edit --add-topic wails
gh repo edit --add-topic cis-benchmark
gh repo edit --add-topic anssi

# Description visible en haut du repo (EN — vise audience internationale) :
gh repo edit --description "Harden Windows 11 in one click. 95 security rules in plain English, fully reversible, mapped to CIS/ANSSI/MS Baseline. CLI + GUI. EN/FR."
```

Sinon **manuellement** sur https://github.com/koff75/harden-win11/settings (icône engrenage en haut à droite du repo) :
- Description : copie le texte ci-dessus
- Topics : ajoute la liste ci-dessus
- Website : optionnel (un site Vercel/Netlify avec démo si tu veux)

## 📦 5. Première Release

Le workflow `.github/workflows/release.yml` se déclenche sur push de tag :

```bash
# Crée le tag v0.3.0 (ou ce que tu veux comme premier release)
git tag -a v0.3.0 -m "Release v0.3.0 — first public release"
git push origin v0.3.0
```

GitHub Actions va automatiquement :
1. Build `harden-engine.exe` + `harden-gui.exe` avec la version dans le binaire
2. Générer le ZIP portable `Harden-Win11-0.3.0.zip` avec engine + GUI + manifests + actions + run-as-admin.bat
3. Calculer les SHA256
4. Créer la release sur GitHub avec les 3 fichiers téléchargeables (ZIP, engine, GUI) + leurs .sha256
5. Auto-générer les release notes depuis les commits

Le user qui arrive sur ta page Releases voit immédiatement le ZIP à télécharger.

## 🔍 6. Indexation externe

Une fois le repo bien présenté, soumets-le à :

- **Awesome Windows** (https://github.com/Awesome-Windows/Awesome) — PR avec ton repo
- **Awesome Security** (https://github.com/sbilly/awesome-security) — PR
- **r/PowerShell** ou **r/cybersecurity** sur Reddit (post sobre, pas de spam)
- **Hacker News** (Show HN) — 1 fois max, post bien rédigé
- **Twitter / Mastodon** avec les bons hashtags : #windows11 #cybersecurity #opensource

**Ne fais pas de spam multi-canal le même jour** — étale sur 1-2 semaines.

## ✅ Critères de "ready to launch"

- [ ] Au moins 4 screenshots dans `docs/screenshots/` (dashboard, action-card, score, coverage)
- [ ] GIF hero en haut du README (15s max)
- [ ] Description repo + topics configurés
- [ ] Première release publiée avec ZIP téléchargeable
- [ ] Smoke test E2E manuel passé sur ta propre machine ([`docs/manual-e2e-checklist.md`](manual-e2e-checklist.md))
- [ ] (Optionnel) Vidéo YouTube de 1-2 min
