package ulid

import (
	"testing"
	"time"
)

func TestNewIsValid(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := New()
		if !Valid(id) {
			t.Fatalf("New() produced invalid ULID %q", id)
		}
	}
}

// String order must equal time order — the event log's ordering rests on it.
func TestTimeOrdering(t *testing.T) {
	t0 := time.UnixMilli(1_700_000_000_000)
	a := At(t0)
	b := At(t0.Add(time.Millisecond))
	if !(a < b) {
		t.Errorf("ULID at t is not lexically before ULID at t+1ms: %q vs %q", a, b)
	}
}

func TestNoCollisionsInTightLoop(t *testing.T) {
	seen := make(map[string]bool, 10_000)
	for i := 0; i < 10_000; i++ {
		id := New()
		if seen[id] {
			t.Fatalf("collision at iteration %d: %q", i, id)
		}
		seen[id] = true
	}
}

func TestValidRejectsJunk(t *testing.T) {
	for _, s := range []string{"", "short", "01ARZ3NDEKTSV4RRFFQ69G5FAI", "01arz3ndektsv4rrffq69g5fav"} {
		if Valid(s) {
			t.Errorf("Valid(%q) = true, want false", s)
		}
	}
}
