package snapshot

import (
	"fmt"
	"math/rand"
	"testing"
	"testing/quick"
)

// Property-based tests : Go testing/quick génère des inputs aléatoires
// et vérifie qu'une propriété (= invariant) tient pour tous.
//
// Ces tests trouvent des bugs que les tests d'exemple ratent — ex: cas
// limites avec valeurs nil, listes vides, doublons, ordres aléatoires.

// Diff(s, s) doit toujours retourner []
func TestProperty_Diff_Reflexive(t *testing.T) {
	f := func(seed int64) bool {
		s := randomSnapshot(seed)
		d := Diff(s, s)
		return len(d) == 0
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Error(err)
	}
}

// Si on diff (a, b) et qu'on inverse, le nb d'entrées doit être conservé
// (un "added" devient "removed" mais c'est toujours 1 entrée).
func TestProperty_Diff_SymmetricCount(t *testing.T) {
	f := func(seedA, seedB int64) bool {
		a := randomSnapshot(seedA)
		b := randomSnapshot(seedB)
		ab := Diff(a, b)
		ba := Diff(b, a)
		// Les counts doivent être identiques (chaque change a son inverse).
		return len(ab) == len(ba)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Diff doit toujours produire des entrées avec change ∈ {added, removed, modified}.
func TestProperty_Diff_ValidChangeKinds(t *testing.T) {
	f := func(seedA, seedB int64) bool {
		a := randomSnapshot(seedA)
		b := randomSnapshot(seedB)
		for _, d := range Diff(a, b) {
			switch d.Change {
			case "added", "removed", "modified":
				continue
			default:
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

func randomSnapshot(seed int64) *Snapshot {
	r := rand.New(rand.NewSource(seed))
	n := r.Intn(20)
	regs := make([]RegEntry, n)
	for i := 0; i < n; i++ {
		regs[i] = RegEntry{
			Path:   "HKLM:\\Foo\\" + fmt.Sprintf("%d", r.Intn(100)),
			Name:   fmt.Sprintf("Name%d", r.Intn(10)),
			Exists: r.Intn(2) == 1,
			Value:  r.Intn(1000),
		}
	}
	return &Snapshot{Registry: regs}
}
