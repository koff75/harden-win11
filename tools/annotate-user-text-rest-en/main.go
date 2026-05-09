// annotate-user-text-rest-en — Adds the English translations for the 66
// rules annotated by annotate-user-text-rest (the FR-only second wave).
// Runs only on rules that have user_today but NOT user_today_en.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ut struct{ Today, After, ForWho, Risk string }

var texts = map[string]ut{
	// ─── DEFENDER (8) ───
	"defender.ioav": {
		Today:  "Defender doesn't scan files downloaded via Internet Explorer / Outlook at download time.",
		After:  "Any file arriving via download is scanned immediately by Defender, before reaching disk.",
		ForWho: "Everyone.",
		Risk:   "None in normal use.",
	},
	"defender.script_scanning": {
		Today:  "Scripts (PowerShell, JS, VBS) running on your PC are not analyzed by Defender.",
		After:  "Defender intercepts and analyzes scripts in memory before they execute.",
		ForWho: "Everyone — modern fileless attack vector.",
		Risk:   "Very rare slowdown on large script suites (CI/CD builds).",
	},
	"defender.nis": {
		Today:  "No network connection inspection system to block known exploits.",
		After:  "Defender inspects network traffic in real-time and blocks packets matching known exploits.",
		ForWho: "Everyone, especially on public Wi-Fi.",
		Risk:   "Very rare false positive on legitimate traffic.",
	},
	"defender.pua": {
		Today:  "Potentially Unwanted Applications (toolbars, crapware, hidden miners) are not detected by default.",
		After:  "Defender automatically blocks downloads and installations of known PUAs.",
		ForWho: "Everyone — useful against installer bundles.",
		Risk:   "Almost none. Very rare case where a legitimate but aggressive app gets flagged.",
	},
	"defender.sample_submission": {
		Today:  "When Defender finds a suspicious but unknown file, it doesn't send it to Microsoft for deeper analysis.",
		After:  "Suspicious samples (without personal data) are sent to Microsoft for cloud analysis — Defender benefits from collaborative detection.",
		ForWho: "Everyone who wants up-to-date protection.",
		Risk:   "Microsoft receives samples (never your personal documents — only suspicious binaries). Skip if very paranoid.",
	},
	"defender.cloud_protection": {
		Today:  "Defender doesn't query the Microsoft cloud for detections (local-only mode).",
		After:  "Defender queries Microsoft's cloud database on every scan — much more up-to-date detection.",
		ForWho: "Everyone with internet.",
		Risk:   "Functionally none. Slight extra latency on some scans (negligible).",
	},
	"defender.signatures": {
		Today:  "No verification that Defender signatures are recently up to date.",
		After:  "We check that signatures were updated in the last 24h. If not, alert.",
		ForWho: "Everyone — Defender with 2-month-old signatures isn't very useful.",
		Risk:   "None.",
	},
	"defender.tamper_protection_check": {
		Today:  "No check that Tamper Protection (anti-disabling Defender by malware) is active.",
		After:  "We verify it's ON. If disabled, we alert (you must re-enable manually in Windows Security).",
		ForWho: "Everyone.",
		Risk:   "None (read-only check).",
	},

	// ─── FIREWALL (2) ───
	"firewall.block_smb_public": {
		Today:  "On public Wi-Fi (café, airport), port 445 (SMB file sharing) remains potentially accessible to other machines on the network.",
		After:  "Port 445 is explicitly blocked inbound on Public profiles — last-resort safety net if another rule accidentally opened it.",
		ForWho: "Everyone, especially when mobile.",
		Risk:   "None for standard use. Prevents SMB sharing on public Wi-Fi (unlikely use case).",
	},
	"firewall.block_netbios_public": {
		Today:  "On public Wi-Fi, NetBIOS ports (137-139) remain potentially accessible — classic lateral attack vector.",
		After:  "NetBIOS ports blocked inbound on Public profile.",
		ForWho: "Everyone, especially when mobile.",
		Risk:   "None. If you used NetBIOS on public Wi-Fi (never), you'd need an exception.",
	},

	// ─── SYSTEM SETTINGS (4) ───
	"system_settings.uac_enable_lua": {
		Today:  "UAC (User Account Control, the mechanism that asks 'do you want to allow this app to modify the system?') may be disabled.",
		After:  "UAC is forced enabled. Any program wanting admin must ask explicitly.",
		ForWho: "Everyone — basic barrier against malware that wants to install.",
		Risk:   "None. A few extra UAC prompts during software installs.",
	},
	"system_settings.uac_prompt_secure_desktop": {
		Today:  "When UAC shows its prompt, the normal desktop stays active behind it — a malware can try to send automated clicks.",
		After:  "UAC displays on an isolated 'secure desktop' — no other program can interact with it while you decide.",
		ForWho: "Everyone.",
		Risk:   "Very brief black flicker on each UAC. No functional impact.",
	},
	"system_settings.rdp_firewall_disable": {
		Today:  "Even if RDP service is disabled, firewall rules for port 3389 (Remote Desktop) may stay open.",
		After:  "RDP firewall rules are removed — port 3389 closed everywhere.",
		ForWho: "Everyone who doesn't use RDP. Consistent with rdp_disable.",
		Risk:   "If you re-enable RDP later, you'll need to re-add the firewall rules.",
	},
	"system_settings.fast_startup_off": {
		Today:  "Fast Startup (Windows 8+) keeps the kernel in hibernation for faster boot — but blocks some driver updates and complicates forensics.",
		After:  "Fast Startup disabled. Boot a bit slower (5-10s) but clean shutdown, drivers up to date on every start.",
		ForWho: "Maximal profile. On modern SSD, the boot difference is negligible.",
		Risk:   "Boot 5-10 seconds slower. No other impact.",
	},

	// ─── PRIVACY (10) ───
	"privacy.advertising_id_machine": {
		Today:  "Windows associates a unique advertising ID with your account — shared with Store apps for profiling.",
		After:  "Advertising ID disabled by machine policy for ALL users on the PC.",
		ForWho: "Everyone who dislikes tracking.",
		Risk:   "Some free Store apps may show less-targeted ads (i.e., a bit more random ads).",
	},
	"privacy.advertising_id_user": {
		Today:  "Your user account has a personal advertising ID used by Store apps.",
		After:  "Disabled for your current account.",
		ForWho: "Everyone — per-user equivalent of the machine-wide option.",
		Risk:   "Same as advertising_id_machine.",
	},
	"privacy.online_speech_off": {
		Today:  "Windows sends your voice (voice keyboard, dictation) to Microsoft for cloud-side transcription.",
		After:  "Online speech recognition disabled. If you use dictation, it stays local (more basic quality).",
		ForWho: "Everyone who doesn't dictate regularly.",
		Risk:   "If you use Windows voice dictation, quality degraded (local model only).",
	},
	"privacy.activity_history_off": {
		Today:  "Windows records your activity history (apps launched, files opened) locally and syncs to Microsoft cloud.",
		After:  "Activity history disabled. No more Timeline in Win+Tab.",
		ForWho: "Everyone who doesn't use Timeline.",
		Risk:   "No more Timeline (Win+Tab only shows currently open apps).",
	},
	"privacy.consumer_features_off": {
		Today:  "Windows automatically installs suggested apps (Candy Crush, Spotify, etc.) on new user logon.",
		After:  "Windows no longer auto-installs anything — you install what you want from the Store.",
		ForWho: "Everyone.",
		Risk:   "None. You can still manually install apps from Microsoft Store.",
	},
	"privacy.silent_apps_off": {
		Today:  "Windows can silently download suggested apps in the background.",
		After:  "No more silent downloads. You decide what to install.",
		ForWho: "Everyone.",
		Risk:   "None.",
	},
	"privacy.settings_suggestions_off": {
		Today:  "Settings shows 'try this' suggestions and proactive notifications.",
		After:  "Settings stays minimal and silent.",
		ForWho: "Everyone who dislikes noise.",
		Risk:   "You might miss a useful feature suggestion (rare).",
	},
	"privacy.start_suggestions_off": {
		Today:  "Start menu shows suggestions for Store apps you haven't installed.",
		After:  "No more ads in Start menu.",
		ForWho: "Everyone.",
		Risk:   "None.",
	},
	"privacy.tips_welcome_off": {
		Today:  "Windows shows 'tips' notifications and welcome screens after updates.",
		After:  "No more tips notifications, no more welcome screens after update.",
		ForWho: "Everyone.",
		Risk:   "None.",
	},
	"privacy.ink_text_collection_off": {
		Today:  "Windows collects your handwriting and keyboard input to train its personalized language models (sent to Microsoft).",
		After:  "No more collection of your input. No personalized keyboard learning.",
		ForWho: "Everyone.",
		Risk:   "Auto-corrections slightly less accurate over time (base model still functional).",
	},

	// ─── ASR (15) ───
	"asr.block_office_child_processes": {
		Today:  "Word/Excel/PowerPoint can launch cmd.exe, PowerShell or any program via macros — classic malware vector.",
		After:  "Blocks any child process creation from Office. A malicious macro can't launch Mimikatz, cmd, etc.",
		ForWho: "Everyone who opens Office files received by email.",
		Risk:   "If you have macros legitimately calling cmd/powershell (rare in personal use), authorize on a case-by-case basis.",
	},
	"asr.block_office_code_injection": {
		Today:  "Office can inject code into other processes — technique used by malware to bypass protections.",
		After:  "Blocks injection attempts. Office malware stays locked in its own process.",
		ForWho: "Everyone.",
		Risk:   "Very rare. Some old Office add-ins may be affected.",
	},
	"asr.block_office_comm_child_processes": {
		Today:  "Outlook can launch programs (cmd, scripts, etc.) — infection vector via booby-trapped email.",
		After:  "Outlook can no longer launch child processes. If you click a booby-trapped attachment by mistake, the chain breaks.",
		ForWho: "Everyone who uses Outlook.",
		Risk:   "Very rare. Some specific corporate workflows.",
	},
	"asr.block_win32_api_office_macros": {
		Today:  "Office VBA macros can call Win32 APIs directly — access to the entire machine.",
		After:  "Macros blocked from calling Win32 APIs. Damage capability strongly reduced.",
		ForWho: "Everyone who doesn't write expert VBA.",
		Risk:   "If you write professional VBA with Win32 APIs, your macros will break.",
	},
	"asr.block_email_executable_content": {
		Today:  "An attachment downloaded from your email can be executed directly (.exe, .bat, .ps1, .vbs...).",
		After:  "Executable content downloaded from email/webmail is blocked. You must explicitly save then run (and Defender scans them then).",
		ForWho: "Everyone.",
		Risk:   "Almost none. If you often download installers from your webmail, slight friction.",
	},
	"asr.block_obfuscated_scripts": {
		Today:  "Obfuscated scripts (Base64-encoded, with scrambled strings) can run without alert.",
		After:  "Defender detects obfuscation patterns and blocks execution.",
		ForWho: "Everyone.",
		Risk:   "Very rare false positive on legitimate but obfuscated scripts (anti-reverse-engineering, packers).",
	},
	"asr.block_js_vbs_launch": {
		Today:  "A downloaded .js or .vbs file can automatically download and launch a .exe (dropper technique).",
		After:  "Blocks .exe launching by downloaded JS/VBS. The attack chain breaks.",
		ForWho: "Everyone.",
		Risk:   "Almost none.",
	},
	"asr.block_unsigned_usb": {
		Today:  "If you plug in a USB stick, any unsigned .exe on it can run.",
		After:  "Defender blocks unsigned/untrusted processes launched from USB.",
		ForWho: "Business / maximal profile. Useful in environments where USB sticks are exchanged.",
		Risk:   "Your unsigned portable tools on USB will not work (Sysinternals signed OK; custom tools unsigned KO).",
	},
	"asr.block_wmi_persistence": {
		Today:  "WMI Event Subscription is a malware persistence technique (runs at boot without leaving traces in Run keys).",
		After:  "Defender blocks new WMI Event Subscription persistence.",
		ForWho: "Maximal profile. Legitimate WMI use rare in personal.",
		Risk:   "If you use WMI Event Subscription for system monitoring, breaks.",
	},
	"asr.block_psexec_wmi": {
		Today:  "PsExec and WMI can launch processes remotely — classic lateral movement techniques used by attackers.",
		After:  "Defender blocks process creation via PsExec and WMI.",
		ForWho: "Maximal profile. Useful unless you do network admin.",
		Risk:   "PsExec no longer works for remote troubleshooting. Classic Sysinternals tools affected.",
	},
	"asr.block_unprevalent_executables": {
		Today:  "Any uncommon .exe (never seen elsewhere in the world) can run without alert.",
		After:  "Rare or unsigned .exe files are blocked automatically. Defender only lets known apps through.",
		ForWho: "Maximal profile. Avoid if testing custom software.",
		Risk:   "Your internal apps / compiled scripts / custom tools will be blocked initially. Whitelist needed.",
	},
	"asr.block_adobe_reader_child": {
		Today:  "Adobe Reader can launch child processes — infection vector via booby-trapped PDF.",
		After:  "Adobe Reader can no longer launch other programs. If you open a malicious PDF, the chain breaks.",
		ForWho: "Everyone who opens unknown PDFs.",
		Risk:   "Very rare PDF workflow that calls an external program.",
	},
	"asr.block_vulnerable_drivers": {
		Today:  "Some Windows drivers signed but vulnerable (BYOVD = Bring Your Own Vulnerable Driver) can be exploited to escalate to kernel.",
		After:  "Defender blocks loading of the known vulnerable drivers list.",
		ForWho: "Everyone.",
		Risk:   "Very rare old hardware driver no longer working (replace with recent version).",
	},
	"asr.block_safe_mode_reboot": {
		Today:  "An attacker can reboot your PC into Safe Mode to disable Defender and do dirty work.",
		After:  "Defender blocks Safe Mode reboots forced by a program.",
		ForWho: "Business / maximal profile.",
		Risk:   "Forced Safe Mode reboot blocked (e.g., via msconfig). Affects troubleshooting workflows.",
	},
	"asr.block_impersonated_tools": {
		Today:  "A malware can copy cmd.exe or PowerShell under another name to evade detection.",
		After:  "Defender detects renamed/copied system tools and blocks their execution.",
		ForWho: "Everyone.",
		Risk:   "Very rare false positive if you use intentionally renamed Sysinternals tools.",
	},
	"asr.block_webshell_servers": {
		Today:  "On a server, a webshell (ASP, PHP) can be created via a web vulnerability for persistence.",
		After:  "Defender blocks webshell creation in web directories.",
		ForWho: "Windows servers. Useless on workstation but harmless.",
		Risk:   "None on a personal PC.",
	},
}

// Bloatware (27) — same template as FR.
type bloat struct{ ID, Name string }

var bloatwares = []bloat{
	{"bloatware.clipchamp", "Clipchamp (Microsoft video editor)"},
	{"bloatware.bing_news", "Bing News"},
	{"bloatware.bing_weather", "Bing Weather"},
	{"bloatware.get_help", "Get Help (Windows support assistant)"},
	{"bloatware.get_started", "Get Started (Windows tutorial)"},
	{"bloatware.solitaire", "Microsoft Solitaire Collection"},
	{"bloatware.mixed_reality", "Mixed Reality Portal (VR headset)"},
	{"bloatware.people", "People (address book)"},
	{"bloatware.skype", "Skype"},
	{"bloatware.feedback_hub", "Feedback Hub"},
	{"bloatware.your_phone", "Your Phone / Phone Link"},
	{"bloatware.zune_music", "Groove Music"},
	{"bloatware.zune_video", "Movies & TV"},
	{"bloatware.disney", "Disney+"},
	{"bloatware.tiktok", "TikTok"},
	{"bloatware.facebook", "Facebook"},
	{"bloatware.instagram", "Instagram"},
	{"bloatware.twitter", "Twitter / X"},
	{"bloatware.linkedin", "LinkedIn"},
	{"bloatware.netflix", "Netflix"},
	{"bloatware.candy_crush", "Candy Crush Saga"},
	{"bloatware.spotify_ab", "Spotify (Store version)"},
	{"bloatware.spotify", "Spotify"},
	{"bloatware.apple_music", "Apple Music"},
	{"bloatware.dolby_access", "Dolby Access"},
	{"bloatware.jimmylin", "JimmyLin (other preinstalled third-party apps)"},
	{"bloatware.pub_5319275a", "Publisher 5319275A app (preinstalled third-party apps)"},
}

func init() {
	for _, b := range bloatwares {
		texts[b.ID] = ut{
			Today:  fmt.Sprintf("The %s app is preinstalled or installed on your PC. Takes space and may run in background.", b.Name),
			After:  "The app is uninstalled. No more notifications, no more updates, no more background processes.",
			ForWho: fmt.Sprintf("Those who don't use %s.", b.Name),
			Risk:   "If you need it later, you can reinstall from the Microsoft Store. No data lost.",
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
			if strings.Contains(block, "user_today_en:") {
				skipped++
				continue
			}

			// Insert before `action:`
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
	fmt.Printf("\nDone. %d rules annotated EN, %d already done.\n", patched, skipped)
}
