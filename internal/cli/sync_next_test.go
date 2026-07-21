package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The cooperative multi-agent loop: a read-only child claims and reports via
// events; the owner's sync materializes both; nothing raced because nothing
// shared a file.
func TestSyncMaterializesChildEvents(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Audit the batch path", "--project", "p", "--accept", "writers listed")

	// Release ownership so the child's claim is meaningful.
	// (Simulate a child: agentid falls back to root when DACLI_AGENT is
	// unset, so we drive the child's side through the event log directly —
	// the same files a real ro child would produce.)
	run(t, dir, 0, "task", "claim", "001")

	// A child finding arrives as an event...
	// Use note add under a fake ro identity by writing the event through the
	// block/finding CLI path is rw here; instead append via events: the ro
	// path is exercised in eventlog itself. Here we verify sync applies a
	// pending finding event end to end.
	w, id, err := openWorkspace(&Ctx{Cwd: dir, Stdout: os.Stdout, Stderr: os.Stderr})
	if err != nil {
		t.Fatal(err)
	}
	_ = id
	tk := findTask(t, dir, "001")
	appendEvent(t, w, "a-child01", "finding", tk, "Batch job bypasses the service layer\n\ncron/settle.go:112 writes directly.")
	appendEvent(t, w, "a-child01", "comment", tk, "starting on the classification pass")

	// Pending events are visible in status before any sync.
	st := run(t, dir, 0, "status")
	if !strings.Contains(st, "pending events: 2") {
		t.Errorf("pending events not surfaced:\n%s", st)
	}

	// And the finding is already in the brief — reads fold pending events.
	briefOut := run(t, dir, 0, "context", "001")
	if !strings.Contains(briefOut, "bypasses the service layer") {
		t.Errorf("pending finding missing from brief:\n%s", briefOut)
	}
	if !strings.Contains(briefOut, "> **a-child01") {
		t.Errorf("finding not attributed to the child:\n%s", briefOut)
	}

	// Sync materializes: finding → note (attributed to the child), comment →
	// Log line, events marked applied.
	out := run(t, dir, 0, "sync")
	if !strings.Contains(out, "2 applied") {
		t.Errorf("sync did not apply both events:\n%s", out)
	}
	show := run(t, dir, 0, "task", "show", "001")
	if !strings.Contains(show, "finding by a-child01") || !strings.Contains(show, "starting on the classification") {
		t.Errorf("log lines missing after sync:\n%s", show)
	}

	// Idempotent: a second sync applies nothing.
	out = run(t, dir, 0, "sync")
	if !strings.Contains(out, "0 applied") {
		t.Errorf("second sync re-applied events:\n%s", out)
	}
}

// next: MoSCoW first, then the critical path; SS overlaps; degradation
// announced when estimates are missing.
func TestNextSchedules(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Audit writers", "--project", "p",
		"--priority", "must", "--estimate", "2,5,14", "--accept", "a")
	run(t, dir, 0, "task", "add", "Build the shim", "--project", "p",
		"--priority", "must", "--estimate", "3,6,15", "--accept", "a", "--depends-on", "001")
	run(t, dir, 0, "task", "add", "Polish docs afterwards", "--project", "p",
		"--priority", "could", "--estimate", "1,2,3", "--accept", "a")

	out := run(t, dir, 0, "next")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.Contains(lines[0], "001-audit-writers") {
		t.Errorf("first pick should be the ready must on the critical path:\n%s", out)
	}
	if strings.Contains(out, "002-build-the-shim") {
		t.Errorf("FS-blocked task recommended:\n%s", out)
	}
	if strings.Contains(out, "polish-docs") {
		t.Errorf("a could recommended while a must is ready:\n%s", out)
	}

	// Finish the audit; the shim becomes the recommendation.
	run(t, dir, 0, "task", "check", "001", "--all")
	run(t, dir, 0, "task", "done", "001")
	out = run(t, dir, 0, "next")
	if !strings.Contains(out, "002-build-the-shim") {
		t.Errorf("unblocked must not recommended:\n%s", out)
	}

	// Missing estimate → announced fallback, never a fake critical path.
	run(t, dir, 0, "task", "add", "Unestimated work item", "--project", "p", "--priority", "must", "--accept", "a")
	out = run(t, dir, 0, "next")
	if !strings.Contains(out, "falling back to MoSCoW") {
		t.Errorf("estimate degradation not announced:\n%s", out)
	}
}

func TestRiskAndGlossaryReachTheBrief(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "a")

	out := run(t, dir, 0, "risk", "add", "Batch may bypass the service layer",
		"--project", "p", "--impact", "high", "--likelihood", "high",
		"--indicator", "diffs only after 02:00 UTC")
	if !strings.Contains(out, "rank 1") || !strings.Contains(out, "no action plan") {
		t.Errorf("rank-1 without action not warned:\n%s", out)
	}
	run(t, dir, 0, "glossary", "p", "--term", "balance", "--def", "the authoritative row, not the API cache")

	briefOut := run(t, dir, 0, "context", "001")
	if !strings.Contains(briefOut, "watch for: diffs only after 02:00 UTC") {
		t.Errorf("risk indicator missing from brief:\n%s", briefOut)
	}
	if !strings.Contains(briefOut, "**balance**") {
		t.Errorf("glossary term missing from brief:\n%s", briefOut)
	}
}

func TestJSONOutputIsParseable(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--priority", "must", "--accept", "a")

	out := runJSON(t, dir, "task", "list")
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("task list --json not parseable: %v\n%s", err, out)
	}
	if len(tasks) != 1 || tasks[0]["priority"] != "must" || tasks[0]["acceptance_total"] != float64(1) {
		t.Errorf("task json = %v", tasks)
	}

	out = runJSON(t, dir, "context", "001")
	var b struct {
		TaskID   string `json:"task_id"`
		Sections []struct{ Title string }
	}
	if err := json.Unmarshal([]byte(out), &b); err != nil {
		t.Fatalf("context --json not parseable: %v\n%s", err, out)
	}
	if !strings.HasPrefix(b.TaskID, "t-") || len(b.Sections) == 0 {
		t.Errorf("context json = %+v", b)
	}
}
