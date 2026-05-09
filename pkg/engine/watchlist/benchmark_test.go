package watchlist

import (
	"strconv"
	"testing"
)

func BenchmarkMergeAlerts_100Sources(b *testing.B) {
	prev := make([]Alert, 100)
	latest := make([]Alert, 100)
	for i := 0; i < 100; i++ {
		s := Source{LogName: "Log" + strconv.Itoa(i)}
		prev[i] = Alert{Source: s, CountSeen: i}
		latest[i] = Alert{Source: s, CountSeen: i + 1}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mergeAlerts(prev, latest)
	}
}

func BenchmarkAdaptiveThreshold(b *testing.B) {
	bl := &Baseline{Sources: map[string]SourceBaseline{
		"Microsoft-Windows-SmbClient/Operational|": {
			Median: 4.5, Stddev: 1.2,
		},
	}}
	src := Source{LogName: "Microsoft-Windows-SmbClient/Operational", Threshold: 5}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bl.AdaptiveThreshold(src)
	}
}

func BenchmarkStats_100samples(b *testing.B) {
	xs := make([]int, 100)
	for i := 0; i < 100; i++ {
		xs[i] = i % 17
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = median(xs)
		_ = mean(xs)
		_ = stddev(xs)
	}
}
