package vcs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// unsetAgentEnv clears DACLI_AGENT for the test, restoring whatever the
// process started with. t.Setenv cannot unset a variable, and since dacli 288
// a present-but-empty DACLI_AGENT is a lost token that fails closed rather
// than resolving to root — so a test wanting the root identity must remove
// the variable entirely, not blank it.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok {
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
}

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

// prIntegrateEnv builds a real git repo on main with a workspace holding one
// DONE task whose branch (dacli/001-slug) carries a commit ahead of main, ready
// to integrate. DACLI_AGENT is cleared so the actor is root (rw).
func prIntegrateEnv(t *testing.T) (string, *workspace.Workspace, *store.Task) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	unsetAgentEnv(t)
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
	// A real task branch with a commit ahead of main so a local-merge fallback
	// has something to merge.
	branch := BranchFor(tk)
	gitAt(t, dir, "checkout", "-q", "-b", branch)
	if err := os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "feature.txt")
	gitAt(t, dir, "commit", "-q", "-m", "feature work")
	gitAt(t, dir, "checkout", "-q", "main")
	return dir, w, tk
}

func prCtx(dir string) (*clikit.Ctx, *bytes.Buffer) {
	var out bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: dir}, &out
}

// stubGH captures gh invocations and returns canned output, so the PR path is
// exercised without a live GitHub. It restores the real runner on cleanup.
func stubGH(t *testing.T, fn func(dir string, args ...string) (string, error)) *[][]string {
	t.Helper()
	var calls [][]string
	orig := runGH
	runGH = func(dir string, args ...string) (string, error) {
		calls = append(calls, args)
		return fn(dir, args...)
	}
	t.Cleanup(func() { runGH = orig })
	return &calls
}

func stubPush(t *testing.T, fn func(root, branch string) (string, error)) *[][]string {
	t.Helper()
	var calls [][]string
	orig := pushBranch
	pushBranch = func(root, branch string) (string, error) {
		calls = append(calls, []string{root, branch})
		return fn(root, branch)
	}
	t.Cleanup(func() { pushBranch = orig })
	return &calls
}

// --pr pushes the branch, opens a PR (recorded), and merges via gh pr merge.
func TestIntegratePRPushesOpensAndMerges(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	push := stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/7", nil
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr: %v\n%s", err, out.String())
	}

	if len(*push) != 1 {
		t.Fatalf("expected one push, got %v", *push)
	}
	var sawCreate, sawMerge bool
	for _, c := range *gh {
		joined := strings.Join(c, " ")
		if strings.HasPrefix(joined, "pr create") {
			sawCreate = true
		}
		if strings.HasPrefix(joined, "pr merge") {
			sawMerge = true
			if !strings.Contains(joined, "--squash") || !strings.Contains(joined, "--delete-branch") {
				t.Errorf("gh pr merge missing --squash/--delete-branch: %q", joined)
			}
		}
	}
	if !sawCreate || !sawMerge {
		t.Fatalf("expected gh pr create and pr merge, got %v", *gh)
	}
	// The PR URL was recorded as a comment event on the task.
	events, _ := eventlog.List(w, eventlog.Query{About: tk.ID, Kinds: []model.EventKind{model.EventComment}})
	found := false
	for _, e := range events {
		if strings.Contains(e.Body, "PR opened: https://github.com/acme/widgets/pull/7") {
			found = true
		}
	}
	if !found {
		t.Errorf("PR URL not recorded as a comment event")
	}
	if !strings.Contains(out.String(), "merged via gh") {
		t.Errorf("expected a merged-via-gh notice:\n%s", out.String())
	}
}

// Naming a task on the command line says which BRANCH to merge; it is not a
// claim that the work was accepted. Without --tasks the done filter is
// structural (ListTasks only returns StatusDone), but a named list used to walk
// straight past it — and that is how fourteen tasks whose code had been merged
// for hours stayed in open/ while `dacli next` went on ranking them `must`
// (dacli 257).
func TestIntegrateRefusesATaskThatIsNotDone(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	// Walk it back to open: the branch still exists and still has the commit.
	if err := store.MoveTask(w, tk, model.StatusOpen); err != nil {
		t.Fatal(err)
	}
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		t.Errorf("gh must not be reached for a task that is not done: %v", args)
		return "", nil
	})

	ctx, out := prCtx(dir)
	err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"})
	if err == nil {
		t.Fatalf("integrate merged a task that is not done:\n%s", out.String())
	}
	if code := clikit.ExitCode(err); code != 3 {
		t.Errorf("exit code = %d, want 3 (refused) — the command line is well-formed, the answer is 'not yet'", code)
	}
	if !strings.Contains(err.Error(), tk.Slug) {
		t.Errorf("the refusal must name the task; got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "dacli accept") {
		t.Errorf("the refusal must say which command closes the task; got %q", err.Error())
	}
	if len(*gh) != 0 {
		t.Errorf("gh was called despite the refusal: %v", *gh)
	}
	// Nothing landed: feature.txt is still absent from main.
	if _, serr := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(serr) {
		t.Errorf("a refused integrate still merged the branch")
	}
}

// --force is the deliberate override: the operator knows the record is behind
// and wants the branch merged anyway.
func TestIntegrateForceMergesANotDoneTask(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	if err := store.MoveTask(w, tk, model.StatusOpen); err != nil {
		t.Fatal(err)
	}
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	stubGH(t, func(dir string, args ...string) (string, error) {
		if strings.HasPrefix(strings.Join(args, " "), "pr create") {
			return "https://github.com/acme/widgets/pull/11", nil
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--force", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --force on a not-done task: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "merged via gh") {
		t.Errorf("--force should have merged:\n%s", out.String())
	}
}

// A DONE task is unaffected — the gate must not add friction to the normal path.
func TestIntegrateStillMergesADoneTaskWithoutForce(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t) // already done
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	stubGH(t, func(dir string, args ...string) (string, error) {
		if strings.HasPrefix(strings.Join(args, " "), "pr create") {
			return "https://github.com/acme/widgets/pull/12", nil
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate on a done task must not need --force: %v\n%s", err, out.String())
	}
}

// --no-merge opens the PR and STOPS: gh pr merge is never called.
func TestIntegratePRNoMergeStopsBeforeMerge(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/9", nil
		}
		return "", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--no-merge", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr --no-merge: %v\n%s", err, out.String())
	}
	for _, c := range *gh {
		if len(c) >= 2 && c[0] == "pr" && c[1] == "merge" {
			t.Errorf("--no-merge still called gh pr merge: %v", c)
		}
	}
	if !strings.Contains(out.String(), "left open for human review") {
		t.Errorf("expected a human-review notice:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "none merged (--no-merge)") {
		t.Errorf("expected a none-merged summary:\n%s", out.String())
	}
	// main did not advance (nothing merged): feature.txt is absent.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
		t.Errorf("--no-merge merged the branch into main (feature.txt present)")
	}
}

// THE case the integrator role exists for: the loop already opened a PR when
// the task landed, and `integrate --pr` is the command you then run to merge
// it. `gh pr create` hard-fails on "already exists", and that failure used to
// abort the run before the merge gate was ever reached — so the sanctioned
// merge path could not merge any PR the loop had opened, which is every PR it
// would ever be pointed at (dacli 255).
func TestIntegratePRMergesAnAlreadyOpenPR(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	const existing = "https://github.com/acme/widgets/pull/287"
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "pr view"):
			return "OPEN " + existing + "\n", nil
		case strings.HasPrefix(joined, "pr create"):
			// Verbatim gh behaviour, and the reason this test exists.
			return "a pull request for branch dacli/001-feature-a into main already exists: " + existing,
				fmt.Errorf("exit status 1")
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr against an existing PR: %v\n%s", err, out.String())
	}

	var sawCreate, sawMerge bool
	for _, c := range *gh {
		joined := strings.Join(c, " ")
		if strings.HasPrefix(joined, "pr create") {
			sawCreate = true
		}
		if strings.HasPrefix(joined, "pr merge") {
			sawMerge = true
		}
	}
	if sawCreate {
		t.Errorf("pr create was attempted even though a PR was already open: %v", *gh)
	}
	if !sawMerge {
		t.Fatalf("the merge gate was never reached — the existing PR cannot be merged: %v", *gh)
	}
	if !strings.Contains(out.String(), existing) {
		t.Errorf("the existing PR's URL should be reported:\n%s", out.String())
	}
}

// A CLOSED or MERGED PR on the branch is not a PR to reuse: the probe must fall
// through to create, so a re-opened branch still gets a PR.
func TestIntegratePRDoesNotReuseAClosedPR(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "pr view"):
			return "CLOSED https://github.com/acme/widgets/pull/1\n", nil
		case strings.HasPrefix(joined, "pr create"):
			return "https://github.com/acme/widgets/pull/2", nil
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr: %v\n%s", err, out.String())
	}
	for _, c := range *gh {
		if strings.HasPrefix(strings.Join(c, " "), "pr create") {
			return
		}
	}
	t.Fatalf("a closed PR was treated as reusable; create was never attempted: %v", *gh)
}

// An unreachable GitHub during the probe must not be read as "a PR exists".
// The probe only ever removes a spurious failure; on any unclear answer the
// caller falls through to create and handles that failure as it always did.
func TestIntegratePRProbeFailureFallsThroughToCreate(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "pr view"):
			return "", fmt.Errorf("could not connect to github.com")
		case strings.HasPrefix(joined, "pr create"):
			return "https://github.com/acme/widgets/pull/3", nil
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr: %v\n%s", err, out.String())
	}
	for _, c := range *gh {
		if strings.HasPrefix(strings.Join(c, " "), "pr create") {
			return
		}
	}
	t.Fatalf("a failed probe was treated as 'a PR exists'; create was never attempted: %v", *gh)
}

// A network failure at push falls back to a LOCAL merge with a warning, so the
// wave still lands offline. gh is never reached.
func TestIntegratePRFallsBackToLocalMergeOnPushNetworkError(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) {
		return "fatal: unable to access 'https://github.com/...': Could not resolve host: github.com", fmt.Errorf("exit status 128")
	})
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		t.Errorf("gh must not be called after a push network failure: %v", args)
		return "", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr (fallback): %v\n%s", err, out.String())
	}
	if len(*gh) != 0 {
		t.Errorf("gh was called despite the push network failure: %v", *gh)
	}
	if !strings.Contains(out.String(), "falling back to a local merge") {
		t.Errorf("expected a fallback warning:\n%s", out.String())
	}
	// The local merge landed: feature.txt is now on main.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Errorf("local-merge fallback did not land the branch: %v", err)
	}
}

// A NON-network push failure (e.g. a protected branch) is surfaced, NOT
// silently local-merged.
func TestIntegratePRSurfacesNonNetworkPushError(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) {
		return "remote: error: GH006: Protected branch update failed", fmt.Errorf("exit status 1")
	})
	stubGH(t, func(dir string, args ...string) (string, error) { return "", nil })

	ctx, out := prCtx(dir)
	err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"})
	if err == nil {
		t.Fatalf("expected a hard error for a non-network push failure\n%s", out.String())
	}
	// It did NOT fall back to a local merge.
	if _, statErr := os.Stat(filepath.Join(dir, "feature.txt")); statErr == nil {
		t.Errorf("a non-network push failure was silently local-merged")
	}
}

// --auto sets GitHub's native auto-merge (gh pr merge --auto --merge
// --delete-branch) and STOPS: GitHub merges the PR when CI goes green, so
// nothing is merged locally now (feature.txt stays off main, the branch is not
// deleted locally). The check gate is NOT consulted — GitHub owns it.
func TestIntegratePRAutoSetsAutoMerge(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/11", nil
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
			t.Errorf("--auto must not consult gh pr checks — GitHub owns the gate: %v", args)
		}
		return "Pull request #11 will be automatically merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--auto", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr --auto: %v\n%s", err, out.String())
	}

	var mergeArgs string
	for _, c := range *gh {
		if len(c) >= 2 && c[0] == "pr" && c[1] == "merge" {
			mergeArgs = strings.Join(c, " ")
		}
	}
	if mergeArgs == "" {
		t.Fatalf("expected a gh pr merge call, got %v", *gh)
	}
	for _, want := range []string{"--auto", "--merge", "--delete-branch"} {
		if !strings.Contains(mergeArgs, want) {
			t.Errorf("gh pr merge missing %q for --auto: %q", want, mergeArgs)
		}
	}
	// Nothing merged locally: main did not advance and the branch still exists.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
		t.Errorf("--auto merged the branch into main locally (feature.txt present)")
	}
	// The branch is NOT torn down locally — GitHub owns the pending merge.
	if got := gitAt(t, dir, "branch", "--list", BranchFor(tk)); !strings.Contains(got, BranchFor(tk)) {
		t.Errorf("--auto deleted the local branch before GitHub merged: %q", got)
	}
	if !strings.Contains(out.String(), "auto-merge set") {
		t.Errorf("expected an auto-merge-set notice:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "queued 1 PR(s) for auto-merge") {
		t.Errorf("expected a queued-for-auto-merge summary:\n%s", out.String())
	}
	_ = w
}

// The default (non-auto) --pr path GATES on gh pr checks: a red or pending
// check leaves the PR OPEN and gh pr merge is never called — dacli never blindly
// merges over a failing gate.
func TestIntegratePRLeavesOpenWhenChecksNotPassing(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/13", nil
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
			// A pending/failing check: gh pr checks exits non-zero, non-network.
			return "build\tpending\t0\thttps://...", fmt.Errorf("exit status 8")
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "merge" {
			t.Errorf("gh pr merge must not run when checks are not passing: %v", args)
		}
		return "", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr (checks pending): %v\n%s", err, out.String())
	}
	for _, c := range *gh {
		if len(c) >= 2 && c[0] == "pr" && c[1] == "merge" {
			t.Errorf("checks-not-passing still called gh pr merge: %v", c)
		}
	}
	// main did not advance: feature.txt is absent.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
		t.Errorf("a checks-not-passing PR was merged into main (feature.txt present)")
	}
	if !strings.Contains(out.String(), "PR left open") {
		t.Errorf("expected a PR-left-open notice:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "left 1 PR(s) open") {
		t.Errorf("expected a left-open summary:\n%s", out.String())
	}
}

// The default --pr path MERGES when gh pr checks passes (exit 0). This locks in
// the "merges only PRs whose checks already pass" half of the acceptance.
func TestIntegratePRMergesWhenChecksPass(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	var sawChecks bool
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/15", nil
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
			sawChecks = true
			return "build\tpass\t0\thttps://...", nil // exit 0: all green
		}
		return "merged", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr (checks pass): %v\n%s", err, out.String())
	}
	if !sawChecks {
		t.Errorf("the gated path did not consult gh pr checks")
	}
	var sawMerge bool
	for _, c := range *gh {
		if len(c) >= 2 && c[0] == "pr" && c[1] == "merge" {
			sawMerge = true
		}
	}
	if !sawMerge {
		t.Fatalf("checks-passing PR was not merged: %v", *gh)
	}
	if !strings.Contains(out.String(), "merged via gh") {
		t.Errorf("expected a merged-via-gh notice:\n%s", out.String())
	}
}

// A repo with NO CI configured (`gh pr checks` reports "no checks reported")
// must NOT be treated as passing — a check gate with nothing to gate would
// otherwise merge every PR as if it were green. The PR is left open, and the
// notice must name the absence, not "checks not passing", so it reads
// distinctly from a red/pending check.
func TestIntegratePRLeavesOpenWhenNoChecksReported(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	gh := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/17", nil
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
			return "no checks reported on this pull request", fmt.Errorf("exit status 1")
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "merge" {
			t.Errorf("gh pr merge must not run when the repo has no checks configured: %v", args)
		}
		return "", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr (no checks reported): %v\n%s", err, out.String())
	}
	for _, c := range *gh {
		if len(c) >= 2 && c[0] == "pr" && c[1] == "merge" {
			t.Errorf("no-checks-reported still called gh pr merge: %v", c)
		}
	}
	// main did not advance: feature.txt is absent.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
		t.Errorf("a no-checks-reported PR was merged into main (feature.txt present)")
	}
	if !strings.Contains(out.String(), "no checks configured") {
		t.Errorf("expected the notice to name the absence of checks, distinct from a failing check:\n%s", out.String())
	}
	if strings.Contains(out.String(), "checks not passing") {
		t.Errorf("absent checks must not be reported as \"checks not passing\" (that's the failing/pending case):\n%s", out.String())
	}
}

// A repo that HAS a CI workflow but whose PR still reports "no checks reported"
// is NOT a CI-less repo — it is the silent pull_request-trigger race (dacli 263):
// the branch got no run at all and is unmergeable with no signal. Integrate must
// name it as NEEDING ATTENTION with the real recovery, never the CI-less "merge
// yourself" advice that would land unverified code.
func TestIntegratePRNamesNoChecksWhenRepoHasCI(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	// The repo has CI configured — the exact state dacli 263 misdiagnosed.
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: ci\non:\n  pull_request:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stubPush(t, func(root, branch string) (string, error) { return "pushed", nil })
	stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/17", nil
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
			return "no checks reported on this pull request", fmt.Errorf("exit status 1")
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "merge" {
			t.Errorf("a no-checks PR must not be merged, CI present or not: %v", args)
		}
		return "", nil
	})

	ctx, out := prCtx(dir)
	if err := cmdIntegrate(ctx, []string{"--pr", "--tasks", tk.ID, "--into", "main"}); err != nil {
		t.Fatalf("integrate --pr (CI present, no checks): %v\n%s", err, out.String())
	}
	// main did not advance: nothing was merged unverified.
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
		t.Errorf("a no-checks PR was merged into main despite CI being present")
	}
	o := out.String()
	if !strings.Contains(o, "NEEDS ATTENTION") {
		t.Errorf("a CI-present repo with no check run must be named as needing attention, not left silent:\n%s", o)
	}
	// It must NOT give the CI-less advice, which would tell the operator to merge
	// unverified code.
	if strings.Contains(o, "no checks configured on this repo") {
		t.Errorf("a CI-present repo must not be reported as having no CI configured:\n%s", o)
	}
}

// --no-merge does NOT fall back to a local merge when offline: the operator
// asked for a PR, so an offline failure is surfaced rather than merged behind
// their back.
func TestIntegratePRNoMergeDoesNotFallBackOffline(t *testing.T) {
	dir, _, tk := prIntegrateEnv(t)
	stubPush(t, func(root, branch string) (string, error) {
		return "Could not resolve host: github.com", fmt.Errorf("exit status 128")
	})
	stubGH(t, func(dir string, args ...string) (string, error) { return "", nil })

	ctx, out := prCtx(dir)
	err := cmdIntegrate(ctx, []string{"--pr", "--no-merge", "--tasks", tk.ID, "--into", "main"})
	if err == nil {
		t.Fatalf("expected an error: --no-merge offline must not local-merge\n%s", out.String())
	}
	if _, statErr := os.Stat(filepath.Join(dir, "feature.txt")); statErr == nil {
		t.Errorf("--no-merge fell back to a local merge while offline")
	}
}
