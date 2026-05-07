// Package manifest contient les types et le loader des manifests YAML
// décrivant la knowledge base des règles de hardening.
package manifest

// Section représente un fichier manifest (1 par section, ex: 01-defender.yaml).
type Section struct {
	Version string      `yaml:"version"`
	Section SectionMeta `yaml:"section"`
	Rules   []Rule      `yaml:"rules"`
}

// SectionMeta : métadonnées de la section.
type SectionMeta struct {
	ID          string `yaml:"id"`
	Order       int    `yaml:"order"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// Rule représente une règle de hardening individuelle.
type Rule struct {
	ID                 string   `yaml:"id"`
	Title              string   `yaml:"title"`
	Description        string   `yaml:"description"`
	Explanation        string   `yaml:"explanation"`
	Severity           string   `yaml:"severity"` // critical | important | nice-to-have
	Impact             string   `yaml:"impact"`
	RequiresReboot     bool     `yaml:"requires_reboot"`
	ProfileWhen        string   `yaml:"profile_when"`
	DependsOn          []string `yaml:"depends_on"`
	Irreversible       bool     `yaml:"irreversible"`
	IrreversibleReason string   `yaml:"irreversible_reason,omitempty"`
	References         []string `yaml:"references,omitempty"`
	Tags               []string `yaml:"tags,omitempty"`
	AddedIn            string   `yaml:"added_in,omitempty"`
	Action             string   `yaml:"action"`
	Test               string   `yaml:"test"`
	Undo               string   `yaml:"undo,omitempty"`
}
