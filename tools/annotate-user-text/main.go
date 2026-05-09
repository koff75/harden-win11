// annotate-user-text — Annote les rules clés avec les 4 champs
// user_today / user_after / user_for_who / user_risk en français normal,
// sans jargon technique.
//
// Insertion juste avant le `action:` de chaque rule. Idempotent.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type userText struct {
	Today  string
	After  string
	ForWho string
	Risk   string
}

// 25 règles annotées avec textes accessibles. Volontairement courts,
// 1 phrase chacun, zéro jargon technique.
var texts = map[string]userText{
	// ─── DEFENDER ───
	"defender.realtime": {
		Today:  "Quand tu télécharges ou ouvres un fichier, rien ne le scanne pour détecter un virus.",
		After:  "Chaque fichier est analysé en arrière-plan dès que tu y touches. Les menaces connues sont bloquées immédiatement.",
		ForWho: "Tout le monde — c'est l'antivirus de base de Windows.",
		Risk:   "Quasi nul. Très rare ralentissement à l'ouverture de fichiers très gros (>1 Go) ou ZIP avec milliers de fichiers.",
	},
	"defender.behavior_monitoring": {
		Today:  "Defender ne regarde que la signature des fichiers (= si on connaît déjà le virus).",
		After:  "Defender surveille aussi le comportement des programmes en cours d'exécution. Détecte les nouveaux malwares qui se cachent en mémoire.",
		ForWho: "Tout le monde, surtout pour résister aux menaces récentes pas encore dans les bases antivirus.",
		Risk:   "Aucun en usage normal.",
	},
	"defender.network_protection": {
		Today:  "Si tu cliques sur un lien malveillant, ton navigateur charge la page (qui peut tenter de t'infecter).",
		After:  "Defender bloque la connexion vers les sites réputés malveillants AVANT que ton navigateur ne charge la page.",
		ForWho: "Tout le monde qui surf sur internet.",
		Risk:   "Très rare faux positif sur un site légitime mais récent. Tu peux le débloquer manuellement si besoin.",
	},
	"defender.controlled_folder_access": {
		Today:  "N'importe quel programme peut écrire dans tes Documents, Photos, Bureau (= zone d'attaque pour les ransomwares).",
		After:  "Seules les apps autorisées (Word, Photoshop, etc.) peuvent y écrire. Un ransomware se prend une erreur et n'arrive pas à chiffrer tes fichiers.",
		ForWho: "Très utile contre les ransomwares. Plutôt en mode 'paranoid' car nécessite un peu de configuration.",
		Risk:   "Au début tu vas devoir autoriser à la main certaines apps légitimes (Steam, OBS, Visual Studio, jeux). Quelques minutes de setup les premiers jours.",
	},

	// ─── FIREWALL ───
	"firewall.profile_public": {
		Today:  "Quand tu te connectes au Wi-Fi d'un café ou d'un aéroport, les autres ordinateurs du réseau peuvent essayer de te joindre.",
		After:  "Sur ces réseaux non-sûrs, ton PC devient invisible — personne ne peut s'y connecter.",
		ForWho: "Indispensable pour un laptop qui sort de la maison. Inutile sur un fixe qui ne quitte jamais ton bureau.",
		Risk:   "Si tu fais du jeu en LAN ou du partage de fichiers dans un café, ça bloquera. Sinon rien.",
	},
	"firewall.profile_private": {
		Today:  "Sur ton réseau Wi-Fi maison (marqué 'privé'), les autres appareils peuvent se connecter à ton PC.",
		After:  "Reste accessible aux machines de ta maison mais on bloque toute tentative inhabituelle.",
		ForWho: "Tout le monde qui a un PC à la maison.",
		Risk:   "Aucun pour usage normal. Si tu héberges un service local (NAS, serveur de jeu), il faudra créer une exception.",
	},
	"firewall.profile_domain": {
		Today:  "Sur un PC en entreprise (joint au domaine), le firewall peut être plus permissif.",
		After:  "On force le blocage par défaut même sur le réseau d'entreprise.",
		ForWho: "Postes en entreprise (domaine Active Directory).",
		Risk:   "Si ton admin réseau a configuré des outils qui ont besoin d'accès entrant, faudra créer des règles.",
	},

	// ─── ACCOUNTS ───
	"accounts.disable_unused": {
		Today:  "Plusieurs comptes systèmes par défaut existent sur Windows (Administrator, Guest…). Souvent activés sans raison.",
		After:  "Ces comptes inutiles sont désactivés. Un attaquant ne peut plus tenter de mots de passe dessus.",
		ForWho: "Tout le monde.",
		Risk:   "Aucun pour 99% des gens. Si tu utilises explicitement le compte 'Invité' (très rare), à ne pas faire.",
	},
	"accounts.rename_admin": {
		Today:  "Le compte super-administrateur s'appelle 'Administrator' — un nom universellement connu des attaquants automatisés.",
		After:  "Renommé en 'AdminLocal_<TonPC>'. Un robot qui tente 1000 mots de passe sur 'Administrator' ne trouve plus le compte.",
		ForWho: "PC perso ou solo. À éviter en entreprise (la GPO peut écraser le rename).",
		Risk:   "Si tu as des scripts qui appellent 'Administrator' explicitement, ils casseront. Très rare en perso.",
	},

	// ─── UAC / RDP / POWER ───
	"system_settings.uac_consent_admin": {
		Today:  "Quand un programme demande les droits administrateur, Windows peut accepter automatiquement (sans te demander).",
		After:  "Tu dois explicitement valider chaque demande d'élévation. Un malware ne peut plus passer en silence.",
		ForWho: "Tout le monde, sauf si tu fais du dev/admin et qu'une fenêtre UAC par jour t'agace.",
		Risk:   "Quelques fenêtres UAC supplémentaires en installant des logiciels. C'est l'objectif : voir ce qui demande à grimper en admin.",
	},
	"system_settings.uac_deny_user_elevation": {
		Today:  "Un compte standard peut demander à devenir admin (en saisissant un mot de passe).",
		After:  "Les comptes standards ne peuvent plus s'élever du tout. Faut se connecter avec le compte admin pour ça.",
		ForWho: "Setup avec plusieurs comptes (parents/enfants, famille). À éviter sur PC perso solo (= ton compte unique est admin).",
		Risk:   "Sur PC perso solo : aucun. Sur PC partagé : impossible pour les comptes standards d'installer un logiciel.",
	},
	"system_settings.rdp_disable": {
		Today:  "Le Bureau à Distance (RDP) est peut-être activé sur ton PC. Si oui, c'est une porte d'entrée connue des attaquants.",
		After:  "RDP est éteint. Plus personne ne peut se connecter à ton PC à distance via cette voie.",
		ForWho: "Tout le monde, sauf si tu utilises RDP pour dépanner ton PC depuis ailleurs.",
		Risk:   "Si tu te connectes à ton PC depuis ailleurs (depuis le boulot, par exemple), tu ne pourras plus.",
	},
	"system_settings.hibernate_off": {
		Today:  "Quand tu mets ton PC en veille prolongée, Windows écrit la mémoire (= secrets, mots de passe en cours) sur le disque.",
		After:  "L'hibernation est désactivée. Soit tu fais Veille (sans copie disque), soit tu éteins.",
		ForWho: "PC fixe principalement. À éviter sur laptop où l'hibernation économise la batterie.",
		Risk:   "Sur laptop : tu perds l'autonomie de l'hibernation longue. Le sleep classique consomme un peu plus.",
	},

	// ─── NETWORK ───
	"network.llmnr_disable": {
		Today:  "Quand ton PC cherche un nom de machine sur le réseau, il demande à voix haute à tout le monde du réseau.",
		After:  "Il n'utilise plus que le DNS classique. Personne sur le réseau ne peut intercepter ses recherches.",
		ForWho: "Tout le monde. C'est un vecteur classique d'attaque sur Wi-Fi public.",
		Risk:   "Sur un réseau familial sans DNS local correctement configuré, certaines découvertes automatiques peuvent ne plus marcher (rare).",
	},
	"network.mdns_disable": {
		Today:  "Ton PC fait de la découverte automatique style Bonjour Apple sur le réseau (cast vers TV, AirPrint, etc.).",
		After:  "Plus de découverte automatique de ce type. Plus discret sur le réseau.",
		ForWho: "Profile 'maximal' (paranoid). À éviter chez toi si tu utilises Chromecast/AirPlay/imprimantes Apple.",
		Risk:   "Plus de Chromecast, plus d'AirPrint. Si tu cast régulièrement vers une TV, ne pas l'activer.",
	},
	"network.netbios_off": {
		Today:  "Un vieux protocole de nommage Windows (NetBIOS) est encore activé. Souvent utilisé pour les attaques sur Wi-Fi public.",
		After:  "Désactivé. Ton PC ne répond plus aux questions NetBIOS d'un attaquant local.",
		ForWho: "Tout le monde, surtout en mobilité.",
		Risk:   "Si tu accèdes à un partage réseau via un nom court (genre \\\\NAS-MAISON), il faudra utiliser l'IP ou le nom complet.",
	},
	"network.wpad_disable": {
		Today:  "Au démarrage, ton PC cherche automatiquement un proxy web. Un attaquant local peut répondre et devenir ton intermédiaire.",
		After:  "Plus de découverte automatique de proxy. Si tu en utilises un, tu le configures explicitement.",
		ForWho: "Tout le monde, surtout en mobilité (Wi-Fi public).",
		Risk:   "Si ton entreprise utilise un proxy auto via WPAD, l'accès web peut casser (situation rare en perso).",
	},
	"network.smbv1_disable": {
		Today:  "Un système de partage de fichiers vieux de 30 ans est encore activé. Utilisé par les ransomwares WannaCry et NotPetya pour se répandre.",
		After:  "Ce vieux système est éteint. Ton PC ne parle plus que les versions modernes (sécurisées).",
		ForWho: "Tout le monde, sauf si tu accèdes à un serveur ou NAS d'avant 2012 que tu ne veux pas remplacer.",
		Risk:   "Si un partage réseau ne marche plus, c'est qu'il était en SMBv1. Faut mettre le NAS à jour ou faire une exception.",
	},
	"network.smb_client_signing": {
		Today:  "Quand tu accèdes à un partage réseau, les paquets ne sont pas signés. Quelqu'un sur le réseau pourrait les modifier.",
		After:  "Les paquets sont signés des 2 côtés. Toute altération est détectée et rejetée.",
		ForWho: "Tout le monde qui utilise des partages réseau (NAS, partages Windows).",
		Risk:   "Rare souci de compatibilité avec très vieux NAS qui ne supportent pas la signature.",
	},
	"network.smb_server_signing": {
		Today:  "Si ton PC partage des fichiers, les paquets sortants ne sont pas signés.",
		After:  "Tes partages forcent la signature. Personne ne peut intercepter et modifier sans que ce soit détecté.",
		ForWho: "Tout le monde qui partage des dossiers depuis son PC.",
		Risk:   "Idem que client signing : très rare souci avec vieux clients.",
	},
	"network.smb_guest_auth_off": {
		Today:  "Ton PC peut se connecter à des partages sans mot de passe (mode 'invité'). Un attaquant local peut faire passer un faux serveur.",
		After:  "Tout partage sans mot de passe est refusé. Faut authentification explicite.",
		ForWho: "Tout le monde.",
		Risk:   "Si tu as un NAS configuré en mode invité (rare), faudra le passer en mode authentifié.",
	},
	"network.ntlm_v2_only": {
		Today:  "Ton PC peut être forcé à parler une vieille version d'authentification (NTLMv1) qui se casse en quelques heures.",
		After:  "Refuse toute négociation vers une version inférieure. Reste en NTLMv2 minimum (résistant).",
		ForWho: "Tout le monde, surtout en entreprise.",
		Risk:   "Très rares vieux serveurs NTLMv1-only ne marcheront plus.",
	},

	// ─── PRIVACY ───
	"privacy.recall_off": {
		Today:  "Recall (Win11 24H2) prend un screenshot de ton écran toutes les quelques secondes et stocke tout localement avec OCR.",
		After:  "Recall est complètement éteint. Plus aucune capture d'écran automatique.",
		ForWho: "Tout le monde sur Win11 24H2+.",
		Risk:   "Aucun. Recall reste une fonctionnalité expérimentale, beaucoup d'utilisateurs préfèrent l'avoir off.",
	},
	"privacy.cortana_off": {
		Today:  "Cortana est activée — assistant vocal Microsoft qui collecte tes recherches, ton calendrier, ta voix.",
		After:  "Cortana est désactivée. La barre de recherche Windows reste fonctionnelle pour les fichiers locaux.",
		ForWho: "Tout le monde qui n'utilise pas Cortana (= 99% des gens en France).",
		Risk:   "Si tu utilises Cortana pour des rappels vocaux, ne pas activer.",
	},
	"privacy.telemetry_required": {
		Today:  "Windows envoie beaucoup d'infos à Microsoft (apps lancées, fréquence, crashs). Niveau 'Enhanced' par défaut sur Pro.",
		After:  "Niveau réduit au minimum requis (juste les crash reports critiques pour Microsoft).",
		ForWho: "Tout le monde qui veut limiter ce qui sort de son PC vers Microsoft.",
		Risk:   "Aucun fonctionnellement. Microsoft a un peu moins de signals pour améliorer Windows à l'avenir.",
	},

	// ─── ASR ───
	"asr.block_lsass_credential_theft": {
		Today:  "Si un attaquant exécute du code sur ton PC, il peut extraire ton mot de passe Windows depuis la mémoire (outil 'Mimikatz').",
		After:  "Defender bloque tout accès à la mémoire d'authentification. Ton mot de passe reste protégé même si une attaque réussit l'étape 1.",
		ForWho: "Tout le monde.",
		Risk:   "Quasi nul. Un seul cas connu : certains outils de gestion d'entreprise très spécifiques peuvent être bloqués (rarement en perso).",
	},
	"asr.block_office_executable_content": {
		Today:  "Word et Excel peuvent ouvrir un fichier qui contient un programme caché et le lancer (vecteur classique des malwares par mail).",
		After:  "Si tu cliques par erreur sur une fausse facture, le programme caché ne pourra pas démarrer. La menace est neutralisée à la source.",
		ForWho: "Tout le monde qui reçoit des PDF/DOCX par mail.",
		Risk:   "Si tu programmes en VBA avec des macros qui appellent des .exe (très rare en perso), il faudra autoriser tes macros au cas par cas.",
	},
	"asr.advanced_ransomware_protection": {
		Today:  "Aucune analyse spécifique des comportements typiques des ransomwares (chiffrement massif rapide).",
		After:  "Defender détecte les patterns de chiffrement massif et bloque le processus avant qu'il ne touche tous tes fichiers.",
		ForWho: "Tout le monde.",
		Risk:   "Très rare. Si tu utilises un outil de chiffrement légitime (VeraCrypt en mode batch), il pourrait être flaggé. Whitelist possible.",
	},
	"asr.block_unprevalent_executables": {
		Today:  "N'importe quel .exe peu commun (jamais vu ailleurs) peut tourner sur ton PC sans alerte.",
		After:  "Les .exe rares ou non signés sont bloqués automatiquement. Defender ne laisse passer que les apps connues.",
		ForWho: "Profile maximal (paranoid). À éviter si tu testes du logiciel custom.",
		Risk:   "Tes apps internes / scripts compilés / outils dev custom seront bloqués au début. Faut whitelister.",
	},
}

func main() {
	root := "manifests"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	patched := 0
	skipped := 0

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

			// Insertion juste avant le `action:`.
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
