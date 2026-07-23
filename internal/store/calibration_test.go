package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func calibWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	return w
}

// writeRun drops one run dir with an invocation.txt made of the given lines.
// Dir names are ULIDs in the real store; the walk reads them in lexical order,
// so a caller controls chronology by choosing lexically ascending ids.
func writeRun(t *testing.T, w *workspace.Workspace, id string, lines string) {
	t.Helper()
	dir := w.RunDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir run %s: %v", id, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invocation.txt"), []byte(lines), 0o644); err != nil {
		t.Fatalf("write invocation %s: %v", id, err)
	}
}

// A verify run is chronologically NEWER than the completing spawn (it checks
// the findings the spawn produced), so under a naive last-run-wins join it
// would clobber the task's real implementation band. It must not: verify seats
// are checks, not actuals.
func TestRunRecordsVerifyDoesNotClobberImplementerBand(t *testing.T) {
	w := calibWorkspace(t)
	writeRun(t, w, "run-a-spawn", "task: t-1\nrole: maintainer\nmodel: opus\nruntime: claude-code\n")
	writeRun(t, w, "run-b-verify", "verify_panel_seat: gemini\ntask: t-1\nrole: verifier\nmodel: -\nruntime: gemini\nclaim: x\n")

	recs := runRecords(w)
	got := recs["t-1"].band
	want := Band{Role: "maintainer", Model: "opus", Runtime: "claude-code"}
	if got != want {
		t.Fatalf("band = %v, want %v (verify seat clobbered the implementer band)", got, want)
	}
}

// A newer run whose invocation predates the role/model lines bands empty; it
// must not wipe the real band an earlier completing run recorded.
func TestRunRecordsEmptyBandDoesNotClobber(t *testing.T) {
	w := calibWorkspace(t)
	writeRun(t, w, "run-a-spawn", "task: t-1\nrole: maintainer\nmodel: opus\nruntime: claude-code\n")
	writeRun(t, w, "run-b-legacy", "run: x\ntask: t-1\n") // no role/model/runtime → empty band

	got := runRecords(w)["t-1"].band
	want := Band{Role: "maintainer", Model: "opus", Runtime: "claude-code"}
	if got != want {
		t.Fatalf("band = %v, want %v (empty band clobbered the real one)", got, want)
	}
}

// A supervise run now records role/model in canonical form, so its band is a
// real, non-empty implementation band that the join keeps.
func TestRunRecordsSuperviseBandIsKept(t *testing.T) {
	w := calibWorkspace(t)
	writeRun(t, w, "run-a", "run: x\nsupervise_turn: 2/3\ntask: t-9\nrole: maintainer\nmodel: sonnet\nruntime: claude-code\n")

	got, ok := TaskBand(w, "t-9")
	if !ok {
		t.Fatalf("TaskBand(t-9) not found — supervise band was dropped")
	}
	want := Band{Role: "maintainer", Model: "sonnet", Runtime: "claude-code"}
	if got != want {
		t.Fatalf("band = %v, want %v", got, want)
	}
}

// A task that ONLY ever had a verify run has no implementation actual, so it
// carries no band (TaskBand reports absent) rather than a bogus verifier band.
func TestRunRecordsVerifyOnlyTaskHasNoBand(t *testing.T) {
	w := calibWorkspace(t)
	writeRun(t, w, "run-a-verify", "verify_panel_seat: gemini\ntask: t-2\nrole: verifier\nmodel: -\nruntime: gemini\n")

	if b, ok := TaskBand(w, "t-2"); ok {
		t.Fatalf("TaskBand(t-2) = %v, want absent (verify-only task must not band)", b)
	}
}

// TokensPerRun groups by ROLE ALONE — coarser than the full role×model×
// runtime Band — so two samples sharing a role but differing model/runtime
// still combine into one figure, which is what a caller sizing a FUTURE spawn
// (no model/runtime picked yet) needs.
func TestTokensPerRunGroupsByRoleAcrossModelsAndRuntimes(t *testing.T) {
	samples := []CalibSample{
		{Te: 2, Tokens: 100, Band: Band{Role: "fixer", Model: "sonnet", Runtime: "claude-code"}},
		{Te: 2, Tokens: 300, Band: Band{Role: "fixer", Model: "opus", Runtime: "codex"}},
		{Te: 2, Tokens: 900, Band: Band{Role: "go-auditor", Model: "sonnet", Runtime: "claude-code"}},
	}
	med, p10, p90, n := TokensPerRun(samples, "fixer")
	if n != 2 {
		t.Fatalf("n = %d, want 2 (both fixer samples, ignoring model/runtime)", n)
	}
	if med != 200 {
		t.Fatalf("median = %v, want 200", med)
	}
	if p10 <= 0 || p90 <= 0 || p10 > p90 {
		t.Fatalf("p10/p90 = %v/%v, want a valid ascending range", p10, p90)
	}

	if _, _, _, n := TokensPerRun(samples, "go-auditor"); n != 1 {
		t.Fatalf("go-auditor n = %d, want 1", n)
	}

	// A sample with no token actual (HasTokens false) must not count.
	untokened := append(append([]CalibSample{}, samples...), CalibSample{Te: 2, Tokens: 0, Band: Band{Role: "fixer"}})
	if _, _, _, n := TokensPerRun(untokened, "fixer"); n != 2 {
		t.Fatalf("n = %d, want 2 (the zero-token sample must not count)", n)
	}
}

// An unknown role has no samples at all — TokensPerRun must degrade to n=0
// rather than panic or return a bogus zero-looking-calibrated figure.
func TestTokensPerRunUnknownRoleIsEmpty(t *testing.T) {
	samples := []CalibSample{{Te: 2, Tokens: 100, Band: Band{Role: "fixer"}}}
	med, p10, p90, n := TokensPerRun(samples, "nonexistent")
	if n != 0 || med != 0 || p10 != 0 || p90 != 0 {
		t.Fatalf("want all-zero for an absent role, got med=%v p10=%v p90=%v n=%v", med, p10, p90, n)
	}
}
