// fix-mojibake — Répare les caractères UTF-8 corrompus (latin-1-as-UTF-8)
// dans les fichiers PowerShell du dossier engine/.
//
// Le pattern est typique d'un copier/coller via PowerShell qui a réinterprété
// du UTF-8 comme du CP1252 puis ré-encodé en UTF-8. On obtient des séquences
// double-encodées (ex: "é" → "Ã©" → "ÃƒÂ©").
//
// Pourquoi en Go : le même outil écrit en PowerShell se mojibake lui-même
// quand on le sauve via Set-Content / Out-File (le shell dépend de la code page).
// Go lit/écrit en bytes UTF-8 sans interprétation, donc neutre.
//
// Usage :
//   go run ./tools/fix-mojibake          (depuis la racine du repo)
//   go run ./tools/fix-mojibake -dry-run  (montre les fichiers sans modifier)

package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Patterns de remplacement, ordre IMPORTANT : double-encoding d'abord
// (sinon on remplace les composants avant la séquence complète).
var replacements = []struct{ from, to string }{
	// Double-encoding : "é" → "Ã©" → "ÃƒÂ©"
	{"ÃƒÂ¨", "è"},
	{"ÃƒÂ©", "é"},
	{"ÃƒÂª", "ê"},
	{"ÃƒÂ«", "ë"},
	{"ÃƒÂ¢", "â"},
	{"ÃƒÂ´", "ô"},
	{"ÃƒÂ®", "î"},
	{"ÃƒÂ¯", "ï"},
	{"ÃƒÂ§", "ç"},
	{"ÃƒÂ¹", "ù"},
	{"ÃƒÂ»", "û"},
	{"ÃƒÂ¦", "æ"},
	{"ÃƒÂ‰", "É"},
	{"Ãƒ‰", "É"},
	// Simple encoding
	{"Ã©", "é"},
	{"Ã¨", "è"},
	{"Ãª", "ê"},
	{"Ã«", "ë"},
	{"Ã¢", "â"},
	{"Ã´", "ô"},
	{"Ã®", "î"},
	{"Ã¯", "ï"},
	{"Ã§", "ç"},
	{"Ã¹", "ù"},
	{"Ã»", "û"},
	{"Ã¦", "æ"},
	{"Ãœ", "Ü"},
	{"Â´", "´"},
}

// "Ã " = "à" (le space après Ã est intentionnel) — on le traite à part
// car le replace global pourrait toucher des séquences légitimes.
const aGraveMojibake = "Ã " // U+00C3 + NBSP — c'est ce que produit le double encode pour "à"

func main() {
	var (
		dryRun bool
		root   string
	)
	flag.BoolVar(&dryRun, "dry-run", false, "ne modifie pas les fichiers, montre juste ce qui serait patché")
	flag.StringVar(&root, "root", "engine", "dossier à scanner (récursif)")
	flag.Parse()

	if !filepath.IsAbs(root) {
		wd, _ := os.Getwd()
		root = filepath.Join(wd, root)
	}

	patched := 0
	scanned := 0
	var errors []string

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ps1" && ext != ".psm1" && ext != ".yaml" && ext != ".yml" {
			return nil
		}
		// Skip *.tests.ps1 — ils sont écrits en code propre et n'ont pas le bug.
		if strings.HasSuffix(strings.ToLower(d.Name()), ".tests.ps1") {
			return nil
		}
		scanned++

		raw, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: read: %v", path, err))
			return nil
		}
		// Strip UTF-8 BOM si présent.
		stripped := raw
		if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
			stripped = raw[3:]
		}

		content := string(stripped)
		original := content
		for _, r := range replacements {
			content = strings.ReplaceAll(content, r.from, r.to)
		}
		// Cas spécial "à" : le double-encoding produit "ÃƒÂ " (Â + NBSP).
		// Un peu plus subtil que les autres.
		content = strings.ReplaceAll(content, "ÃƒÂ ", "à")
		content = strings.ReplaceAll(content, aGraveMojibake, "à")
		// Le simple "Ã " (Ã + space U+0020) est rare en bon UTF-8 → on patch.
		content = strings.ReplaceAll(content, "Ã ", "à ")

		if content == original {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		if dryRun {
			fmt.Printf("[would-fix] %s\n", rel)
		} else {
			// UTF-8 sans BOM, line endings préservés.
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				errors = append(errors, fmt.Sprintf("%s: write: %v", path, err))
				return nil
			}
			fmt.Printf("[fixed] %s\n", rel)
		}
		patched++
		return nil
	})
	if walkErr != nil {
		fmt.Fprintf(os.Stderr, "walk error: %v\n", walkErr)
		os.Exit(2)
	}

	fmt.Println()
	fmt.Printf("Scanned: %d  Patched: %d  Errors: %d\n", scanned, patched, len(errors))
	for _, e := range errors {
		fmt.Fprintln(os.Stderr, "  ", e)
	}
	if dryRun && patched > 0 {
		fmt.Println("(dry-run : aucun fichier modifié)")
	}
}
