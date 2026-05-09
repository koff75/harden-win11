// Package watchlist implémente la surveillance Event Viewer post-apply.
//
// Pourquoi : la plupart des problèmes après un hardening n'apparaissent pas
// immédiatement. Ils arrivent le lendemain quand l'utilisateur essaye
// d'imprimer ou de monter un partage. Sans monitoring, il met 3 semaines
// à faire le lien.
//
// Stratégie : à la fin d'un apply réel, on enregistre une tâche planifiée
// Windows qui exécute `harden-engine watch-events --run-id X --duration 24h`
// 5 minutes plus tard. Cette commande lit Event Viewer toutes les heures
// pendant 24h, compte les erreurs sur les sources concernées (SMB, Defender,
// NetBIOS, RDP), et écrit les anomalies dans
// %ProgramData%\Harden-Win11\watchlist\<runID>.json.
//
// La GUI au prochain boot lit ce dossier et affiche un bandeau si elle voit
// des alertes récentes.
package watchlist

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

// Source : un Log + Provider + level que la watchlist surveille.
type Source struct {
	LogName  string `json:"log_name"`
	Provider string `json:"provider,omitempty"`
	EventIDs []int  `json:"event_ids,omitempty"`
	// MaxLevel : 1=Critical, 2=Error, 3=Warning. On compte les events <= ce niveau.
	MaxLevel int `json:"max_level"`
	// Threshold : nombre d'events suspect au-delà duquel on alerte.
	Threshold int    `json:"threshold"`
	Reason    string `json:"reason"` // user-facing
}

// DefaultSources : configuration par défaut couvrant les domaines touchés
// par les règles harden-win11. Chaque source est un canari pour un type
// de casse fonctionnelle.
var DefaultSources = []Source{
	{
		LogName:   "Microsoft-Windows-SmbClient/Operational",
		MaxLevel:  2, // Error
		Threshold: 5,
		Reason:    "Erreurs SMB Client (partages réseau, NAS, lecteurs mappés). Si ça monte après apply, network.smbv1_disable ou network.smb_*_signing peut casser un partage legacy.",
	},
	{
		LogName:   "Microsoft-Windows-SMBServer/Operational",
		MaxLevel:  2,
		Threshold: 5,
		Reason:    "Erreurs SMB Server (autres machines qui ne peuvent plus accéder à tes partages).",
	},
	{
		LogName:   "Microsoft-Windows-Windows Defender/Operational",
		Provider:  "Microsoft-Windows-Windows Defender",
		EventIDs:  []int{1121, 1122}, // ASR rule blocked / audited
		MaxLevel:  4,                 // Information OK — on les compte tous
		Threshold: 20,
		Reason:    "ASR Defender bloque ou audite des actions. Pic = peut-être une rule ASR trop stricte pour une app que tu utilises.",
	},
	{
		LogName:   "System",
		Provider:  "NETBT",
		MaxLevel:  3, // Warning
		Threshold: 5,
		Reason:    "Problèmes NetBIOS. Un partage qui passe par un nom NetBIOS ne s'y retrouve plus.",
	},
	{
		LogName:   "System",
		Provider:  "Schannel",
		MaxLevel:  2,
		Threshold: 10,
		Reason:    "Erreurs TLS/SSL côté client. Peut indiquer une casse de cipher/protocol après hardening.",
	},
	{
		LogName:   "Microsoft-Windows-PrintService/Admin",
		MaxLevel:  2,
		Threshold: 5,
		Reason:    "Erreurs Print Service. Un partage d'imprimante qui ne marche plus après désactivation NetBIOS / mDNS / SMBv1.",
	},
}

// Alert : une anomalie détectée par la watchlist.
type Alert struct {
	Source         Source   `json:"source"`
	CountSeen      int      `json:"count_seen"`
	WindowStart    string   `json:"window_start"`
	WindowEnd      string   `json:"window_end"`
	SampleMessages []string `json:"sample_messages,omitempty"`
}

// Report : sortie complète du watch.
type Report struct {
	RunID      string  `json:"run_id"`
	Started    string  `json:"started"`
	Completed  string  `json:"completed,omitempty"`
	Duration   string  `json:"duration"`
	Alerts     []Alert `json:"alerts"`
	Polls      int     `json:"polls"`       // nb de scans effectués
	BaselineAt string  `json:"baseline_at"` // timestamp à partir duquel on lit
}

// DefaultDir : %ProgramData%\Harden-Win11\watchlist
func DefaultDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return filepath.Join(pd, "Harden-Win11", "watchlist")
}

// Path retourne le chemin du report pour un runID donné.
func Path(runID string) string {
	return filepath.Join(DefaultDir(), runID+".json")
}

// Watch lance la boucle de polling pour la durée demandée.
// Idempotent : si un report existe déjà pour ce runID, il est mis à jour.
// Best-effort : ne plante jamais (les erreurs PS sont écrites dans le report).
func Watch(ctx context.Context, runID string, baselineAt time.Time, duration time.Duration, pollEvery time.Duration) (*Report, error) {
	if err := os.MkdirAll(DefaultDir(), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir watchlist: %w", err)
	}

	report := &Report{
		RunID:      runID,
		Started:    time.Now().UTC().Format(time.RFC3339),
		BaselineAt: baselineAt.UTC().Format(time.RFC3339),
		Duration:   duration.String(),
		Alerts:     []Alert{},
	}

	deadline := time.Now().Add(duration)
	if pollEvery <= 0 {
		pollEvery = 1 * time.Hour
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			report.Completed = time.Now().UTC().Format(time.RFC3339)
			report.Alerts = mergeAlerts(report.Alerts, scanOnce(ctx, baselineAt, DefaultSources))
			if err := persist(report); err != nil {
				fmt.Fprintf(os.Stderr, "watchlist: final persist on cancel failed: %v\n", err)
			}
			return report, ctx.Err()
		default:
		}

		alerts := scanOnce(ctx, baselineAt, DefaultSources)
		report.Alerts = mergeAlerts(report.Alerts, alerts)
		report.Polls++
		if err := persist(report); err != nil {
			fmt.Fprintf(os.Stderr, "watchlist: persist failed: %v\n", err)
		}

		// Sleep jusqu'au prochain poll (ou jusqu'à context cancel).
		t := time.NewTimer(pollEvery)
		select {
		case <-ctx.Done():
			t.Stop()
			report.Completed = time.Now().UTC().Format(time.RFC3339)
			if err := persist(report); err != nil {
				fmt.Fprintf(os.Stderr, "watchlist: final persist on cancel failed: %v\n", err)
			}
			return report, ctx.Err()
		case <-t.C:
		}
	}

	report.Completed = time.Now().UTC().Format(time.RFC3339)
	if err := persist(report); err != nil {
		fmt.Fprintf(os.Stderr, "watchlist: final persist failed: %v\n", err)
	}
	return report, nil
}

// scanOnce lance un scan ponctuel via PowerShell Get-WinEvent.
// Si une baseline est dispo, utilise des seuils adaptatifs (jamais plus
// stricts que le seuil statique — uniquement plus laxistes pour les
// machines bruyantes).
func scanOnce(ctx context.Context, since time.Time, sources []Source) []Alert {
	bl, _ := LoadBaseline()
	return scanOnceWithBaseline(ctx, since, sources, bl)
}

func scanOnceWithBaseline(ctx context.Context, since time.Time, sources []Source, bl *Baseline) []Alert {
	out := []Alert{}
	for _, src := range sources {
		count, samples, err := countEvents(ctx, src, since)
		if err != nil {
			continue
		}
		threshold := bl.AdaptiveThreshold(src)
		if count >= threshold {
			a := Alert{
				Source:         src,
				CountSeen:      count,
				WindowStart:    since.UTC().Format(time.RFC3339),
				WindowEnd:      time.Now().UTC().Format(time.RFC3339),
				SampleMessages: samples,
			}
			// Si baseline a été utilisée, l'inclure dans les samples pour
			// que l'utilisateur sache que c'est anormal MALGRÉ la baseline.
			if reason := bl.AdaptiveReason(src); reason != "" {
				a.SampleMessages = append([]string{reason}, a.SampleMessages...)
			}
			out = append(out, a)
		}
	}
	return out
}

func countEvents(ctx context.Context, src Source, since time.Time) (int, []string, error) {
	// Build le hashtable -FilterHashtable de Get-WinEvent.
	parts := []string{fmt.Sprintf("LogName='%s'", src.LogName)}
	if src.Provider != "" {
		parts = append(parts, fmt.Sprintf("ProviderName='%s'", src.Provider))
	}
	if src.MaxLevel > 0 {
		levels := []string{}
		for i := 1; i <= src.MaxLevel; i++ {
			levels = append(levels, fmt.Sprintf("%d", i))
		}
		parts = append(parts, "Level=@("+strings.Join(levels, ",")+")")
	}
	if len(src.EventIDs) > 0 {
		ids := []string{}
		for _, id := range src.EventIDs {
			ids = append(ids, fmt.Sprintf("%d", id))
		}
		parts = append(parts, "Id=@("+strings.Join(ids, ",")+")")
	}
	parts = append(parts, fmt.Sprintf("StartTime=([datetime]'%s')", since.Format("2006-01-02T15:04:05Z")))
	filter := "@{" + strings.Join(parts, ";") + "}"

	script := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$evts = @(Get-WinEvent -FilterHashtable %s -MaxEvents 50 -ErrorAction SilentlyContinue)
$samples = $evts | Select-Object -First 3 | ForEach-Object { $_.Message.Substring(0, [Math]::Min($_.Message.Length, 140)) }
@{ count = $evts.Count; samples = @($samples) } | ConvertTo-Json -Compress
`, filter)

	rctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(rctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = hideConsoleAttr()
	raw, err := cmd.Output()
	if err != nil {
		return 0, nil, err
	}
	var parsed struct {
		Count   int      `json:"count"`
		Samples []string `json:"samples"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Get-WinEvent retourne nothing → vide → unmarshal échoue. OK.
		return 0, nil, nil
	}
	return parsed.Count, parsed.Samples, nil
}

// mergeAlerts garde le plus haut count par source (les events sont cumulatifs
// dans la fenêtre baseline → maintenant, donc le dernier scan = max).
func mergeAlerts(prev, latest []Alert) []Alert {
	keyed := map[string]Alert{}
	for _, a := range prev {
		keyed[a.Source.LogName+"\x00"+a.Source.Provider] = a
	}
	for _, a := range latest {
		k := a.Source.LogName + "\x00" + a.Source.Provider
		if existing, ok := keyed[k]; ok {
			if a.CountSeen > existing.CountSeen {
				keyed[k] = a
			}
		} else {
			keyed[k] = a
		}
	}
	out := make([]Alert, 0, len(keyed))
	for _, v := range keyed {
		out = append(out, v)
	}
	return out
}

func persist(r *Report) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(r.RunID), b, 0o644)
}

// LoadReport lit un report depuis le disque.
func LoadReport(runID string) (*Report, error) {
	b, err := os.ReadFile(Path(runID))
	if err != nil {
		return nil, err
	}
	var r Report
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRecent retourne les reports récents (modifiés depuis maxAge).
// Utilisé par la GUI au boot pour afficher un bandeau si des alertes existent.
func ListRecent(maxAge time.Duration) ([]*Report, error) {
	dir := DefaultDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	cutoff := time.Now().Add(-maxAge)
	out := []*Report{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		runID := strings.TrimSuffix(e.Name(), ".json")
		r, err := LoadReport(runID)
		if err == nil && len(r.Alerts) > 0 {
			out = append(out, r)
		}
	}
	return out, nil
}

// ScheduleTask enregistre une tâche planifiée Windows qui lance
// `harden-engine watch-events --run-id X` 5 minutes après l'apply.
// La tâche s'exécute en SYSTEM, log dans %ProgramData%\Harden-Win11\watchlist\,
// et se désinscrit elle-même à la fin de l'exécution (-DeleteExpiredTaskAfter).
//
// Best-effort : si l'enregistrement échoue, on retourne l'erreur mais l'apply
// peut continuer (l'utilisateur peut lancer la commande à la main plus tard).
func ScheduleTask(ctx context.Context, runID, exePath string, delayMinutes int, durationHours int) error {
	taskName := "harden-watchlist-" + runID
	startAt := time.Now().Add(time.Duration(delayMinutes) * time.Minute)
	args := fmt.Sprintf("watch-events --run-id %s --duration %dh", runID, durationHours)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$action = New-ScheduledTaskAction -Execute '%s' -Argument '%s'
$trigger = New-ScheduledTaskTrigger -Once -At ([datetime]'%s')
$settings = New-ScheduledTaskSettingsSet -DeleteExpiredTaskAfter ([timespan]'1.00:00:00') -ExecutionTimeLimit ([timespan]'%d:00:00') -StartWhenAvailable
$principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
Register-ScheduledTask -TaskName '%s' -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
'ok'
`, escapeQuote(exePath), escapeQuote(args), startAt.Format("2006-01-02T15:04:05"), durationHours+1, taskName)

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
