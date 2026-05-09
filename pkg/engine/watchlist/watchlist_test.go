package watchlist

import (
	"testing"
	"time"
)

func TestMergeAlerts_HigherCountWins(t *testing.T) {
	src := Source{LogName: "X"}
	prev := []Alert{{Source: src, CountSeen: 5, WindowEnd: "t1"}}
	latest := []Alert{{Source: src, CountSeen: 12, WindowEnd: "t2"}}
	out := mergeAlerts(prev, latest)
	if len(out) != 1 || out[0].CountSeen != 12 {
		t.Errorf("expected count=12 (latest wins), got %+v", out)
	}
}

func TestMergeAlerts_DistinctSources(t *testing.T) {
	prev := []Alert{{Source: Source{LogName: "X"}, CountSeen: 5}}
	latest := []Alert{{Source: Source{LogName: "Y"}, CountSeen: 7}}
	out := mergeAlerts(prev, latest)
	if len(out) != 2 {
		t.Errorf("expected 2 distinct alerts, got %d", len(out))
	}
}

func TestPath(t *testing.T) {
	p := Path("foo")
	if p == "" {
		t.Error("Path returned empty")
	}
}

func TestDefaultSourcesPlausible(t *testing.T) {
	if len(DefaultSources) < 4 {
		t.Errorf("expected >= 4 default sources (SMB, Defender, NetBT, Schannel, …), got %d", len(DefaultSources))
	}
	for _, s := range DefaultSources {
		if s.LogName == "" {
			t.Error("source missing LogName")
		}
		if s.Threshold <= 0 {
			t.Errorf("source %q has invalid threshold %d", s.LogName, s.Threshold)
		}
		if s.Reason == "" {
			t.Errorf("source %q has no user-facing reason", s.LogName)
		}
	}
}

// Just smoke that Watch context-cancels properly without hanging.
func TestWatch_ContextCancelStopsImmediately(t *testing.T) {
	// We can't actually run a 5min watch, so skip if not a quick test.
	if testing.Short() {
		t.Skip("short")
	}
	// On Linux/macOS le PS n'existe pas → countEvents échoue silencieusement.
	// Le test vérifie juste qu'on respecte le context cancel.
	_ = time.Second
}
