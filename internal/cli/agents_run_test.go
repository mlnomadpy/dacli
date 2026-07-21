package cli

import (
	"strings"
	"testing"
)

// The full identity loop: spawn a read-only child, act as it via the token,
// verify its writes become events, and that attenuation holds.
func TestSpawnedChildIdentity(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Audit the batch path", "--project", "p", "--accept", "a")

	// Spawn: token on stdout alone, so $(...) captures it.
	out := run(t, dir, 0, "agent", "spawn", "--role", "auditor", "--grant", "ro")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	token := strings.TrimSpace(lines[0])
	if len(token) != 48 {
		t.Fatalf("token line = %q, want 48 hex chars", token)
	}

	// Act as the child.
	t.Setenv("DACLI_AGENT", token)

	who := run(t, dir, 0, "whoami")
	if !strings.Contains(who, "grant: ro") || !strings.Contains(who, "role: auditor") {
		t.Errorf("child identity not resolved: %s", who)
	}

	// A read-only child finishing a task produces a proposal, not a move.
	done := run(t, dir, 0, "task", "done", "001")
	if !strings.Contains(done, "proposed as event") {
		t.Errorf("ro done should be an event: %s", done)
	}

	// Regression (found by the live mock-child demo): an ro finding filed
	// against a seq ref like "001" must resolve to the task id at the write
	// site, or the brief's about-filter never matches it.
	run(t, dir, 0, "note", "add", "finding", "Batch bypasses the service layer",
		"--project", "p", "--about", "001", "--body", "settle.go:112 writes directly")
	briefOut := run(t, dir, 0, "context", "001")
	if !strings.Contains(briefOut, "settle.go:112") {
		t.Errorf("ro child finding (seq-ref about) missing from brief:\n%s", briefOut)
	}

	// A read-only parent cannot mint a read-write child: attenuation, exit 3.
	run(t, dir, 3, "agent", "spawn", "--grant", "rw")

	// Back to root: the tree shows lineage and the child's writes.
	t.Setenv("DACLI_AGENT", "")
	tree := run(t, dir, 0, "agent", "tree")
	if !strings.Contains(tree, "a-root (rw") || !strings.Contains(tree, "auditor") {
		t.Errorf("tree missing lineage:\n%s", tree)
	}
	if !strings.Contains(tree, "2 events") {
		t.Errorf("child's events (proposal + finding) not attributed in tree:\n%s", tree)
	}

	// The owner's sync applies the child's proposal... which is refused-shaped:
	// the task still has an unchecked box, but propose-status applies the
	// move regardless — the owner asked for it by running sync.
	syncOut := run(t, dir, 0, "sync")
	if !strings.Contains(syncOut, "2 applied") {
		t.Errorf("child proposal + finding not applied:\n%s", syncOut)
	}
}

func TestBadTokenRejected(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	t.Setenv("DACLI_AGENT", "deadbeef")
	out := run(t, dir, 1, "whoami")
	if !strings.Contains(out, "not recognized") {
		t.Errorf("bad token error unclear: %s", out)
	}
}

// The shortcut loop: define, dry-run, guarded execution, injection safety,
// and the run event feeding the catalog.
func TestShortcutRun(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "a")

	// No effect → refused at creation, not silently defaulted.
	run(t, dir, 1, "shortcut", "add", "bad", "--command", "true")

	run(t, dir, 0, "shortcut", "add", "greet",
		"--command", "echo hello {{name}}",
		"--effect", "read",
		"--param", "name=world",
		"--summary", "echo test",
		"--why", "exists for the test suite")

	// Dry-run shows the expansion without executing.
	dry := run(t, dir, 0, "run", "greet", "--dry-run")
	if strings.TrimSpace(dry) != "echo hello world" {
		t.Errorf("dry-run = %q", dry)
	}

	// Injection attempt arrives quoted: the shell echoes it as a literal.
	out := run(t, dir, 0, "run", "greet", "--name", "world; touch /tmp/pwned")
	if !strings.Contains(out, "hello world; touch /tmp/pwned") {
		t.Errorf("param not passed as a literal: %q", out)
	}

	// Destructive requires rw AND --confirm.
	run(t, dir, 0, "shortcut", "add", "nuke", "--command", "echo boom", "--effect", "destructive")
	run(t, dir, 3, "run", "nuke")
	if got := run(t, dir, 0, "run", "nuke", "--confirm"); !strings.Contains(got, "boom") {
		t.Errorf("confirmed destructive did not run: %q", got)
	}

	// Uses are derived from run events; the catalog ranks by them.
	list := run(t, dir, 0, "run", "--list")
	if !strings.HasPrefix(strings.TrimSpace(list), "- `dacli run greet`") {
		t.Errorf("catalog should lead with the used shortcut:\n%s", list)
	}

	// And the catalog reaches the brief.
	briefOut := run(t, dir, 0, "context", "001")
	if !strings.Contains(briefOut, "## Shortcuts") || !strings.Contains(briefOut, "dacli run greet") {
		t.Errorf("shortcut catalog missing from brief:\n%s", briefOut)
	}
}

// ask blocks the task; answer unblocks it and leaves a durable note that
// reaches future briefs. The question is transient; the answer is permanent.
func TestAskAnswerLoop(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Build the shim", "--project", "p", "--accept", "a")
	run(t, dir, 0, "task", "claim", "001")

	out := run(t, dir, 0, "ask", "Does the batch job write balances directly?", "--about", "001")
	if !strings.Contains(out, "blocked until answered") {
		t.Errorf("ask did not block: %s", out)
	}
	qid := strings.Fields(strings.TrimPrefix(out, "asked "))[0]

	st := run(t, dir, 0, "task", "list", "--status", "blocked")
	if !strings.Contains(st, "build-the-shim") {
		t.Errorf("task not in blocked:\n%s", st)
	}

	run(t, dir, 0, "answer", qid, "Yes — it bypasses the service layer entirely.", "--as", "finding")

	// Unblocked, and the answer is in the next brief.
	st = run(t, dir, 0, "task", "list", "--status", "active")
	if !strings.Contains(st, "build-the-shim") {
		t.Errorf("task not unblocked after answer:\n%s", st)
	}
	briefOut := run(t, dir, 0, "context", "001")
	if !strings.Contains(briefOut, "bypasses the service layer") {
		t.Errorf("answer not in the brief:\n%s", briefOut)
	}
}
