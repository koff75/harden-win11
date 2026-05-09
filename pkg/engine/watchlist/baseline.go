package watchlist

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Baseline : seuils appris depuis l'historique Event Viewer de la machine.
// Idée : au lieu d'utiliser un seuil statique (5 SMB errors = alerte) qui
// produit beaucoup de faux positifs sur les machines bruyantes (NAS legacy,
// imprimantes ancienne génération), on observe la "normale" sur 7 jours et
// on alerte uniquement quand on dépasse `médiane + N × écart-type`.
//
// La baseline est PER-MACHINE (jamais par-run) et écrite dans
// %ProgramData%\Harden-Win11\watchlist\baseline.json. Refresh recommandé
// tous les 30 jours via `harden-engine watchlist baseline learn`.
type Baseline struct {
	LearnedAt   string                    `json:"learned_at"`
	WindowDays  int                       `json:"window_days"`
	SamplesUsed int                       `json:"samples_used"`
	Sources     map[string]SourceBaseline `json:"sources"`
}

// SourceBaseline : stats par source. La key dans Baseline.Sources est
// "<LogName>|<Provider>" (provider peut être vide).
type SourceBaseline struct {
	DailyCounts []int   `json:"daily_counts"` // count par jour (ordre chronologique)
	Median      float64 `json:"median"`
	Mean        float64 `json:"mean"`
	Stddev      float64 `json:"stddev"`
	Max         int     `json:"max"`
}

// AdaptiveThreshold renvoie le seuil à utiliser pour cette source. Si la
// baseline est absente OU sa stddev est nulle (pas assez de signal),
// retourne le seuil statique. Sinon : max(staticThreshold, médiane + 3σ).
//
// L'idée : les seuils adaptatifs ne deviennent JAMAIS plus stricts que les
// statiques (sinon on crée des faux positifs sur les machines silencieuses).
// Ils peuvent uniquement assouplir si la machine génère normalement plus
// d'events que le seuil statique.
func (b *Baseline) AdaptiveThreshold(src Source) int {
	if b == nil {
		return src.Threshold
	}
	key := src.LogName + "|" + src.Provider
	sb, ok := b.Sources[key]
	if !ok || sb.Stddev == 0 {
		return src.Threshold
	}
	adaptive := int(math.Ceil(sb.Median + 3*sb.Stddev))
	if adaptive < src.Threshold {
		return src.Threshold
	}
	return adaptive
}

// AdaptiveReason produit une explication user-facing si l'on alerte malgré
// un seuil adaptatif. Utile pour le bandeau GUI qui veut dire pourquoi
// c'est suspect malgré la baseline.
func (b *Baseline) AdaptiveReason(src Source) string {
	if b == nil {
		return ""
	}
	key := src.LogName + "|" + src.Provider
	sb, ok := b.Sources[key]
	if !ok || sb.Stddev == 0 {
		return ""
	}
	return fmt.Sprintf("baseline: médiane=%.1f, σ=%.1f, max=%d sur %d échantillon(s).",
		sb.Median, sb.Stddev, sb.Max, len(sb.DailyCounts))
}

// BaselinePath : %ProgramData%\Harden-Win11\watchlist\baseline.json
func BaselinePath() string {
	return filepath.Join(DefaultDir(), "baseline.json")
}

func baselinePath() string { return BaselinePath() }

// SaveBaseline persiste sur disque.
func SaveBaseline(b *Baseline) error {
	if err := os.MkdirAll(DefaultDir(), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(baselinePath(), out, 0o644)
}

// LoadBaseline lit le baseline.json. Renvoie nil si absent (pas une erreur).
func LoadBaseline() (*Baseline, error) {
	b, err := os.ReadFile(baselinePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var bl Baseline
	if err := json.Unmarshal(b, &bl); err != nil {
		return nil, err
	}
	return &bl, nil
}

// Learn observe les events des `daysBack` derniers jours sur les sources
// fournies, calcule médiane/mean/σ par source, et persiste.
//
// Best-effort : si une source ne répond pas, on la skip. Si la machine n'a
// pas assez d'historique (Event Viewer purge anciens events), on prend ce
// qu'il y a.
func Learn(ctx context.Context, sources []Source, daysBack int) (*Baseline, error) {
	if daysBack <= 0 {
		daysBack = 7
	}
	bl := &Baseline{
		LearnedAt:  time.Now().UTC().Format(time.RFC3339),
		WindowDays: daysBack,
		Sources:    map[string]SourceBaseline{},
	}

	totalSamples := 0
	now := time.Now()
	for _, src := range sources {
		dailyCounts, err := learnSource(ctx, src, now, daysBack)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: baseline source %s: %v (skip)\n", src.LogName, err)
			continue
		}
		if len(dailyCounts) == 0 {
			continue
		}
		key := src.LogName + "|" + src.Provider
		bl.Sources[key] = SourceBaseline{
			DailyCounts: dailyCounts,
			Median:      median(dailyCounts),
			Mean:        mean(dailyCounts),
			Stddev:      stddev(dailyCounts),
			Max:         maxInt(dailyCounts),
		}
		totalSamples += len(dailyCounts)
	}
	bl.SamplesUsed = totalSamples
	return bl, nil
}

// learnSource : pour une source, lance Get-WinEvent par jour sur les daysBack
// derniers jours, retourne le count par jour (ordre chronologique : 0 = jour
// le plus ancien, len-1 = jour le plus récent complet).
func learnSource(ctx context.Context, src Source, now time.Time, daysBack int) ([]int, error) {
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
	baseFilter := strings.Join(parts, ";")

	counts := make([]int, 0, daysBack)
	for d := daysBack; d >= 1; d-- {
		end := now.Add(-time.Duration(d-1) * 24 * time.Hour)
		start := end.Add(-24 * time.Hour)
		filter := "@{" + baseFilter +
			fmt.Sprintf(";StartTime=([datetime]'%s');EndTime=([datetime]'%s')", start.Format("2006-01-02T15:04:05Z"), end.Format("2006-01-02T15:04:05Z")) +
			"}"
		script := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'
$evts = @(Get-WinEvent -FilterHashtable %s -ErrorAction SilentlyContinue)
$evts.Count
`, filter)

		rctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		cmd := exec.CommandContext(rctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
		cmd.SysProcAttr = hideConsoleAttr()
		out, err := cmd.Output()
		cancel()
		if err != nil {
			// Probablement pas d'historique ce jour-là (Event Viewer purgé).
			counts = append(counts, 0)
			continue
		}
		var n int
		_, _ = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n)
		counts = append(counts, n)
	}
	return counts, nil
}

func median(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := make([]int, len(xs))
	copy(cp, xs)
	sort.Ints(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return float64(cp[mid])
	}
	return float64(cp[mid-1]+cp[mid]) / 2
}

func mean(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0
	for _, x := range xs {
		sum += x
	}
	return float64(sum) / float64(len(xs))
}

func stddev(xs []int) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	sum := 0.0
	for _, x := range xs {
		d := float64(x) - m
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(xs)-1))
}

func maxInt(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}
