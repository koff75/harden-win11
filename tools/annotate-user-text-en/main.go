// annotate-user-text-en — Adds English versions of user_* fields next to
// the existing French ones in the YAML manifests.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type userTextEn struct {
	Today  string
	After  string
	ForWho string
	Risk   string
}

var texts = map[string]userTextEn{
	// ─── DEFENDER ───
	"defender.realtime": {
		Today:  "When you download or open a file, nothing scans it for viruses.",
		After:  "Every file is analyzed in the background as soon as you touch it. Known threats are blocked immediately.",
		ForWho: "Everyone — this is Windows' built-in antivirus.",
		Risk:   "Almost none. Very rare slowdown when opening huge files (>1 GB) or ZIPs with thousands of files.",
	},
	"defender.behavior_monitoring": {
		Today:  "Defender only checks file signatures (i.e., previously known viruses).",
		After:  "Defender also monitors the behavior of running programs. Detects new malware that hides in memory.",
		ForWho: "Everyone, especially to defend against recent threats not yet in antivirus databases.",
		Risk:   "None in normal use.",
	},
	"defender.network_protection": {
		Today:  "If you click a malicious link, your browser loads the page (which can try to infect you).",
		After:  "Defender blocks the connection to known-malicious sites BEFORE your browser loads anything.",
		ForWho: "Everyone who browses the web.",
		Risk:   "Very rare false positive on a legitimate but newly registered site. You can manually unblock if needed.",
	},
	"defender.controlled_folder_access": {
		Today:  "Any program can write to your Documents, Pictures, Desktop folders (= attack surface for ransomware).",
		After:  "Only authorized apps (Word, Photoshop, etc.) can write there. A ransomware gets an access error and can't encrypt your files.",
		ForWho: "Strong defense against ransomware. More of a 'paranoid' setting because it requires some configuration.",
		Risk:   "You'll need to manually authorize some legitimate apps at first (Steam, OBS, Visual Studio, games). A few minutes of setup the first days.",
	},

	// ─── FIREWALL ───
	"firewall.profile_public": {
		Today:  "When you connect to a café or airport Wi-Fi, other computers on the network can try to reach your PC.",
		After:  "On these untrusted networks, your PC becomes invisible — nobody can connect to it.",
		ForWho: "Essential for laptops that leave home. Useless on a desktop that never moves.",
		Risk:   "If you do LAN gaming or file sharing in a café, it'll block. Otherwise nothing.",
	},
	"firewall.profile_private": {
		Today:  "On your home Wi-Fi (marked 'private'), other devices can connect to your PC.",
		After:  "Stays accessible to your home machines but blocks any unusual connection attempt.",
		ForWho: "Everyone with a home PC.",
		Risk:   "None for normal use. If you host a local service (NAS, game server), you'll need to add an exception.",
	},
	"firewall.profile_domain": {
		Today:  "On a corporate PC (domain-joined), the firewall can be more permissive.",
		After:  "Forces default-deny even on the corporate network.",
		ForWho: "Corporate workstations (Active Directory domain).",
		Risk:   "If your network admin has tools that need inbound access, you'll need explicit rules.",
	},

	// ─── ACCOUNTS ───
	"accounts.disable_unused": {
		Today:  "Several default system accounts exist on Windows (Administrator, Guest…), often enabled for no reason.",
		After:  "These unused accounts are disabled. An attacker can't try passwords on them.",
		ForWho: "Everyone.",
		Risk:   "None for 99% of users. If you explicitly use the 'Guest' account (very rare), don't apply this.",
	},
	"accounts.rename_admin": {
		Today:  "The super-administrator account is named 'Administrator' — a name universally known by automated attackers.",
		After:  "Renamed to 'AdminLocal_<YourPC>'. A bot trying 1000 passwords on 'Administrator' no longer finds the account.",
		ForWho: "Personal or solo PC. Avoid in corporate environments (GPO can override the rename).",
		Risk:   "If you have scripts that explicitly call 'Administrator', they'll break. Very rare in personal use.",
	},

	// ─── UAC / RDP / POWER ───
	"system_settings.uac_consent_admin": {
		Today:  "When a program requests admin rights, Windows can accept automatically (without asking you).",
		After:  "You must explicitly approve every elevation request. Malware can no longer slip through silently.",
		ForWho: "Everyone, unless you do dev/admin work and one UAC popup per day annoys you.",
		Risk:   "A few extra UAC prompts when installing software. That's the goal: see what's asking to elevate.",
	},
	"system_settings.uac_deny_user_elevation": {
		Today:  "A standard account can request to become admin (by entering an admin password).",
		After:  "Standard accounts can no longer elevate at all. You must log in with the admin account directly.",
		ForWho: "Multi-account setups (parents/kids, family). Avoid on solo personal PC (= your only account is admin).",
		Risk:   "On solo personal PC: none. On shared PC: standard accounts can't install software anymore.",
	},
	"system_settings.rdp_disable": {
		Today:  "Remote Desktop (RDP) might be enabled on your PC. If so, it's a known entry point for attackers.",
		After:  "RDP is off. No one can connect to your PC remotely through this channel anymore.",
		ForWho: "Everyone, unless you use RDP to remote into your PC from elsewhere.",
		Risk:   "If you remote into your PC from somewhere else (e.g., from work), you won't be able to anymore.",
	},
	"system_settings.hibernate_off": {
		Today:  "When you put your PC into hibernation, Windows writes memory contents (= secrets, passwords in use) to disk.",
		After:  "Hibernation is disabled. Either you Sleep (no disk copy) or shut down.",
		ForWho: "Mainly desktop PCs. Avoid on laptops where hibernation saves battery.",
		Risk:   "On laptop: you lose long hibernation autonomy. Regular sleep consumes a bit more.",
	},

	// ─── NETWORK ───
	"network.llmnr_disable": {
		Today:  "When your PC looks for a machine name on the network, it shouts the question to everyone on the network.",
		After:  "It only uses standard DNS now. Nobody on the network can intercept these lookups.",
		ForWho: "Everyone. Classic attack vector on public Wi-Fi.",
		Risk:   "On a home network without correctly configured local DNS, some auto-discoveries may stop working (rare).",
	},
	"network.mdns_disable": {
		Today:  "Your PC does Apple-Bonjour-style auto-discovery on the network (cast to TV, AirPrint, etc.).",
		After:  "No more such auto-discovery. More discreet on the network.",
		ForWho: "'Maximal' (paranoid) profile. Avoid at home if you use Chromecast/AirPlay/Apple printers.",
		Risk:   "No more Chromecast, no more AirPrint. If you cast to a TV regularly, don't enable this.",
	},
	"network.netbios_off": {
		Today:  "An old Windows naming protocol (NetBIOS) is still enabled. Often used in attacks on public Wi-Fi.",
		After:  "Disabled. Your PC no longer answers NetBIOS questions from a local attacker.",
		ForWho: "Everyone, especially when on the move.",
		Risk:   "If you access a network share via short name (e.g., \\\\HOME-NAS), you'll need to use the IP or full name.",
	},
	"network.wpad_disable": {
		Today:  "On startup, your PC automatically searches for a web proxy. A local attacker can answer and become your middleman.",
		After:  "No more auto-discovery of proxy. If you use one, you configure it explicitly.",
		ForWho: "Everyone, especially when mobile (public Wi-Fi).",
		Risk:   "If your company uses an auto WPAD proxy, web access may break (rare situation in personal use).",
	},
	"network.smbv1_disable": {
		Today:  "A 30-year-old file sharing system is still enabled. Used by WannaCry and NotPetya ransomware to spread.",
		After:  "This old system is off. Your PC only speaks modern (secure) versions.",
		ForWho: "Everyone, unless you access a server or NAS from before 2012 you don't want to replace.",
		Risk:   "If a network share stops working, it was on SMBv1. Update the NAS or make an exception.",
	},
	"network.smb_client_signing": {
		Today:  "When you access a network share, packets aren't signed. Someone on the network could modify them.",
		After:  "Packets are signed on both ends. Any tampering is detected and rejected.",
		ForWho: "Everyone using network shares (NAS, Windows shares).",
		Risk:   "Rare compatibility issue with very old NAS that don't support signing.",
	},
	"network.smb_server_signing": {
		Today:  "If your PC shares files, outgoing packets aren't signed.",
		After:  "Your shares enforce signing. Nobody can intercept and modify without detection.",
		ForWho: "Everyone who shares folders from their PC.",
		Risk:   "Same as client signing: very rare issue with old clients.",
	},
	"network.smb_guest_auth_off": {
		Today:  "Your PC can connect to shares without password ('guest' mode). A local attacker can run a fake server.",
		After:  "Any password-less share is refused. Explicit authentication required.",
		ForWho: "Everyone.",
		Risk:   "If you have a NAS configured in guest mode (rare), you'll need to switch it to authenticated mode.",
	},
	"network.ntlm_v2_only": {
		Today:  "Your PC can be forced to speak an old authentication version (NTLMv1) which cracks in hours.",
		After:  "Refuses any downgrade negotiation. Stays at NTLMv2 minimum (resistant).",
		ForWho: "Everyone, especially in corporate settings.",
		Risk:   "Very rare NTLMv1-only old servers will stop working.",
	},

	// ─── PRIVACY ───
	"privacy.recall_off": {
		Today:  "Recall (Win11 24H2) takes a screenshot of your screen every few seconds and stores everything locally with OCR.",
		After:  "Recall is fully off. No more automatic screen captures.",
		ForWho: "Everyone on Win11 24H2+.",
		Risk:   "None. Recall is still experimental, many users prefer it off.",
	},
	"privacy.cortana_off": {
		Today:  "Cortana is enabled — Microsoft's voice assistant that collects your searches, calendar, voice.",
		After:  "Cortana is disabled. The Windows search bar still works for local files.",
		ForWho: "Everyone who doesn't use Cortana (= 99% of people).",
		Risk:   "If you use Cortana for voice reminders, don't enable this.",
	},
	"privacy.telemetry_required": {
		Today:  "Windows sends a lot of info to Microsoft (apps launched, frequency, crashes). Defaults to 'Enhanced' on Pro.",
		After:  "Reduced to minimum required (just critical crash reports for Microsoft).",
		ForWho: "Everyone who wants to limit what leaves their PC for Microsoft.",
		Risk:   "Functionally none. Microsoft has slightly fewer signals to improve Windows in the future.",
	},

	// ─── ASR ───
	"asr.block_lsass_credential_theft": {
		Today:  "If an attacker runs code on your PC, they can extract your Windows password from authentication memory (Mimikatz).",
		After:  "Defender blocks any access to authentication memory. Your password stays protected even if step 1 of an attack succeeds.",
		ForWho: "Everyone.",
		Risk:   "Almost none. One known case: very specific corporate management tools may be blocked (rarely in personal use).",
	},
	"asr.block_office_executable_content": {
		Today:  "Word and Excel can open a file containing a hidden program and launch it (classic malware-by-email vector).",
		After:  "If you click on a fake invoice by mistake, the hidden program can't start. The threat is neutralized at the source.",
		ForWho: "Everyone who receives PDF/DOCX files via email.",
		Risk:   "If you program in VBA with macros that call .exe files (very rare in personal use), you'll need to authorize macros case by case.",
	},
	"asr.advanced_ransomware_protection": {
		Today:  "No specific analysis of typical ransomware behavior (massive fast encryption).",
		After:  "Defender detects mass encryption patterns and blocks the process before it touches all your files.",
		ForWho: "Everyone.",
		Risk:   "Very rare. If you use a legitimate encryption tool (VeraCrypt in batch mode), it could be flagged. Whitelist available.",
	},
	"asr.block_unprevalent_executables": {
		Today:  "Any unusual .exe (never seen elsewhere) can run on your PC without alert.",
		After:  "Rare or unsigned .exe files are blocked automatically. Defender only lets known apps through.",
		ForWho: "Maximal (paranoid) profile. Avoid if you test custom software.",
		Risk:   "Your internal apps / compiled scripts / custom dev tools will be blocked at first. You need to whitelist.",
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
			if strings.Contains(block, "user_today_en:") {
				skipped++
				continue
			}

			// Insertion juste avant `action:`.
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
			insertion := indent + `user_today_en: "` + esc(txt.Today) + "\"\n" +
				indent + `user_after_en: "` + esc(txt.After) + "\"\n" +
				indent + `user_for_who_en: "` + esc(txt.ForWho) + "\"\n" +
				indent + `user_risk_en: "` + esc(txt.Risk) + "\"\n"

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
	fmt.Printf("\nDone. %d rules annotated EN, %d already present (skip).\n", patched, skipped)
}
