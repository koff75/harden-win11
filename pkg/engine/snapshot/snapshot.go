// Package snapshot capture l'état système pertinent avant + après un apply.
//
// Pourquoi : 1 semaine après un apply, l'utilisateur revient en disant
// "depuis votre truc, mon imprimante ne marche plus" ou "mon partage SMB
// s'est cassé". Sans snapshots, on devine. Avec, on peut diffuser
// pre/post snapshot et identifier la rule qui a touché à la valeur reg
// concernée.
//
// Les snapshots vivent en local dans %ProgramData%\Harden-Win11\snapshots\.
// 0 envoi réseau. L'utilisateur peut tout supprimer via simple delete du
// dossier — pas de service en arrière-plan.
package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Phase indique si on est avant ou après l'apply.
type Phase string

const (
	PhasePre  Phase = "pre"
	PhasePost Phase = "post"
)

// Snapshot agrège l'état observé à un instant t.
type Snapshot struct {
	Timestamp string            `json:"timestamp"`
	RunID     string            `json:"run_id"`
	Phase     Phase             `json:"phase"`
	Registry  []RegEntry        `json:"registry"`
	Defender  map[string]any    `json:"defender,omitempty"`
	Services  []ServiceEntry    `json:"services,omitempty"`
	OSInfo    map[string]string `json:"os_info,omitempty"`
	Errors    []string          `json:"errors,omitempty"`
}

// RegEntry : une clé HKLM avec sa valeur (et un flag exists pour distinguer
// "absent" de "present mais vide").
type RegEntry struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
	Value  any    `json:"value,omitempty"`
}

// ServiceEntry : nom + StartType + Status d'un service watch-listé.
type ServiceEntry struct {
	Name      string `json:"name"`
	StartType string `json:"start_type,omitempty"`
	Status    string `json:"status,omitempty"`
}

// DefaultDir : %ProgramData%\Harden-Win11\snapshots
func DefaultDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return filepath.Join(pd, "Harden-Win11", "snapshots")
}

// Path retourne le chemin du fichier snapshot pour un (runID, phase) donné.
func Path(runID string, phase Phase) string {
	return filepath.Join(DefaultDir(), fmt.Sprintf("%s-%s.json", runID, phase))
}

// Capture lance le helper PowerShell et écrit le JSON sur disque.
// Best-effort : ne plante jamais l'apply, retourne juste les erreurs comme
// metadata dans le snapshot lui-même.
func Capture(ctx context.Context, runID string, phase Phase, timeout time.Duration) (string, error) {
	if err := os.MkdirAll(DefaultDir(), 0o755); err != nil {
		return "", fmt.Errorf("mkdir snapshots: %w", err)
	}

	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(rctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", scanScript)
	cmd.SysProcAttr = hideConsoleAttr()

	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return "", fmt.Errorf("ps scan: %v %s", err, strings.TrimSpace(stderr))
	}

	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		return "", fmt.Errorf("parse ps output: %w", err)
	}

	snap := Snapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RunID:     runID,
		Phase:     phase,
	}
	if reg, ok := raw["registry"].([]any); ok {
		for _, r := range reg {
			if m, ok := r.(map[string]any); ok {
				e := RegEntry{}
				if v, ok := m["path"].(string); ok {
					e.Path = v
				}
				if v, ok := m["name"].(string); ok {
					e.Name = v
				}
				if v, ok := m["exists"].(bool); ok {
					e.Exists = v
				}
				e.Value = m["value"]
				snap.Registry = append(snap.Registry, e)
			}
		}
	}
	if def, ok := raw["defender"].(map[string]any); ok {
		snap.Defender = def
	}
	if svcs, ok := raw["services"].([]any); ok {
		for _, s := range svcs {
			if m, ok := s.(map[string]any); ok {
				e := ServiceEntry{}
				if v, ok := m["name"].(string); ok {
					e.Name = v
				}
				if v, ok := m["start_type"].(string); ok {
					e.StartType = v
				}
				if v, ok := m["status"].(string); ok {
					e.Status = v
				}
				snap.Services = append(snap.Services, e)
			}
		}
	}
	if os, ok := raw["os_info"].(map[string]any); ok {
		snap.OSInfo = map[string]string{}
		for k, v := range os {
			if s, ok := v.(string); ok {
				snap.OSInfo[k] = s
			}
		}
	}
	if errs, ok := raw["errors"].([]any); ok {
		for _, e := range errs {
			if s, ok := e.(string); ok {
				snap.Errors = append(snap.Errors, s)
			}
		}
	}

	out2, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	dest := Path(runID, phase)
	if err := os.WriteFile(dest, out2, 0o644); err != nil {
		return "", fmt.Errorf("write snapshot: %w", err)
	}
	return dest, nil
}

// LoadSnapshot lit un snapshot JSON.
func LoadSnapshot(path string) (*Snapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Diff agrège les changements entre 2 snapshots.
type DiffEntry struct {
	Kind   string `json:"kind"` // "registry" | "service" | "defender"
	Key    string `json:"key"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
	Change string `json:"change"` // "added" | "removed" | "modified"
}

// Diff compare 2 snapshots et retourne la liste des changements.
func Diff(before, after *Snapshot) []DiffEntry {
	out := []DiffEntry{}

	// Registry : keyer sur path+name.
	bMap := map[string]RegEntry{}
	for _, e := range before.Registry {
		bMap[e.Path+"\x00"+e.Name] = e
	}
	aMap := map[string]RegEntry{}
	for _, e := range after.Registry {
		aMap[e.Path+"\x00"+e.Name] = e
	}
	for k, a := range aMap {
		b, hadBefore := bMap[k]
		if !hadBefore || !b.Exists {
			if a.Exists {
				out = append(out, DiffEntry{Kind: "registry", Key: a.Path + "\\" + a.Name, After: a.Value, Change: "added"})
			}
			continue
		}
		if !a.Exists {
			out = append(out, DiffEntry{Kind: "registry", Key: a.Path + "\\" + a.Name, Before: b.Value, Change: "removed"})
			continue
		}
		if !equalAny(b.Value, a.Value) {
			out = append(out, DiffEntry{Kind: "registry", Key: a.Path + "\\" + a.Name, Before: b.Value, After: a.Value, Change: "modified"})
		}
	}
	for k, b := range bMap {
		if _, ok := aMap[k]; !ok && b.Exists {
			out = append(out, DiffEntry{Kind: "registry", Key: b.Path + "\\" + b.Name, Before: b.Value, Change: "removed"})
		}
	}

	// Services : keyer sur name.
	bSvc := map[string]ServiceEntry{}
	for _, s := range before.Services {
		bSvc[s.Name] = s
	}
	for _, s := range after.Services {
		if b, ok := bSvc[s.Name]; ok {
			if b.StartType != s.StartType || b.Status != s.Status {
				out = append(out, DiffEntry{
					Kind: "service", Key: s.Name,
					Before: fmt.Sprintf("%s/%s", b.StartType, b.Status),
					After:  fmt.Sprintf("%s/%s", s.StartType, s.Status),
					Change: "modified",
				})
			}
		}
	}

	// Defender : keyer sur le nom du field.
	for k, av := range after.Defender {
		bv, ok := before.Defender[k]
		if !ok || !equalAny(bv, av) {
			out = append(out, DiffEntry{Kind: "defender", Key: k, Before: bv, After: av, Change: "modified"})
		}
	}
	return out
}

func equalAny(a, b any) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

// hideConsoleAttr est défini par les fichiers _windows.go / _other.go
func hideConsoleAttr() *syscall.SysProcAttr {
	return hideConsoleAttrImpl()
}
