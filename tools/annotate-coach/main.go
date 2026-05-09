// annotate-coach — Annote les rules clés des manifests YAML avec un champ
// coach_example. Insertion juste avant le 'action:' de chaque rule ciblée.
// Idempotent : skip si la rule a déjà un coach_example.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Map rule_id → coach_example (français, scénario concret).
var examples = map[string]string{
	"defender.realtime": `Tu télécharges un PDF de facture qui s'avère être un dropper Emotet. Sans cette protection, le malware s'installe en arrière-plan dès que tu ouvres le PDF. Avec, Defender le détecte à la lecture du fichier et bloque l'exécution avant qu'il fasse des dégâts.`,

	"defender.controlled_folder_access": `Un ransomware se fait passer pour Word et tente de chiffrer ton dossier Documents. Cette règle interdit aux processus non whitelistés d'écrire dans tes dossiers personnels — Word marche, le ransomware se prend une erreur d'accès. C'est la dernière ligne de défense quand tout le reste a échoué.`,

	"asr.block_lsass_credential_theft": `Un attaquant a réussi à exécuter un code dans ta session. Il tente Mimikatz pour extraire ton mot de passe Windows depuis la mémoire LSASS. Cette règle ASR bloque tout accès au processus LSASS depuis du code non signé Microsoft. Sans elle, ton mot de passe AD/local part dans la nature.`,

	"asr.block_office_executable_content": `Tu reçois une "facture" en .docm. Tu cliques "activer les macros" parce que ça affiche un message convaincant. La macro tente de télécharger emotet.exe et de l'exécuter. Cette règle bloque Word/Excel/PowerPoint d'écrire ou lancer un .exe — la chaîne d'attaque casse au step 2.`,

	"firewall.profile_public": `Tu te connectes au Wi-Fi de l'aéroport. Un autre passager scanne le réseau et trouve des partages SMB ouverts sur ta machine. Cette règle force le profil Public en "deny inbound" — quand Windows détecte un Wi-Fi non sûr, plus rien ne rentre. Tes shares restent invisibles.`,

	"system_settings.uac_consent_admin": `Un .exe douteux te demande des droits admin. Sans cette règle, UAC peut accepter sans demander de mot de passe (Auto-Elevate). Avec, tu dois explicitement valider la fenêtre UAC avec ton mot de passe — ça met une vraie barrière entre un drive-by-download et l'élévation de privilèges.`,

	"system_settings.rdp_disable": `Tu as activé RDP "juste pour dépanner ton oncle". 6 mois plus tard, ton port 3389 est public sur Internet et un bot tente 10 000 mots de passe par jour. Désactiver RDP ferme ce vecteur connu massivement utilisé par les ransomwares (BlueKeep, BadRabbit, etc).`,

	"network.smbv1_disable": `EternalBlue / WannaCry se sont propagés en 2017 via SMBv1. Cette version du protocole a 30 ans et est toujours présente sur Windows pour compat avec NAS antédiluviens. Désactiver SMBv1 ferme un vecteur d'infection latéral majeur. Si ton NAS de 2010 ne marche plus, c'est l'occasion de le remplacer.`,

	"network.ntlm_v2_only": `Un attaquant sur ton réseau force ton PC à parler NTLMv1 (Responder, MITM6, etc.) puis crack ton hash en quelques heures. NTLMv2 résiste mieux aux attaques de relay et au cracking offline. Cette règle force NTLMv2 only — refuse les downgrade silencieux.`,

	"privacy.recall_off": `Recall (Win11 24H2) prend un screenshot de ton écran toutes les quelques secondes et le stocke localement avec OCR. Pratique en théorie, désastreux en pratique : tout malware qui lit ce dossier a un historique complet de tes mots de passe, RIB, Slack privé. Cette règle désactive Recall complètement.`,

	"firewall.block_smb_public": `Variant ciblé du Wi-Fi public : interdit explicitement le port 445 (SMB) en inbound sur le profil Public. C'est ceinture-bretelles avec firewall.profile_public — au cas où une autre rule ouvrirait par inadvertance le partage de fichiers.`,

	"network.wpad_disable": `WPAD (Web Proxy Auto-Discovery) cherche un serveur de proxy via DNS/NetBIOS au démarrage. Un attaquant sur ton réseau peut répondre à cette requête et devenir ton proxy → MITM sur tout ton trafic web. Désactiver WPAD ferme ce vecteur.`,
}

var insertBefore = regexp.MustCompile(`(\n)(\s+action:\s+)`)

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

		// Pour chaque rule annotée, trouver son bloc et insérer coach_example
		// juste avant le `action:` (au même niveau d'indentation).
		for ruleID, example := range examples {
			// Pattern : - id: <ruleID>\n ... action:
			ruleStart := regexp.MustCompile(`(?m)^(\s+)- id:\s+` + regexp.QuoteMeta(ruleID) + `\s*$`)
			loc := ruleStart.FindStringSubmatchIndex(content)
			if loc == nil {
				continue
			}
			indent := content[loc[2]:loc[3]] + "  " // indent du bloc + 2 espaces
			ruleBlockStart := loc[1]

			// Si déjà annotée, skip.
			searchEnd := len(content)
			if next := regexp.MustCompile(`(?m)^\s+- id:`).FindStringIndex(content[ruleBlockStart:]); next != nil {
				searchEnd = ruleBlockStart + next[0]
			}
			block := content[ruleBlockStart:searchEnd]
			if strings.Contains(block, "coach_example:") {
				skipped++
				continue
			}

			// Trouver l'action: dans ce bloc.
			rel := regexp.MustCompile(`(?m)^(\s+)action:\s+`)
			al := rel.FindStringSubmatchIndex(block)
			if al == nil {
				continue
			}
			abs := ruleBlockStart + al[0]

			// Échapper " et \ pour YAML quoted string.
			esc := strings.ReplaceAll(example, `\`, `\\`)
			esc = strings.ReplaceAll(esc, `"`, `\"`)
			insertion := indent + `coach_example: "` + esc + "\"\n"
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

	fmt.Printf("\nDone. %d coach_example added, %d already present.\n", patched, skipped)
}
