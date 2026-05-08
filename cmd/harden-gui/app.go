package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/executor"
	"github.com/koff75/harden-win11/pkg/engine/journal"
	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
	"github.com/koff75/harden-win11/pkg/engine/winadmin"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App expose les méthodes appelables depuis le frontend JS.
//
// Convention : toutes les méthodes retournent (résultat, erreur). En cas
// d'erreur, le frontend reçoit une promise rejected ; sinon le résultat
// JSON de la valeur.
type App struct {
	ctx context.Context

	manifestDir string
	schemaPath  string
	basePath    string
}

// NewApp construit une App avec les paths par défaut résolus depuis le
// CWD du process. Si on lance depuis dist/ (cas binaire packagé), on
// remonte d'un niveau pour trouver manifests/.
func NewApp() *App {
	cwd, _ := os.Getwd()
	manifestDir := "manifests"
	schemaPath := filepath.Join("schemas", "manifest.schema.json")

	// Si CWD\manifests n'existe pas mais ..\manifests existe, on remonte.
	if _, err := os.Stat(filepath.Join(cwd, manifestDir)); err != nil {
		parent := filepath.Dir(cwd)
		if _, err := os.Stat(filepath.Join(parent, manifestDir)); err == nil {
			cwd = parent
		}
	}
	absManifestDir, _ := filepath.Abs(filepath.Join(cwd, manifestDir))
	base := filepath.Dir(absManifestDir)

	return &App{
		manifestDir: filepath.Join(cwd, manifestDir),
		schemaPath:  filepath.Join(cwd, schemaPath),
		basePath:    base,
	}
}

// Startup est appelé par Wails quand la fenêtre est prête. On capture le
// ctx pour pouvoir émettre des events vers le frontend.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// SectionInfo est le payload retourné par GetSections() : assez d'info
// pour afficher la sidebar (id, title, rule_count) sans surcharge.
type SectionInfo struct {
	ID         string `json:"id"`
	Order      int    `json:"order"`
	Title      string `json:"title"`
	RuleCount  int    `json:"ruleCount"`
	ManifestID string `json:"manifestId"`
}

// EngineInfo est retourné par GetEngineInfo() pour le header.
type EngineInfo struct {
	EngineVersion   string `json:"engineVersion"`
	ManifestVersion string `json:"manifestVersion"`
	IsAdmin         bool   `json:"isAdmin"`
	JournalDir      string `json:"journalDir"`
}

// RunSummary est retourné par DryRun/Apply : agrégats finaux.
type RunSummary struct {
	RunID      string `json:"runId"`
	Mode       string `json:"mode"`
	Skipped    int    `json:"skipped"`
	Applied    int    `json:"applied"`
	Failed     int    `json:"failed"`
	RolledBack int    `json:"rolledBack"`
	Aborted    bool   `json:"aborted"`
}

// GetEngineInfo retourne les métadonnées du moteur pour le header.
func (a *App) GetEngineInfo() EngineInfo {
	isAdmin, _ := winadmin.IsElevated()
	return EngineInfo{
		EngineVersion:   "0.1.0-dev",
		ManifestVersion: "1.0",
		IsAdmin:         isAdmin,
		JournalDir:      journal.DefaultDir(),
	}
}

// GetSections charge + valide tous les manifests et retourne la liste
// triée par section.order.
func (a *App) GetSections() ([]SectionInfo, error) {
	validator, err := manifest.NewValidator(a.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	entries, err := os.ReadDir(a.manifestDir)
	if err != nil {
		return nil, fmt.Errorf("read manifest dir: %w", err)
	}

	var sections []SectionInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(a.manifestDir, e.Name())
		if err := validator.ValidateFile(path); err != nil {
			return nil, fmt.Errorf("validate %s: %w", e.Name(), err)
		}
		s, err := manifest.Load(path)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", e.Name(), err)
		}
		sections = append(sections, SectionInfo{
			ID:         s.Section.ID,
			Order:      s.Section.Order,
			Title:      s.Section.Title,
			RuleCount:  len(s.Rules),
			ManifestID: e.Name(),
		})
	}
	sort.Slice(sections, func(i, j int) bool {
		if sections[i].Order != sections[j].Order {
			return sections[i].Order < sections[j].Order
		}
		return sections[i].ID < sections[j].ID
	})
	return sections, nil
}

// DryRun lance un dry-run sur les sections demandées. sectionIDs vide = toutes.
// Émet des events Wails 'event' avec le payload NDJSON pour le live progress.
func (a *App) DryRun(sectionIDs []string) (*RunSummary, error) {
	return a.runEngine(executor.ModeDry, sectionIDs)
}

// Apply lance un apply réel. Refuse si non-admin (le frontend doit gérer
// le UX de l'erreur).
func (a *App) Apply(sectionIDs []string) (*RunSummary, error) {
	isAdmin, err := winadmin.IsElevated()
	if err != nil {
		return nil, fmt.Errorf("admin check: %w", err)
	}
	if !isAdmin {
		return nil, errors.New("apply requires Administrator privileges. Re-launch the GUI from an elevated PowerShell")
	}
	return a.runEngine(executor.ModeApply, sectionIDs)
}

// runEngine est le code commun à DryRun et Apply.
func (a *App) runEngine(mode executor.Mode, sectionIDs []string) (*RunSummary, error) {
	allSections, err := a.GetSections()
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, id := range sectionIDs {
		wanted[id] = true
	}
	var sections []SectionInfo
	if len(wanted) == 0 {
		sections = allSections
	} else {
		for _, s := range allSections {
			if wanted[s.ID] {
				sections = append(sections, s)
			}
		}
		if len(sections) == 0 {
			return nil, fmt.Errorf("none of the requested sections found")
		}
	}

	runID := time.Now().UTC().Format("2006-01-02T15-04-05")
	w := newEventWriter(a.ctx, mode, runID)

	a.emit("run_start", map[string]any{
		"runId":         runID,
		"mode":          modeName(mode),
		"sectionCount":  len(sections),
		"sections":      collectIDs(sections),
	})

	r := runner.New()
	ctx := context.Background()

	var total executor.Summary
	var aborted bool
	for _, sct := range sections {
		summary, err := executor.Run(ctx, filepath.Join(a.manifestDir, sct.ManifestID), executor.Options{
			Mode:        mode,
			ManifestDir: a.manifestDir,
			BasePath:    a.basePath,
			Runner:      r,
			Writer:      w,
			RunID:       runID,
		})
		total.Skipped += summary.Skipped
		total.Applied += summary.Applied
		total.Failed += summary.Failed
		total.RolledBack += summary.RolledBack

		if errors.Is(err, executor.ErrAborted) {
			aborted = true
			break
		}
		if err != nil {
			return nil, fmt.Errorf("section %s: %w", sct.ID, err)
		}
	}

	res := &RunSummary{
		RunID:      runID,
		Mode:       modeName(mode),
		Skipped:    total.Skipped,
		Applied:    total.Applied,
		Failed:     total.Failed,
		RolledBack: total.RolledBack,
		Aborted:    aborted,
	}
	a.emit("run_end", res)
	return res, nil
}

// ListRuns retourne les run IDs disponibles dans le journal (du plus récent
// au plus ancien). Utilisé pour la sidebar History.
func (a *App) ListRuns() ([]string, error) {
	dir := journal.DefaultDir()
	if _, err := os.Stat(dir); err != nil {
		return []string{}, nil
	}
	return journal.ListRuns(dir)
}

// emit envoie un event Wails au frontend (eventName, payload JSON).
func (a *App) emit(name string, payload any) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, name, payload)
}

// eventWriter implémente io.Writer mais transforme chaque ligne NDJSON
// en event Wails.
type eventWriter struct {
	ctx    context.Context
	mode   executor.Mode
	runID  string
	buffer []byte
}

func newEventWriter(ctx context.Context, mode executor.Mode, runID string) *ndjson.Writer {
	ew := &eventWriter{ctx: ctx, mode: mode, runID: runID}
	return ndjson.NewWriter(ew)
}

func (e *eventWriter) Write(p []byte) (int, error) {
	e.buffer = append(e.buffer, p...)
	for {
		nl := indexByte(e.buffer, '\n')
		if nl < 0 {
			break
		}
		line := e.buffer[:nl]
		e.buffer = e.buffer[nl+1:]
		if len(line) == 0 {
			continue
		}
		// Le payload est déjà du JSON ; on le ré-émet brut comme string vers
		// le frontend qui le parsera. Plus simple que de faire un round-trip
		// décode→encode.
		if e.ctx != nil {
			wailsruntime.EventsEmit(e.ctx, "engine_event", string(line))
		}
	}
	return len(p), nil
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

func collectIDs(s []SectionInfo) []string {
	ids := make([]string, len(s))
	for i, sct := range s {
		ids[i] = sct.ID
	}
	return ids
}

func modeName(m executor.Mode) string {
	switch m {
	case executor.ModeDry:
		return "dry-run"
	case executor.ModeApply:
		return "apply"
	default:
		return "unknown"
	}
}
