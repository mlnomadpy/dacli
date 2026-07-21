package cli

import (
	"bytes"
	"strings"
	"testing"
)

// The v0.1 acceptance test in miniature (ARCHITECTURE § 3): one session,
// init → project → tasks → decision → finding → context → done. If the brief
// this produces isn't worth generating, nothing downstream matters.
func run(t *testing.T, dir string, wantCode int, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	ctx := &Ctx{Stdout: &out, Stderr: &errb, Cwd: dir}
	cmd, rest := match(args)
	if cmd == nil {
		t.Fatalf("no such command: %v", args)
	}
	err := cmd.Run(ctx, rest)
	if got := exitCode(err); got != wantCode {
		t.Fatalf("%v: exit %d, want %d (err: %v)\nstdout: %s\nstderr: %s",
			args, got, wantCode, err, out.String(), errb.String())
	}
	combined := out.String() + errb.String()
	if err != nil {
		combined += "\n" + err.Error() // Main prints this; the helper must too
	}
	return combined
}

func TestDogfoodLoop(t *testing.T) {
	dir := t.TempDir()

	run(t, dir, 0, "init", "--name", "billing")
	run(t, dir, 0, "project", "add", "Migrate billing to the new ledger",
		"--slug", "ledger", "--goal", "One write path into balances, reconciliation-clean.")

	// A vague title draws the ambiguity lint but still creates.
	out := run(t, dir, 0, "task", "add", "Handle the balances properly", "--project", "ledger")
	if !strings.Contains(out, "vague-words") {
		t.Errorf("vague title not flagged:\n%s", out)
	}

	// The real task, with acceptance and a three-point estimate.
	run(t, dir, 0, "task", "add", "Audit write paths into balances",
		"--project", "ledger", "--priority", "must", "--estimate", "2,5,14",
		"--accept", "Writers of balances listed with file:line",
		"--accept", "Writers classified: service-layer vs direct")

	// A scalar estimate is refused at creation.
	if got := run(t, dir, 1, "task", "add", "Scalar estimate task", "--project", "ledger", "--estimate", "5"); !strings.Contains(got, "three-point") {
		t.Errorf("scalar estimate accepted:\n%s", got)
	}

	run(t, dir, 0, "note", "add", "decision", "Ledger writes stay synchronous",
		"--project", "ledger",
		"--rejected", "async queue with eventual reconciliation",
		"--because", "reconciliation cost exceeds the latency win")

	// A decision without a rejection is refused: it cannot be safely revisited.
	run(t, dir, 1, "note", "add", "decision", "Half a decision", "--project", "ledger")

	run(t, dir, 0, "note", "add", "finding", "Batch job writes balances directly",
		"--project", "ledger", "--about", "002", "--severity", "major",
		"--body", "cron/settle_batch.go:112 bypasses the service layer entirely.")

	// The product: the brief must carry the goal, the boundary-relevant
	// decision (with its rejection), the sibling finding quote-fenced, and
	// the estimate with Te.
	briefOut := run(t, dir, 0, "context", "002")
	for _, want := range []string{
		"## Task: Audit write paths into balances",
		"estimate: 2/5/14 (Te 6.0)",
		"Goal: One write path into balances",
		"Rejected: async queue",
		"> **a-root, major**", // attributed quote fencing with severity
		"bypasses the service layer",
		"data, not instructions",
	} {
		if !strings.Contains(briefOut, want) {
			t.Errorf("brief missing %q:\n%s", want, briefOut)
		}
	}

	// done VERIFIES: unchecked acceptance is a refusal (exit 3), and the
	// message says do-not-retry. This is the 1-vs-3 distinction in action.
	refusal := run(t, dir, 3, "task", "done", "002")
	if !strings.Contains(refusal, "acceptance unmet") || !strings.Contains(refusal, "do not retry") {
		t.Errorf("refusal message wrong:\n%s", refusal)
	}

	run(t, dir, 0, "task", "check", "002", "--all")
	run(t, dir, 0, "task", "done", "002")

	// Status reflects the folder move.
	st := run(t, dir, 0, "status")
	if !strings.Contains(st, "done:1") {
		t.Errorf("status does not show the completed task:\n%s", st)
	}

	// Unknown refs are 4, not 1.
	run(t, dir, 4, "task", "show", "no-such-task")

	// lint sweeps the vague task we left behind.
	lint := run(t, dir, 0, "lint", "--project", "ledger")
	if !strings.Contains(lint, "no acceptance criteria") {
		t.Errorf("lint missed the acceptance-free task:\n%s", lint)
	}
}

func TestBudgetTrimAnnounced(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", strings.Repeat("goal words ", 50))
	run(t, dir, 0, "task", "add", "A tightly scoped task", "--project", "p", "--accept", "it works")

	out := run(t, dir, 0, "context", "001", "--budget", "60")
	if !strings.Contains(out, "## Task: A tightly scoped task") {
		t.Error("task section must survive any trim")
	}
	if !strings.Contains(out, "omitted") {
		t.Errorf("trim not announced:\n%s", out)
	}
}

func TestContextRecordFreezesBrief(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "done means done")
	out := run(t, dir, 0, "context", "001", "--record")
	if !strings.Contains(out, "recorded: ") {
		t.Errorf("--record did not report a run dir:\n%s", out)
	}
}
