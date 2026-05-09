// add-utf8-bom — Ajoute un BOM UTF-8 (EF BB BF) au début d'un fichier
// pour que PowerShell 5.1 le lise correctement en UTF-8 (au lieu de CP1252
// par défaut). Idempotent : ne fait rien si le BOM est déjà là.
//
// Usage : go run ./tools/add-utf8-bom <file1> <file2> ...

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: add-utf8-bom <file>...")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			continue
		}
		if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
			fmt.Printf("%s already has BOM, skip\n", path)
			continue
		}
		out := append([]byte{0xEF, 0xBB, 0xBF}, raw...)
		if err := os.WriteFile(path, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			continue
		}
		fmt.Printf("BOM added: %s\n", path)
	}
}
