package maturity

import "testing"

func BenchmarkCompute(b *testing.B) {
	in := Inputs{
		CriticalTotal: 25, CriticalCompliant: 18,
		ImportantTotal: 29, ImportantCompliant: 20,
		NiceTotal: 41, NiceCompliant: 30,
		HasRecentRestorePoint: true,
		HasWatchlistRunning:   false,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Compute(in)
	}
}
