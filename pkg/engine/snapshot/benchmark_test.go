package snapshot

import "testing"

// Benchmark Diff sur snapshots de taille croissante. Détecte une régression
// si la complexité passe de O(n) à O(n²) par exemple.

func BenchmarkDiff_Small(b *testing.B) {
	bench := makeBenchmarkSnapshots(20, 0.5) // 20 keys, 50% changed
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(bench.before, bench.after)
	}
}

func BenchmarkDiff_Medium(b *testing.B) {
	bench := makeBenchmarkSnapshots(200, 0.3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(bench.before, bench.after)
	}
}

func BenchmarkDiff_Large(b *testing.B) {
	bench := makeBenchmarkSnapshots(2000, 0.1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(bench.before, bench.after)
	}
}

type benchPair struct {
	before, after *Snapshot
}

func makeBenchmarkSnapshots(n int, changeRatio float64) benchPair {
	beforeReg := make([]RegEntry, n)
	afterReg := make([]RegEntry, n)
	for i := 0; i < n; i++ {
		beforeReg[i] = RegEntry{
			Path:   "HKLM:\\Path\\" + itoaInt(i),
			Name:   "Name",
			Exists: true,
			Value:  i,
		}
		afterReg[i] = beforeReg[i]
	}
	// Modifie une fraction des entries.
	changes := int(float64(n) * changeRatio)
	for i := 0; i < changes; i++ {
		afterReg[i].Value = i + 9999
	}
	return benchPair{
		before: &Snapshot{Registry: beforeReg},
		after:  &Snapshot{Registry: afterReg},
	}
}

func itoaInt(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}
