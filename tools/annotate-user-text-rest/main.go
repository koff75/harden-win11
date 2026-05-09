// annotate-user-text-rest — Annote les 66 rules NON couvertes par
// annotate-user-text initial. Mêmes 4 champs user_today / user_after /
// user_for_who / user_risk en français accessible.
//
// Idempotent : skip si user_today déjà présent.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ut struct {
	Today, After, ForWho, Risk string
}

var texts = map[string]ut{
	// ─── DEFENDER (8 restantes) ───
	"defender.ioav": {
		Today:  "Defender ne scanne pas les fichiers téléchargés via Internet Explorer / Outlook au moment du téléchargement.",
		After:  "Tout fichier qui arrive via un download est analysé immédiatement par Defender, avant d'arriver sur ton disque.",
		ForWho: "Tout le monde.",
		Risk:   "Aucun en usage normal.",
	},
	"defender.script_scanning": {
		Today:  "Les scripts (PowerShell, JS, VBS) lancés sur ton PC ne sont pas analysés par Defender.",
		After:  "Defender intercepte et analyse les scripts en mémoire avant qu'ils ne s'exécutent.",
		ForWho: "Tout le monde — vrai vecteur de malware moderne (fileless attacks).",
		Risk:   "Très rare ralentissement sur des grosses suites de scripts (build CI/CD).",
	},
	"defender.nis": {
		Today:  "Pas de système d'inspection des connexions réseau pour bloquer les exploits connus.",
		After:  "Defender inspecte le trafic réseau en temps réel et bloque les paquets correspondant à des exploits connus.",
		ForWho: "Tout le monde, surtout sur Wi-Fi public.",
		Risk:   "Très rare faux positif sur du trafic légitime.",
	},
	"defender.pua": {
		Today:  "Les Potentially Unwanted Applications (toolbars, crapware, miners cachés) ne sont pas détectées par défaut.",
		After:  "Defender bloque automatiquement le téléchargement et l'installation des PUA connues.",
		ForWho: "Tout le monde — utile contre les bundles d'installeurs.",
		Risk:   "Quasi nul. Très rare cas d'app légitime mais agressive (CCleaner ancienne version) qui se prend la croix.",
	},
	"defender.sample_submission": {
		Today:  "Quand Defender trouve un fichier suspect mais inconnu, il ne l'envoie pas à Microsoft pour analyse approfondie.",
		After:  "Les samples suspects (sans données perso) sont envoyés à Microsoft pour analyse cloud — Defender bénéficie de la détection collaborative.",
		ForWho: "Tout le monde qui veut une protection à jour.",
		Risk:   "Microsoft reçoit des samples (jamais tes documents personnels — uniquement des binaires suspects). Skip si tu es vraiment paranoïaque.",
	},
	"defender.cloud_protection": {
		Today:  "Defender ne consulte pas le cloud Microsoft pour les détections (mode local-only).",
		After:  "Defender consulte la base cloud de Microsoft à chaque analyse — détection beaucoup plus à jour.",
		ForWho: "Tout le monde avec une connexion internet.",
		Risk:   "Aucun fonctionnel. Petite latence supplémentaire sur certains scans (négligeable).",
	},
	"defender.signatures": {
		Today:  "Pas de vérification que les signatures Defender sont à jour récemment.",
		After:  "On vérifie que les signatures ont été mises à jour dans les dernières 24h. Si pas, alerte.",
		ForWho: "Tout le monde — un Defender avec signatures vieilles de 2 mois ne sert plus à grand chose.",
		Risk:   "Aucun.",
	},
	"defender.tamper_protection_check": {
		Today:  "Aucune vérification que la Tamper Protection (anti-désactivation Defender par malware) est active.",
		After:  "On vérifie qu'elle est ON. Si désactivée, on alerte (tu dois aller la réactiver dans Windows Security manuellement).",
		ForWho: "Tout le monde.",
		Risk:   "Aucun (vérification read-only).",
	},

	// ─── FIREWALL (2 restantes) ───
	"firewall.block_smb_public": {
		Today:  "Sur Wi-Fi public (café, aéroport), le port 445 (partage de fichiers SMB) reste potentiellement accessible aux autres machines du réseau.",
		After:  "Le port 445 est explicitement bloqué en entrée sur les réseaux Public — ceinture-bretelles si une autre rule ouvrait par erreur.",
		ForWho: "Tout le monde, surtout en mobilité.",
		Risk:   "Aucun en usage standard. Empêche partage SMB sur Wi-Fi public (peu probable cas d'usage).",
	},
	"firewall.block_netbios_public": {
		Today:  "Sur Wi-Fi public, les ports NetBIOS (137-139) restent potentiellement accessibles — vecteur classique d'attaque latérale.",
		After:  "Ports NetBIOS bloqués en entrée sur le profil Public.",
		ForWho: "Tout le monde, surtout en mobilité.",
		Risk:   "Aucun. Si tu utilisais NetBIOS sur Wi-Fi public (jamais), il faudrait ajouter une exception.",
	},

	// ─── SYSTEM SETTINGS (4 restantes) ───
	"system_settings.uac_enable_lua": {
		Today:  "L'UAC (User Account Control, le mécanisme qui te demande \"voulez-vous autoriser cette app à modifier le système ?\") est peut-être désactivé.",
		After:  "L'UAC est forcé activé. Tout programme qui veut admin doit demander explicitement.",
		ForWho: "Tout le monde — c'est la barrière de base contre les malwares qui veulent s'installer.",
		Risk:   "Aucun. Quelques fenêtres UAC en plus à l'install de softs.",
	},
	"system_settings.uac_prompt_secure_desktop": {
		Today:  "Quand l'UAC affiche sa fenêtre, le bureau normal reste actif derrière — un malware peut tenter d'envoyer des clics auto.",
		After:  "L'UAC s'affiche sur un \"secure desktop\" isolé — aucun autre programme ne peut interagir avec lui pendant que tu décides.",
		ForWho: "Tout le monde.",
		Risk:   "Très bref clignotement noir à chaque UAC. Aucun impact fonctionnel.",
	},
	"system_settings.rdp_firewall_disable": {
		Today:  "Même si RDP service est désactivé, les règles firewall pour le port 3389 (Bureau à distance) peuvent rester ouvertes.",
		After:  "Les règles firewall RDP sont supprimées — port 3389 fermé partout.",
		ForWho: "Tout le monde qui n'utilise pas RDP. Cohérent avec rdp_disable.",
		Risk:   "Si tu réactives RDP plus tard, faudra ré-ajouter les règles firewall.",
	},
	"system_settings.fast_startup_off": {
		Today:  "Fast Startup (Windows 8+) garde le noyau en hibernation pour booter plus vite — mais empêche certaines mises à jour drivers et complique le forensics.",
		After:  "Fast Startup désactivé. Boot un peu plus lent (5-10s) mais shutdown propre, drivers à jour à chaque démarrage.",
		ForWho: "Profile maximal. Sur SSD moderne, la différence de boot est négligeable.",
		Risk:   "Boot 5-10 secondes plus lent. Aucun autre impact.",
	},

	// ─── PRIVACY (10 restantes) ───
	"privacy.advertising_id_machine": {
		Today:  "Windows associe un identifiant publicitaire unique à ton compte — partagé avec les apps Store pour profiling.",
		After:  "L'Advertising ID est désactivé par stratégie machine pour TOUS les utilisateurs du PC.",
		ForWho: "Tout le monde qui n'aime pas le tracking.",
		Risk:   "Quelques apps Store gratuites peuvent montrer des pubs moins ciblées (= un peu plus de pubs random).",
	},
	"privacy.advertising_id_user": {
		Today:  "Ton compte utilisateur a un identifiant publicitaire personnel utilisé par les apps Store.",
		After:  "Désactivé pour ton compte courant.",
		ForWho: "Tout le monde — équivalent par-utilisateur de l'option machine-wide.",
		Risk:   "Idem advertising_id_machine.",
	},
	"privacy.online_speech_off": {
		Today:  "Windows envoie ta voix (clavier vocal, dictée) à Microsoft pour transcription côté cloud.",
		After:  "La reconnaissance vocale en ligne est désactivée. Si tu utilises la dictée, ça reste local (qualité plus basique).",
		ForWho: "Tout le monde qui ne fait pas de dictée régulière.",
		Risk:   "Si tu utilises la dictée vocale Windows, qualité dégradée (modèle local seulement).",
	},
	"privacy.activity_history_off": {
		Today:  "Windows enregistre ton historique d'activités (apps lancées, fichiers ouverts) localement et le synchronise vers le cloud Microsoft.",
		After:  "L'historique d'activités est désactivé. Plus de Timeline dans Win+Tab.",
		ForWho: "Tout le monde qui n'utilise pas la Timeline.",
		Risk:   "Plus de Timeline (Win+Tab montre seulement les apps ouvertes maintenant).",
	},
	"privacy.consumer_features_off": {
		Today:  "Windows installe automatiquement des apps suggérées (Candy Crush, Spotify, etc.) à la connexion d'un nouvel utilisateur.",
		After:  "Windows n'installe plus rien automatiquement — tu installes ce que tu veux depuis le Store.",
		ForWho: "Tout le monde.",
		Risk:   "Aucun. Tu peux toujours installer manuellement les apps depuis le Microsoft Store.",
	},
	"privacy.silent_apps_off": {
		Today:  "Windows peut télécharger silencieusement des apps suggérées en arrière-plan.",
		After:  "Plus de download silencieux. Tu décides quoi installer.",
		ForWho: "Tout le monde.",
		Risk:   "Aucun.",
	},
	"privacy.settings_suggestions_off": {
		Today:  "Settings affiche des suggestions \"essayez ceci\" et notifications proactives.",
		After:  "Settings reste minimal et silencieux.",
		ForWho: "Tout le monde qui n'aime pas le bruit.",
		Risk:   "Tu pourrais manquer une suggestion de feature utile (rare).",
	},
	"privacy.start_suggestions_off": {
		Today:  "Le menu Start affiche des suggestions d'apps Store que tu n'as pas installées.",
		After:  "Plus de pubs dans le menu Start.",
		ForWho: "Tout le monde.",
		Risk:   "Aucun.",
	},
	"privacy.tips_welcome_off": {
		Today:  "Windows affiche des notifications \"astuces\" et écrans d'accueil après les mises à jour.",
		After:  "Plus de notifications de tips, plus d'écran d'accueil après update.",
		ForWho: "Tout le monde.",
		Risk:   "Aucun.",
	},
	"privacy.ink_text_collection_off": {
		Today:  "Windows collecte ton écriture manuscrite et tes saisies clavier pour entraîner ses modèles linguistiques personnalisés (envoyés à Microsoft).",
		After:  "Plus de collecte de tes saisies. Pas d'apprentissage personnalisé du clavier.",
		ForWho: "Tout le monde.",
		Risk:   "Auto-corrections un peu moins précises avec le temps (le modèle de base reste fonctionnel).",
	},

	// ─── ASR (15 restantes) ───
	"asr.block_office_child_processes": {
		Today:  "Word/Excel/PowerPoint peuvent lancer cmd.exe, PowerShell ou n'importe quel programme via macros — vecteur classique de malware.",
		After:  "Bloque toute création de processus enfant depuis Office. Une macro malveillante ne peut plus lancer Mimikatz, cmd, etc.",
		ForWho: "Tout le monde qui ouvre des Office reçus par mail.",
		Risk:   "Si tu as des macros qui appellent cmd/powershell légitimement (rare en perso), à autoriser au cas par cas.",
	},
	"asr.block_office_code_injection": {
		Today:  "Office peut injecter du code dans d'autres processus — technique utilisée par des malwares pour contourner les protections.",
		After:  "Bloque les tentatives d'injection. Un malware Office reste prisonnier de son propre processus.",
		ForWho: "Tout le monde.",
		Risk:   "Très rare. Quelques add-ins Office anciens peuvent être affectés.",
	},
	"asr.block_office_comm_child_processes": {
		Today:  "Outlook peut lancer des programmes (cmd, scripts, etc.) — vecteur d'infection par mail piégé.",
		After:  "Outlook ne peut plus lancer de processus enfant. Si tu cliques par erreur sur une PJ piégée, la chaîne casse.",
		ForWho: "Tout le monde qui utilise Outlook.",
		Risk:   "Très rare. Quelques workflows entreprise spécifiques.",
	},
	"asr.block_win32_api_office_macros": {
		Today:  "Les macros VBA Office peuvent appeler les API Win32 directement — accès à toute la machine.",
		After:  "Macros bloquées d'appeler les Win32 API. Capacité de nuisance fortement réduite.",
		ForWho: "Tout le monde qui n'écrit pas du VBA expert.",
		Risk:   "Si tu programmes en VBA professionnel avec Win32 API, tes macros casseront.",
	},
	"asr.block_email_executable_content": {
		Today:  "Une PJ téléchargée depuis ton mail peut être exécutée directement (.exe, .bat, .ps1, .vbs...).",
		After:  "Les contenus exécutables téléchargés depuis email/webmail sont bloqués. Tu dois explicitement les sauver puis les lancer (et là Defender les scanne).",
		ForWho: "Tout le monde.",
		Risk:   "Quasi nul. Si tu télécharges souvent des installeurs depuis ton webmail, légère friction.",
	},
	"asr.block_obfuscated_scripts": {
		Today:  "Les scripts obfusqués (encodés en Base64, avec des chaînes brouillées) peuvent s'exécuter sans alerte.",
		After:  "Defender détecte les patterns d'obfuscation et bloque l'exécution.",
		ForWho: "Tout le monde.",
		Risk:   "Très rare faux positif sur des scripts légitimes mais obfusqués (anti-reverse engineering, packers).",
	},
	"asr.block_js_vbs_launch": {
		Today:  "Un fichier .js ou .vbs téléchargé peut télécharger et lancer un .exe automatiquement (technique des droppers).",
		After:  "Bloque le lancement d'exe par JS/VBS téléchargés. La chaîne d'attaque casse.",
		ForWho: "Tout le monde.",
		Risk:   "Quasi nul.",
	},
	"asr.block_unsigned_usb": {
		Today:  "Si tu branches une clé USB, n'importe quel .exe non signé dessus peut s'exécuter.",
		After:  "Defender bloque les processus non-signés/non-trusted lancés depuis une clé USB.",
		ForWho: "Profile business / maximal. Utile en environnement où on échange des clés USB.",
		Risk:   "Tes outils portables non-signés sur clé USB ne marcheront plus (utilitaires sysinternals signés OK ; tools custom non-signés KO).",
	},
	"asr.block_wmi_persistence": {
		Today:  "WMI Event Subscription est une technique de persistance malware (s'exécute au démarrage sans laisser de trace dans Run keys).",
		After:  "Defender bloque les nouvelles persistances WMI Event Subscription.",
		ForWho: "Profile maximal. Usage WMI légitime rare en perso.",
		Risk:   "Si tu utilises WMI Event Subscription pour de la supervision système, casse.",
	},
	"asr.block_psexec_wmi": {
		Today:  "PsExec et WMI peuvent lancer des processus à distance — techniques classiques de mouvement latéral des attaquants.",
		After:  "Defender bloque la création de processus via PsExec et WMI.",
		ForWho: "Profile maximal. Utile sauf si tu fais de l'admin réseau.",
		Risk:   "PsExec ne marche plus pour le dépannage distant. Outils Sysinternals classiques affectés.",
	},
	"asr.block_unprevalent_executables": {
		Today:  "N'importe quel .exe peu commun (jamais vu ailleurs dans le monde) peut tourner sans alerte.",
		After:  "Les .exe rares ou non-signés sont bloqués automatiquement. Defender ne laisse passer que les apps connues.",
		ForWho: "Profile maximal. À éviter si tu testes du logiciel custom.",
		Risk:   "Tes apps internes / scripts compilés / outils custom seront bloqués au début. Faut whitelister.",
	},
	"asr.block_adobe_reader_child": {
		Today:  "Adobe Reader peut lancer des processus enfant — vecteur d'infection via PDF piégé.",
		After:  "Adobe Reader ne peut plus lancer d'autre programme. Si tu ouvres un PDF malveillant, la chaîne casse.",
		ForWho: "Tout le monde qui ouvre des PDF inconnus.",
		Risk:   "Très rare workflow PDF qui appelle un programme externe.",
	},
	"asr.block_vulnerable_drivers": {
		Today:  "Certains drivers Windows signés mais vulnérables (BYOVD = Bring Your Own Vulnerable Driver) peuvent être exploités pour grimper en kernel.",
		After:  "Defender bloque le chargement de la liste connue de drivers vulnérables.",
		ForWho: "Tout le monde.",
		Risk:   "Très rare driver hardware ancien qui ne marche plus (à remplacer par version récente).",
	},
	"asr.block_safe_mode_reboot": {
		Today:  "Un attaquant peut reboot ton PC en Safe Mode pour désactiver Defender et faire ses sales coups.",
		After:  "Defender bloque les reboots en Safe Mode forcés par un programme.",
		ForWho: "Profile business / maximal.",
		Risk:   "Reboot en Safe Mode bloqué (par ex via msconfig). Affecte les workflows de troubleshooting.",
	},
	"asr.block_impersonated_tools": {
		Today:  "Un malware peut copier cmd.exe ou PowerShell sous un autre nom pour échapper à la détection.",
		After:  "Defender détecte les outils système renommés/copiés et bloque leur exécution.",
		ForWho: "Tout le monde.",
		Risk:   "Très rare faux positif si tu utilises des outils sysinternals renommés volontairement.",
	},
	"asr.block_webshell_servers": {
		Today:  "Sur un serveur, un webshell (ASP, PHP) peut être créé via une vulnérabilité web pour persister.",
		After:  "Defender bloque la création de webshells dans les répertoires web.",
		ForWho: "Serveurs Windows. Inutile sur poste de travail mais sans dommage.",
		Risk:   "Aucun sur un poste perso.",
	},
}

// Bloatware (27) → tous ont la même logique : "Cette app Microsoft est
// installée. Si tu désinstalles, libère X Mo, plus de notifs. Pour qui :
// ceux qui ne l'utilisent pas. Ce qui peut t'embêter : tu devras la
// réinstaller via le Store si tu en as besoin plus tard."
//
// On boucle sur les noms friendly d'apps pour générer.
type bloat struct{ ID, Name string }

var bloatwares = []bloat{
	{"bloatware.clipchamp", "Clipchamp (éditeur vidéo Microsoft)"},
	{"bloatware.bing_news", "Bing News (l'app actualités)"},
	{"bloatware.bing_weather", "Bing Weather (la météo)"},
	{"bloatware.get_help", "Get Help (assistant de support Windows)"},
	{"bloatware.get_started", "Get Started (tutoriel Windows)"},
	{"bloatware.solitaire", "Microsoft Solitaire Collection"},
	{"bloatware.mixed_reality", "Mixed Reality Portal (casque VR)"},
	{"bloatware.people", "People (carnet d'adresses)"},
	{"bloatware.skype", "Skype (messagerie/visio Microsoft)"},
	{"bloatware.feedback_hub", "Feedback Hub (envoi de feedback à Microsoft)"},
	{"bloatware.your_phone", "Your Phone / Phone Link (lien Android)"},
	{"bloatware.zune_music", "Groove Music (lecteur audio)"},
	{"bloatware.zune_video", "Films & TV (lecteur vidéo)"},
	{"bloatware.disney", "Disney+"},
	{"bloatware.tiktok", "TikTok"},
	{"bloatware.facebook", "Facebook"},
	{"bloatware.instagram", "Instagram"},
	{"bloatware.twitter", "Twitter / X"},
	{"bloatware.linkedin", "LinkedIn"},
	{"bloatware.netflix", "Netflix"},
	{"bloatware.candy_crush", "Candy Crush Saga"},
	{"bloatware.spotify_ab", "Spotify (version Store)"},
	{"bloatware.spotify", "Spotify"},
	{"bloatware.apple_music", "Apple Music"},
	{"bloatware.dolby_access", "Dolby Access"},
	{"bloatware.jimmylin", "JimmyLin (autres apps tierces préinstallées)"},
	{"bloatware.pub_5319275a", "App publisher 5319275A (apps tierces préinstallées)"},
}

func init() {
	// Auto-génère les textes bloatware avec template.
	for _, b := range bloatwares {
		texts[b.ID] = ut{
			Today:  fmt.Sprintf("L'app %s est préinstallée ou installée sur ton PC. Elle prend de la place et peut tourner en arrière-plan.", b.Name),
			After:  "L'app est désinstallée. Plus de notifications, plus de mises à jour, plus de processus en arrière-plan.",
			ForWho: fmt.Sprintf("Ceux qui n'utilisent pas %s.", b.Name),
			Risk:   "Si tu en as besoin plus tard, tu peux la réinstaller depuis le Microsoft Store. Aucune donnée perdue.",
		}
	}
}

func main() {
	root := "manifests"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	patched, skipped := 0, 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(raw)
		original := content

		for ruleID, txt := range texts {
			ruleStart := regexp.MustCompile(`(?m)^(\s+)- id:\s+` + regexp.QuoteMeta(ruleID) + `\s*$`)
			loc := ruleStart.FindStringSubmatchIndex(content)
			if loc == nil {
				continue
			}
			indent := content[loc[2]:loc[3]] + "  "
			ruleBlockStart := loc[1]

			searchEnd := len(content)
			if next := regexp.MustCompile(`(?m)^\s+- id:`).FindStringIndex(content[ruleBlockStart:]); next != nil {
				searchEnd = ruleBlockStart + next[0]
			}
			block := content[ruleBlockStart:searchEnd]
			if strings.Contains(block, "user_today:") {
				skipped++
				continue
			}

			rel := regexp.MustCompile(`(?m)^(\s+)action:\s+`)
			al := rel.FindStringSubmatchIndex(block)
			if al == nil {
				continue
			}
			abs := ruleBlockStart + al[0]

			esc := func(s string) string {
				s = strings.ReplaceAll(s, `\`, `\\`)
				s = strings.ReplaceAll(s, `"`, `\"`)
				return s
			}
			insertion := indent + `user_today: "` + esc(txt.Today) + "\"\n" +
				indent + `user_after: "` + esc(txt.After) + "\"\n" +
				indent + `user_for_who: "` + esc(txt.ForWho) + "\"\n" +
				indent + `user_risk: "` + esc(txt.Risk) + "\"\n"

			content = content[:abs] + insertion + content[abs:]
			patched++
		}

		if content != original {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
			fmt.Printf("patched: %s\n", path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walk: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("\nDone. %d rules annotées, %d déjà annotées (skip).\n", patched, skipped)
}
