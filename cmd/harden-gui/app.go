package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/baseline"
	"github.com/koff75/harden-win11/pkg/engine/executor"
	"github.com/koff75/harden-win11/pkg/engine/journal"
	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/maturity"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/restorepoint"
	"github.com/koff75/harden-win11/pkg/engine/runner"
	"github.com/koff75/harden-win11/pkg/engine/watchlist"
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
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Explanation        string   `json:"explanation"`
	Impact             string   `json:"impact"`
	Severity           string   `json:"severity"`
	RequiresReboot     bool     `json:"requiresReboot"`
	Irreversible       bool     `json:"irreversible"`
	IrreversibleReason string   `json:"irreversibleReason,omitempty"`
	Profiles           []string `json:"profiles,omitempty"`
	Breaks             []string `json:"breaks,omitempty"`
	CoachExample       string   `json:"coachExample,omitempty"`
}

// ProfileInfo : descripteur d'un profil pour le sélecteur GUI.
type ProfileInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
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

// RelaunchAsAdmin relance la GUI avec UAC puis quitte l'instance courante.
// Best-effort : si le spawn échoue, l'app courante reste ouverte.
func (a *App) RelaunchAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		logf("app.RelaunchAsAdmin: os.Executable: %v", err)
		return fmt.Errorf("get current exe path: %w", err)
	}
	logf("app.RelaunchAsAdmin: spawn UAC %s", exe)

	// Utilise PowerShell Start-Process -Verb RunAs pour déclencher UAC.
	// Si l'utilisateur clique "No" sur l'UAC, exit 1 mais on quitte quand même
	// proprement (l'instance courante reste valide jusqu'à ce point).
	cmd := exec.Command("powershell.exe",
		"-NoProfile", "-Command",
		fmt.Sprintf("Start-Process -FilePath '%s' -Verb RunAs", strings.ReplaceAll(exe, "'", "''")))
	cmd.SysProcAttr = hideWindowAttr()
	if err := cmd.Start(); err != nil {
		logf("app.RelaunchAsAdmin: spawn failed: %v", err)
		return fmt.Errorf("spawn elevated: %w", err)
	}
	// Détache puis quitte la GUI courante.
	go func() {
		time.Sleep(200 * time.Millisecond)
		wailsruntime.Quit(a.ctx)
	}()
	return nil
}

// MaturityInputs : payload du frontend pour calculer le score.
type MaturityInputs struct {
	CriticalTotal      int  `json:"criticalTotal"`
	CriticalCompliant  int  `json:"criticalCompliant"`
	ImportantTotal     int  `json:"importantTotal"`
	ImportantCompliant int  `json:"importantCompliant"`
	NiceTotal          int  `json:"niceTotal"`
	NiceCompliant      int  `json:"niceCompliant"`
	UndeterminedCount  int  `json:"undeterminedCount"`
}

// ComputeMaturity calcule le score à partir des inputs frontend (compteurs
// par sévérité agrégés depuis les action_results affichés).
// Le backend ajoute les facteurs système (Restore Point récent, watchlist
// active) qui ne sont pas observables côté frontend.
func (a *App) ComputeMaturity(in MaturityInputs) maturity.Report {
	hasRP := false
	hasWatch := false
	if reports, _ := watchlist.ListRecent(7 * 24 * time.Hour); len(reports) > 0 {
		hasWatch = true
	}
	if hasRecentRestorePoint() {
		hasRP = true
	}
	rep := maturity.Compute(maturity.Inputs{
		CriticalTotal:         in.CriticalTotal,
		CriticalCompliant:     in.CriticalCompliant,
		ImportantTotal:        in.ImportantTotal,
		ImportantCompliant:    in.ImportantCompliant,
		NiceTotal:             in.NiceTotal,
		NiceCompliant:         in.NiceCompliant,
		HasRecentRestorePoint: hasRP,
		HasWatchlistRunning:   hasWatch,
		UndeterminedCount:     in.UndeterminedCount,
	})
	logf("ComputeMaturity: grade=%s score=%d hasRP=%t hasWatch=%t", rep.Grade, rep.Score, hasRP, hasWatch)
	return rep
}

// hasRecentRestorePoint : interroge Get-ComputerRestorePoint via PS et regarde
// si un point harden-win11 < 30 jours existe. Best-effort, swallowed errors.
func hasRecentRestorePoint() bool {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command",
		`$rp = Get-ComputerRestorePoint -ErrorAction SilentlyContinue | Where-Object { $_.Description -like '*harden-win11*' } | Sort-Object CreationTime -Descending | Select-Object -First 1
if ($rp) {
    $ts = [Management.ManagementDateTimeConverter]::ToDateTime($rp.CreationTime)
    $age = (Get-Date) - $ts
    if ($age.TotalDays -le 30) { 'yes' } else { 'no' }
} else { 'no' }`)
	cmd.SysProcAttr = hideWindowAttr()
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "yes"
}

// WatchlistAlert : un alert résumé pour le frontend.
type WatchlistAlert struct {
	RunID      string `json:"runId"`
	LogName    string `json:"logName"`
	Provider   string `json:"provider,omitempty"`
	CountSeen  int    `json:"countSeen"`
	Reason     string `json:"reason"`
	WindowEnd  string `json:"windowEnd"`
}

// GetWatchlistAlerts retourne les alertes Event Viewer remontées par les
// runs récents (modifiés dans les 7 derniers jours). La GUI affiche un
// bandeau si la liste est non-vide.
func (a *App) GetWatchlistAlerts() []WatchlistAlert {
	reports, err := watchlist.ListRecent(7 * 24 * time.Hour)
	if err != nil {
		logf("GetWatchlistAlerts: %v", err)
		return nil
	}
	out := []WatchlistAlert{}
	for _, r := range reports {
		for _, al := range r.Alerts {
			out = append(out, WatchlistAlert{
				RunID:     r.RunID,
				LogName:   al.Source.LogName,
				Provider:  al.Source.Provider,
				CountSeen: al.CountSeen,
				Reason:    al.Source.Reason,
				WindowEnd: al.WindowEnd,
			})
		}
	}
	logf("GetWatchlistAlerts: %d alerts across %d reports", len(out), len(reports))
	return out
}

// loadAllManifests charge tous les *.yaml d'un dossier sans validation schema
// (pour les usages internes : coverage, etc).
func loadAllManifests(dir string) ([]*manifest.Section, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []*manifest.Section
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		s, err := manifest.Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", e.Name(), err)
		}
		out = append(out, s)
	}
	return out, nil
}

// GetCoverage retourne la couverture des règles vs CIS / ANSSI / MS Security Baseline.
// Source : mappings/baselines.yaml. Retourne nil + log si le fichier est absent
// (le frontend doit gérer l'absence gracefulement).
func (a *App) GetCoverage() *baseline.CoverageReport {
	mappingPath := filepath.Join(a.basePath, "mappings", "baselines.yaml")
	doc, err := baseline.Load(mappingPath)
	if err != nil {
		logf("app.GetCoverage: load %s: %v", mappingPath, err)
		return nil
	}
	sections, err := loadAllManifests(a.manifestDir)
	if err != nil {
		logf("app.GetCoverage: loadAllManifests: %v", err)
		return nil
	}
	ids := []string{}
	for _, s := range sections {
		for _, r := range s.Rules {
			ids = append(ids, r.ID)
		}
	}
	rep := baseline.Compute(doc, ids)
	logf("app.GetCoverage: total=%d mapped=%d", rep.TotalRules, rep.MappedRules)
	return rep
}

// ContextDetection : info de contexte machine pour suggérer un profil
// + des rules à auto-exclure (qui casseraient des choses sur cette machine).
type ContextDetection struct {
	ADJoined         bool     `json:"adJoined"`
	IsLaptop         bool     `json:"isLaptop"`
	HasBattery       bool     `json:"hasBattery"`
	PrinterCount     int      `json:"printerCount"`
	NetworkPrinters  bool     `json:"networkPrinters"`
	SuggestedProfile string   `json:"suggestedProfile"`
	Reason           string   `json:"reason"`
	// AutoSkipRules : rule_ids que la GUI doit pré-cocher comme exclus parce
	// qu'ils casseraient une fonctionnalité utilisée sur cette machine
	// (ex: rename_admin sur AD-joined, hibernate_off sur laptop).
	AutoSkipRules []AutoSkipEntry `json:"autoSkipRules"`
}

// AutoSkipEntry décrit une rule auto-exclue + la raison user-facing.
type AutoSkipEntry struct {
	RuleID string `json:"ruleId"`
	Reason string `json:"reason"`
}

// DetectContext spawn un PS qui détecte le contexte machine (AD-joined,
// imprimantes réseau présentes) et propose un profil.
func (a *App) DetectContext() ContextDetection {
	logf("app.DetectContext: start")

	// Heuristiques hardcodées en PS (pas un fichier séparé pour éviter
	// le besoin de manifest dédié).
	psScript := `
$adJoined = $false
$isLaptop = $false
$hasBattery = $false
$printerCount = 0
$networkPrinters = $false
try {
    $cs = Get-CimInstance Win32_ComputerSystem -ErrorAction SilentlyContinue
    if ($cs) { $adJoined = [bool]$cs.PartOfDomain }
} catch {}
try {
    # ChassisTypes : 8=Portable, 9=Laptop, 10=Notebook, 14=Sub Notebook, 30=Tablet, 31=Convertible, 32=Detachable
    $enc = Get-CimInstance Win32_SystemEnclosure -ErrorAction SilentlyContinue
    if ($enc) {
        $laptopTypes = @(8, 9, 10, 14, 30, 31, 32)
        $isLaptop = [bool]($enc.ChassisTypes | Where-Object { $laptopTypes -contains $_ })
    }
} catch {}
try {
    $bat = @(Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue)
    $hasBattery = ($bat.Count -gt 0)
} catch {}
try {
    $printers = @(Get-Printer -ErrorAction SilentlyContinue)
    $printerCount = $printers.Count
    $networkPrinters = [bool](@($printers | Where-Object { $_.Type -eq 'Connection' -or $_.PortName -like 'WSD-*' -or $_.PortName -like 'IP_*' }).Count -gt 0)
} catch {}
@{
    adJoined = $adJoined
    isLaptop = $isLaptop
    hasBattery = $hasBattery
    printerCount = $printerCount
    networkPrinters = $networkPrinters
} | ConvertTo-Json -Compress
`
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("harden-detect-%d.ps1", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, []byte(psScript), 0o644); err != nil {
		logf("DetectContext: write tmp script: %v", err)
		return ContextDetection{SuggestedProfile: "personal", Reason: "détection impossible — défaut PC personnel"}
	}
	defer os.Remove(tmpFile)

	r := runner.New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := r.RunPS(ctx, tmpFile, nil)
	if err != nil {
		logf("DetectContext: PS error: %v", err)
		return ContextDetection{SuggestedProfile: "personal", Reason: "détection impossible — défaut PC personnel"}
	}

	d := ContextDetection{}
	if v, ok := out["adJoined"].(bool); ok {
		d.ADJoined = v
	}
	if v, ok := out["isLaptop"].(bool); ok {
		d.IsLaptop = v
	}
	if v, ok := out["hasBattery"].(bool); ok {
		d.HasBattery = v
	}
	if v, ok := out["printerCount"].(float64); ok {
		d.PrinterCount = int(v)
	}
	if v, ok := out["networkPrinters"].(bool); ok {
		d.NetworkPrinters = v
	}

	// Heuristique du suggestedProfile :
	if d.ADJoined {
		d.SuggestedProfile = "business"
		d.Reason = "Machine jointe à un domaine Active Directory."
	} else if d.NetworkPrinters || d.PrinterCount >= 3 {
		d.SuggestedProfile = "business"
		d.Reason = fmt.Sprintf("%d imprimante(s) détectée(s) dont au moins une réseau — usage probable petite entreprise.", d.PrinterCount)
	} else {
		d.SuggestedProfile = "personal"
		d.Reason = "Pas de domaine AD, peu d'imprimantes — usage perso probable."
	}

	// Auto-skip rules : règles incompatibles avec ce contexte spécifique.
	if d.ADJoined {
		d.AutoSkipRules = append(d.AutoSkipRules, AutoSkipEntry{
			RuleID: "accounts.rename_admin",
			Reason: "Machine AD-joined : la GPO du domaine peut écraser le rename. Renommage à faire via politique AD plutôt qu'en local.",
		})
	}
	if d.IsLaptop || d.HasBattery {
		d.AutoSkipRules = append(d.AutoSkipRules, AutoSkipEntry{
			RuleID: "system_settings.hibernate_off",
			Reason: "Laptop détecté : l'hibernation est utile pour préserver la batterie en mobilité (sleep prolongé consomme).",
		})
	}

	logf("app.DetectContext: %+v", d)
	return d
}

// GetProfiles retourne la liste des profils utilisables. Hardcodé pour
// l'instant — les profils sont une convention partagée entre les manifests
// (champ rule.profiles) et la GUI (sélecteur).
func (a *App) GetProfiles() []ProfileInfo {
	return []ProfileInfo{
		{
			ID:          "personal",
			Title:       "PC personnel",
			Description: "Usage perso, pas de domaine AD, pas de NAS, pas de RDP. Règles agressives OK.",
		},
		{
			ID:          "business",
			Title:       "Petite entreprise",
			Description: "Workgroup, NAS / imprimante réseau, possible RDP support. On évite les règles qui cassent les partages locaux.",
		},
		{
			ID:          "maximal",
			Title:       "Maximal (paranoid)",
			Description: "Toutes les règles sans exception. Pour machine isolée à protéger au maximum.",
		},
	}
}

// GetSections charge + valide tous les manifests et retourne la liste
// triée par section.order. Si profile est non-vide, ne retourne que les
// rules applicables à ce profil.
func (a *App) GetSections(profile string) ([]SectionInfo, error) {
	logf("app.GetSections: profile=%q", profile)
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
			if profile != "" && !r.AppliesToProfile(profile) {
				continue
			}
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
				Profiles:           r.Profiles,
				Breaks:             r.Breaks,
				CoachExample:       r.CoachExample,
			})
		}
		// Skip les sections qui n'ont aucune rule pour le profil sélectionné.
		if len(rules) == 0 {
			continue
		}
		sections = append(sections, SectionInfo{
			ID:          s.Section.ID,
			Order:       s.Section.Order,
			Title:       s.Section.Title,
			Description: s.Section.Description,
			RuleCount:   len(rules),
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

func (a *App) DryRun(sectionIDs []string, profile string, auditMode bool, excludedRules []string) (*RunSummary, error) {
	logf("app.DryRun: sections=%v profile=%q audit=%t excluded=%d", sectionIDs, profile, auditMode, len(excludedRules))
	return a.runEngine(executor.ModeDry, sectionIDs, profile, auditMode, excludedRules)
}

func (a *App) Apply(sectionIDs []string, profile string, auditMode bool, excludedRules []string) (*RunSummary, error) {
	logf("app.Apply: sections=%v profile=%q audit=%t excluded=%d", sectionIDs, profile, auditMode, len(excludedRules))
	isAdmin, err := winadmin.IsElevated()
	if err != nil {
		return nil, fmt.Errorf("admin check: %w", err)
	}
	if !isAdmin {
		return nil, errors.New("apply requires Administrator privileges. Re-launch the GUI from an elevated PowerShell")
	}
	return a.runEngine(executor.ModeApply, sectionIDs, profile, auditMode, excludedRules)
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

func (a *App) runEngine(mode executor.Mode, sectionIDs []string, profile string, auditMode bool, excludedRules []string) (*RunSummary, error) {
	excluded := map[string]bool{}
	for _, id := range excludedRules {
		excluded[id] = true
	}

	allSections, err := a.GetSections(profile)
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

	// Apply réel : ouvrir un fichier journal sur disque pour audit + undo
	// futur via 'harden-engine.exe undo'. Dry-run : pas de journal disque
	// (on n'a rien modifié).
	var journalFile *os.File
	var journalPath string
	if mode == executor.ModeApply {
		dir := journal.DefaultDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logf("runEngine: cannot create journal dir %s: %v", dir, err)
		} else {
			journalPath = filepath.Join(dir, runID+".ndjson")
			f, err := os.OpenFile(journalPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				logf("runEngine: cannot open journal %s: %v", journalPath, err)
			} else {
				journalFile = f
				defer journalFile.Close()
			}
		}
	}
	w := newEventWriter(a.ctx, mode, runID, journalFile)

	a.emit("run_start", map[string]any{
		"runId":        runID,
		"mode":         modeName(mode),
		"sectionCount": len(sections),
		"ruleCount":    totalRules,
		"sections":     collectIDs(sections),
		"journalPath":  journalPath,
	})

	// Émettre run_start aussi dans le journal disque (cohérent avec la CLI).
	if journalFile != nil {
		runStartJSON := fmt.Sprintf(`{"type":"run_start","run_id":%q,"mode":%q,"engine_version":"0.1.0-dev","manifest_version":"1.0","section_count":%d,"sections":%s,"journal_path":%q,"emitter":"gui"}`+"\n",
			runID, modeName(mode), len(sections), jsonStringSlice(collectIDs(sections)), journalPath)
		_, _ = journalFile.WriteString(runStartJSON)
		_ = journalFile.Sync()
	}

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

	// Apply réel : tente un Restore Point Windows (best-effort).
	if mode == executor.ModeApply {
		a.emit("restore_point_started", map[string]any{"runId": runID})
		st := restorepoint.Create(runCtx, runID, 90*time.Second)
		rpPayload := map[string]any{
			"runId":      runID,
			"created":    st.Created,
			"durationMs": st.Duration.Milliseconds(),
			"reason":     st.Reason,
			"error":      st.Error,
		}
		a.emit("restore_point_done", rpPayload)
		if journalFile != nil {
			rpJSON := fmt.Sprintf(`{"type":"restore_point","run_id":%q,"created":%t,"reason":%q,"duration_ms":%d}`+"\n",
				runID, st.Created, st.Reason, st.Duration.Milliseconds())
			_, _ = journalFile.WriteString(rpJSON)
			_ = journalFile.Sync()
		}
		logf("runEngine: restore point created=%v reason=%q duration=%v", st.Created, st.Reason, st.Duration)
	}

	r := runner.New()
	if auditMode {
		r.Env = map[string]string{"HARDEN_ASR_MODE": "audit"}
	}

	var total executor.Summary
	var aborted, cancelled bool
	for _, sct := range sections {
		summary, err := executor.Run(runCtx, filepath.Join(a.manifestDir, sct.ManifestID), executor.Options{
			Mode:            mode,
			ManifestDir:     a.manifestDir,
			BasePath:        a.basePath,
			Runner:          r,
			Writer:          w,
			RunID:           runID,
			Profile:         profile,
			ExcludedRuleIDs: excluded,
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

	// Émettre run_end dans le journal disque pour clore le fichier proprement.
	if journalFile != nil {
		runEndJSON := fmt.Sprintf(`{"type":"run_end","run_id":%q,"skipped":%d,"applied":%d,"failed":%d,"rolled_back":%d,"aborted":%t,"cancelled":%t}`+"\n",
			runID, total.Skipped, total.Applied, total.Failed, total.RolledBack, aborted, cancelled)
		_, _ = journalFile.WriteString(runEndJSON)
		_ = journalFile.Sync()
	}

	a.emit("run_end", res)
	return res, nil
}

// jsonStringSlice retourne ["a","b","c"] pour ["a","b","c"] (encoder à la
// main pour éviter d'importer encoding/json juste pour ça).
func jsonStringSlice(s []string) string {
	b := make([]byte, 0, 64)
	b = append(b, '[')
	for i, v := range s {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '"')
		for _, c := range []byte(v) {
			if c == '"' || c == '\\' {
				b = append(b, '\\')
			}
			b = append(b, c)
		}
		b = append(b, '"')
	}
	b = append(b, ']')
	return string(b)
}

// LoadRun retourne les events action_result d'un run précédent depuis le
// journal disque. Utilisé pour rejouer un run dans le tableau quand
// l'utilisateur clique sur un item de l'historique.
func (a *App) LoadRun(runID string) ([]map[string]any, error) {
	logf("app.LoadRun: runID=%s", runID)
	dir := journal.DefaultDir()
	events, err := journal.ReadRun(dir, runID)
	if err != nil {
		logf("app.LoadRun: %v", err)
		return nil, err
	}
	// On ne renvoie que les action_result au frontend (et pas les
	// run_start/section_start/run_end qui sont du méta).
	var out []map[string]any
	for _, ev := range events {
		if ev["type"] == "action_result" {
			out = append(out, ev)
		}
	}
	logf("app.LoadRun: %d action_result events", len(out))
	return out, nil
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

// eventWriter dual-écrit chaque ligne NDJSON :
//   1. émet un event Wails 'engine_event' vers le frontend (live progress)
//   2. (optionnel) append au fichier journal sur disque (audit trail)
type eventWriter struct {
	ctx     context.Context
	mode    executor.Mode
	runID   string
	buffer  []byte
	journal *os.File // peut être nil (mode dry-run = pas de fichier)
}

func newEventWriter(ctx context.Context, mode executor.Mode, runID string, journalFile *os.File) *ndjson.Writer {
	ew := &eventWriter{ctx: ctx, mode: mode, runID: runID, journal: journalFile}
	return ndjson.NewWriter(ew)
}

func (e *eventWriter) Write(p []byte) (int, error) {
	// Tjs append d'abord au buffer pour gérer le split par newline.
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
		// 1. Event Wails vers le frontend (best-effort — on ignore les erreurs).
		if e.ctx != nil {
			wailsruntime.EventsEmit(e.ctx, "engine_event", string(line))
		}
		// 2. Persistance sur disque + fsync (crash-safe).
		if e.journal != nil {
			_, _ = e.journal.Write(line)
			_, _ = e.journal.Write([]byte{'\n'})
			_ = e.journal.Sync()
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
