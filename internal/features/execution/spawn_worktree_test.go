package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

func initExecGitRepo(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"add", "."},
		{"commit", "-qm", "initial"},
	} {
		if out, err := gitx.Run(root, args...); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}

func TestSpawnWorktreeStartsFromConfiguredBaseNotOperatorHead(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	remote := filepath.Join(t.TempDir(), "origin.git")
	if _, err := gitx.Run(w.Root, "init", "--bare", "-q", remote); err != nil {
		t.Fatal(err)
	}
	base, err := gitx.Run(w.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "remote", "add", "origin", remote); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "push", "-q", "-u", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "checkout", "-q", "-b", "operator-feature"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, "unrelated.txt"), []byte("operator only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "add", "unrelated.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "commit", "-q", "-m", "unrelated operator work"); err != nil {
		t.Fatal(err)
	}

	project, err := store.LoadProject(w, testProject)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConfigureProjectLanding(project, model.LandingPolicy{Mode: model.LandingPR, Base: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	task := mustTask(t, w, "Exact base spawn", store.TaskOpts{Claims: []string{"observed-base.txt"}})
	bin := filepath.Join(t.TempDir(), "record-base")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\ngit rev-parse HEAD > observed-base.txt\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustRuntime(t, w, store.Runtime{Name: "base-recorder", Binary: bin, Mode: "stdin"})

	ctx, _, _ := newCtx(w.Root)
	if err := cmdSpawn(ctx, []string{"--task", task.ID, "--runtime", "base-recorder", "--grant", "rw", "--cooperative", "--worktree"}); err != nil {
		t.Fatal(err)
	}
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	observed, err := os.ReadFile(filepath.Join(wt, "observed-base.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(observed)) != strings.TrimSpace(base) {
		t.Fatalf("spawn base = %s, want configured main %s", strings.TrimSpace(string(observed)), strings.TrimSpace(base))
	}
	if _, err := os.Stat(filepath.Join(wt, "unrelated.txt")); !os.IsNotExist(err) {
		t.Fatalf("spawn inherited operator feature content: %v", err)
	}
	runs, err := os.ReadDir(w.RunsDir())
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs = %d, err %v", len(runs), err)
	}
	raw, err := os.ReadFile(filepath.Join(w.RunDir(runs[0].Name()), "worktree-base.json"))
	if err != nil {
		t.Fatal(err)
	}
	var recorded store.TaskWorktreeBase
	if err := json.Unmarshal(raw, &recorded); err != nil {
		t.Fatal(err)
	}
	if recorded.Commit != strings.TrimSpace(base) || recorded.Branch != "main" || recorded.Ref != "refs/remotes/origin/main" || recorded.Source != "fresh-origin" {
		t.Fatalf("recorded worktree base = %#v", recorded)
	}
}

func TestResolveSpawnWorkDirResumesSameTaskWorktreeWithoutFlag(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Resume here", store.TaskOpts{})
	branch := taskBranch(task)
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if _, err := gitx.AddWorktree(w.Root, wt, branch, "main"); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(wt, "internal", "features")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, isolated, err := resolveSpawnWorkDir(w, task, nested, false)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedPath(got) != resolvedPath(wt) || !isolated {
		t.Fatalf("resolved (%q, %v), want same-task worktree (%q, true)", got, isolated, wt)
	}
	if head, err := gitx.Run(got, "branch", "--show-current"); err != nil || strings.TrimSpace(head) != branch {
		t.Fatalf("resolved checkout branch = %q, err %v; want %q", strings.TrimSpace(head), err, branch)
	}
}

func TestValidateReviewTargetRefusesMissingLocalTaskBranch(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Missing review branch", store.TaskOpts{})
	f, err := clikit.ParseFlags([]string{"--review"})
	if err != nil {
		t.Fatal(err)
	}

	err = validateReviewTarget(w, task, f)
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("exit = %d, want policy refusal 3: %v", clikit.ExitCode(err), err)
	}
	for _, want := range []string{task.ID, taskBranch(task), "create or restore it"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal missing %q: %v", want, err)
		}
	}
}

func TestSpawnFromTaskWorktreeRunsAndEditsOnlyThere(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Resume runtime", store.TaskOpts{Claims: []string{"runtime-cwd.txt", "runtime-branch.txt", "resumed-edit.txt"}})
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if _, err := gitx.AddWorktree(w.Root, wt, taskBranch(task), "main"); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "record-cwd")
	script := "#!/bin/sh\npwd > runtime-cwd.txt\ngit branch --show-current > runtime-branch.txt\nprintf resumed > resumed-edit.txt\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	mustRuntime(t, w, store.Runtime{Name: "cwd-recorder", Binary: bin, Mode: "stdin"})

	ctx, _, _ := newCtx(wt)
	if err := cmdSpawn(ctx, []string{"--task", task.ID, "--runtime", "cwd-recorder", "--grant", "rw", "--cooperative"}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"runtime-cwd.txt":    "claim-sandbox",
		"runtime-branch.txt": taskBranch(task),
		"resumed-edit.txt":   "resumed",
	} {
		raw, err := os.ReadFile(filepath.Join(wt, name))
		if err != nil {
			t.Fatalf("worktree %s: %v", name, err)
		}
		got := strings.TrimSpace(string(raw))
		if name == "runtime-cwd.txt" {
			got = resolvedPath(got)
			if got == resolvedPath(wt) || !strings.HasPrefix(got, resolvedPath(filepath.Join(w.WorktreesDir(), ".claim-sandboxes"))+string(filepath.Separator)) {
				t.Errorf("%s = %q, want independent claim sandbox rather than canonical %q", name, got, wt)
			}
			continue
		}
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
		if _, err := os.Stat(filepath.Join(w.Root, name)); !os.IsNotExist(err) {
			t.Errorf("%s escaped into main checkout (stat err %v)", name, err)
		}
	}
}

// A reclaimed terminal child leaves an audited root transfer on the task
// worktree. A later supervise correction must run in that same checkout and
// publish a fresh run record for it: otherwise its child starts in main and
// its governed commit is inevitably refused as an unrelated owner.
func TestSuperviseCorrectionResumesRootReclaimedTaskWorktreeAcrossTurns(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Correct reclaimed work", store.TaskOpts{Accept: []string{"verified"}})
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if _, err := gitx.AddWorktree(w.Root, wt, taskBranch(task), "main"); err != nil {
		t.Fatal(err)
	}

	// The historical terminal child and its root transfer model the recovery
	// state that prompted this correction; the supervision child must not be
	// sent back to the main checkout just because workspace.Find redirects
	// durable state there.
	if err := os.MkdirAll(w.RunDir("01KZRECLAIMED000000000001"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir("01KZRECLAIMED000000000001"), "worktree.txt"), []byte(wt+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir("01KZRECLAIMED000000000001"), "proc.txt"), procmon.Record{
		RunID: "01KZRECLAIMED000000000001", Child: "a-terminal", Task: task.ID, Claims: []string{"claimed.txt"}, Outcome: "failed",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir("01KZRECLAIMED000000000001"), "worktree-transfer.txt"), []byte("version: 1\nworktree: "+wt+"\nbranch: "+taskBranch(task)+"\nprior_run: 01KZRECLAIMED000000000001\nprior_owner: a-terminal\nnew_owner: a-root\nclaims: claimed.txt\ntransferred_at: now\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(t.TempDir(), "record-supervise-cwd")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\npwd >> supervise-cwds.txt\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustRuntime(t, w, store.Runtime{Name: "cwd-recorder", Binary: bin, Mode: "stdin"})

	// Root owns the recovery transfer and invokes supervise from the shared
	// checkout. The terminal run remains the durable evidence that identifies
	// this task's registered checkout.
	ctx, _, _ := newCtx(w.Root)
	err := cmdSupervise(ctx, []string{"--task", task.ID, "--runtime", "cwd-recorder", "--grant", "rw", "--claim", "claimed.txt,supervise-cwds.txt", "--cooperative", "--max-turns", "2"})
	if err == nil || !strings.Contains(err.Error(), "stalled after 2 turns") {
		t.Fatalf("supervise result = %v, want bounded unmet correction loop", err)
	}

	raw, err := os.ReadFile(filepath.Join(wt, "supervise-cwds.txt"))
	if err != nil {
		t.Fatal(err)
	}
	turns := strings.Fields(string(raw))
	if len(turns) != 2 {
		t.Fatalf("supervise turns = %q, want two runtime invocations", raw)
	}
	for turn, got := range turns {
		resolved := resolvedPath(got)
		if resolved == resolvedPath(wt) || !strings.HasPrefix(resolved, resolvedPath(filepath.Join(w.WorktreesDir(), ".claim-sandboxes"))+string(filepath.Separator)) {
			t.Fatalf("turn %d cwd = %q, want independent claim sandbox for canonical %q", turn+1, got, wt)
		}
	}
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		t.Fatal(err)
	}
	worktreeRuns := 0
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(w.RunDir(entry.Name()), "worktree.txt"))
		if err == nil && resolvedPath(strings.TrimSpace(string(raw))) == resolvedPath(wt) {
			worktreeRuns++
		}
	}
	if worktreeRuns != 3 { // reclaimed terminal run plus both correction turns
		t.Fatalf("runs recorded for reclaimed worktree = %d, want 3", worktreeRuns)
	}
}

func TestResolveSpawnWorkDirKeepsMainCheckoutBehavior(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Main stays main", store.TaskOpts{})
	other := mustTask(t, w, "Unrelated", store.TaskOpts{})
	otherWT := w.WorktreePath(other.Project, other.Seq, other.Slug)
	if _, err := gitx.AddWorktree(w.Root, otherWT, taskBranch(other), "main"); err != nil {
		t.Fatal(err)
	}

	got, isolated, err := resolveSpawnWorkDir(w, task, w.Root, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != w.Root || isolated {
		t.Fatalf("resolved (%q, %v), want main checkout (%q, false)", got, isolated, w.Root)
	}
}

func TestResolveSpawnWorkDirRefusesMismatchedTaskWorktree(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Wanted", store.TaskOpts{})
	other := mustTask(t, w, "Wrong checkout", store.TaskOpts{})
	otherWT := w.WorktreePath(other.Project, other.Seq, other.Slug)
	if _, err := gitx.AddWorktree(w.Root, otherWT, taskBranch(other), "main"); err != nil {
		t.Fatal(err)
	}

	_, _, err := resolveSpawnWorkDir(w, task, otherWT, false)
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("exit = %d, want policy refusal 3: %v", clikit.ExitCode(err), err)
	}
	for _, want := range []string{taskBranch(other), "dacli spawn --task " + task.ID + " --worktree"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not name %q: %v", want, err)
		}
	}
}

func TestResolveSpawnWorkDirRefusesAmbiguousTaskWorktrees(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "Ambiguous", store.TaskOpts{})
	branch := taskBranch(task)
	wts := []gitx.Worktree{{Path: "/tmp/a", Branch: branch}, {Path: "/tmp/b", Branch: branch}}

	_, _, err := resolveSpawnWorkDirFrom(w, task, "/tmp/a", false, wts)
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("exit = %d, want policy refusal 3: %v", clikit.ExitCode(err), err)
	}
	if want := "git worktree remove <duplicate-path>"; !strings.Contains(err.Error(), want) {
		t.Fatalf("refusal does not name recovery command %q: %v", want, err)
	}
}
