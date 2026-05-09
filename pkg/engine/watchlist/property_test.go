package watchlist

import (
	"math/rand"
	"strconv"
	"testing"
	"testing/quick"
)

// Property : mergeAlerts est commutative au niveau du nb d'alertes finales
// (les sources distinctes restent distinctes, peu importe l'ordre).
func TestProperty_MergeAlerts_LengthCommutative(t *testing.T) {
	f := func(seedA, seedB int64) bool {
		a := randomAlerts(seedA)
		b := randomAlerts(seedB)
		ab := mergeAlerts(a, b)
		ba := mergeAlerts(b, a)
		return len(ab) == len(ba)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Property : merge avec une liste vide est l'identité (nb sources préservé).
func TestProperty_MergeAlerts_EmptyIsIdentity(t *testing.T) {
	f := func(seed int64) bool {
		a := randomAlerts(seed)
		merged := mergeAlerts(a, []Alert{})
		return len(merged) == uniqueSourceCount(a)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Property : AdaptiveThreshold ≥ Threshold statique. Toujours.
func TestProperty_AdaptiveThreshold_NeverStricter(t *testing.T) {
	f := func(seed int64, statThres uint16, median uint16, stddev uint16) bool {
		bl := &Baseline{Sources: map[string]SourceBaseline{
			"X|": {Median: float64(median), Stddev: float64(stddev)},
		}}
		src := Source{LogName: "X", Threshold: int(statThres) % 1000}
		got := bl.AdaptiveThreshold(src)
		// Adaptive doit JAMAIS être plus strict que static.
		return got >= src.Threshold
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}

func randomAlerts(seed int64) []Alert {
	r := rand.New(rand.NewSource(seed))
	n := r.Intn(15)
	out := make([]Alert, n)
	for i := 0; i < n; i++ {
		out[i] = Alert{
			Source:    Source{LogName: "Log" + strconv.Itoa(r.Intn(10))},
			CountSeen: r.Intn(100),
		}
	}
	return out
}

func uniqueSourceCount(alerts []Alert) int {
	seen := map[string]bool{}
	for _, a := range alerts {
		seen[a.Source.LogName+"\x00"+a.Source.Provider] = true
	}
	return len(seen)
}
