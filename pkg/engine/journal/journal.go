// Package journal gère la persistence des runs sur disque sous forme de
// fichiers NDJSON. Un run = un fichier <run_id>.ndjson contenant les events
// run_start / section_start / action_result / section_end / run_end.
//
// Path par défaut : %ProgramData%\Harden-Win11\runs\.
package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultDir retourne le path par défaut du journal sur la machine courante.
// Sur Windows : %ProgramData%\Harden-Win11\runs\. Si ProgramData n'est pas
// défini, fallback sur C:\ProgramData\... (cas qui ne devrait pas arriver
// mais on est défensif).
func DefaultDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return filepath.Join(pd, "Harden-Win11", "runs")
}

// AppliedRule est une vue dérivée d'un event action_result (status=applied)
// utilisée par la commande undo pour retrouver les .undo.ps1 à rejouer.
type AppliedRule struct {
	RuleID    string
	SectionID string
	Before    any // payload JSON arbitraire issu de .action.ps1 'before'
}

// LatestRunID retourne l'ID du run le plus récent dans dir (= nom du fichier
// <run_id>.ndjson le plus récent par mtime). Retourne une erreur si dir n'existe
// pas ou est vide.
func LatestRunID(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read journal dir %s: %w", dir, err)
	}
	type fileMeta struct {
		runID string
		mtime int64
	}
	var files []fileMeta
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".ndjson") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		runID := strings.TrimSuffix(name, filepath.Ext(name))
		files = append(files, fileMeta{runID: runID, mtime: info.ModTime().UnixNano()})
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no journal files found in %s", dir)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mtime > files[j].mtime })
	return files[0].runID, nil
}

// ReadRun lit + parse tous les events d'un journal pour le run_id donné.
// Retourne les events comme []map[string]any (parsed JSON).
func ReadRun(dir, runID string) ([]map[string]any, error) {
	path := filepath.Join(dir, runID+".ndjson")
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("run %q not found in %s", runID, dir)
		}
		return nil, fmt.Errorf("open journal %s: %w", path, err)
	}
	defer f.Close()

	var events []map[string]any
	scanner := bufio.NewScanner(f)
	// Augmenter le buffer pour gérer les lignes longues (ex: ASR avec liste de GUIDs).
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, fmt.Errorf("parse line %d of %s: %w", lineNum, path, err)
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan journal %s: %w", path, err)
	}
	return events, nil
}

// ListRuns retourne tous les run IDs du dossier journal, triés du plus récent
// au plus ancien.
// RunsModifiedSince retourne les runIDs dont le fichier journal a été modifié
// depuis `cutoff`. Triés par mtime décroissant (plus récent en tête) — utile
// pour le time-aware rollback qui veut undo en LIFO sur plusieurs runs.
func RunsModifiedSince(dir string, cutoff time.Time) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read journal dir %s: %w", dir, err)
	}
	type fm struct {
		runID string
		mtime time.Time
	}
	var files []fm
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".ndjson") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		runID := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		// Skip les "undo-*" runs (ce sont des runs d'annulation, pas d'apply).
		if strings.HasPrefix(runID, "undo-") {
			continue
		}
		files = append(files, fm{runID, info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mtime.After(files[j].mtime) })
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.runID)
	}
	return out, nil
}

func ListRuns(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read journal dir %s: %w", dir, err)
	}
	type fileMeta struct {
		runID string
		mtime int64
	}
	var files []fileMeta
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".ndjson") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		runID := strings.TrimSuffix(name, filepath.Ext(name))
		files = append(files, fileMeta{runID: runID, mtime: info.ModTime().UnixNano()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mtime > files[j].mtime })
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.runID)
	}
	return out, nil
}
