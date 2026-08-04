package brief

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
)

func doc(fields map[string]string) *mdstore.Doc {
	d := &mdstore.Doc{}
	for k, v := range fields {
		if v != "" {
			d.Front.Set(k, v)
		}
	}
	return d
}

func ids(notes []*mdstore.Doc) []string {
	out := make([]string, len(notes))
	for i, n := range notes {
		out[i], _ = n.Front.Get("id")
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSortFindingsOrdersBySeverityThenTrustThenRecency(t *testing.T) {
	notes := []*mdstore.Doc{
		doc(map[string]string{"id": "f-minor", "severity": "minor", "trust": "confirmed", "created": "2026-01-01T00:00:00Z"}),
		doc(map[string]string{"id": "f-major-old", "severity": "major", "trust": "unverified", "created": "2026-01-01T00:00:00Z"}),
		doc(map[string]string{"id": "f-major-new", "severity": "major", "trust": "unverified", "created": "2026-02-01T00:00:00Z"}),
		doc(map[string]string{"id": "f-major-confirmed", "severity": "major", "trust": "confirmed", "created": "2026-01-01T00:00:00Z"}),
	}
	sortFindings(notes)
	// severity first (major before minor); within major, trust (confirmed beats
	// unverified) beats recency; the two unverified majors break to the newest.
	want := []string{"f-major-confirmed", "f-major-new", "f-major-old", "f-minor"}
	if got := ids(notes); !equalStrings(got, want) {
		t.Fatalf("sortFindings order = %v, want %v", got, want)
	}
}

func TestSortByRecencyNewestFirst(t *testing.T) {
	notes := []*mdstore.Doc{
		doc(map[string]string{"id": "d-old", "created": "2026-01-01T00:00:00Z"}),
		doc(map[string]string{"id": "d-new", "created": "2026-03-01T00:00:00Z"}),
		doc(map[string]string{"id": "d-mid", "created": "2026-02-01T00:00:00Z"}),
		doc(map[string]string{"id": "d-nostamp"}),
	}
	sortByRecency(notes)
	want := []string{"d-new", "d-mid", "d-old", "d-nostamp"}
	if got := ids(notes); !equalStrings(got, want) {
		t.Fatalf("sortByRecency order = %v, want %v", got, want)
	}
}

func TestNamedOmissionCapsEnumeration(t *testing.T) {
	names := make([]string, omissionNameCap+3)
	for i := range names {
		names[i] = fmt.Sprintf("[[f-%d]]", i)
	}
	got := namedOmission("findings", len(names), names)
	if !strings.Contains(got, "+3 more") {
		t.Fatalf("expected a summarized tail, got %q", got)
	}
	if strings.Contains(got, fmt.Sprintf("[[f-%d]]", omissionNameCap)) {
		t.Fatalf("enumerated beyond the cap: %q", got)
	}
	small := namedOmission("decisions", 2, []string{"[[d-a]]", "[[d-b]]"})
	if !strings.Contains(small, "[[d-a]]") || strings.Contains(small, "more") {
		t.Fatalf("a short list must be fully named without a tail summary: %q", small)
	}
	if got := namedOmission("findings", 0, nil); strings.Contains(got, ":") {
		t.Fatalf("an empty name list must degrade to a bare count: %q", got)
	}
}
