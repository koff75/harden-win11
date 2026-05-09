//go:build windows && smoke

package restorepoint

import (
	"context"
	"testing"
	"time"
)

// TestSmoke_RealCreate vérifie que le helper s'invoque sans crash face à un
// vrai Win11. Lance avec : go test -tags=smoke ./pkg/engine/restorepoint/
//
// Note : ne nécessite pas d'admin (Checkpoint-Computer demande admin et
// retournera "spawn_failed" ou "error" en non-admin). Le test passe quand
// même — on vérifie juste que le helper renvoie une Status structurée.
func TestSmoke_RealCreate(t *testing.T) {
	ctx := context.Background()
	st := Create(ctx, "smoketest-"+time.Now().Format("150405"), 30*time.Second)

	t.Logf("Status: created=%v reason=%q error=%q duration=%v",
		st.Created, st.Reason, st.Error, st.Duration)

	if st.Description == "" {
		t.Error("Description vide — devrait toujours contenir le runID")
	}
	if st.Duration <= 0 {
		t.Error("Duration <= 0 — le helper doit toujours mesurer le temps")
	}
}
