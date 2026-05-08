package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
type App struct {
	ctx context.Context

	manifestDir string
	schemaPath  string
	basePath    string

	// Pour permettre l'annulation d'un run en cours via CancelRun.
	runMu     sync.Mutex
	runCancel context.CancelFunc
}

// NewApp construit une App avec les paths par défaut. Cherche manifests/
// et schemas/ en partant du dossier du binaire (os.Executable) puis du
// CWD, en remontant plusieurs niveaux. Permet de lancer la GUI depuis
// n'importe où (Explorer Windows, raccourci, terminal, Start-Job...).
func NewApp() *App {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}

	for _, start := range candidates {
		mDir, sPath, base, ok := findRepoLayout(start)
		if ok {
			logf("NewApp: resolved repo from %s → manifestDir=%s", start, mDir)
			return &App{manifestDir: mDir, schemaPath: sPath, basePath: base}
		}
	}

	// Fallback : 1er candidat + log warning. La 1re méthode call retournera
	// une erreur claire au frontend.
	fallback := "."
	if len(candidates) > 0 {
		fallback = candidates[0]
	}
	logf("NewApp: WARNING — no manifests/ found from candidates %v, using fallback=%s", candidates, fallback)
	return &App{
		manifestDir: filepath.Join(fallback, "manifests"),
		schemaPath:  filepath.Join(fallback, "schemas", "manifest.schema.json"),
		basePath:    fallback,
	}
}

// findRepoLayout cherche manifests/ + schemas/ en remontant l'arborescence
// depuis start, jusqu'à 8 niveaux (assez pour cmd/harden-gui/build/bin/ +
// marge).
func findRepoLayout(start string) (manifestDir, schemaPath, basePath string, ok bool) {
	dir := start
	for i := 0; i < 8; i++ {
		mDir := filepath.Join(dir, "manifests")
		sPath := filepath.Join(dir, "schemas", "manifest.schema.json")
		if _, err := os.Stat(mDir); err == nil {
			if _, err := os.Stat(sPath); err == nil {
				return mDir, sPath, dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", "", false
}

// Startup est appelé par Wails quand la fenêtre est prête.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	logf("app.Startup: manifestDir=%s schemaPath=%s basePath=%s", a.manifestDir, a.schemaPath, a.basePath)
}

// ─────────────────────────────────────────────────────────────────
// Types exposés au frontend
// ─────────────────────────────────────────────────────────────────

type EngineInfo struct {
	EngineVersion   string `json:"engineVersion"`
	ManifestVersion string `json:"manifestVersion"`
	IsAdmin         bool   `json:"isAdmin"`
	JournalDir      string `json:"journalDir"`
	LogPath         string `json:"logPath"`
	ManifestDir     string `json:"manifestDir"`
}

type RuleInfo struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Explanation    string `json:"explanation"`
	Impact         string `json:"impact"`
	Severity       string `json:"severity"`
	RequiresReboot bool   `json:"requiresReboot"`
	Irreversible   bool   `json:"irreversible"`
	IrreversibleReason string `json:"irreversibleReason,omitempty"`
}

type SectionInfo struct {
	ID         string     `json:"id"`
	Order      int        `json:"order"`
	Title      string     `json:"title"`
	Description string    `json:"description"`
	RuleCount  int        `json:"ruleCount"`
	ManifestID string     `json:"manifestId"`
	Rules      []RuleInfo `json:"rules"`
}

type RunSummary struct {
	RunID      string `json:"runId"`
	Mode       string `json:"mode"`
	Skipped    int    `json:"skipped"`
	Applied    int    `json:"applied"`
	Failed     int    `json:"failed"`
	RolledBack int    `json:"rolledBack"`
	Aborted    bool   `json:"aborted"`
	Cancelled  bool   `json:"cancelled"`
}

// ─────────────────────────────────────────────────────────────────
// Méthodes exposées
// ─────────────────────────────────────────────────────────────────

func (a *App) GetEngineInfo() EngineInfo {
	isAdmin, _ := winadmin.IsElevated()
	info := EngineInfo{
		EngineVersion:   "0.1.0-dev",
		ManifestVersion: "1.0",
		IsAdmin:         isAdmin,
		JournalDir:      journal.DefaultDir(),
		LogPath:         LogPath(),
		ManifestDir:     a.manifestDir,
	}
	logf("app.GetEngineInfo: %+v", info)
	return info
}

// GetSections charge + valide tous les manifests et retourne la liste
// triée par section.order, avec toutes les rules détaillées (titre,
// impact, severity, etc.) pour que le frontend puisse afficher des
// tooltips et des badges.
func (a *App) GetSections() ([]SectionInfo, error) {
	logf("app.GetSections: start")
	validator, err := manifest.NewValidator(a.schemaPath)
	if err != nil {
		logf("app.GetSections: schema compile error: %v", err)
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	entries, err := os.ReadDir(a.manifestDir)
	if err != nil {
		logf("app.GetSections: read dir error: %v", err)
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
			logf("app.GetSections: validate %s failed: %v", e.Name(), err)
			return nil, fmt.Errorf("validate %s: %w", e.Name(), err)
		}
		s, err := manifest.Load(path)
		if err != nil {
			logf("app.GetSections: load %s failed: %v", e.Name(), err)
			return nil, fmt.Errorf("load %s: %w", e.Name(), err)
		}
		rules := make([]RuleInfo, 0, len(s.Rules))
		for _, r := range s.Rules {
			rules = append(rules, RuleInfo{
				ID:                 r.ID,
				Title:              r.Title,
				Description:        r.Description,
				Explanation:        strings.TrimSpace(r.Explanation),
				Impact:             r.Impact,
				Severity:           r.Severity,
				RequiresReboot:     r.RequiresReboot,
				Irreversible:       r.Irreversible,
				IrreversibleReason: r.IrreversibleReason,
			})
		}
		sections = append(sections, SectionInfo{
			ID:          s.Section.ID,
			Order:       s.Section.Order,
			Title:       s.Section.Title,
			Description: s.Section.Description,
			RuleCount:   len(s.Rules),
			ManifestID:  e.Name(),
			Rules:       rules,
		})
	}
	sort.Slice(sections, func(i, j int) bool {
		if sections[i].Order != sections[j].Order {
			return sections[i].Order < sections[j].Order
		}
		return sections[i].ID < sections[j].ID
	})
	logf("app.GetSections: %d sections loaded", len(sections))
	return sections, nil
}

func (a *App) DryRun(sectionIDs []string) (*RunSummary, error) {
	logf("app.DryRun: sections=%v", sectionIDs)
	return a.runEngine(executor.ModeDry, sectionIDs)
}

func (a *App) Apply(sectionIDs []string) (*RunSummary, error) {
	logf("app.Apply: sections=%v", sectionIDs)
	isAdmin, err := winadmin.IsElevated()
	if err != nil {
		return nil, fmt.Errorf("admin check: %w", err)
	}
	if !isAdmin {
		return nil, errors.New("apply requires Administrator privileges. Re-launch the GUI from an elevated PowerShell")
	}
	return a.runEngine(executor.ModeApply, sectionIDs)
}

// CancelRun annule le run en cours s'il y en a un. Le run termine
// proprement avec status=cancelled dans le RunSummary.
func (a *App) CancelRun() {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.runCancel != nil {
		logf("app.CancelRun: cancelling current run")
		a.runCancel()
		a.runCancel = nil
	} else {
		logf("app.CancelRun: no run in progress")
	}
}

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

	totalRules := 0
	for _, s := range sections {
		totalRules += s.RuleCount
	}

	runID := time.Now().UTC().Format("2006-01-02T15-04-05")
	w := newEventWriter(a.ctx, mode, runID)

	a.emit("run_start", map[string]any{
		"runId":        runID,
		"mode":         modeName(mode),
		"sectionCount": len(sections),
		"ruleCount":    totalRules,
		"sections":     collectIDs(sections),
	})

	// Setup cancel context.
	runCtx, cancel := context.WithCancel(context.Background())
	a.runMu.Lock()
	a.runCancel = cancel
	a.runMu.Unlock()
	defer func() {
		a.runMu.Lock()
		a.runCancel = nil
		a.runMu.Unlock()
		cancel()
	}()

	r := runner.New()

	var total executor.Summary
	var aborted, cancelled bool
	for _, sct := range sections {
		summary, err := executor.Run(runCtx, filepath.Join(a.manifestDir, sct.ManifestID), executor.Options{
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

		// Détection de cancel : si runCtx.Err() est Canceled, c'est qu'on a
		// été annulé en cours de route.
		if runCtx.Err() == context.Canceled {
			cancelled = true
			break
		}
		if errors.Is(err, executor.ErrAborted) {
			aborted = true
			break
		}
		if err != nil {
			logf("app.runEngine: section %s error: %v", sct.ID, err)
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
		Cancelled:  cancelled,
	}
	logf("app.runEngine: done %+v", res)
	a.emit("run_end", res)
	return res, nil
}

// ListRuns retourne les run IDs disponibles dans le journal (du plus récent
// au plus ancien). Filtre les runs 'undo-*' pour ne pas polluer la sidebar.
func (a *App) ListRuns() ([]string, error) {
	logf("app.ListRuns: start")
	dir := journal.DefaultDir()
	if _, err := os.Stat(dir); err != nil {
		return []string{}, nil
	}
	all, err := journal.ListRuns(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(all))
	for _, r := range all {
		if strings.HasPrefix(r, "undo-") {
			continue
		}
		out = append(out, r)
	}
	logf("app.ListRuns: %d runs", len(out))
	return out, nil
}

// emit envoie un event Wails au frontend.
func (a *App) emit(name string, payload any) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, name, payload)
}

// ─────────────────────────────────────────────────────────────────
// eventWriter : transforme NDJSON en events Wails côté frontend
// ─────────────────────────────────────────────────────────────────

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
