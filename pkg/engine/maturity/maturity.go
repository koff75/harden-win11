// Package maturity calcule un score de maturité du hardening Win11 sur cette
// machine, sous forme de note A/B/C/D + détail.
//
// Pas juste "X règles à renforcer" : un vrai score pondéré qui tient compte
// de :
//   - % critical compliant (poids 50)
//   - % important compliant (poids 25)
//   - % nice-to-have compliant (poids 10)
//   - existence d'un Restore Point récent (poids 8)
//   - watchlist 24h disponible (poids 7)
//
// Total max = 100. Bandes :
//   A : ≥ 90  (excellent)
//   B : 75-89 (solide)
//   C : 50-74 (moyen, à améliorer)
//   D : < 50  (à renforcer urgemment)
package maturity

import (
	"sort"
	"strings"
	"time"
)

type Grade string

const (
	GradeA Grade = "A"
	GradeB Grade = "B"
	GradeC Grade = "C"
	GradeD Grade = "D"
)

// Inputs : ce qu'on a observé sur cette machine.
type Inputs struct {
	// Comptes par sévérité (parmi les rules effectivement évaluées —
	// would_skip = compliant, would_apply = non-compliant, would_fail =
	// indéterminé donc ignoré pour le scoring).
	CriticalTotal     int
	CriticalCompliant int
	ImportantTotal    int
	ImportantCompliant int
	NiceTotal         int
	NiceCompliant     int

	// HasRecentRestorePoint : true si un RP < 30 jours existe.
	HasRecentRestorePoint bool
	// HasWatchlistRunning : true si un report watchlist récent existe.
	HasWatchlistRunning bool

	// Optionnel : nb de rules avec status=would_fail (admin requis ou erreur).
	// On les compte pour info mais on ne pénalise pas le score (l'utilisateur
	// peut juste manquer d'admin).
	UndeterminedCount int
}

// Component : un sous-score qui contribue à la note finale.
type Component struct {
	Name      string  `json:"name"`
	Weight    int     `json:"weight"`     // sur 100
	Earned    float64 `json:"earned"`     // 0 → Weight
	Detail    string  `json:"detail"`
	Status    string  `json:"status"`     // "ok" | "partial" | "missing"
}

// Report : le rapport complet retourné à la GUI.
type Report struct {
	Grade       Grade       `json:"grade"`
	Score       int         `json:"score"`        // 0-100 entier
	Headline    string      `json:"headline"`
	Components  []Component `json:"components"`
	NextActions []string    `json:"next_actions"` // 3 max, priorisées
	Computed    string      `json:"computed_at"`
}

// Compute calcule le score à partir des inputs.
func Compute(in Inputs) Report {
	r := Report{Computed: time.Now().UTC().Format(time.RFC3339)}

	// Sous-score 1 : critical compliant (poids 50).
	cr := pct(in.CriticalCompliant, in.CriticalTotal)
	c1 := Component{
		Name:   "Règles critiques",
		Weight: 50,
		Earned: 50.0 * float64(cr) / 100.0,
		Detail: humanCount(in.CriticalCompliant, in.CriticalTotal, cr),
		Status: bandStatus(cr),
	}
	r.Components = append(r.Components, c1)

	// Sous-score 2 : important compliant (poids 25).
	ir := pct(in.ImportantCompliant, in.ImportantTotal)
	c2 := Component{
		Name:   "Règles importantes",
		Weight: 25,
		Earned: 25.0 * float64(ir) / 100.0,
		Detail: humanCount(in.ImportantCompliant, in.ImportantTotal, ir),
		Status: bandStatus(ir),
	}
	r.Components = append(r.Components, c2)

	// Sous-score 3 : nice-to-have (poids 10).
	nr := pct(in.NiceCompliant, in.NiceTotal)
	c3 := Component{
		Name:   "Règles optionnelles",
		Weight: 10,
		Earned: 10.0 * float64(nr) / 100.0,
		Detail: humanCount(in.NiceCompliant, in.NiceTotal, nr),
		Status: bandStatus(nr),
	}
	r.Components = append(r.Components, c3)

	// Sous-score 4 : Restore Point (poids 8).
	c4 := Component{Name: "Restore Point récent (< 30 jours)", Weight: 8}
	if in.HasRecentRestorePoint {
		c4.Earned = 8
		c4.Status = "ok"
		c4.Detail = "Un Restore Point Windows existe — rollback OS-level disponible."
	} else {
		c4.Earned = 0
		c4.Status = "missing"
		c4.Detail = "Aucun Restore Point récent trouvé. Lance un apply réel ou Checkpoint-Computer manuellement."
	}
	r.Components = append(r.Components, c4)

	// Sous-score 5 : watchlist (poids 7).
	c5 := Component{Name: "Surveillance Event Viewer 24h", Weight: 7}
	if in.HasWatchlistRunning {
		c5.Earned = 7
		c5.Status = "ok"
		c5.Detail = "Une watchlist 24h existe pour un apply récent."
	} else {
		c5.Earned = 0
		c5.Status = "missing"
		c5.Detail = "Pas de watchlist active. Sera créée au prochain apply réel."
	}
	r.Components = append(r.Components, c5)

	total := 0.0
	for _, c := range r.Components {
		total += c.Earned
	}
	r.Score = int(total + 0.5) // arrondi
	r.Grade = scoreToGrade(r.Score)
	r.Headline = headline(r.Grade, r.Score)
	r.NextActions = nextActions(r.Components, in)
	return r
}

func scoreToGrade(s int) Grade {
	switch {
	case s >= 90:
		return GradeA
	case s >= 75:
		return GradeB
	case s >= 50:
		return GradeC
	default:
		return GradeD
	}
}

func headline(g Grade, s int) string {
	switch g {
	case GradeA:
		return "Excellent — ta machine est solidement durcie."
	case GradeB:
		return "Solide — quelques ajustements pour passer en niveau A."
	case GradeC:
		return "Moyen — plusieurs leviers à activer pour vraiment renforcer."
	default:
		return "À renforcer urgemment — peu de protections actives actuellement."
	}
}

// nextActions : top 3 des actions à priorité décroissante pour gagner du score.
func nextActions(comps []Component, in Inputs) []string {
	// Trie par "deficit" décroissant (poids - earned).
	type def struct {
		comp    Component
		deficit float64
	}
	defs := []def{}
	for _, c := range comps {
		defs = append(defs, def{c, float64(c.Weight) - c.Earned})
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].deficit > defs[j].deficit })

	out := []string{}
	for _, d := range defs {
		if d.deficit < 1 {
			continue
		}
		switch {
		case strings.Contains(d.comp.Name, "critiques"):
			out = append(out, "Activer les règles critical restantes (gain potentiel : ~"+round(d.deficit)+" points). Filtre Niveau=Critique dans la table.")
		case strings.Contains(d.comp.Name, "importantes"):
			out = append(out, "Renforcer les règles important (gain : ~"+round(d.deficit)+" points). Filtre Niveau=Important.")
		case strings.Contains(d.comp.Name, "Restore Point"):
			out = append(out, "Lancer un apply réel (au moins une rule) pour créer un Windows Restore Point. Gain : 8 points.")
		case strings.Contains(d.comp.Name, "Surveillance"):
			out = append(out, "Lancer un apply réel — la watchlist 24h sera planifiée automatiquement. Gain : 7 points.")
		case strings.Contains(d.comp.Name, "optionnelles"):
			out = append(out, "Activer quelques rules nice-to-have selon ton usage (gain : ~"+round(d.deficit)+" points).")
		}
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func pct(num, denom int) int {
	if denom == 0 {
		return 100 // pas de rule = "vacuously compliant"
	}
	return int(float64(num) / float64(denom) * 100)
}

func humanCount(comp, total, pct int) string {
	if total == 0 {
		return "Aucune rule de cette catégorie évaluée."
	}
	return fmtPct(comp, total, pct)
}

func bandStatus(p int) string {
	switch {
	case p >= 90:
		return "ok"
	case p >= 50:
		return "partial"
	default:
		return "missing"
	}
}

func round(f float64) string {
	// arrondi simple, pas de package fmt pour rester explicite.
	if f < 0 {
		return "0"
	}
	n := int(f + 0.5)
	return itoa(n)
}

func itoa(n int) string {
	return fmtInt(n)
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

func fmtPct(num, total, pct int) string {
	return fmtInt(num) + " / " + fmtInt(total) + " (" + fmtInt(pct) + "%)"
}
