// audit-i18n — Static audit of i18n consistency :
//   - Every t('...') key referenced in app.js exists in i18n.js (FR + EN)
//   - Every data-i18n="..." in index.html resolves to a key in i18n.js
//   - Every i18n.js key is used somewhere (no dead key)
//   - Every YAML rule has user_today / user_after / user_for_who / user_risk
//     AND their _en variants (95 rules × 8 fields each)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	bugs := 0

	// 1. t('...') keys in app.js
	appJS := mustRead("cmd/harden-gui/frontend/app.js")
	tCalls := regexp.MustCompile(`\bt\(\s*['"]([^'"]+)['"]\s*[,)]`).FindAllStringSubmatch(appJS, -1)
	usedKeys := map[string]bool{}
	for _, m := range tCalls {
		usedKeys[m[1]] = true
	}

	// 2. data-i18n in index.html
	indexHTML := mustRead("cmd/harden-gui/frontend/index.html")
	dataKeys := regexp.MustCompile(`data-i18n="([^"]+)"`).FindAllStringSubmatch(indexHTML, -1)
	for _, m := range dataKeys {
		usedKeys[m[1]] = true
	}

	// 3. i18n.js keys (FR + EN)
	i18nJS := mustRead("cmd/harden-gui/frontend/i18n.js")
	frBlock, enBlock := splitFrEn(i18nJS)
	frKeys := extractKeys(frBlock)
	enKeys := extractKeys(enBlock)

	// Audit 1 : key utilisée mais pas définie
	missingFr := []string{}
	missingEn := []string{}
	for k := range usedKeys {
		if !frKeys[k] {
			missingFr = append(missingFr, k)
		}
		if !enKeys[k] {
			missingEn = append(missingEn, k)
		}
	}
	for _, k := range missingFr {
		fmt.Printf("[BUG] key '%s' used but missing in FR dict\n", k)
		bugs++
	}
	for _, k := range missingEn {
		fmt.Printf("[BUG] key '%s' used but missing in EN dict\n", k)
		bugs++
	}

	// Audit 2 : key définie mais jamais utilisée (dead code)
	deadKeys := []string{}
	for k := range frKeys {
		if !usedKeys[k] {
			deadKeys = append(deadKeys, k)
		}
	}
	for _, k := range deadKeys {
		fmt.Printf("[WARN] key '%s' defined in FR but never used\n", k)
		// pas un bug bloquant
	}

	// Audit 3 : parité FR ↔ EN
	for k := range frKeys {
		if !enKeys[k] {
			fmt.Printf("[BUG] key '%s' in FR but missing in EN\n", k)
			bugs++
		}
	}
	for k := range enKeys {
		if !frKeys[k] {
			fmt.Printf("[BUG] key '%s' in EN but missing in FR\n", k)
			bugs++
		}
	}

	// Audit 4 : user_*_en sur tous les rules YAML
	manifestErrors := auditManifests("manifests")
	for _, e := range manifestErrors {
		fmt.Printf("[BUG] %s\n", e)
		bugs++
	}

	fmt.Println()
	fmt.Printf("=== Summary ===\n")
	fmt.Printf("  t() calls: %d\n", len(tCalls))
	fmt.Printf("  data-i18n: %d\n", len(dataKeys))
	fmt.Printf("  used keys: %d\n", len(usedKeys))
	fmt.Printf("  fr keys defined: %d\n", len(frKeys))
	fmt.Printf("  en keys defined: %d\n", len(enKeys))
	fmt.Printf("  dead keys (defined unused): %d\n", len(deadKeys))
	fmt.Printf("  bugs: %d\n", bugs)
	if bugs > 0 {
		os.Exit(1)
	}
}

func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(2)
	}
	return string(b)
}

func splitFrEn(src string) (string, string) {
	frStart := strings.Index(src, "fr:")
	enStart := strings.Index(src, "en:")
	if frStart < 0 || enStart < 0 {
		return "", ""
	}
	return src[frStart:enStart], src[enStart:]
}

func extractKeys(block string) map[string]bool {
	out := map[string]bool{}
	matches := regexp.MustCompile(`'([a-zA-Z][\w.]*)':`).FindAllStringSubmatch(block, -1)
	for _, m := range matches {
		out[m[1]] = true
	}
	return out
}

func auditManifests(root string) []string {
	out := []string{}
	requiredFields := []string{"user_today:", "user_after:", "user_for_who:", "user_risk:",
		"user_today_en:", "user_after_en:", "user_for_who_en:", "user_risk_en:"}

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		raw, _ := os.ReadFile(path)
		content := string(raw)

		// Trouver tous les `- id: <id>` blocks
		ruleStart := regexp.MustCompile(`(?m)^(\s+)- id:\s+(\S+)\s*$`)
		matches := ruleStart.FindAllStringSubmatchIndex(content, -1)

		for i, m := range matches {
			ruleID := content[m[4]:m[5]]
			blockStart := m[1]
			var blockEnd int
			if i+1 < len(matches) {
				blockEnd = matches[i+1][0]
			} else {
				blockEnd = len(content)
			}
			block := content[blockStart:blockEnd]

			for _, field := range requiredFields {
				if !strings.Contains(block, field) {
					out = append(out, fmt.Sprintf("rule %s in %s : missing %s",
						ruleID, filepath.Base(path), field))
				}
			}
		}
		return nil
	})
	return out
}
