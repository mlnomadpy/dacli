package spm

import (
	"math"
	"testing"
)

func hasCat(fs []Finding, c Category, term string) bool {
	for _, f := range fs {
		if f.Category == c && equalFold(f.Term, term) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 32
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// The canonical bad task description. Three agents given this produce three
// different implementations, which is the entire argument for the linter.
func TestScanCatchesTheCanonicalVagueTask(t *testing.T) {
	const text = "The system should handle all the errors properly and process them efficiently."
	fs := Scan(text, Options{})

	for _, tc := range []struct {
		cat  Category
		term string
	}{
		{CatVague, "handle"},
		{CatVague, "properly"},
		{CatVague, "process"},
		{CatVague, "efficiently"},
		{CatQualifier, "all"},
		{CatIndirect, "should"},
	} {
		if !hasCat(fs, tc.cat, tc.term) {
			t.Errorf("missed %s %q", tc.cat, tc.term)
		}
	}
}

func TestScanDefaultsToModerateAndAbove(t *testing.T) {
	fs := Scan("It should be done.", Options{})
	if hasCat(fs, CatPronoun, "It") {
		t.Error("pronoun surfaced at default severity; minor categories are too noisy to default on")
	}
	fs = Scan("It should be done.", Options{MinSeverity: SevMinor})
	if !hasCat(fs, CatPronoun, "It") {
		t.Error("pronoun not surfaced even at MinSeverity=minor")
	}
}

func TestScanIgnoresCode(t *testing.T) {
	const text = "Write the parser.\n\n```go\nfunc process(all items []Item) {}\n```\n\nCall `handle()` when done.\n"
	fs := Scan(text, Options{})
	for _, f := range fs {
		if f.Line >= 3 && f.Line <= 5 {
			t.Errorf("flagged %q inside a fenced code block at line %d", f.Term, f.Line)
		}
	}
	if hasCat(fs, CatVague, "handle") {
		t.Error("flagged `handle()` inside an inline code span")
	}
}

func TestScanPositionsAreCorrect(t *testing.T) {
	const text = "ok line\nthe system should work\n"
	fs := Scan(text, Options{})
	var found bool
	for _, f := range fs {
		if f.Term == "should" {
			found = true
			if f.Line != 2 {
				t.Errorf("line = %d, want 2", f.Line)
			}
			if f.Col != 12 {
				t.Errorf("col = %d, want 12", f.Col)
			}
		}
	}
	if !found {
		t.Fatal(`"should" not found`)
	}
}

func TestScanPrefersLongestMatch(t *testing.T) {
	fs := Scan("Make it more efficient than the old one.", Options{})
	if !hasCat(fs, CatComparative, "more efficient") {
		t.Error(`"more efficient" should match the comparative rule as a whole phrase`)
	}
}

func TestPERT(t *testing.T) {
	p := ThreePoint{Optimistic: 2, Probable: 5, Pessimistic: 14}
	if got := p.Expected(); math.Abs(got-6) > 1e-9 {
		t.Errorf("Te = %g, want 6", got)
	}
	if got := p.Sigma(); math.Abs(got-2) > 1e-9 {
		t.Errorf("sigma = %g, want 2", got)
	}
	if got := p.Triangular(); math.Abs(got-7) > 1e-9 {
		t.Errorf("triangular = %g, want 7", got)
	}
	lo, hi := p.ConfidenceRange(1)
	if math.Abs(lo-4) > 1e-9 || math.Abs(hi-8) > 1e-9 {
		t.Errorf("1-sigma range = %g..%g, want 4..8", lo, hi)
	}
}

func TestPERTRejectsInvalidOrdering(t *testing.T) {
	if err := (ThreePoint{Optimistic: 9, Probable: 5, Pessimistic: 14}).Valid(); err == nil {
		t.Error("optimistic > probable should be rejected")
	}
	if err := (ThreePoint{Optimistic: 1, Probable: 15, Pessimistic: 14}).Valid(); err == nil {
		t.Error("probable > pessimistic should be rejected")
	}
}

// A 6-unit estimate at elicitation stage must report as 3–12. Reporting it as
// "6" is the specific dishonesty the cone exists to prevent.
func TestConeOfUncertainty(t *testing.T) {
	lo, hi := ConeRange(6, StageElicitation)
	if math.Abs(lo-3) > 1e-9 || math.Abs(hi-12) > 1e-9 {
		t.Errorf("elicitation range = %g..%g, want 3..12", lo, hi)
	}
	lo, hi = ConeRange(6, StageApproach)
	if math.Abs(lo-4.8) > 1e-9 || math.Abs(hi-7.5) > 1e-9 {
		t.Errorf("approach range = %g..%g, want 4.8..7.5", lo, hi)
	}
}

func TestIsEpic(t *testing.T) {
	p := ThreePoint{Optimistic: 2, Probable: 5, Pessimistic: 14}
	if !p.IsEpic(10) {
		t.Error("pessimistic 14 against a 10 budget should be an epic")
	}
	if p.IsEpic(20) {
		t.Error("pessimistic 14 against a 20 budget should not be an epic")
	}
}

// Diamond: A(3) → B(2) → D(4) and A(3) → C(5) → D(4).
// A-C-D = 12 is critical; B carries 3 units of slack.
func TestCPMDiamond(t *testing.T) {
	nodes := []Node{{"A", 3}, {"B", 2}, {"C", 5}, {"D", 4}}
	edges := []Edge{
		{From: "A", To: "B", Type: FS},
		{From: "A", To: "C", Type: FS},
		{From: "B", To: "D", Type: FS},
		{From: "C", To: "D", Type: FS},
	}
	net, err := ComputeCPM(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(net.Duration-12) > 1e-9 {
		t.Errorf("duration = %g, want 12", net.Duration)
	}
	if got := net.Schedules["B"].Slack; math.Abs(got-3) > 1e-9 {
		t.Errorf("B slack = %g, want 3", got)
	}
	for _, id := range []string{"A", "C", "D"} {
		if !net.Schedules[id].Critical {
			t.Errorf("%s should be on the critical path", id)
		}
	}
	if net.Schedules["B"].Critical {
		t.Error("B should not be on the critical path")
	}
	want := []string{"A", "C", "D"}
	if len(net.CriticalPath) != len(want) {
		t.Fatalf("critical path = %v, want %v", net.CriticalPath, want)
	}
	for i := range want {
		if net.CriticalPath[i] != want[i] {
			t.Fatalf("critical path = %v, want %v", net.CriticalPath, want)
		}
	}
}

// Start-start is what makes two tasks genuinely parallel-safe; a plain
// blocked_by would have serialized these.
func TestCPMStartStartAllowsOverlap(t *testing.T) {
	net, err := ComputeCPM(
		[]Node{{"A", 5}, {"B", 3}},
		[]Edge{{From: "A", To: "B", Type: SS}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := net.Schedules["B"].EarlyStart; got != 0 {
		t.Errorf("B early start = %g, want 0 (SS permits immediate overlap)", got)
	}
	if math.Abs(net.Duration-5) > 1e-9 {
		t.Errorf("duration = %g, want 5", net.Duration)
	}
}

func TestCPMFinishFinish(t *testing.T) {
	net, err := ComputeCPM(
		[]Node{{"A", 5}, {"B", 3}},
		[]Edge{{From: "A", To: "B", Type: FF}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := net.Schedules["B"].EarlyFinish; math.Abs(got-5) > 1e-9 {
		t.Errorf("B early finish = %g, want 5", got)
	}
	if got := net.Schedules["B"].EarlyStart; math.Abs(got-2) > 1e-9 {
		t.Errorf("B early start = %g, want 2", got)
	}
}

func TestCPMDetectsCycle(t *testing.T) {
	_, err := ComputeCPM(
		[]Node{{"A", 1}, {"B", 1}},
		[]Edge{{From: "A", To: "B"}, {From: "B", To: "A"}},
	)
	if err != ErrCycle {
		t.Errorf("err = %v, want ErrCycle", err)
	}
}

func TestCPMRejectsUnknownTask(t *testing.T) {
	_, err := ComputeCPM([]Node{{"A", 1}}, []Edge{{From: "A", To: "ghost"}})
	if err == nil {
		t.Error("edge to an unknown task should be rejected")
	}
}

// The scheduling primitive: what should my next N children work on.
func TestParallelizablePrefersZeroSlack(t *testing.T) {
	net, err := ComputeCPM(
		[]Node{{"A", 3}, {"B", 2}, {"C", 5}, {"D", 4}},
		[]Edge{{From: "A", To: "B"}, {From: "A", To: "C"}, {From: "B", To: "D"}, {From: "C", To: "D"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	got := net.Parallelizable(map[string]bool{"A": true}, 2)
	if len(got) != 2 {
		t.Fatalf("got %v, want 2 tasks", got)
	}
	if got[0] != "C" {
		t.Errorf("first pick = %q, want \"C\" (zero slack beats B's slack of 3)", got[0])
	}
}
