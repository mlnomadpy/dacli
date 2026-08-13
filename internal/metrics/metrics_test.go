package metrics

import (
	"testing"
	"time"
)

// Named windows are the comparison unit used by the deterministic scenario
// harness. Missing observations stay missing: zero is a real measured value,
// not a substitute for absent runtime data.
func TestCompareNamedScenarioWindowsRejectsMissingOrFabricatedData(t *testing.T) {
	before := Summarise(Window{Name: "before"}, []Run{
		{Task: "one", Outcome: "failed", Elapsed: time.Minute},
	})
	after := Summarise(Window{Name: "after"}, []Run{
		{Task: "one", Outcome: "ok", Elapsed: 30 * time.Second, OutputTokens: ptr(1200)},
	})

	if before.Window.Name != "before" || after.Window.Name != "after" {
		t.Fatalf("named windows lost in reports: %q, %q", before.Window.Name, after.Window.Name)
	}
	if before.Tokens.Output != nil || before.Tokens.Samples != 0 {
		t.Fatalf("missing tokens were fabricated: %+v", before.Tokens)
	}
	if after.Tokens.Output == nil || *after.Tokens.Output != 1200 || after.Tokens.Samples != 1 {
		t.Fatalf("measured tokens missing: %+v", after.Tokens)
	}
	if before.Failures.Samples != 1 || before.Failures.Classes["failed"] != 1 {
		t.Fatalf("failure class data missing: %+v", before.Failures)
	}
	if after.Failures.Samples != 0 || after.Failures.Classes == nil {
		t.Fatalf("zero failures must be measured, with an object not missing data: %+v", after.Failures)
	}
}

func ptr[T any](v T) *T { return &v }
