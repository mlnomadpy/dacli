package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fault injection: force the failures the code normally never sees, and check
// that the tool says so instead of quietly carrying on. Several of this
// codebase's worst bugs were swallowed errors that became silent wrong answers
// — a corrupt file skipped by every list path, a task that "exists" with no
// content — and none of them were reachable by a test that only exercises the
// happy path (dacli 204).

// A task file whose frontmatter is gone still appears in `task list`, because
// status comes from the folder and seq/slug from the filename. It lists as a
// hollow row: no id, no title, no acceptance criteria. Nothing surfaced this —
// `doctor`, whose entire job is workspace integrity, reported "no anti-patterns
// detected" over a workspace whose task had silently lost its identity. That is
// the exact shape of the CRLF and newline-injection data-loss bugs.
func TestDoctorDetectsAHollowTaskFile(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a genuine goal for this project")
	run(t, dir, 0, "task", "add", "a real task", "--project", "p", "--accept", "it works")

	open := filepath.Join(dir, ".dacli", "projects", "p", "tasks", "open")
	entries, err := os.ReadDir(open)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected a task file in %s: %v", open, err)
	}
	victim := filepath.Join(open, entries[0].Name())

	// Clean workspace: doctor must be quiet.
	if out := run(t, dir, 0, "doctor"); !strings.Contains(out, "no anti-patterns") {
		t.Fatalf("a healthy workspace should be quiet, got:\n%s", out)
	}

	// Now destroy the file's structure, the way a bad rewrite or a CRLF
	// checkout would.
	if err := os.WriteFile(victim, []byte("this file lost its frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := run(t, dir, 0, "doctor")
	if strings.Contains(out, "no anti-patterns") {
		t.Fatalf("doctor must not call a workspace healthy when a task file has lost its identity; got:\n%s", out)
	}
	if !strings.Contains(out, "corrupt-object") {
		t.Errorf("doctor should report the integrity problem as corrupt-object; got:\n%s", out)
	}
	if !strings.Contains(out, entries[0].Name()) && !strings.Contains(out, "001") {
		t.Errorf("doctor should name the offending task; got:\n%s", out)
	}
}

// A write into a directory the process cannot write must FAIL, loudly. The
// danger is not the error — it is a command that reports success on a path
// where it wrote nothing.
func TestTaskAddFailsLoudlyWhenTheStoreIsUnwritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission bits do not constrain writes")
	}
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a genuine goal for this project")
	// Status folders are created lazily on first write, so seed one task to
	// bring open/ into existence before making it unwritable.
	run(t, dir, 0, "task", "add", "the first task", "--project", "p", "--accept", "it works")

	open := filepath.Join(dir, ".dacli", "projects", "p", "tasks", "open")
	before, err := os.ReadDir(open)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", open, err)
	}
	if err := os.Chmod(open, 0o555); err != nil {
		t.Skipf("cannot make the directory read-only here: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(open, 0o755) })

	// Any non-zero exit is acceptable; silently reporting success is not.
	var out, errb strings.Builder
	ctx := &Ctx{Stdout: &out, Stderr: &errb, Cwd: dir}
	cmd, rest := match([]string{"task", "add", "a task that cannot be written", "--project", "p", "--accept", "it works"})
	if cmd == nil {
		t.Fatal("no task add command")
	}
	if err := cmd.Run(ctx, rest); err == nil {
		t.Fatalf("task add into an unwritable store must fail; it reported success:\n%s", out.String())
	}
	// And it must not have left a half-written artifact behind — the atomic
	// write's temp file included.
	if after, rerr := os.ReadDir(open); rerr == nil && len(after) != len(before) {
		t.Errorf("a failed write changed the store: %d file(s) before, %d after", len(before), len(after))
	}
}
