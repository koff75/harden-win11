# SP1 Walking Skeleton — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Construire un PoC end-to-end du moteur v2 — un binaire `harden-engine.exe` qui parse un manifest YAML, valide contre JSONSchema, exécute un dry-run sur **une seule règle migrée** (`defender.realtime`), et émet de la sortie NDJSON conforme au spec. Preuve que l'architecture Go + PowerShell tient la route avant d'investir dans la migration des 50 règles.

**Architecture:** Go 1.26 (module à la racine `github.com/koff75/harden-win11`) avec `pkg/engine/` (lib) et `cmd/harden-engine/` (CLI). Cobra pour la CLI, `yaml.v3` pour YAML, `santhosh-tekuri/jsonschema/v6` pour validation. Snippets PowerShell exécutés via `os/exec` avec JSON sur stdin/stdout. Pester 5 pour les tests des snippets, Go test pour la lib.

**Tech Stack:** Go 1.26 (windows/arm64), PowerShell 5.1+, Pester 5.x (à installer), Cobra v1.8, yaml.v3, jsonschema/v6.

**Cible de complétion :** un PoC qui répond à `harden-engine apply --dry-run --section defender` en émettant des events NDJSON valides pour la règle `defender.realtime`. Tout le reste (autres règles, journal réel, restore point, undo, GUI) est **hors scope** de ce plan — voir Plan 2 et Plan 3.

---

## Prerequisites

À faire **avant** d'attaquer la Task 1 :

- [ ] **Pré-req 1 : Vérifier Go 1.26+**

```powershell
go version
# Attendu : go version go1.26.0 windows/arm64 (ou plus récent)
```

- [ ] **Pré-req 2 : Installer Pester 5**

```powershell
# La version embarquée Win11 est Pester 3.4 (très ancienne). On veut Pester 5.
Install-Module -Name Pester -RequiredVersion 5.7.1 -Force -SkipPublisherCheck -Scope CurrentUser
Get-Module -ListAvailable Pester | Sort-Object Version -Descending | Select-Object -First 1
# Attendu : Version 5.7.1 (ou plus récent dans la branche 5.x)
```

- [ ] **Pré-req 3 : Vérifier que PowerShell sait faire JSON in/out**

```powershell
'{"foo":42}' | ConvertFrom-Json | ConvertTo-Json -Compress
# Attendu : {"foo":42}
```

---

## File Structure

Vue d'ensemble des fichiers créés par ce plan (relatifs à la racine du repo) :

```
harden-win11/
├── go.mod                                          [Task 1]
├── go.sum                                          [Task 1, auto-géré]
├── .gitignore                                      [Task 1, modifié]
├── cmd/
│   └── harden-engine/
│       ├── main.go                                 [Task 7]
│       └── main_test.go                            [Task 7]
├── pkg/
│   └── engine/
│       ├── ndjson/
│       │   ├── writer.go                           [Task 2]
│       │   └── writer_test.go                      [Task 2]
│       ├── runner/
│       │   ├── runner.go                           [Task 3]
│       │   └── runner_test.go                      [Task 3]
│       ├── manifest/
│       │   ├── types.go                            [Task 4]
│       │   ├── loader.go                           [Task 4]
│       │   ├── loader_test.go                      [Task 4]
│       │   ├── validator.go                        [Task 5]
│       │   └── validator_test.go                   [Task 5]
│       └── dryrun/
│           ├── dryrun.go                           [Task 9]
│           └── dryrun_test.go                      [Task 9]
├── schemas/
│   └── manifest.schema.json                        [Task 5]
├── manifests/
│   └── 01-defender.yaml                            [Task 6]
├── engine/
│   └── actions/
│       └── defender/
│           ├── realtime.action.ps1                 [Task 6]
│           ├── realtime.test.ps1                   [Task 6]
│           ├── realtime.undo.ps1                   [Task 6]
│           └── realtime.tests.ps1                  [Task 8]
└── docs/
    └── DEVELOPING.md                               [Task 10]
```

**Décision : on ne touche pas à `Harden-Win11.ps1` v1.** Il reste à la racine, fonctionnel. Plus tard (Plan 3 ou similaire) il sera déplacé dans `legacy/`. Pour l'instant, on construit la v2 à côté sans rien casser.

---

## Tasks

### Task 1 : Bootstrap Go module + structure répertoires

**Files:**
- Create: `go.mod`
- Modify: `.gitignore`
- Create: `cmd/harden-engine/` (répertoire)
- Create: `pkg/engine/` (répertoire)

- [ ] **Step 1.1 : Initialiser le module Go**

```powershell
go mod init github.com/koff75/harden-win11
```

Attendu : crée `go.mod` avec :
```
module github.com/koff75/harden-win11

go 1.26
```

- [ ] **Step 1.2 : Mettre à jour `.gitignore`**

Lire `.gitignore` actuel puis ajouter à la fin :

```gitignore
# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.sum
/dist/
/build/
__debug_bin*

# IDE
.vscode/
.idea/
*.swp
*.swo
```

Note : on commit pas `go.sum` au début pour éviter les diffs inutiles pendant le bootstrap. À reactiver plus tard quand on stabilise les deps.

- [ ] **Step 1.3 : Créer la structure de répertoires**

```powershell
New-Item -ItemType Directory -Force -Path `
    cmd/harden-engine, `
    pkg/engine/ndjson, `
    pkg/engine/runner, `
    pkg/engine/manifest, `
    pkg/engine/dryrun, `
    schemas, `
    manifests, `
    engine/actions/defender | Out-Null

# Vérifier
Get-ChildItem -Recurse -Directory -Depth 3 | Select-Object FullName
```

Attendu : tous les répertoires listés existent.

- [ ] **Step 1.4 : Test smoke "Hello, world" en Go pour vérifier la toolchain**

Créer `cmd/harden-engine/main.go` (sera réécrit en Task 7) :

```go
package main

import "fmt"

func main() {
	fmt.Println("harden-engine bootstrap OK")
}
```

Build :

```powershell
go build -o dist/harden-engine.exe ./cmd/harden-engine
.\dist\harden-engine.exe
```

Attendu : `harden-engine bootstrap OK`

- [ ] **Step 1.5 : Commit**

```powershell
git add go.mod .gitignore cmd/harden-engine/main.go
git commit -m "chore(v2): bootstrap Go module + directory structure"
```

---

### Task 2 : NDJSON writer

**Files:**
- Create: `pkg/engine/ndjson/writer.go`
- Test: `pkg/engine/ndjson/writer_test.go`

Ce module sérialise des events JSON ligne par ligne sur un `io.Writer` (typiquement `os.Stdout`). Foundation pour tout ce qui suit.

- [ ] **Step 2.1 : Écrire le test qui échoue**

Créer `pkg/engine/ndjson/writer_test.go` :

```go
package ndjson

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriter_Emit_SingleEvent(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	event := map[string]any{
		"type":    "run_start",
		"run_id":  "2026-05-07T14-23-00",
		"dry_run": false,
	}

	if err := w.Emit(event); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	got := buf.String()
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline, got %q", got)
	}
	// JSON content (any field order is OK, json.Marshal sorts map keys deterministically since Go 1.12)
	if !strings.Contains(got, `"type":"run_start"`) {
		t.Errorf("expected type=run_start in output, got %q", got)
	}
	if !strings.Contains(got, `"dry_run":false`) {
		t.Errorf("expected dry_run=false in output, got %q", got)
	}
}

func TestWriter_Emit_MultipleEvents_OnePerLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	_ = w.Emit(map[string]any{"type": "a"})
	_ = w.Emit(map[string]any{"type": "b"})
	_ = w.Emit(map[string]any{"type": "c"})

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d : %q", len(lines), buf.String())
	}
}
```

- [ ] **Step 2.2 : Lancer le test pour vérifier qu'il échoue**

```powershell
go test ./pkg/engine/ndjson/...
```

Attendu : FAIL avec `undefined: NewWriter` (le code n'existe pas encore).

- [ ] **Step 2.3 : Implémenter le writer minimal**

Créer `pkg/engine/ndjson/writer.go` :

```go
// Package ndjson écrit des events NDJSON (Newline Delimited JSON) sur un io.Writer.
// Utilisé pour la sortie streaming du moteur, consommée par la GUI ou par jq.
package ndjson

import (
	"encoding/json"
	"io"
)

// Writer émet des events JSON ligne par ligne.
type Writer struct {
	w io.Writer
}

// NewWriter retourne un Writer qui écrit sur w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Emit sérialise event en JSON compact suivi d'un \n.
// Retourne la première erreur rencontrée (Marshal ou Write).
func (w *Writer) Emit(event any) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := w.w.Write(b); err != nil {
		return err
	}
	_, err = w.w.Write([]byte{'\n'})
	return err
}
```

- [ ] **Step 2.4 : Lancer le test pour vérifier qu'il passe**

```powershell
go test ./pkg/engine/ndjson/... -v
```

Attendu : `--- PASS: TestWriter_Emit_SingleEvent` et `--- PASS: TestWriter_Emit_MultipleEvents_OnePerLine`.

- [ ] **Step 2.5 : Commit**

```powershell
git add pkg/engine/ndjson/
git commit -m "feat(v2/engine): NDJSON event writer"
```

---

### Task 3 : PowerShell runner

**Files:**
- Create: `pkg/engine/runner/runner.go`
- Test: `pkg/engine/runner/runner_test.go`
- Create: `pkg/engine/runner/testdata/echo.ps1` (fixture pour les tests)

Le runner est responsable d'exécuter un snippet PowerShell avec un JSON en entrée (stdin) et de récupérer le JSON en sortie (stdout). C'est le morceau le plus risqué techniquement — si ça merde, rien ne marche.

- [ ] **Step 3.1 : Créer la fixture PS pour les tests**

Créer `pkg/engine/runner/testdata/echo.ps1` :

```powershell
# Fixture de test : lit un JSON sur stdin, le renvoie inchangé sur stdout
# avec un champ "echoed":true ajouté.
$inputJson = [Console]::In.ReadToEnd()
$obj = if ($inputJson.Trim()) { $inputJson | ConvertFrom-Json -AsHashtable } else { @{} }
$obj.echoed = $true
$obj | ConvertTo-Json -Compress -Depth 10
```

Note : `ConvertFrom-Json -AsHashtable` (PowerShell 6+) ne marche pas sur PS 5.1. **Le PS embarqué Win11 est PS 5.1**. Adapter :

```powershell
# Fixture de test compatible PS 5.1
$inputJson = [Console]::In.ReadToEnd()
if ($inputJson.Trim()) {
    $obj = $inputJson | ConvertFrom-Json
    # Convertir PSCustomObject en hashtable (5.1 friendly)
    $hash = @{}
    $obj.PSObject.Properties | ForEach-Object { $hash[$_.Name] = $_.Value }
} else {
    $hash = @{}
}
$hash.echoed = $true
$hash | ConvertTo-Json -Compress -Depth 10
```

- [ ] **Step 3.2 : Écrire le test qui échoue**

Créer `pkg/engine/runner/runner_test.go` :

```go
package runner

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestRunner_RunPS_EchoFixture(t *testing.T) {
	r := New()
	scriptPath, _ := filepath.Abs("testdata/echo.ps1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := r.RunPS(ctx, scriptPath, map[string]any{"hello": "world", "n": 42})
	if err != nil {
		t.Fatalf("RunPS error: %v", err)
	}

	if out["hello"] != "world" {
		t.Errorf("expected hello=world, got %v", out["hello"])
	}
	if out["echoed"] != true {
		t.Errorf("expected echoed=true, got %v", out["echoed"])
	}
}

func TestRunner_RunPS_NoInput(t *testing.T) {
	r := New()
	scriptPath, _ := filepath.Abs("testdata/echo.ps1")

	ctx := context.Background()
	out, err := r.RunPS(ctx, scriptPath, nil)
	if err != nil {
		t.Fatalf("RunPS error: %v", err)
	}
	if out["echoed"] != true {
		t.Errorf("expected echoed=true, got %v", out["echoed"])
	}
}

func TestRunner_RunPS_MissingScript(t *testing.T) {
	r := New()
	ctx := context.Background()
	_, err := r.RunPS(ctx, "nonexistent.ps1", nil)
	if err == nil {
		t.Fatal("expected error for missing script, got nil")
	}
}
```

- [ ] **Step 3.3 : Lancer le test pour vérifier qu'il échoue**

```powershell
go test ./pkg/engine/runner/... -v
```

Attendu : FAIL avec `undefined: New` ou similaire.

- [ ] **Step 3.4 : Implémenter le runner**

Créer `pkg/engine/runner/runner.go` :

```go
// Package runner exécute des snippets PowerShell avec I/O JSON via stdin/stdout.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Runner exécute des scripts PowerShell.
type Runner struct {
	// Path vers powershell.exe. Vide = utilise "powershell.exe" (PATH).
	PowerShellPath string
}

// New retourne un Runner avec les défauts.
func New() *Runner {
	return &Runner{PowerShellPath: "powershell.exe"}
}

// RunPS exécute le script PS au chemin scriptPath en lui passant input
// sérialisé en JSON sur stdin. Retourne le JSON parsé depuis stdout.
//
// Le script doit lire stdin via [Console]::In.ReadToEnd() et émettre du JSON
// en une ligne sur stdout (via ConvertTo-Json -Compress).
//
// stderr du process est capturé et inclus dans l'erreur si le process échoue.
func (r *Runner) RunPS(ctx context.Context, scriptPath string, input any) (map[string]any, error) {
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, r.PowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
	)

	// Sérialise input en JSON (ou string vide si nil)
	var stdin []byte
	if input != nil {
		var err error
		stdin, err = json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("marshal input: %w", err)
		}
	}
	cmd.Stdin = bytes.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("powershell failed: %w (stderr: %s)", err, stderr.String())
	}

	out := stdout.Bytes()
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, fmt.Errorf("powershell produced empty stdout (stderr: %s)", stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse stdout as JSON: %w (stdout: %s)", err, string(out))
	}
	return result, nil
}
```

- [ ] **Step 3.5 : Lancer le test pour vérifier qu'il passe**

```powershell
go test ./pkg/engine/runner/... -v
```

Attendu : 3 tests PASS. Note : ce test invoque réellement `powershell.exe` — c'est un integration test, il ne tournera pas sur Linux/macOS. C'est OK, le projet est Windows-only.

- [ ] **Step 3.6 : Commit**

```powershell
git add pkg/engine/runner/
git commit -m "feat(v2/engine): PowerShell runner with JSON I/O"
```

---

### Task 4 : Manifest types + YAML loader

**Files:**
- Create: `pkg/engine/manifest/types.go`
- Create: `pkg/engine/manifest/loader.go`
- Test: `pkg/engine/manifest/loader_test.go`
- Create: `pkg/engine/manifest/testdata/valid-defender.yaml` (fixture)

Cette task installe `gopkg.in/yaml.v3` et définit les types Go correspondant au schéma manifest, puis un loader qui parse un fichier YAML.

- [ ] **Step 4.1 : Installer yaml.v3**

```powershell
go get gopkg.in/yaml.v3@latest
```

Attendu : ajout dans `go.mod`.

- [ ] **Step 4.2 : Définir les types**

Créer `pkg/engine/manifest/types.go` :

```go
// Package manifest contient les types et le loader des manifests YAML
// décrivant la knowledge base des règles de hardening.
package manifest

// Section représente un fichier manifest (1 par section, ex: 01-defender.yaml).
type Section struct {
	Version string      `yaml:"version"`
	Section SectionMeta `yaml:"section"`
	Rules   []Rule      `yaml:"rules"`
}

// SectionMeta : métadonnées de la section.
type SectionMeta struct {
	ID          string `yaml:"id"`
	Order       int    `yaml:"order"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// Rule représente une règle de hardening individuelle.
type Rule struct {
	ID                 string   `yaml:"id"`
	Title              string   `yaml:"title"`
	Description        string   `yaml:"description"`
	Explanation        string   `yaml:"explanation"`
	Severity           string   `yaml:"severity"` // critical | important | nice-to-have
	Impact             string   `yaml:"impact"`
	RequiresReboot     bool     `yaml:"requires_reboot"`
	ProfileWhen        string   `yaml:"profile_when"`
	DependsOn          []string `yaml:"depends_on"`
	Irreversible       bool     `yaml:"irreversible"`
	IrreversibleReason string   `yaml:"irreversible_reason,omitempty"`
	References         []string `yaml:"references,omitempty"`
	Tags               []string `yaml:"tags,omitempty"`
	AddedIn            string   `yaml:"added_in,omitempty"`
	Action             string   `yaml:"action"`
	Test               string   `yaml:"test"`
	Undo               string   `yaml:"undo,omitempty"`
}
```

- [ ] **Step 4.3 : Créer les fixtures (valide + invalide)**

Créer `pkg/engine/manifest/testdata/valid-defender.yaml` :

```yaml
version: "1.0"

section:
  id: defender
  order: 1
  title: "Microsoft Defender"
  description: "Antivirus intégré à Windows."

rules:
  - id: defender.realtime
    title: "Protection temps réel"
    description: "Scanner les fichiers à chaque ouverture."
    explanation: |
      Defender scanne en arrière-plan chaque fichier que tu ouvres.
    severity: critical
    impact: "Aucun. Activé par défaut sur Win11."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    references:
      - "https://learn.microsoft.com/example"
    tags: [malware, defender]
    added_in: "1.0"
    action: ./engine/actions/defender/realtime.action.ps1
    test: ./engine/actions/defender/realtime.test.ps1
    undo: ./engine/actions/defender/realtime.undo.ps1
```

Et `pkg/engine/manifest/testdata/invalid-syntax.yaml` :

```yaml
version: "1.0"
section:
  id: broken
  order: not-a-number       # erreur: int attendu
  title: "Test"
rules: [
  malformed
```

- [ ] **Step 4.4 : Écrire les tests qui échouent**

Créer `pkg/engine/manifest/loader_test.go` :

```go
package manifest

import (
	"path/filepath"
	"testing"
)

func TestLoad_ValidDefender(t *testing.T) {
	path, _ := filepath.Abs("testdata/valid-defender.yaml")

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if s.Version != "1.0" {
		t.Errorf("expected version 1.0, got %q", s.Version)
	}
	if s.Section.ID != "defender" {
		t.Errorf("expected section.id=defender, got %q", s.Section.ID)
	}
	if s.Section.Order != 1 {
		t.Errorf("expected section.order=1, got %d", s.Section.Order)
	}
	if len(s.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(s.Rules))
	}

	r := s.Rules[0]
	if r.ID != "defender.realtime" {
		t.Errorf("expected rule.id=defender.realtime, got %q", r.ID)
	}
	if r.Severity != "critical" {
		t.Errorf("expected severity=critical, got %q", r.Severity)
	}
	if r.ProfileWhen != "always" {
		t.Errorf("expected profile_when=always, got %q", r.ProfileWhen)
	}
	if r.Action != "./engine/actions/defender/realtime.action.ps1" {
		t.Errorf("unexpected action path: %q", r.Action)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path, _ := filepath.Abs("testdata/invalid-syntax.yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
```

- [ ] **Step 4.5 : Lancer le test pour vérifier qu'il échoue**

```powershell
go test ./pkg/engine/manifest/... -v
```

Attendu : FAIL avec `undefined: Load`.

- [ ] **Step 4.6 : Implémenter le loader**

Créer `pkg/engine/manifest/loader.go` :

```go
package manifest

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load lit et parse un fichier manifest YAML.
//
// Le YAML doit respecter la structure définie par les types Section/SectionMeta/Rule.
// Cette fonction NE valide PAS contre le JSONSchema — utiliser Validate() pour ça.
//
// KnownFields(true) force l'échec sur tout champ YAML non mappé dans les types,
// ce qui détecte les fautes de frappe (ex: 'severty' au lieu de 'severity').
func Load(path string) (*Section, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}

	var s Section
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	return &s, nil
}
```

- [ ] **Step 4.7 : Lancer tous les tests**

```powershell
go test ./pkg/engine/manifest/... -v
```

Attendu : 3 tests PASS.

- [ ] **Step 4.8 : Commit**

```powershell
git add pkg/engine/manifest/ go.mod
git commit -m "feat(v2/engine): YAML manifest loader with strict parsing"
```

---

### Task 5 : JSONSchema validator

**Files:**
- Create: `schemas/manifest.schema.json`
- Create: `pkg/engine/manifest/validator.go`
- Test: `pkg/engine/manifest/validator_test.go`

Le validator vérifie qu'un manifest YAML respecte un schéma JSON strict, AVANT le parsing en types Go. Permet de détecter `severity: super-critical` (au lieu de `critical`) ou un champ obligatoire manquant avec un message clair.

- [ ] **Step 5.1 : Installer la lib JSONSchema**

```powershell
go get github.com/santhosh-tekuri/jsonschema/v6@latest
```

- [ ] **Step 5.2 : Écrire le schéma JSONSchema**

Créer `schemas/manifest.schema.json` :

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/koff75/harden-win11/schemas/manifest.schema.json",
  "title": "Harden-Win11 Manifest Section",
  "type": "object",
  "required": ["version", "section", "rules"],
  "additionalProperties": false,
  "properties": {
    "version": {
      "type": "string",
      "pattern": "^[0-9]+\\.[0-9]+$"
    },
    "section": {
      "type": "object",
      "required": ["id", "order", "title", "description"],
      "additionalProperties": false,
      "properties": {
        "id": { "type": "string", "pattern": "^[a-z][a-z0-9_-]*$" },
        "order": { "type": "integer", "minimum": 1, "maximum": 99 },
        "title": { "type": "string", "minLength": 1 },
        "description": { "type": "string", "minLength": 1 }
      }
    },
    "rules": {
      "type": "array",
      "minItems": 1,
      "items": { "$ref": "#/$defs/rule" }
    }
  },
  "$defs": {
    "rule": {
      "type": "object",
      "required": [
        "id", "title", "description", "explanation",
        "severity", "impact", "requires_reboot",
        "profile_when", "depends_on", "irreversible",
        "action", "test"
      ],
      "additionalProperties": false,
      "properties": {
        "id": { "type": "string", "pattern": "^[a-z][a-z0-9_-]*\\.[a-z][a-z0-9_.-]*$" },
        "title": { "type": "string", "minLength": 1 },
        "description": { "type": "string", "minLength": 1 },
        "explanation": { "type": "string", "minLength": 1 },
        "severity": { "enum": ["critical", "important", "nice-to-have"] },
        "impact": { "type": "string" },
        "requires_reboot": { "type": "boolean" },
        "profile_when": { "type": "string", "minLength": 1 },
        "depends_on": {
          "type": "array",
          "items": { "type": "string" }
        },
        "irreversible": { "type": "boolean" },
        "irreversible_reason": { "type": "string" },
        "references": {
          "type": "array",
          "items": { "type": "string", "format": "uri" }
        },
        "tags": {
          "type": "array",
          "items": { "type": "string" }
        },
        "added_in": { "type": "string" },
        "action": { "type": "string", "pattern": "\\.action\\.ps1$" },
        "test": { "type": "string", "pattern": "\\.test\\.ps1$" },
        "undo": { "type": "string", "pattern": "\\.undo\\.ps1$" }
      },
      "if": {
        "properties": { "irreversible": { "const": false } }
      },
      "then": {
        "required": ["undo"]
      }
    }
  }
}
```

- [ ] **Step 5.3 : Écrire le test qui échoue**

Créer `pkg/engine/manifest/validator_test.go` :

```go
package manifest

import (
	"path/filepath"
	"testing"
)

func TestValidate_ValidManifest(t *testing.T) {
	manifestPath, _ := filepath.Abs("testdata/valid-defender.yaml")
	schemaPath, _ := filepath.Abs("../../../schemas/manifest.schema.json")

	if err := Validate(manifestPath, schemaPath); err != nil {
		t.Fatalf("Validate returned error on valid manifest: %v", err)
	}
}

func TestValidate_MissingRequiredField(t *testing.T) {
	manifestPath, _ := filepath.Abs("testdata/missing-action.yaml")
	schemaPath, _ := filepath.Abs("../../../schemas/manifest.schema.json")

	err := Validate(manifestPath, schemaPath)
	if err == nil {
		t.Fatal("expected validation error for missing 'action', got nil")
	}
}

func TestValidate_InvalidSeverity(t *testing.T) {
	manifestPath, _ := filepath.Abs("testdata/invalid-severity.yaml")
	schemaPath, _ := filepath.Abs("../../../schemas/manifest.schema.json")

	err := Validate(manifestPath, schemaPath)
	if err == nil {
		t.Fatal("expected validation error for invalid severity, got nil")
	}
}
```

- [ ] **Step 5.4 : Créer les fixtures invalides**

Créer `pkg/engine/manifest/testdata/missing-action.yaml` :

```yaml
version: "1.0"
section:
  id: defender
  order: 1
  title: "Defender"
  description: "."
rules:
  - id: defender.broken
    title: "Sans action"
    description: "."
    explanation: "."
    severity: critical
    impact: "."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    test: ./engine/actions/defender/broken.test.ps1
    undo: ./engine/actions/defender/broken.undo.ps1
    # action: missing
```

Créer `pkg/engine/manifest/testdata/invalid-severity.yaml` :

```yaml
version: "1.0"
section:
  id: defender
  order: 1
  title: "Defender"
  description: "."
rules:
  - id: defender.broken
    title: "Severité invalide"
    description: "."
    explanation: "."
    severity: super-critical              # invalide
    impact: "."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    action: ./engine/actions/defender/broken.action.ps1
    test: ./engine/actions/defender/broken.test.ps1
    undo: ./engine/actions/defender/broken.undo.ps1
```

- [ ] **Step 5.5 : Implémenter le validator**

Créer `pkg/engine/manifest/validator.go` :

```go
package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Validate vérifie qu'un manifest YAML respecte le schéma JSONSchema fourni.
//
// Le YAML est d'abord parsé en `any`, puis converti en JSON, puis validé
// (le validateur JSONSchema ne lit pas YAML directement).
func Validate(manifestPath, schemaPath string) error {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var raw any
	if err := yaml.Unmarshal(manifestData, &raw); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	// yaml.Unmarshal retourne map[any]any (legacy YAML behavior) ou
	// map[string]any (yaml.v3 default for top-level). On normalise en
	// re-marshalant en JSON puis re-parsant.
	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("convert YAML→JSON: %w", err)
	}
	var instance any
	if err := json.Unmarshal(jsonBytes, &instance); err != nil {
		return fmt.Errorf("parse converted JSON: %w", err)
	}

	c := jsonschema.NewCompiler()
	schema, err := c.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}
	return nil
}
```

- [ ] **Step 5.6 : Lancer les tests**

```powershell
go test ./pkg/engine/manifest/... -v
```

Attendu : 3 tests Validate PASS, plus les 3 de Load. Total 6 PASS.

- [ ] **Step 5.7 : Commit**

```powershell
git add schemas/ pkg/engine/manifest/validator.go pkg/engine/manifest/validator_test.go pkg/engine/manifest/testdata/missing-action.yaml pkg/engine/manifest/testdata/invalid-severity.yaml go.mod
git commit -m "feat(v2/engine): JSONSchema validator for manifests"
```

---

### Task 6 : Premier rule migrée — `defender.realtime` (snippets PS + manifest)

**Files:**
- Create: `engine/actions/defender/realtime.action.ps1`
- Create: `engine/actions/defender/realtime.test.ps1`
- Create: `engine/actions/defender/realtime.undo.ps1`
- Create: `manifests/01-defender.yaml`

On migre la 1re règle du script v1 (lignes 319-323) vers la nouvelle structure. Pas de tests Pester ici — Task 8.

- [ ] **Step 6.1 : Créer `realtime.action.ps1`**

```powershell
# realtime.action.ps1
# Active la protection temps réel de Microsoft Defender.
# Input stdin : { "before": { "DisableRealtimeMonitoring": <bool> } }
# Output stdout : { "ok": true, "before": {...}, "after": {...} }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# État avant
$before = @{
    DisableRealtimeMonitoring = (Get-MpPreference).DisableRealtimeMonitoring
}

# Action
Set-MpPreference -DisableRealtimeMonitoring $false

# État après
$after = @{
    DisableRealtimeMonitoring = (Get-MpPreference).DisableRealtimeMonitoring
}

@{
    ok     = $true
    before = $before
    after  = $after
} | ConvertTo-Json -Compress -Depth 10
```

- [ ] **Step 6.2 : Créer `realtime.test.ps1`**

```powershell
# realtime.test.ps1
# Vérifie si la règle defender.realtime est déjà conforme.
# Output stdout : { "compliant": <bool>, "current": {...} }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$current = (Get-MpPreference).DisableRealtimeMonitoring
$compliant = -not $current   # conforme si DisableRealtimeMonitoring = $false

@{
    compliant = $compliant
    current   = @{ DisableRealtimeMonitoring = $current }
} | ConvertTo-Json -Compress -Depth 10
```

- [ ] **Step 6.3 : Créer `realtime.undo.ps1`**

```powershell
# realtime.undo.ps1
# Revient à l'état "before" passé en stdin.
# Input stdin : { "DisableRealtimeMonitoring": <bool> }
# Output stdout : { "ok": true }

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$inputJson = [Console]::In.ReadToEnd()
if (-not $inputJson.Trim()) {
    Write-Error "undo requires JSON input on stdin with DisableRealtimeMonitoring field"
    exit 1
}
$input = $inputJson | ConvertFrom-Json

Set-MpPreference -DisableRealtimeMonitoring ([bool]$input.DisableRealtimeMonitoring)

@{ ok = $true } | ConvertTo-Json -Compress
```

- [ ] **Step 6.4 : Créer le manifest `01-defender.yaml` (1 règle uniquement)**

```yaml
version: "1.0"

section:
  id: defender
  order: 1
  title: "Microsoft Defender"
  description: "Antivirus intégré à Windows. On active toutes ses protections."

rules:
  - id: defender.realtime
    title: "Protection temps réel"
    description: "Scanner les fichiers à chaque ouverture / téléchargement."
    explanation: |
      Defender scanne en arrière-plan chaque fichier que tu ouvres.
      Sans ça, un malware peut s'exécuter sans alerte.
    severity: critical
    impact: "Aucun. Activé par défaut sur Win11."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    references:
      - "https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/configure-real-time-protection-microsoft-defender-antivirus"
    tags: [malware, defender]
    added_in: "1.0"
    action: ./engine/actions/defender/realtime.action.ps1
    test: ./engine/actions/defender/realtime.test.ps1
    undo: ./engine/actions/defender/realtime.undo.ps1
```

- [ ] **Step 6.5 : Test smoke manuel des snippets**

Lancer le test snippet directement (n'exécute pas l'action, juste lit l'état) :

```powershell
& .\engine\actions\defender\realtime.test.ps1
# Attendu (sortie JSON) : {"compliant":true,"current":{"DisableRealtimeMonitoring":false}}
# (compliant=true si Real-time est déjà actif sur ta machine, ce qui est probable)
```

- [ ] **Step 6.6 : (skip) — la validation E2E sera testée par la CLI en Task 9**

Pas de vérification ici. Task 9 ajoute la commande `harden-engine validate` qui validera `manifests/01-defender.yaml` contre `schemas/manifest.schema.json`. Si la CLI passe, ce manifest est OK.

- [ ] **Step 6.7 : Commit**

```powershell
git add engine/actions/defender/ manifests/
git commit -m "feat(v2/manifests): migrate defender.realtime as PoC"
```

---

### Task 7 : CLI scaffold avec Cobra + `version`

**Files:**
- Modify: `cmd/harden-engine/main.go` (le bootstrap "hello world" devient une vraie CLI)
- Test: `cmd/harden-engine/main_test.go`

- [ ] **Step 7.1 : Installer Cobra**

```powershell
go get github.com/spf13/cobra@latest
```

- [ ] **Step 7.2 : Écrire le test pour `version` (smoke test E2E)**

Créer `cmd/harden-engine/main_test.go` :

```go
package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot remonte jusqu'à la racine du repo (où vit go.mod).
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// cmd/harden-engine/main_test.go → racine = ../..
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// buildEngine compile le binaire dans dist/ et retourne son chemin absolu.
func buildEngine(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	out := filepath.Join(root, "dist", "harden-engine.exe")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/harden-engine")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	return out
}

func TestCLI_Version(t *testing.T) {
	bin := buildEngine(t)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, "version")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("version command failed: %v (stderr: %s)", err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("expected JSON output, got %q : %v", out, err)
	}
	if _, ok := v["version"]; !ok {
		t.Errorf("expected 'version' field in output, got %v", v)
	}
}
```

- [ ] **Step 7.3 : Lancer le test (échoue, version subcommand n'existe pas)**

```powershell
go test ./cmd/harden-engine/... -v
```

Attendu : FAIL (le binaire compile mais `version` n'est pas une commande connue).

- [ ] **Step 7.4 : Implémenter la CLI minimale**

Réécrire `cmd/harden-engine/main.go` :

```go
// harden-engine est le moteur CLI v2 du projet harden-win11.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version est le numéro d'engine (override-able au build via -ldflags).
	Version = "0.1.0-dev"
	// ManifestVersion est la version du schéma manifest supportée.
	ManifestVersion = "1.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "harden-engine",
		Short: "Moteur de hardening Windows 11",
		Long:  "harden-engine — moteur de la baseline de sécurité Windows 11 v2.",
	}
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Affiche la version (engine + manifest + OS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := map[string]any{
				"version":          Version,
				"manifest_version": ManifestVersion,
				"go":               runtime.Version(),
				"os":               runtime.GOOS,
				"arch":             runtime.GOARCH,
			}
			b, err := json.Marshal(out)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		},
	}
}
```

- [ ] **Step 7.5 : Lancer le test pour vérifier qu'il passe**

```powershell
go test ./cmd/harden-engine/... -v
```

Attendu : PASS.

- [ ] **Step 7.6 : Test manuel**

```powershell
go build -o dist/harden-engine.exe ./cmd/harden-engine
.\dist\harden-engine.exe version
.\dist\harden-engine.exe version | jq .   # si tu as jq
```

- [ ] **Step 7.7 : Commit**

```powershell
git add cmd/harden-engine/ go.mod
git commit -m "feat(v2/cli): scaffold harden-engine CLI with version subcommand"
```

---

### Task 8 : Tests Pester pour `realtime.action.ps1`

**Files:**
- Create: `engine/actions/defender/realtime.tests.ps1`

Pester teste directement le snippet PS isolement. Mock `Set-MpPreference` et `Get-MpPreference` pour ne pas dépendre du système réel.

- [ ] **Step 8.1 : Vérifier Pester 5 dispo**

```powershell
Import-Module Pester -MinimumVersion 5.0
Get-Module Pester
```

Attendu : module chargé, version 5.x.

- [ ] **Step 8.2 : Écrire le test Pester**

Créer `engine/actions/defender/realtime.tests.ps1` :

```powershell
# Tests Pester 5 pour realtime.action.ps1 et realtime.test.ps1
# Lancer : Invoke-Pester engine/actions/defender/realtime.tests.ps1

BeforeAll {
    $script:ActionScript = Join-Path $PSScriptRoot 'realtime.action.ps1'
    $script:TestScript   = Join-Path $PSScriptRoot 'realtime.test.ps1'
    $script:UndoScript   = Join-Path $PSScriptRoot 'realtime.undo.ps1'
}

Describe 'realtime.test.ps1' {
    It 'returns compliant=true when DisableRealtimeMonitoring is false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableRealtimeMonitoring = $false }
        }

        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $true
        $output.current.DisableRealtimeMonitoring | Should -Be $false
    }

    It 'returns compliant=false when DisableRealtimeMonitoring is true' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableRealtimeMonitoring = $true }
        }

        $output = & $TestScript | ConvertFrom-Json
        $output.compliant | Should -Be $false
        $output.current.DisableRealtimeMonitoring | Should -Be $true
    }
}

Describe 'realtime.action.ps1' {
    It 'calls Set-MpPreference -DisableRealtimeMonitoring $false' {
        Mock -CommandName Get-MpPreference -MockWith {
            [PSCustomObject]@{ DisableRealtimeMonitoring = $true }
        }
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = & $ActionScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableRealtimeMonitoring -eq $false }
    }
}

Describe 'realtime.undo.ps1' {
    It 'restores DisableRealtimeMonitoring from input' {
        Mock -CommandName Set-MpPreference -MockWith { } -Verifiable

        $output = '{"DisableRealtimeMonitoring":true}' | & $UndoScript | ConvertFrom-Json
        $output.ok | Should -Be $true

        Should -Invoke -CommandName Set-MpPreference -Times 1 `
            -ParameterFilter { $DisableRealtimeMonitoring -eq $true }
    }
}
```

- [ ] **Step 8.3 : Lancer les tests Pester**

```powershell
Invoke-Pester engine/actions/defender/realtime.tests.ps1 -Output Detailed
```

Attendu : tous les `It` passent (4 tests).

**Note sur le mocking** : si Pester émet un warning du type *"Cannot find Set-MpPreference"*, c'est qu'il essaie de mocker une commande non chargée. Solution :

```powershell
# Force le module Defender à se charger AVANT de mocker
Import-Module Defender -ErrorAction SilentlyContinue
```

Ajouter ça dans `BeforeAll` si nécessaire.

- [ ] **Step 8.4 : Commit**

```powershell
git add engine/actions/defender/realtime.tests.ps1
git commit -m "test(v2): Pester tests for defender.realtime snippets"
```

---

### Task 9 : Subcommands `validate` et `apply --dry-run`

**Files:**
- Modify: `cmd/harden-engine/main.go` (ajout subcommands)
- Create: `pkg/engine/dryrun/dryrun.go`
- Test: `pkg/engine/dryrun/dryrun_test.go`

`validate` parse + valide tous les manifests dans `--manifest-dir`. Code retour 0 si OK, 3 si invalide.

`apply --dry-run` : pour chaque règle, lance `.test.ps1`, émet un event `action_result` avec status `would_skip` (si conforme) ou `would_apply` (si non conforme).

- [ ] **Step 9.1 : Écrire la logique dryrun pure**

Créer `pkg/engine/dryrun/dryrun.go` :

```go
// Package dryrun implémente la logique du dry-run : pour chaque règle,
// lancer .test.ps1 et émettre un event NDJSON would_apply/would_skip.
package dryrun

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
)

// Options configure une exécution dryrun.
type Options struct {
	ManifestDir string         // dossier contenant les fichiers YAML
	BasePath    string         // chemin de base pour résoudre les paths relatifs (./engine/actions/...)
	Runner      *runner.Runner // injectable pour les tests
	Writer      *ndjson.Writer // sortie NDJSON
	RunID       string
}

// Run exécute le dry-run sur toutes les règles du manifest unique fourni.
// (Chargement multi-section sera ajouté plus tard.)
func Run(ctx context.Context, sectionPath string, opts Options) error {
	s, err := manifest.Load(sectionPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// run_start
	_ = opts.Writer.Emit(map[string]any{
		"type":             "run_start",
		"run_id":           opts.RunID,
		"manifest_version": s.Version,
		"dry_run":          true,
	})

	for _, rule := range s.Rules {
		testPath := filepath.Join(opts.BasePath, rule.Test)

		start := time.Now()
		out, err := opts.Runner.RunPS(ctx, testPath, nil)
		duration := time.Since(start)

		ev := map[string]any{
			"type":        "action_result",
			"run_id":      opts.RunID,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"rule_id":     rule.ID,
			"duration_ms": duration.Milliseconds(),
			"dry_run":     true,
		}

		if err != nil {
			ev["status"] = "would_fail"
			ev["error"] = err.Error()
		} else if compliant, _ := out["compliant"].(bool); compliant {
			ev["status"] = "would_skip"
			ev["reason"] = "already_compliant"
			ev["current_state"] = out["current"]
		} else {
			ev["status"] = "would_apply"
			ev["current_state"] = out["current"]
		}

		_ = opts.Writer.Emit(ev)
	}

	_ = opts.Writer.Emit(map[string]any{
		"type":   "run_end",
		"run_id": opts.RunID,
	})
	return nil
}
```

- [ ] **Step 9.2 : Écrire le test du dryrun (avec un Runner stubbé)**

Créer `pkg/engine/dryrun/dryrun_test.go` (un seul bloc, imports en tête) :

```go
package dryrun

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
)

// Note : ce test invoque powershell.exe via le runner réel.
// Pour un vrai unit test, on stubberait le runner. Pour le walking skeleton
// on accepte l'integration test (rapide sur Windows, < 5s).

func TestRun_DefenderRealtime(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}

	repo, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	manifestPath := filepath.Join(repo, "manifests", "01-defender.yaml")

	var buf bytes.Buffer
	w := ndjson.NewWriter(&buf)

	opts := Options{
		ManifestDir: filepath.Join(repo, "manifests"),
		BasePath:    repo,
		Runner:      runner.New(),
		Writer:      w,
		RunID:       "test-run",
	}

	if err := Run(context.Background(), manifestPath, opts); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// On doit voir au moins : run_start, action_result(defender.realtime), run_end
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected >= 3 events, got %d : %s", len(lines), buf.String())
	}

	var sawAction bool
	for _, line := range lines {
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("invalid JSON line: %q : %v", line, err)
			continue
		}
		if ev["type"] == "action_result" && ev["rule_id"] == "defender.realtime" {
			sawAction = true
			status, _ := ev["status"].(string)
			if status != "would_skip" && status != "would_apply" && status != "would_fail" {
				t.Errorf("unexpected status: %q", status)
			}
		}
	}
	if !sawAction {
		t.Error("did not see action_result for defender.realtime")
	}
}

// findRepoRoot remonte les répertoires jusqu'à trouver go.mod.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
```

- [ ] **Step 9.3 : Lancer le test (échoue : package nouveau)**

```powershell
go test ./pkg/engine/dryrun/... -v
```

Attendu : compile et tourne. Le test peut PASS ou FAIL selon l'état Defender de la machine, mais doit produire au moins des events bien formés.

- [ ] **Step 9.4 : Ajouter les subcommands `validate` et `apply --dry-run`**

Modifier `cmd/harden-engine/main.go` :

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/dryrun"
	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
	"github.com/spf13/cobra"
)

var (
	Version         = "0.1.0-dev"
	ManifestVersion = "1.0"
)

var (
	flagManifestDir string
	flagSchemaPath  string
	flagDryRun      bool
	flagSection     string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "harden-engine",
		Short: "Moteur de hardening Windows 11",
	}
	rootCmd.PersistentFlags().StringVar(&flagManifestDir, "manifest-dir", "manifests", "Dossier contenant les manifests YAML")
	rootCmd.PersistentFlags().StringVar(&flagSchemaPath, "schema", "schemas/manifest.schema.json", "Chemin du JSONSchema")

	rootCmd.AddCommand(versionCmd(), validateCmd(), applyCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFor(err))
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Affiche la version",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := map[string]any{
				"version":          Version,
				"manifest_version": ManifestVersion,
				"go":               runtime.Version(),
				"os":               runtime.GOOS,
				"arch":             runtime.GOARCH,
			}
			b, _ := json.Marshal(out)
			fmt.Println(string(b))
			return nil
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Valide tous les manifests contre le JSONSchema",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := os.ReadDir(flagManifestDir)
			if err != nil {
				return fmt.Errorf("read manifest dir: %w", err)
			}
			var failed int
			for _, e := range entries {
				if e.IsDir() || (filepath.Ext(e.Name()) != ".yaml" && filepath.Ext(e.Name()) != ".yml") {
					continue
				}
				path := filepath.Join(flagManifestDir, e.Name())
				if err := manifest.Validate(path, flagSchemaPath); err != nil {
					fmt.Fprintf(os.Stderr, "[FAIL] %s : %v\n", e.Name(), err)
					failed++
				} else {
					fmt.Fprintf(os.Stderr, "[OK]   %s\n", e.Name())
				}
			}
			if failed > 0 {
				return &exitError{code: 3, msg: fmt.Sprintf("%d manifests invalid", failed)}
			}
			return nil
		},
	}
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Exécute (ou dry-run) les règles d'une section",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flagDryRun {
				return fmt.Errorf("only --dry-run is supported in this walking skeleton (use --dry-run)")
			}
			if flagSection == "" {
				return fmt.Errorf("--section is required (e.g. --section defender)")
			}

			// Trouve le manifest de la section : NN-<id>.yaml
			entries, err := os.ReadDir(flagManifestDir)
			if err != nil {
				return fmt.Errorf("read manifest dir: %w", err)
			}
			var sectionPath string
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if filepath.Ext(e.Name()) != ".yaml" {
					continue
				}
				p := filepath.Join(flagManifestDir, e.Name())
				s, err := manifest.Load(p)
				if err != nil {
					continue
				}
				if s.Section.ID == flagSection {
					sectionPath = p
					break
				}
			}
			if sectionPath == "" {
				return fmt.Errorf("section %q not found in %s", flagSection, flagManifestDir)
			}

			// BasePath = parent du flagManifestDir (les paths YAML sont relatifs à la racine repo)
			absManifestDir, _ := filepath.Abs(flagManifestDir)
			base := filepath.Dir(absManifestDir)

			runID := time.Now().UTC().Format("2006-01-02T15-04-05")
			w := ndjson.NewWriter(os.Stdout)
			ctx := context.Background()

			return dryrun.Run(ctx, sectionPath, dryrun.Options{
				ManifestDir: flagManifestDir,
				BasePath:    base,
				Runner:      runner.New(),
				Writer:      w,
				RunID:       runID,
			})
		},
	}
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Mode dry-run (rien d'exécuté)")
	cmd.Flags().StringVar(&flagSection, "section", "", "ID de la section à dry-runner (ex: defender)")
	return cmd
}

// exitError porte un code retour custom. exitCodeFor le mappe.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func exitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exitError); ok {
		fmt.Fprintln(os.Stderr, ee.msg)
		return ee.code
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}
```

- [ ] **Step 9.5 : Build et test smoke**

```powershell
go build -o dist/harden-engine.exe ./cmd/harden-engine

# Test validate
.\dist\harden-engine.exe validate --manifest-dir manifests
# Attendu : "[OK]   01-defender.yaml" sur stderr, exit 0

# Test apply --dry-run
.\dist\harden-engine.exe apply --dry-run --section defender
# Attendu : 3+ lignes JSON (run_start, action_result, run_end)
```

- [ ] **Step 9.6 : Commit**

```powershell
git add cmd/harden-engine/main.go pkg/engine/dryrun/ go.mod
git commit -m "feat(v2/cli): validate and apply --dry-run subcommands"
```

---

### Task 10 : Documentation développeur + commit final

**Files:**
- Create: `docs/DEVELOPING.md`

- [ ] **Step 10.1 : Écrire `docs/DEVELOPING.md`**

```markdown
# Developing harden-win11 v2

## Prérequis

- Windows 11
- Go 1.26+ (`go version`)
- PowerShell 5.1+ (intégré Win11)
- Pester 5+ : `Install-Module Pester -RequiredVersion 5.7.1 -Force -SkipPublisherCheck -Scope CurrentUser`

## Build

```powershell
go build -o dist/harden-engine.exe ./cmd/harden-engine
```

## Tests

```powershell
# Tests Go
go test ./...

# Tests Pester (snippets PowerShell)
Invoke-Pester engine/actions/defender/realtime.tests.ps1 -Output Detailed
```

## Lancer le walking skeleton

```powershell
# Validation des manifests
.\dist\harden-engine.exe validate

# Dry-run sur la section defender
.\dist\harden-engine.exe apply --dry-run --section defender
```

## Structure

- `cmd/harden-engine/` : binaire CLI
- `pkg/engine/` : library partagée (parser, runner, dryrun, ndjson)
- `manifests/` : YAML descriptifs des règles
- `engine/actions/` : snippets PowerShell par règle
- `schemas/` : JSONSchema de validation
- `legacy/Harden-Win11.ps1` : script v1 (pas encore déplacé, à la racine pour l'instant)

## Conventions

- Tests Go : nom de fichier `*_test.go`, fonctions `TestXxx(t *testing.T)`
- Tests Pester : nom de fichier `*.tests.ps1`, dans le même dossier que le snippet testé
- Snippets PS : doivent lire JSON sur stdin, émettre JSON compact sur stdout (1 ligne)
- Encodage : tous les fichiers en UTF-8 (avec BOM pour les `.ps1` pour bon parsing PS 5.1)

## Prochaines étapes

Voir `docs/superpowers/plans/2026-05-07-sp1-walking-skeleton.md` pour le plan en cours, et `docs/superpowers/specs/2026-05-07-v2-sp1-core-engine-design.md` pour le design global de SP1.
```

- [ ] **Step 10.2 : Vérifier que tout build et que tous les tests passent**

```powershell
# Tests Go
go test ./...
# Attendu : tous les packages PASS

# Build du binaire
go build -o dist/harden-engine.exe ./cmd/harden-engine
# Attendu : succès, dist/harden-engine.exe créé

# Smoke test
.\dist\harden-engine.exe version
.\dist\harden-engine.exe validate
.\dist\harden-engine.exe apply --dry-run --section defender

# Tests Pester
Invoke-Pester engine/actions/defender/ -Output Detailed
# Attendu : tous les It PASS
```

- [ ] **Step 10.3 : Commit final**

```powershell
git add docs/DEVELOPING.md
git commit -m "docs(v2): developing guide for walking skeleton"
```

---

## Self-Review (à exécuter par l'agent à la fin)

1. **Spec coverage** : ce plan implémente uniquement le walking skeleton de SP1. Sont **explicitement out-of-scope** (à reporter aux Plans 2/3) :
   - Apply réel (Tasks 9 ne fait que dry-run)
   - Journal NDJSON sur disque (la sortie va sur stdout, pas dans `%ProgramData%\Harden-Win11\`)
   - System Restore Point
   - Détecteurs et profilage
   - Undo
   - Migration des 49 autres règles
   - JSONSchema entièrement strict (le schéma actuel valide la structure mais pas tous les invariants)

2. **Tests à la fin** : 4 unit tests Go (ndjson, runner, manifest loader×3, manifest validator×3, dryrun×1) + 4 tests Pester. Total ~12 tests verts.

3. **Critère de succès du Walking Skeleton** : les 3 commandes ci-dessous tournent sans erreur et produisent une sortie cohérente :
```powershell
.\dist\harden-engine.exe version
.\dist\harden-engine.exe validate
.\dist\harden-engine.exe apply --dry-run --section defender
```

4. **Risques connus à surveiller pendant l'exécution** :
   - **Encoding PS 5.1** : si les `.ps1` ne sont pas en UTF-8 BOM, les caractères accentués peuvent merder. Save explicitement en UTF-8 BOM.
   - **`KnownFields(true)` sur YAML** : strict, peut piéger sur des champs YAML que le manifest contient mais pas le struct Go. Si ça casse, retirer temporairement et logger les champs ignorés.
   - **JSONSchema vs YAML** : la conversion YAML→JSON peut perdre des types subtils (timestamps). Pour ce plan, pas de pièges (tous nos champs sont string/int/bool/array). Surveiller en Plan 2 quand on aura plus de complexité.
   - **`Defender` module non-chargé** dans Pester : ajouter `Import-Module Defender` dans `BeforeAll` si les mocks échouent.

5. **Estimation effort réel** : 8-15 heures focalisées (1.5-3 semaines en évening side-project). Tasks 3 (runner) et 9 (dryrun) sont les plus risquées ; les autres sont quasi-mécaniques.

---

**Fin du plan Walking Skeleton.**
