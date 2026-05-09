// annotate-section-en — adds title_en + description_en to section meta
// of each manifest YAML. Idempotent.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type entry struct{ TitleEn, DescEn string }

var sections = map[string]entry{
	"defender":        {"Microsoft Defender", "Built-in Windows antivirus. We turn on every protection it offers."},
	"firewall":        {"Windows Firewall", "OS-level network protection — profiles + specific rules."},
	"accounts":        {"Local Accounts", "Disable rarely-used default accounts and rename the built-in Administrator."},
	"system_settings": {"UAC, RDP, Power", "User Account Control, Remote Desktop, hibernation — settings that make a real difference for everyday safety."},
	"network":         {"Network Protocol Hardening", "Cuts off legacy name-resolution protocols (LLMNR, mDNS, NetBIOS, WPAD) and enforces NTLMv2 + SMB signing."},
	"privacy":         {"Privacy & Telemetry", "Reduces the data Windows sends to Microsoft and turns off attention-stealing default features."},
	"asr":             {"Attack Surface Reduction (ASR)", "Defender rules that block common offensive behaviors (Office macros, LSASS dump, obfuscated scripts, USB, WMI, ransomware)."},
	"bloatware":       {"Microsoft Store Bloatware", "Removes preinstalled Store apps. You can pick individually which ones to keep."},
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

		// Find section.id
		idRe := regexp.MustCompile(`(?m)^section:\s*\n\s+id:\s*(\S+)\s*$`)
		m := idRe.FindStringSubmatch(content)
		if m == nil {
			return nil
		}
		secID := m[1]
		entry, ok := sections[secID]
		if !ok {
			return nil
		}

		if strings.Contains(content, "title_en:") {
			skipped++
			return nil
		}

		// Insert after `description:` of the section header (first occurrence,
		// inside section: block).
		descRe := regexp.MustCompile(`(?m)^(\s+description:\s*"[^"]*"\s*\n)`)
		loc := descRe.FindStringSubmatchIndex(content)
		if loc == nil {
			return nil
		}
		indent := "  "
		insertion := indent + `title_en: "` + esc(entry.TitleEn) + "\"\n" +
			indent + `description_en: "` + esc(entry.DescEn) + "\"\n"
		content = content[:loc[1]] + insertion + content[loc[1]:]

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
		fmt.Printf("patched: %s (section=%s)\n", path, secID)
		patched++
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walk: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("\nDone. %d sections annotated, %d already done.\n", patched, skipped)
}

func esc(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
