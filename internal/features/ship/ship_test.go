package ship

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// shipEnv sets up a git repo on main with a workspace holding one DONE task,
// and returns the repo dir. DACLI_AGENT is cleared so the acting identity is
// root (rw) regardless of who runs the suite.
func shipEnv(t *testing.T) (string, *workspace.Workspace) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("DACLI_AGENT", "")
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")

	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	return dir, w
}

func newCtx(dir string) (*clikit.Ctx, *bytes.Buffer) {
	var out bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: dir}, &out
}

// The pipeline shells accept and integrate (stubbed here), then commits the
// .dacli record staging ONLY .dacli — never `git add -A`. The proof: an
// untracked non-.dacli file is left untouched by the record commit.
func TestShipPipelineRecordsOnlyDacli(t *testing.T) {
	dir, w := shipEnv(t)

	// A stray untracked code file that a `git add -A` would sweep in.
	if err := os.WriteFile(filepath.Join(dir, "stray.txt"), []byte("uncommitted\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, w *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, nil); err != nil {
		t.Fatalf("ship: %v\n%s", err, out.String())
	}

	// accept --all then integrate --tasks <ulid> --into main. The ref is the
	// task's globally-unique ULID, never a bare (per-project) seq.
	tk := findDone(t, w)
	if len(calls) != 2 {
		t.Fatalf("expected accept + integrate, got %d calls: %v", len(calls), calls)
	}
	if got := strings.Join(calls[0], " "); got != "accept --all" {
		t.Errorf("step 1 = %q, want \"accept --all\"", got)
	}
	wantIntegrate := "integrate --tasks " + tk.ID + " --into main"
	if got := strings.Join(calls[1], " "); got != wantIntegrate {
		t.Errorf("step 2 = %q, want %q", got, wantIntegrate)
	}

	// The record commit landed on main and its message names ship.
	if msg := gitAt(t, dir, "log", "-1", "--format=%s"); !strings.Contains(msg, "ship: record") {
		t.Errorf("no ship record commit on HEAD: %q", msg)
	}
	// It staged the workspace record...
	files := gitAt(t, dir, "show", "--name-only", "--format=", "HEAD")
	if !strings.Contains(files, ".dacli/") {
		t.Errorf("record commit did not include .dacli: %q", files)
	}
	// ...and NOT the stray code file: it is still untracked (never git add -A).
	if status := gitAt(t, dir, "status", "--porcelain", "stray.txt"); !strings.HasPrefix(status, "??") {
		t.Errorf("stray.txt was swept into the commit (status %q) — ship must stage only .dacli", status)
	}
	// No --push: the command is printed, not run.
	if !strings.Contains(out.String(), "not pushed") {
		t.Errorf("expected a not-pushed notice:\n%s", out.String())
	}
}

// findDone returns the (single) done task shipEnv seeds, so a test can name its
// ULID — the ref ship now passes to integrate.
func findDone(t *testing.T, w *workspace.Workspace) *store.Task {
	t.Helper()
	tk, err := store.FindTask(w, "1")
	if err != nil {
		t.Fatalf("find done task: %v", err)
	}
	return tk
}

// --dry-run prints the plan and executes nothing: no shell-out, no commit.
func TestShipDryRunExecutesNothing(t *testing.T) {
	dir, w := shipEnv(t)
	before := gitAt(t, dir, "rev-parse", "HEAD")

	var called bool
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, w *workspace.Workspace, args ...string) (string, error) {
		called = true
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--dry-run"}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if called {
		t.Error("dry-run shelled a subcommand — it must execute nothing")
	}
	if after := gitAt(t, dir, "rev-parse", "HEAD"); after != before {
		t.Error("dry-run created a commit")
	}
	tk := findDone(t, w)
	s := out.String()
	for _, want := range []string{"dry-run", "accept --all", "integrate --tasks " + tk.ID + " --into main", "git add .dacli"} {
		if !strings.Contains(s, want) {
			t.Errorf("dry-run plan missing %q:\n%s", want, s)
		}
	}
}

// A conflict (integrate blocks the task) stops ship BEFORE the record commit and
// push — never a half-ship. Simulated by the integrate stub moving the task to
// blocked, exactly as a real conflict would.
func TestShipStopsOnConflict(t *testing.T) {
	dir, w := shipEnv(t)
	before := gitAt(t, dir, "rev-parse", "HEAD")

	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "integrate" {
			tk, err := store.FindTask(wk, "1")
			if err != nil {
				t.Fatal(err)
			}
			if err := store.MoveTask(wk, tk, model.StatusBlocked); err != nil {
				t.Fatal(err)
			}
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	err := cmdShip(ctx, nil)
	if err == nil {
		t.Fatalf("expected a stop on conflict; ship returned nil\n%s", out.String())
	}
	if code := clikit.ExitCode(err); code != 3 {
		t.Errorf("conflict stop exit = %d, want 3 (refused)", code)
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("stop reason not surfaced: %v", err)
	}
	// Nothing committed, nothing pushed.
	if after := gitAt(t, dir, "rev-parse", "HEAD"); after != before {
		t.Error("ship committed the record despite a conflict — that is a half-ship")
	}
	_ = w
}

// A genuine (non-conflict) integrate failure — reported by integrate as a
// non-zero exit — stops ship BEFORE the record commit and push. Nothing is
// half-shipped even though no task is blocked (the old bug swallowed this to
// exit 0 and shipped a partial record anyway).
func TestShipStopsOnIntegrateError(t *testing.T) {
	dir, _ := shipEnv(t)
	before := gitAt(t, dir, "rev-parse", "HEAD")

	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "integrate" {
			return "integrated 0 branch(es) into main before the error\n", fmt.Errorf("exit status 1")
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, nil); err == nil {
		t.Fatalf("expected a stop on integrate error; ship returned nil\n%s", out.String())
	}
	if after := gitAt(t, dir, "rev-parse", "HEAD"); after != before {
		t.Error("ship committed the record despite an integrate failure — that is a half-ship")
	}
	if strings.Contains(out.String(), "pushed") && !strings.Contains(out.String(), "not pushed") {
		t.Errorf("ship pushed despite an integrate failure:\n%s", out.String())
	}
}

// The record commit message reports branches ACTUALLY merged (parsed from
// integrate's output), not the raw done-task count. Here two tasks are done but
// integrate reports only one merged (the other had no branch), so the message
// must say 1, not 2.
func TestShipRecordMessageReportsActualMerges(t *testing.T) {
	dir, w := shipEnv(t)
	// Seed a second done task so the done set is 2.
	tk2, err := store.CreateTask(w, "a-root", "p", "Feature B", store.TaskOpts{Accept: []string{"b"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk2, model.StatusDone); err != nil {
		t.Fatal(err)
	}

	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "integrate" {
			// Two done tasks, but only one branch actually merged.
			return "integrated 1 branch(es) into main, no conflicts\n", nil
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, nil); err != nil {
		t.Fatalf("ship: %v\n%s", err, out.String())
	}
	msg := gitAt(t, dir, "log", "-1", "--format=%s")
	if !strings.Contains(msg, "integrating 1 task(s)") {
		t.Errorf("record message = %q, want it to report 1 branch actually merged (not the done count 2)", msg)
	}
}

// doneRefs emits each task's globally-unique ULID, so a done set spanning two
// projects that both have a seq-1 task does NOT collapse to an ambiguous bare
// "1" — the regression that made `dacli ship` unable to integrate in any
// multi-project workspace.
func TestDoneRefsQualifiesAcrossProjects(t *testing.T) {
	dir, w := shipEnv(t)
	_ = dir
	// A second project, whose first task is also seq 1.
	if _, err := store.CreateProject(w, "a-root", "Q", "q", "g", ""); err != nil {
		t.Fatal(err)
	}
	tq, err := store.CreateTask(w, "a-root", "q", "Other", store.TaskOpts{Accept: []string{"c"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tq, model.StatusDone); err != nil {
		t.Fatal(err)
	}

	done, err := store.ListTasks(w, "", model.StatusDone)
	if err != nil {
		t.Fatal(err)
	}
	refs := doneRefs(done)
	if len(refs) != 2 {
		t.Fatalf("expected 2 done tasks across projects, got %d: %v", len(refs), refs)
	}
	for _, r := range refs {
		if r == "1" {
			t.Errorf("doneRefs emitted a bare per-project seq %q — ambiguous across projects", r)
		}
		// Each ref must resolve to exactly one task (no ambiguity error).
		if _, err := store.FindTask(w, r); err != nil {
			t.Errorf("ref %q does not resolve unambiguously: %v", r, err)
		}
	}
}
