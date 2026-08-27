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
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/eventlog"
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

// shipEnv sets up a git repo on main with a workspace holding one OPEN task —
// the wave `dacli accept` closes during ship (closeAllOpen simulates that in the
// shellDacli stub) — and returns the repo dir. Seeding the task open, not done,
// mirrors reality: an agent leaves its task proposed-for-acceptance, and ship's
// own accept step is what closes it. DACLI_AGENT is cleared so the acting
// identity is root (rw) regardless of who runs the suite.
func shipEnv(t *testing.T) (string, *workspace.Workspace) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	// Unset, not blank: since dacli 288 a present-but-empty DACLI_AGENT is a
	// lost token that fails closed, so only an actually-unset var resolves root.
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok {
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	// Disable git's background maintenance in THIS repo (task 256). A later
	// `git commit` can trigger auto gc/maintenance, which git detaches into a
	// child process (gc.autoDetach defaults true) that keeps writing under
	// .git AFTER the git command — and the test body — has returned. t.TempDir's
	// deferred RemoveAll then races that detached writer and fails the whole test
	// at cleanup with "directory not empty", even though every assertion passed.
	// It only flaked in CI (whose global git config enables the auto path) and
	// stayed green locally, so it looked like a defect in the change under test.
	// These repo-local settings override any global config: gc never runs, and if
	// it somehow did it runs in the FOREGROUND, so no git subprocess can outlive
	// the test to race cleanup. This is a real teardown fix, not a retry.
	gitAt(t, dir, "config", "gc.auto", "0")
	gitAt(t, dir, "config", "gc.autoDetach", "false")
	gitAt(t, dir, "config", "maintenance.auto", "false")
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
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	return dir, w
}

// closeAllOpen simulates `dacli accept` closing the wave: every open task moves
// to done, exactly as accept does after it verifies a proposed task. Ship's wave
// scope (done-after-accept minus done-before) is then those tasks. Tests wire it
// into the shellDacli stub's "accept" branch.
func closeAllOpen(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	open, err := store.ListTasks(w, "", model.StatusOpen)
	if err != nil {
		t.Fatal(err)
	}
	for _, tk := range open {
		if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
			t.Fatal(err)
		}
	}
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
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk) // accept closes the wave (the seeded open task)
		case "integrate":
			ctx.Result = commandresult.Integration{}
		}
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
	if got := strings.Join(calls[0], " "); got != "accept --all --force --defer-landing" {
		t.Errorf("step 1 = %q, want \"accept --all --force --defer-landing\"", got)
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

// `ship --pr [--no-merge]` forwards the PR-first flags to `dacli integrate`, so
// the wave lands as reviewable PRs instead of a local merge. The default (no
// --pr) forwards nothing, keeping the local-merge path unchanged.
func TestShipForwardsPRFlagsToIntegrate(t *testing.T) {
	dir, w := shipEnv(t)

	var integrateArgs []string
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk)
		case "integrate":
			ctx.Result = commandresult.Integration{Merged: 1}
			integrateArgs = args
			return "integrated 1 branch(es) into main, no conflicts\n", nil
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--pr", "--no-merge"}); err != nil {
		t.Fatalf("ship --pr --no-merge: %v\n%s", err, out.String())
	}
	joined := strings.Join(integrateArgs, " ")
	if !strings.Contains(joined, "--pr") {
		t.Errorf("ship --pr did not forward --pr to integrate: %q", joined)
	}
	if !strings.Contains(joined, "--no-merge") {
		t.Errorf("ship --no-merge did not forward --no-merge to integrate: %q", joined)
	}
	_ = w
}

// `ship --pr --auto` forwards --auto to `dacli integrate`, so the wave's PRs are
// set to GitHub auto-merge (merge when CI passes) — hands-off integration.
func TestShipForwardsAutoToIntegrate(t *testing.T) {
	dir, _ := shipEnv(t)

	var integrateArgs []string
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk)
		case "integrate":
			ctx.Result = commandresult.Integration{Open: 1}
			integrateArgs = args
			return "queued 1 PR(s) for auto-merge targeting main — GitHub merges each when CI passes (hands-off)\n", nil
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--pr", "--auto"}); err != nil {
		t.Fatalf("ship --pr --auto: %v\n%s", err, out.String())
	}
	joined := strings.Join(integrateArgs, " ")
	if !strings.Contains(joined, "--pr") || !strings.Contains(joined, "--auto") {
		t.Errorf("ship --pr --auto did not forward --pr --auto to integrate: %q", joined)
	}
}

// The default ship (no --pr) forwards NO PR flags — the local-merge path is
// unchanged.
func TestShipDefaultForwardsNoPRFlags(t *testing.T) {
	dir, _ := shipEnv(t)

	var integrateArgs []string
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk)
		case "integrate":
			ctx.Result = commandresult.Integration{}
			integrateArgs = args
		}
		return "", nil
	}

	ctx, _ := newCtx(dir)
	if err := cmdShip(ctx, nil); err != nil {
		t.Fatalf("ship: %v", err)
	}
	if joined := strings.Join(integrateArgs, " "); strings.Contains(joined, "--pr") {
		t.Errorf("default ship forwarded a PR flag: %q", joined)
	}
}

// TestShipIntegratesOnlyTheWaveNotFullHistory is the task-261 regression: on a
// workspace whose done set is MUCH larger than the wave — many tasks closed by
// prior runs, already integrated — ship must pass ONLY the task this run closes
// to integrate, never the whole history. Before the fix it integrated every done
// task ever, getting more dangerous the longer a project ran.
func TestShipIntegratesOnlyTheWaveNotFullHistory(t *testing.T) {
	dir, w := shipEnv(t) // seeds one OPEN task — the wave accept will close
	// A long history: many tasks closed (and integrated) by PRIOR runs.
	var historical []string
	for i := 0; i < 6; i++ {
		tk, err := store.CreateTask(w, "a-root", "p", fmt.Sprintf("Old %d", i), store.TaskOpts{Accept: []string{"x"}})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
			t.Fatal(err)
		}
		historical = append(historical, tk.ID)
	}

	var integrateArgs []string
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk) // closes only the seeded open task — the wave
		case "integrate":
			ctx.Result = commandresult.Integration{Merged: 1}
			integrateArgs = args
			return "integrated 1 branch(es) into main, no conflicts\n", nil
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, nil); err != nil {
		t.Fatalf("ship: %v\n%s", err, out.String())
	}

	joined := strings.Join(integrateArgs, " ")
	wave := findDone(t, w) // the seeded task, now done
	if !strings.Contains(joined, "--tasks "+wave.ID) {
		t.Errorf("integrate did not target the wave task %s: %q", wave.ID, joined)
	}
	for _, h := range historical {
		if strings.Contains(joined, h) {
			t.Errorf("ship integrated a long-settled done task %s — it must integrate only the wave: %q", h, joined)
		}
	}
}

// TestShipExplicitTasksWindow covers the acceptance's "or an explicit window":
// `ship --tasks <ref>` integrates exactly the named done task even when it was
// closed by a prior run (so it is NOT in this run's wave) — the operator's
// escape hatch to re-integrate a specific already-done task.
func TestShipExplicitTasksWindow(t *testing.T) {
	dir, w := shipEnv(t)
	// task1 was closed by a PRIOR run: already done, so not in this run's wave.
	tk, err := store.FindTask(w, "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusDone); err != nil {
		t.Fatal(err)
	}

	var integrateArgs []string
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		if args[0] == "integrate" {
			ctx.Result = commandresult.Integration{Merged: 1}
			integrateArgs = args
			return "integrated 1 branch(es) into main, no conflicts\n", nil
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--tasks", tk.ID}); err != nil {
		t.Fatalf("ship --tasks: %v\n%s", err, out.String())
	}
	if joined := strings.Join(integrateArgs, " "); !strings.Contains(joined, "--tasks "+tk.ID) {
		t.Errorf("explicit --tasks window not integrated: %q", joined)
	}
}

// A --tasks ref that resolves to a NOT-done task is a usage error (exit 2): ship
// integrates a done task's branch, and a not-done ref has none to land.
func TestShipExplicitTasksRejectsNotDone(t *testing.T) {
	dir, w := shipEnv(t) // the seeded task is OPEN
	tk, err := store.FindTask(w, "1")
	if err != nil {
		t.Fatal(err)
	}

	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		return "", nil
	}

	ctx, _ := newCtx(dir)
	err = cmdShip(ctx, []string{"--tasks", tk.ID})
	if err == nil {
		t.Fatal("expected a usage error for a not-done --tasks ref")
	}
	if code := clikit.ExitCode(err); code != 2 {
		t.Errorf("not-done --tasks exit = %d, want 2 (usage)", code)
	}
}

// findDone returns the (single) task shipEnv seeds, so a test can name its ULID
// — the ref ship passes to integrate once the wave closes it.
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
	_ = w
	s := out.String()
	// Without --tasks, accept has not run yet, so the plan cannot name the wave;
	// it says so honestly and never claims it would integrate the full done set.
	for _, want := range []string{"dry-run", "accept --all", "integrate --tasks", "the wave accept closes this run", "not re-integrated", "git add .dacli"} {
		if !strings.Contains(s, want) {
			t.Errorf("dry-run plan missing %q:\n%s", want, s)
		}
	}
}

// The real ship pipeline accepts a pending proposal before it resolves an
// explicit window. Its dry-run must project that same transition rather than
// rejecting the task's pre-accept active status.
func TestShipDryRunExplicitProposedActiveWindowProjectsAccept(t *testing.T) {
	dir, w := shipEnv(t)
	tk, err := store.FindTask(w, "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusActive); err != nil {
		t.Fatal(err)
	}
	proposal, err := eventlog.Append(w, "a-root", model.EventProposeStatus, tk.ID, "", "propose: done")
	if err != nil {
		t.Fatal(err)
	}
	taskBefore, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatal(err)
	}
	proposalBefore, err := os.ReadFile(proposal.Path)
	if err != nil {
		t.Fatal(err)
	}

	before := gitAt(t, dir, "rev-parse", "HEAD")
	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--dry-run", "--tasks", tk.ID}); err != nil {
		t.Fatalf("ship --dry-run --tasks active proposed task: %v\n%s", err, out.String())
	}
	if after := gitAt(t, dir, "rev-parse", "HEAD"); after != before {
		t.Error("dry-run created a commit")
	}
	fresh, err := store.FindTask(w, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Status != model.StatusActive {
		t.Errorf("dry-run changed task status to %s, want active", fresh.Status)
	}
	if taskAfter, err := os.ReadFile(tk.Path); err != nil || !bytes.Equal(taskAfter, taskBefore) {
		t.Errorf("dry-run changed task record: %v", err)
	}
	if proposalAfter, err := os.ReadFile(proposal.Path); err != nil || !bytes.Equal(proposalAfter, proposalBefore) {
		t.Errorf("dry-run changed pending proposal: %v", err)
	}
	for _, want := range []string{"accept --all", "integrate --tasks " + tk.ID, "explicit --tasks window"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("dry-run plan missing %q:\n%s", want, out.String())
		}
	}
}

func TestShipDryRunExplicitWindowReportsAcceptanceRefusal(t *testing.T) {
	dir, w := shipEnv(t)
	tk, err := store.CreateTask(w, "a-root", "p", "Unverifiable", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tk, model.StatusActive); err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-root", model.EventProposeStatus, tk.ID, "", "propose: done"); err != nil {
		t.Fatal(err)
	}

	ctx, _ := newCtx(dir)
	err = cmdShip(ctx, []string{"--dry-run", "--tasks", tk.ID})
	if err == nil {
		t.Fatal("expected dry-run to refuse an active task accept would skip")
	}
	if got := err.Error(); !strings.Contains(got, "no acceptance criteria — nothing to verify") {
		t.Errorf("dry-run error = %q, want accept's no-acceptance reason", got)
	}
}

func TestShipDryRunReportsConfiguredPRPolicyAndGates(t *testing.T) {
	dir, w := shipEnv(t)
	gitAt(t, dir, "branch", "release")
	p, _ := store.LoadProject(w, "p")
	_ = store.ConfigureProjectLanding(p, model.LandingPolicy{Mode: model.LandingPR, Base: "release"})
	_ = store.SaveProject(p)
	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--dry-run", "--project", "p", "--pr"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"mode=pr", "base=release", "override=true", "PR action=", "required checks and reviews"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("dry-run missing %q:\n%s", want, out.String())
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
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk) // the wave task is now done — in this run's wave
		case "integrate":
			ctx.Result = commandresult.Integration{}
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
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk)
		case "integrate":
			ctx.Result = commandresult.Integration{}
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
	// Seed a second task so the wave accept closes is 2.
	if _, err := store.CreateTask(w, "a-root", "p", "Feature B", store.TaskOpts{Accept: []string{"b"}}); err != nil {
		t.Fatal(err)
	}

	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		switch args[0] {
		case "accept":
			closeAllOpen(t, wk) // wave of 2
		case "integrate":
			// Two tasks in the wave, but only one branch actually merged.
			ctx.Result = commandresult.Integration{Merged: 1}
			return "landed a single branch cleanly\n", nil
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

// The shipEnv repo must disable git's background auto-maintenance, or a
// detached gc/maintenance child can outlive a `git commit` and write under
// .git while t.TempDir's RemoveAll is deleting the tree — the "directory not
// empty" cleanup flake that failed these tests in CI (task 256). This guards
// the mitigation so a future edit cannot silently reintroduce the race; it
// fails before the shipEnv config lines exist (the keys read back empty).
func TestShipEnvDisablesGitAutoMaintenance(t *testing.T) {
	dir, _ := shipEnv(t)
	for _, kv := range [][2]string{
		{"gc.auto", "0"},
		{"gc.autoDetach", "false"},
		{"maintenance.auto", "false"},
	} {
		if got := gitAt(t, dir, "config", "--get", kv[0]); got != kv[1] {
			t.Errorf("shipEnv repo %s = %q, want %q — a detached git process can race t.TempDir cleanup", kv[0], got, kv[1])
		}
	}
}

// doneRefs emits each task's globally-unique ULID, so a done set spanning two
// projects that both have a seq-1 task does NOT collapse to an ambiguous bare
// "1" — the regression that made `dacli ship` unable to integrate in any
// multi-project workspace.
func TestDoneRefsQualifiesAcrossProjects(t *testing.T) {
	dir, w := shipEnv(t)
	_ = dir
	// shipEnv seeds project p's task open; close it so the done set spans both
	// projects' seq-1 tasks.
	tp, err := store.FindTask(w, "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveTask(w, tp, model.StatusDone); err != nil {
		t.Fatal(err)
	}
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

// `ship --push --release <tag>` cuts a tagged release AFTER the push, shelling
// `dacli github release <project> <tag> --target <into>` so the release tags the
// branch that just reached origin (task 223). The push is real (a bare remote),
// integrate is stubbed.
func TestShipCutsReleaseAfterPush(t *testing.T) {
	dir, _ := shipEnv(t)
	remote := t.TempDir()
	gitAt(t, remote, "init", "--bare", "-q")
	gitAt(t, dir, "remote", "add", "origin", remote)

	var releaseArgs []string
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "integrate" {
			ctx.Result = commandresult.Integration{Merged: 1}
			return "integrated 1 branch(es) into main, no conflicts\n", nil
		}
		if len(args) > 1 && args[0] == "github" && args[1] == "release" {
			releaseArgs = args
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--project", "p", "--push", "--release", "v1.0.0"}); err != nil {
		t.Fatalf("ship --push --release: %v\n%s", err, out.String())
	}
	joined := strings.Join(releaseArgs, " ")
	for _, want := range []string{"github release p v1.0.0", "--target main"} {
		if !strings.Contains(joined, want) {
			t.Errorf("release call %q missing %q", joined, want)
		}
	}
}

// --release requires --push: a release of un-pushed commits would tag a state
// the remote does not have. The refusal fires UP FRONT (exit 3), before accept
// integrates anything — so a wave is never half-shipped then refused.
func TestShipReleaseRequiresPush(t *testing.T) {
	dir, _ := shipEnv(t)
	before := gitAt(t, dir, "rev-parse", "HEAD")

	var called bool
	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		called = true
		return "", nil
	}

	ctx, out := newCtx(dir)
	err := cmdShip(ctx, []string{"--project", "p", "--release", "v1.0.0"})
	if err == nil {
		t.Fatalf("expected a refusal for --release without --push\n%s", out.String())
	}
	if code := clikit.ExitCode(err); code != 3 {
		t.Errorf("--release-without-push exit = %d, want 3 (refused)", code)
	}
	if called {
		t.Error("ship shelled a step before refusing the release precondition — it must refuse up front")
	}
	if after := gitAt(t, dir, "rev-parse", "HEAD"); after != before {
		t.Error("ship committed despite refusing the release precondition")
	}
}

// --release cannot ride --pr: PR-first merges to the target asynchronously on
// GitHub's clock, so a release cut now could tag the target before the wave's
// PRs merge. Refused up front.
func TestShipReleaseRefusesWithPR(t *testing.T) {
	dir, _ := shipEnv(t)
	ctx, out := newCtx(dir)
	err := cmdShip(ctx, []string{"--project", "p", "--push", "--pr", "--release", "v1.0.0"})
	if err == nil {
		t.Fatalf("expected a refusal for --release with --pr\n%s", out.String())
	}
	if code := clikit.ExitCode(err); code != 3 {
		t.Errorf("--release-with-pr exit = %d, want 3 (refused)", code)
	}
}

// --release needs --project to resolve the linked repo — a usage error (exit 2)
// when it is omitted.
func TestShipReleaseRequiresProject(t *testing.T) {
	dir, _ := shipEnv(t)
	ctx, out := newCtx(dir)
	err := cmdShip(ctx, []string{"--push", "--release", "v1.0.0"})
	if err == nil {
		t.Fatalf("expected a usage error for --release without --project\n%s", out.String())
	}
	if code := clikit.ExitCode(err); code != 2 {
		t.Errorf("--release-without-project exit = %d, want 2 (usage)", code)
	}
}

// The dry-run plan shows the release step (nothing executed) when --release is
// passed, so the operator can preview the tag and target.
func TestShipDryRunShowsReleaseStep(t *testing.T) {
	dir, _ := shipEnv(t)
	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--dry-run", "--project", "p", "--push", "--release", "v1.0.0"}); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	s := out.String()
	for _, want := range []string{"5. release", "github release p v1.0.0", "--target main", "generated notes"} {
		if !strings.Contains(s, want) {
			t.Errorf("dry-run plan missing %q:\n%s", want, s)
		}
	}
}

// The record branch must reach the REMOTE. Step 3 puts the record on its own
// ref precisely so trunk stays code-only, which means the record is not an
// ancestor of the current branch — and step 4 pushed only the current branch,
// so every commit of the trajectory stayed on the machine while the output
// said "pushed main to origin". Silent history loss that reads as a completed
// push (dacli 323).
func TestShipPushesTheRecordBranchNotJustTrunk(t *testing.T) {
	dir, _ := shipEnv(t)
	remote := t.TempDir()
	gitAt(t, remote, "init", "--bare", "-q")
	gitAt(t, dir, "remote", "add", "origin", remote)

	orig := shellDacli
	defer func() { shellDacli = orig }()
	shellDacli = func(ctx *clikit.Ctx, wk *workspace.Workspace, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "integrate" {
			ctx.Result = commandresult.Integration{Merged: 1}
			return "integrated 1 branch(es) into main, no conflicts\n", nil
		}
		return "", nil
	}

	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--project", "p", "--push", "--record-branch", "dacli-record"}); err != nil {
		t.Fatalf("ship --push --record-branch: %v\n%s", err, out.String())
	}

	// The local ref exists only if step 3 actually recorded something; without
	// that this test would pass vacuously on two absent refs.
	local := gitAt(t, dir, "rev-parse", "dacli-record")
	pushed := gitAt(t, remote, "rev-parse", "dacli-record")
	if local != pushed {
		t.Errorf("record branch did not reach origin: local %s, remote %s", local, pushed)
	}
	// Trunk still goes out too — fixing the record push must not cost the one
	// that already worked.
	if got, want := gitAt(t, remote, "rev-parse", "main"), gitAt(t, dir, "rev-parse", "main"); got != want {
		t.Errorf("main did not reach origin: local %s, remote %s", want, got)
	}
	// And the operator is told BOTH refs went, so the line cannot read as a
	// full push while half the state stayed local.
	if s := out.String(); !strings.Contains(s, "main and dacli-record") {
		t.Errorf("push line must name every ref pushed:\n%s", s)
	}
}

// The --dry-run plan must describe the branch the record ACTUALLY lands on.
// It printed "git add .dacli && git commit" and "git push -u origin main"
// regardless, so the preview of a record-branch workspace described a
// different command than the one that would run.
func TestShipDryRunPlanNamesTheRecordBranch(t *testing.T) {
	dir, _ := shipEnv(t)
	ctx, out := newCtx(dir)
	if err := cmdShip(ctx, []string{"--project", "p", "--push", "--record-branch", "dacli-record", "--dry-run"}); err != nil {
		t.Fatalf("ship --dry-run: %v\n%s", err, out.String())
	}
	s := out.String()
	for _, want := range []string{"onto dacli-record", "git push -u origin main && git push -u origin dacli-record"} {
		if !strings.Contains(s, want) {
			t.Errorf("dry-run plan missing %q:\n%s", want, s)
		}
	}
}
