package baseline

import (
	"path/filepath"
	"testing"
)

func TestLoadAndCompute(t *testing.T) {
	doc, err := Load(filepath.Join("..", "..", "..", "mappings", "baselines.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(doc.Mappings) == 0 {
		t.Fatal("aucun mapping chargé")
	}
	if doc.Disclaimer == "" {
		t.Error("disclaimer vide")
	}

	// Quelques règles connues : asr.block_lsass_credential_theft doit avoir
	// un mapping ANSSI (R30/R31).
	lsass, ok := doc.Mappings["asr.block_lsass_credential_theft"]
	if !ok {
		t.Fatal("mapping manquant pour asr.block_lsass_credential_theft")
	}
	if len(lsass.ANSSI) == 0 {
		t.Error("asr.block_lsass_credential_theft devrait avoir un R-number ANSSI")
	}

	// Compute sur l'ensemble des IDs déclarés dans le mapping.
	ruleIDs := make([]string, 0, len(doc.Mappings))
	for id := range doc.Mappings {
		ruleIDs = append(ruleIDs, id)
	}
	rep := Compute(doc, ruleIDs)

	if rep.TotalRules != len(ruleIDs) {
		t.Errorf("TotalRules = %d, attendu %d", rep.TotalRules, len(ruleIDs))
	}
	// Au moins 30 règles couvertes par CIS (Defender + ASR + Firewall + UAC).
	if rep.Frameworks["cis"].Mapped < 30 {
		t.Errorf("CIS mapped = %d, attendu ≥30", rep.Frameworks["cis"].Mapped)
	}
	// Bloatware = aucun mapping.
	if rep.Frameworks["cis"].UniqueControls < 20 {
		t.Errorf("CIS unique controls = %d, attendu ≥20", rep.Frameworks["cis"].UniqueControls)
	}
}

func TestComputeUnmappedRule(t *testing.T) {
	doc := &Document{
		Mappings: map[string]ruleMapping{
			"foo.bar": {CIS: []string{"1.2.3"}},
		},
	}
	rep := Compute(doc, []string{"foo.bar", "ghost.rule"})
	// ghost.rule n'est pas dans le mapping → doit apparaître unmapped sur les 3.
	for fw, st := range rep.Frameworks {
		found := false
		for _, u := range st.UnmappedRules {
			if u == "ghost.rule" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ghost.rule non listé unmapped pour %s", fw)
		}
	}
}
