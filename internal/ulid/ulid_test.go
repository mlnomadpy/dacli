package ulid

import (
	"strings"
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

// A ULID carries its own creation time, so a reader can window or order events
// by time without opening a single file — the id is already in the filename.
func TestTimeRoundTripsTheEncodedTimestamp(t *testing.T) {
	want := time.UnixMilli(time.Now().UnixMilli()).UTC()
	id := At(want)
	got, ok := Time(id)
	if !ok {
		t.Fatalf("Time(%q) reported the id as malformed", id)
	}
	if !got.Equal(want) {
		t.Errorf("Time = %v, want %v (millisecond precision must survive)", got, want)
	}
}

// Malformed input must report false rather than silently decoding to the
// epoch, which a caller would read as "very old" and filter out.
func TestTimeRejectsMalformedIDs(t *testing.T) {
	for _, bad := range []string{"", "TOOSHORT", strings.Repeat("!", 26)} {
		if _, ok := Time(bad); ok {
			t.Errorf("Time(%q) accepted a malformed id", bad)
		}
	}
}

// Lexical order is time order — the property the whole event log depends on.
func TestTimeAgreesWithLexicalOrder(t *testing.T) {
	early := At(time.UnixMilli(1_700_000_000_000))
	late := At(time.UnixMilli(1_700_000_060_000))
	if !(early < late) {
		t.Fatalf("lexical order broken: %q should sort before %q", early, late)
	}
	et, _ := Time(early)
	lt, _ := Time(late)
	if !et.Before(lt) {
		t.Errorf("decoded times disagree with lexical order: %v vs %v", et, lt)
	}
}
