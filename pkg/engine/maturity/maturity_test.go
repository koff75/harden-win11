package maturity

import "testing"

func TestCompute_PerfectGrade(t *testing.T) {
	r := Compute(Inputs{
		CriticalTotal: 10, CriticalCompliant: 10,
		ImportantTotal: 10, ImportantCompliant: 10,
		NiceTotal: 10, NiceCompliant: 10,
		HasRecentRestorePoint: true,
		HasWatchlistRunning:   true,
	})
	if r.Grade != GradeA {
		t.Errorf("expected A, got %s (score=%d)", r.Grade, r.Score)
	}
	if r.Score < 95 {
		t.Errorf("expected score ≥ 95, got %d", r.Score)
	}
}

func TestCompute_FailGrade(t *testing.T) {
	r := Compute(Inputs{
		CriticalTotal: 10, CriticalCompliant: 0,
		ImportantTotal: 10, ImportantCompliant: 0,
		NiceTotal: 10, NiceCompliant: 0,
		HasRecentRestorePoint: false,
		HasWatchlistRunning:   false,
	})
	if r.Grade != GradeD {
		t.Errorf("expected D, got %s (score=%d)", r.Grade, r.Score)
	}
	if r.Score > 10 {
		t.Errorf("expected score ≤ 10, got %d", r.Score)
	}
}

func TestCompute_CriticalDominates(t *testing.T) {
	// 100% critical compliant mais rien d'autre → devrait quand même être ≥ C.
	r := Compute(Inputs{
		CriticalTotal: 10, CriticalCompliant: 10,
		ImportantTotal: 10, ImportantCompliant: 0,
		NiceTotal: 10, NiceCompliant: 0,
		HasRecentRestorePoint: false,
		HasWatchlistRunning:   false,
	})
	if r.Grade != GradeC {
		t.Errorf("expected C (50 pts critical only), got %s (score=%d)", r.Grade, r.Score)
	}
}

func TestCompute_NextActionsTop3(t *testing.T) {
	r := Compute(Inputs{
		CriticalTotal: 10, CriticalCompliant: 5,
		ImportantTotal: 10, ImportantCompliant: 5,
		NiceTotal: 10, NiceCompliant: 5,
	})
	if len(r.NextActions) > 3 {
		t.Errorf("expected ≤ 3 next actions, got %d", len(r.NextActions))
	}
	if len(r.NextActions) == 0 {
		t.Error("expected at least 1 next action")
	}
}

func TestScoreToGrade_Bands(t *testing.T) {
	cases := map[int]Grade{0: GradeD, 49: GradeD, 50: GradeC, 74: GradeC, 75: GradeB, 89: GradeB, 90: GradeA, 100: GradeA}
	for s, want := range cases {
		got := scoreToGrade(s)
		if got != want {
			t.Errorf("scoreToGrade(%d) = %s, want %s", s, got, want)
		}
	}
}
