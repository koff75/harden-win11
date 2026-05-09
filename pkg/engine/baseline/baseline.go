// Package baseline parse mappings/baselines.yaml et calcule la couverture
// des règles harden-win11 par rapport à CIS / ANSSI / MS Security Baseline.
package baseline

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type ruleMapping struct {
	CIS        []string `yaml:"cis"`
	ANSSI      []string `yaml:"anssi"`
	MSBaseline []string `yaml:"ms_baseline"`
	Notes      string   `yaml:"notes,omitempty"`
}

type Document struct {
	Version    string                 `yaml:"version"`
	Title      string                 `yaml:"title"`
	Sources    map[string]string      `yaml:"sources"`
	Disclaimer string                 `yaml:"disclaimer"`
	Mappings   map[string]ruleMapping `yaml:"mappings"`
}

// Load lit le fichier mappings/baselines.yaml.
func Load(path string) (*Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc Document
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &doc, nil
}

// FrameworkStat agrège la couverture par référentiel.
type FrameworkStat struct {
	Framework      string   `json:"framework"`
	Mapped         int      `json:"mapped"`          // règles harden-win11 avec ≥1 ID dans ce framework
	UnmappedRules  []string `json:"unmapped_rules"`  // rules sans aucun ID (toutes catégories confondues, listées 1×)
	UniqueControls int      `json:"unique_controls"` // nb d'IDs distincts couverts dans le framework
	SampleControls []string `json:"sample_controls"` // 5 premiers IDs (pour aperçu)
}

// CoverageReport est le résultat global.
type CoverageReport struct {
	TotalRules  int                       `json:"total_rules"`  // total des rules dans le mapping
	MappedRules int                       `json:"mapped_rules"` // rules ayant ≥1 ID dans ≥1 framework
	Frameworks  map[string]*FrameworkStat `json:"frameworks"`
	Disclaimer  string                    `json:"disclaimer"`
	Sources     map[string]string         `json:"sources"`
}

// Compute calcule la couverture à partir d'un document chargé et de la liste
// des rule_ids effectivement présents dans les manifests (pour détecter les
// IDs orphelins du mapping).
func Compute(doc *Document, manifestRuleIDs []string) *CoverageReport {
	rep := &CoverageReport{
		TotalRules: len(manifestRuleIDs),
		Frameworks: map[string]*FrameworkStat{
			"cis":         {Framework: "CIS Win11 Enterprise v3.0.0"},
			"anssi":       {Framework: "ANSSI Windows (R-numbers)"},
			"ms_baseline": {Framework: "MS Security Baseline Win11 24H2"},
		},
		Disclaimer: doc.Disclaimer,
		Sources:    doc.Sources,
	}

	cisControls := map[string]bool{}
	anssiControls := map[string]bool{}
	msControls := map[string]bool{}

	mappedRuleIDs := map[string]bool{}

	for _, ruleID := range manifestRuleIDs {
		entry, ok := doc.Mappings[ruleID]
		if !ok {
			// Pas dans le mapping du tout = rule non documentée côté baseline.
			continue
		}
		rep.Frameworks["cis"].Mapped += boolToInt(len(entry.CIS) > 0)
		rep.Frameworks["anssi"].Mapped += boolToInt(len(entry.ANSSI) > 0)
		rep.Frameworks["ms_baseline"].Mapped += boolToInt(len(entry.MSBaseline) > 0)
		if len(entry.CIS)+len(entry.ANSSI)+len(entry.MSBaseline) > 0 {
			mappedRuleIDs[ruleID] = true
		}
		for _, c := range entry.CIS {
			cisControls[c] = true
		}
		for _, c := range entry.ANSSI {
			anssiControls[c] = true
		}
		for _, c := range entry.MSBaseline {
			msControls[c] = true
		}
	}
	rep.MappedRules = len(mappedRuleIDs)

	rep.Frameworks["cis"].UniqueControls = len(cisControls)
	rep.Frameworks["anssi"].UniqueControls = len(anssiControls)
	rep.Frameworks["ms_baseline"].UniqueControls = len(msControls)
	rep.Frameworks["cis"].SampleControls = sortedSample(cisControls, 5)
	rep.Frameworks["anssi"].SampleControls = sortedSample(anssiControls, 5)
	rep.Frameworks["ms_baseline"].SampleControls = sortedSample(msControls, 5)

	// Rules sans aucun mapping (utilisé pour signaler les trous).
	for _, ruleID := range manifestRuleIDs {
		entry, ok := doc.Mappings[ruleID]
		if !ok {
			rep.Frameworks["cis"].UnmappedRules = append(rep.Frameworks["cis"].UnmappedRules, ruleID)
			rep.Frameworks["anssi"].UnmappedRules = append(rep.Frameworks["anssi"].UnmappedRules, ruleID)
			rep.Frameworks["ms_baseline"].UnmappedRules = append(rep.Frameworks["ms_baseline"].UnmappedRules, ruleID)
			continue
		}
		if len(entry.CIS) == 0 {
			rep.Frameworks["cis"].UnmappedRules = append(rep.Frameworks["cis"].UnmappedRules, ruleID)
		}
		if len(entry.ANSSI) == 0 {
			rep.Frameworks["anssi"].UnmappedRules = append(rep.Frameworks["anssi"].UnmappedRules, ruleID)
		}
		if len(entry.MSBaseline) == 0 {
			rep.Frameworks["ms_baseline"].UnmappedRules = append(rep.Frameworks["ms_baseline"].UnmappedRules, ruleID)
		}
	}
	return rep
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func sortedSample(m map[string]bool, n int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > n {
		keys = keys[:n]
	}
	return keys
}
