// Package drift detects post-Windows-Update configuration drift.
//
// Why : Microsoft Cumulative Updates regularly reset registry settings that
// harden-win11 had applied. Without monitoring, a user's machine silently
// loses some of its hardening over time. This package re-runs the test
// scripts and compares against the last known-good post-apply snapshot.
package drift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/snapshot"
)

// DefaultDir : %ProgramData%\Harden-Win11\drift
func DefaultDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return filepath.Join(pd, "Harden-Win11", "drift")
}

// Report : output of a drift check.
type Report struct {
	CheckedAt        string             `json:"checked_at"`
	BaselineRunID    string             `json:"baseline_run_id"`
	BaselineSnapshot string             `json:"baseline_snapshot"`
	DriftedRegistry  []snapshot.RegEntry `json:"drifted_registry"`
	DriftedDefender  []DefenderDrift    `json:"drifted_defender,omitempty"`
	DriftedServices  []ServiceDrift     `json:"drifted_services,omitempty"`
	TotalDrifted     int                `json:"total_drifted"`
}

type DefenderDrift struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	Now    any    `json:"now"`
}

type ServiceDrift struct {
	Name   string `json:"name"`
	Before string `json:"before"`
	Now    string `json:"now"`
}

// Check compares the current system state against the baseline post-apply
// snapshot identified by runID. Returns a Report listing what changed.
//
// Best-effort : if the baseline snapshot is missing, returns an error.
// If the system state can't be captured (PS down), returns the inner error.
func Check(ctx context.Context, baselineRunID string) (*Report, error) {
	baselinePath := snapshot.Path(baselineRunID, snapshot.PhasePost)
	baseline, err := snapshot.LoadSnapshot(baselinePath)
	if err != nil {
		return nil, fmt.Errorf("baseline snapshot missing for run %s: %w", baselineRunID, err)
	}

	// Capture current state using a one-off snapshot.
	currentRunID := "drift-check-" + time.Now().UTC().Format("2006-01-02T15-04-05")
	if _, err := snapshot.Capture(ctx, currentRunID, snapshot.PhasePre, 60*time.Second); err != nil {
		return nil, fmt.Errorf("capture current state: %w", err)
	}
	current, err := snapshot.LoadSnapshot(snapshot.Path(currentRunID, snapshot.PhasePre))
	if err != nil {
		return nil, fmt.Errorf("load current snapshot: %w", err)
	}

	rep := &Report{
		CheckedAt:        time.Now().UTC().Format(time.RFC3339),
		BaselineRunID:    baselineRunID,
		BaselineSnapshot: baselinePath,
	}

	// Registry drift : value was X in baseline, now != X.
	bMap := map[string]snapshot.RegEntry{}
	for _, e := range baseline.Registry {
		bMap[e.Path+"\x00"+e.Name] = e
	}
	for _, c := range current.Registry {
		key := c.Path + "\x00" + c.Name
		b, ok := bMap[key]
		if !ok {
			continue
		}
		if !sameValue(b, c) {
			rep.DriftedRegistry = append(rep.DriftedRegistry, c)
		}
	}

	// Defender drift.
	for k, bv := range baseline.Defender {
		if cv, ok := current.Defender[k]; ok && fmt.Sprint(cv) != fmt.Sprint(bv) {
			rep.DriftedDefender = append(rep.DriftedDefender, DefenderDrift{Field: k, Before: bv, Now: cv})
		}
	}

	// Services drift.
	bSvc := map[string]snapshot.ServiceEntry{}
	for _, s := range baseline.Services {
		bSvc[s.Name] = s
	}
	for _, s := range current.Services {
		if b, ok := bSvc[s.Name]; ok {
			before := fmt.Sprintf("%s/%s", b.StartType, b.Status)
			now := fmt.Sprintf("%s/%s", s.StartType, s.Status)
			if before != now {
				rep.DriftedServices = append(rep.DriftedServices, ServiceDrift{Name: s.Name, Before: before, Now: now})
			}
		}
	}

	rep.TotalDrifted = len(rep.DriftedRegistry) + len(rep.DriftedDefender) + len(rep.DriftedServices)
	return rep, nil
}

func sameValue(a, b snapshot.RegEntry) bool {
	if a.Exists != b.Exists {
		return false
	}
	return fmt.Sprint(a.Value) == fmt.Sprint(b.Value)
}

// Save writes the report to %ProgramData%\Harden-Win11\drift\<timestamp>.json.
func Save(r *Report) (string, error) {
	if err := os.MkdirAll(DefaultDir(), 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(DefaultDir(), strings.ReplaceAll(r.CheckedAt, ":", "-")+".json")
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Latest reads the most recent drift report (or nil if none).
func Latest() (*Report, error) {
	dir := DefaultDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type fm struct {
		name  string
		mtime time.Time
	}
	var files []fm
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fm{e.Name(), info.ModTime()})
	}
	if len(files) == 0 {
		return nil, nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mtime.After(files[j].mtime) })
	b, err := os.ReadFile(filepath.Join(dir, files[0].name))
	if err != nil {
		return nil, err
	}
	var r Report
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ScheduleAutoCheck registers a Windows scheduled task that fires after
// each successful Windows Update install (event ID 19 of
// Microsoft-Windows-WindowsUpdateClient/Operational). The task runs
// `harden-engine drift-check --baseline-run-id <id>` to detect post-update
// drift on the rules that were known compliant after `id`.
//
// Best-effort. Replaces any existing harden-drift-check task.
func ScheduleAutoCheck(ctx context.Context, exePath, baselineRunID string) error {
	taskName := "harden-drift-check"
	args := fmt.Sprintf("drift-check --baseline-run-id %s", baselineRunID)

	// Trigger XML : EventTrigger sur le log Windows Update Client, event ID 19
	// (= update successfully installed). On utilise un -XmlText raw pour
	// PowerShell parce que New-ScheduledTaskTrigger n'a pas de helper EventTrigger.
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$xml = @"
<QueryList>
  <Query Id="0" Path="Microsoft-Windows-WindowsUpdateClient/Operational">
    <Select Path="Microsoft-Windows-WindowsUpdateClient/Operational">*[System[(EventID=19)]]</Select>
  </Query>
</QueryList>
"@
$class = Get-CimClass -ClassName MSFT_TaskEventTrigger -Namespace Root/Microsoft/Windows/TaskScheduler
$trigger = New-CimInstance -CimClass $class -ClientOnly
$trigger.Enabled = $true
$trigger.Subscription = $xml

$action = New-ScheduledTaskAction -Execute '%s' -Argument '%s'
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries
$principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
Register-ScheduledTask -TaskName '%s' -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
'ok'
`, escapeQuote(exePath), escapeQuote(args), taskName)

	rctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(rctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = hideConsoleAttr()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Register-ScheduledTask failed: %v\n%s", err, string(out))
	}
	return nil
}

func escapeQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func hideConsoleAttr() *syscall.SysProcAttr {
	return hideConsoleAttrImpl()
}
