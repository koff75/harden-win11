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
	ID            string `yaml:"id"`
	Order         int    `yaml:"order"`
	Title         string `yaml:"title"`
	Description   string `yaml:"description"`
	TitleEn       string `yaml:"title_en,omitempty"`
	DescriptionEn string `yaml:"description_en,omitempty"`
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
	// Profiles : liste des profils où cette règle s'applique. Valeurs
	// autorisées : "personal", "business", "maximal". Si vide ou absent,
	// la règle est considérée applicable à tous les profils.
	Profiles []string `yaml:"profiles,omitempty"`
	// Breaks : liste user-facing de ce que la règle peut casser
	// (ex: "Chromecast", "NAS legacy", "RDP support distant"). Affichée
	// dans la GUI pour avertir l'utilisateur.
	Breaks []string `yaml:"breaks,omitempty"`
	// CoachExample : scénario concret en français qui illustre la valeur
	// de la rule. Mode pédagogique : pour la 1ère découverte d'une rule,
	// la GUI affiche cet encart au lieu du jargon "severity: critical".
	// Optionnel — seules les rules importantes en ont. Pas un texte théorique :
	// un cas concret type "imagine que tu cliques sur un PDF reçu par mail…".
	CoachExample string `yaml:"coach_example,omitempty"`
	// User-facing text fields : explication zéro-jargon pour l'utilisateur
	// final. Tous optionnels, affichés tels quels dans la cellule action de
	// la GUI quand présents (sinon fallback sur description/state).
	// Format : 1 phrase courte, française, sans nom de regkey ni chiffre.
	UserToday  string `yaml:"user_today,omitempty"`  // situation présente sans la règle
	UserAfter  string `yaml:"user_after,omitempty"`  // ce que la règle change concrètement
	UserForWho string `yaml:"user_for_who,omitempty"` // quel profil en bénéficie
	UserRisk   string `yaml:"user_risk,omitempty"`   // ce que la règle peut t'empêcher de faire
	// Versions anglaises (i18n). Optionnelles. Si présentes, la GUI les
	// utilise quand l'utilisateur a sélectionné la langue 'en'.
	UserTodayEn  string `yaml:"user_today_en,omitempty"`
	UserAfterEn  string `yaml:"user_after_en,omitempty"`
	UserForWhoEn string `yaml:"user_for_who_en,omitempty"`
	UserRiskEn   string `yaml:"user_risk_en,omitempty"`
}

// AppliesToProfile retourne true si la règle s'applique au profil donné.
// Profiles vide = applicable à tous (compat avec les manifests pré-SP3
// qui n'ont pas le champ profiles).
func (r Rule) AppliesToProfile(profile string) bool {
	if len(r.Profiles) == 0 {
		return true
	}
	for _, p := range r.Profiles {
		if p == profile {
			return true
		}
	}
	return false
}
