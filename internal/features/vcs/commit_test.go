package vcs

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestCommitUsageMatchesCommandTable(t *testing.T) {
	unsetAgentEnv(t)
	dir := t.TempDir()
	if _, err := workspace.Init(dir, "x"); err != nil {
		t.Fatal(err)
	}

	var command clikit.Command
	for _, candidate := range Commands {
		if candidate.Path == "commit" {
			command = candidate
			break
		}
	}
	if command.Usage != commitUsage {
		t.Fatalf("commit command Usage = %q, want %q", command.Usage, commitUsage)
	}
	ctx, _ := commitCtx(dir)
	err := cmdCommit(ctx, nil)
	if got, want := err.Error(), "usage: "+command.Usage; got != want {
		t.Fatalf("commit missing-argument output = %q, want %q", got, want)
	}
}

func TestCommitJSONDisambiguatesTaskOwnershipWorktreeAndMissingPathScope(t *testing.T) {
	unsetAgentEnv(t)
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q", "-b", "main")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, agentid.RootID, "p", "Manual owner work", store.TaskOpts{Accept: []string{"behavior works"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-qm", "base")
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	gitAt(t, dir, "worktree", "add", "-q", "-b", BranchFor(task), wt, "HEAD")
	if err := os.WriteFile(filepath.Join(wt, "manual.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, output := commitCtx(wt)
	ctx.JSON = true
	if err := cmdCommit(ctx, []string{"manual work", "--task", task.ID}); err != nil {
		t.Fatal(err)
	}
	var result commitResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("commit JSON = %v\n%s", err, output.String())
	}
	if result.Diagnostic == nil || result.Diagnostic.Code != "path_scope_unavailable" || result.Diagnostic.Remediation == "" {
		t.Fatalf("diagnostic = %+v", result.Diagnostic)
	}
	c := result.Controls
	if c.TaskRef != task.ID || c.TaskOwner != agentid.RootID || !c.TaskOwnedByActor || !c.CanonicalWorktree || !c.CanonicalBranch || c.PathScopePresent {
		t.Fatalf("independent controls = %+v", c)
	}
	if strings.Contains(output.String(), "no recorded --claim") {
		t.Fatalf("ambiguous legacy wording remains: %s", output.String())
	}
	if err := os.WriteFile(filepath.Join(wt, "second.txt"), []byte("more\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, output = commitCtx(wt)
	if err := cmdCommit(ctx, []string{"second manual change", "--task", task.ID}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"warning[path_scope_unavailable]", "task=" + task.ID, "owner=a-root", "owned-by-actor=true", "canonical-worktree=true", "canonical-branch=true", "dacli context " + task.ID + " --json"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("human diagnostic missing %q:\n%s", want, output.String())
		}
	}
}

// Tasks 422/423 reproduced the same destructive sequence: an rw child staged
// its claimed files in an isolated worktree, a lost-token invocation accepted
// `-m` as the message and committed them as a-root with subject "-m", then the
// child's documented commit returned "nothing staged". Both refusals below
// must happen before the index changes; removing both commit guards recreates
// that sequence and makes the malformed call exit 0 as a-root.
func TestCommitPreservesSpawnedWorktreeStagingAndChildAttribution(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	unsetAgentEnv(t)
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, agentid.RootID, "p", "Protect child staging", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	gitAt(t, dir, "worktree", "add", "-q", "-b", BranchFor(task), wt, "HEAD")

	child, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}, "fixer", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	runID := "01KZTESTCOMMITOWNER0000000"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worktree.txt"), []byte(wt+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
		RunID: runID, Child: child, Task: task.ID, Role: "fixer", Started: time.Now(), Claims: []string{"claimed.txt"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, "claimed.txt"), []byte("verified child work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, wt, "add", "claimed.txt")
	before := gitAt(t, wt, "rev-parse", "HEAD")

	ctx, output := commitCtx(wt)
	if err := cmdCommit(ctx, []string{"-m", "requested subject", "--task", "001"}); clikit.ExitCode(err) != 2 {
		t.Fatalf("malformed commit exit = %d, want 2: %v\n%s", clikit.ExitCode(err), err, output)
	}
	ctx, output = commitCtx(wt)
	if err := cmdCommit(ctx, []string{"requested subject", "--task", "001"}); clikit.ExitCode(err) != 3 ||
		!strings.Contains(err.Error(), child) || !strings.Contains(err.Error(), agentid.RootID) || !strings.Contains(err.Error(), "staged work was preserved") {
		t.Fatalf("owner refusal did not name both actors and preservation: exit %d, %v\n%s", clikit.ExitCode(err), err, output)
	}
	if got := gitAt(t, wt, "rev-parse", "HEAD"); got != before {
		t.Fatalf("refused invocation created commit %s, want unchanged %s", got, before)
	}
	if got := gitAt(t, wt, "diff", "--cached", "--name-only"); got != "claimed.txt" {
		t.Fatalf("refused invocation changed staged work: %q", got)
	}

	t.Setenv(agentid.EnvVar, token)
	ctx, output = commitCtx(wt)
	if err := cmdCommit(ctx, []string{"requested subject", "--task", "001", "--no-add"}); err != nil {
		t.Fatalf("child commit: %v\n%s", err, output)
	}
	result, ok := ctx.Result.(commitResult)
	if !ok || !result.Controls.PathScopePresent || len(result.Controls.PathScope) != 1 || result.Controls.PathScope[0] != "claimed.txt" || result.Controls.PathScopeSource == "" {
		t.Fatalf("spawned path-scope controls = %+v", ctx.Result)
	}
	log := gitAt(t, wt, "log", "-1", "--format=%s%n%(trailers:key=Dacli-Agent,valueonly)%n%(trailers:key=Dacli-Role,valueonly)%n%(trailers:key=Dacli-Task,valueonly)")
	for _, want := range []string{"requested subject", child, "fixer", "001-protect-child-staging"} {
		if !strings.Contains(log, want) {
			t.Fatalf("child commit missing %q:\n%s", want, log)
		}
	}
}

func TestCommitAcceptsTrailingRecursiveClaimDescendantsOnly(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	unsetAgentEnv(t)
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, agentid.RootID, "p", "Honor recursive claim", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	gitAt(t, dir, "worktree", "add", "-q", "-b", BranchFor(task), wt, "HEAD")

	child, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}, "fixer", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	runID := "01KZTESTRECURSIVECLAIM000"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worktree.txt"), []byte(wt+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
		RunID: runID, Child: child, Task: task.ID, Role: "fixer", Started: time.Now(), Claims: []string{"supabase/**", "docs/**"},
	}); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"supabase/config.toml":                 "project_id = 'test'\n",
		"supabase/tests/database/rls.test.sql": "select true;\n",
		"docs/claimed.md":                      "claimed by the second path\n",
		"scripts/verify-supabase-types.mjs":    "export {};\n",
	} {
		path := filepath.Join(wt, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitAt(t, wt, "add", "-A")
	t.Setenv(agentid.EnvVar, token)
	ctx, output := commitCtx(wt)
	err = cmdCommit(ctx, []string{"recursive claim", "--task", "001", "--no-add"})
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "scripts/verify-supabase-types.mjs") || strings.Contains(err.Error(), "supabase/config.toml") || strings.Contains(err.Error(), "docs/claimed.md") {
		t.Fatalf("claim refusal = exit %d, %v\n%s", clikit.ExitCode(err), err, output)
	}
	gitAt(t, wt, "restore", "--staged", "scripts/verify-supabase-types.mjs")
	ctx, output = commitCtx(wt)
	if err := cmdCommit(ctx, []string{"recursive claim", "--task", "001", "--no-add"}); err != nil {
		t.Fatalf("commit recursive descendants: %v\n%s", err, output)
	}
}

// Issue #694: a runtime can exit before it reads the prompt, leaving its
// one-time child identity as the newest worktree owner. Recovery must use the
// finalized run record rather than impersonating that lost identity, and the
// resulting root commit must retain a real claim boundary.
func TestWorktreeReclaimTerminalFailedOwnersAndCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	unsetAgentEnv(t)
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, agentid.RootID, "p", "Recover failed runtime", store.TaskOpts{Accept: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")
	wt := w.WorktreePath(task.Project, task.Seq, task.Slug)
	gitAt(t, dir, "worktree", "add", "-q", "-b", BranchFor(task), wt, "HEAD")

	firstRun, firstChild := seedWorktreeRun(t, w, wt, "01KZRECLAIMFAILED000000001", task.ID, 0, "no visible result")
	if err := os.WriteFile(filepath.Join(wt, "claimed.txt"), []byte("verified retained work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, "outside.txt"), []byte("not claimed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, output := commitCtx(wt)
	if err := cmdWorktreeReclaim(ctx, []string{"--claim", "claimed.txt"}); err != nil {
		t.Fatalf("preview: %v\n%s", err, output)
	}
	preview := output.String()
	for _, want := range []string{wt, BranchFor(task), firstChild, firstRun, "claimed.txt", "outside.txt", "preview only"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q:\n%s", want, preview)
		}
	}
	if _, err := os.Stat(filepath.Join(w.RunDir(firstRun), worktreeTransferFile)); !os.IsNotExist(err) {
		t.Fatalf("preview mutated transfer state: %v", err)
	}

	ctx, output = commitCtx(wt)
	if err := cmdWorktreeReclaim(ctx, []string{"--claim", "claimed.txt", "--apply"}); err != nil {
		t.Fatalf("apply reclaim: %v\n%s", err, output)
	}
	original, err := procmon.ReadRecord(filepath.Join(w.RunDir(firstRun), "proc.txt"))
	if err != nil || original.Child != firstChild || original.Outcome != "no visible result" {
		t.Fatalf("historical run changed: %+v, %v", original, err)
	}
	transfer, err := readWorktreeTransfer(filepath.Join(w.RunDir(firstRun), worktreeTransferFile))
	if err != nil || transfer.Owner != agentid.RootID || strings.Join(transfer.Claims, ",") != "claimed.txt" {
		t.Fatalf("durable transfer = %+v, %v", transfer, err)
	}
	audit, err := os.ReadFile(filepath.Join(w.RunDir(firstRun), worktreeTransferFile))
	if err != nil || !strings.Contains(string(audit), "dirty: ?? claimed.txt") || !strings.Contains(string(audit), "prior_owner: "+firstChild) {
		t.Fatalf("transfer audit omitted prior owner or dirty paths: %v\n%s", err, audit)
	}

	gitAt(t, wt, "add", "-A")
	ctx, output = commitCtx(wt)
	err = cmdCommit(ctx, []string{"recovered work", "--task", "001", "--no-add"})
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "outside.txt") {
		t.Fatalf("transferred claim refusal = exit %d, %v\n%s", clikit.ExitCode(err), err, output)
	}
	gitAt(t, wt, "restore", "--staged", "outside.txt")
	ctx, output = commitCtx(wt)
	if err := cmdCommit(ctx, []string{"recovered work", "--task", "001", "--no-add"}); err != nil {
		t.Fatalf("governed root commit: %v\n%s", err, output)
	}

	secondRun, secondChild := seedWorktreeRun(t, w, wt, "01KZRECLAIMFAILED000000002", task.ID, 0, "no visible result")
	ctx, output = commitCtx(wt)
	if err := cmdWorktreeReclaim(ctx, []string{"--claim", "outside.txt", "--apply"}); err != nil {
		t.Fatalf("reclaim after repeated failed correction: %v\n%s", err, output)
	}
	if !strings.Contains(output.String(), secondRun) || !strings.Contains(output.String(), secondChild) {
		t.Fatalf("second reclaim did not name newest failed owner:\n%s", output)
	}
	if owner, ok := agentWorktreeOwner(w, wt); !ok || owner != agentid.RootID {
		t.Fatalf("owner after repeated recovery = %q, %v", owner, ok)
	}
}

func TestWorktreeOwnerUsesPrelaunchIntentBeforeProcessRegistration(t *testing.T) {
	dir := t.TempDir()
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}

	oldRun := "01KZPRELAUNCHOLD000000001"
	oldDir := w.RunDir(oldRun)
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "worktree.txt"), []byte(dir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(oldDir, "proc.txt"), procmon.Record{RunID: oldRun, Child: "a-old", Outcome: "failed"}); err != nil {
		t.Fatal(err)
	}
	transfer := "version: 1\nworktree: " + dir + "\nbranch: dacli/001-fast\nprior_run: " + oldRun + "\nprior_owner: a-old\nnew_owner: " + agentid.RootID + "\nclaims: claimed.txt\n"
	if err := os.WriteFile(filepath.Join(oldDir, worktreeTransferFile), []byte(transfer), 0o644); err != nil {
		t.Fatal(err)
	}

	newRun := "01MZPRELAUNCHNEW000000001"
	newDir := w.RunDir(newRun)
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "worktree.txt"), []byte(dir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "invocation.txt"), []byte("run: "+newRun+"\nchild: a-fast-child\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if owner, ok := agentWorktreeOwner(w, dir); !ok || owner != "a-fast-child" {
		t.Fatalf("owner during process-registration window = %q, %v; want prelaunch child", owner, ok)
	}
}

func TestWorktreeReclaimRefusesLiveOrUnreadableRun(t *testing.T) {
	unsetAgentEnv(t)
	for _, tc := range []struct {
		name       string
		pid        int
		outcome    string
		removeProc bool
		want       string
	}{
		{name: "live", pid: os.Getpid(), outcome: "", want: "live process"},
		{name: "unreadable", outcome: "failed", removeProc: true, want: "process state"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			gitAt(t, dir, "init", "-q")
			gitAt(t, dir, "checkout", "-q", "-b", "recovery-test")
			w, err := workspace.Init(dir, "x")
			if err != nil {
				t.Fatal(err)
			}
			run, _ := seedWorktreeRun(t, w, dir, "01KZRECLAIMREFUSE00000001", "t-1", tc.pid, tc.outcome)
			if tc.removeProc {
				if err := os.Remove(filepath.Join(w.RunDir(run), "proc.txt")); err != nil {
					t.Fatal(err)
				}
			}
			ctx, output := commitCtx(dir)
			err = cmdWorktreeReclaim(ctx, []string{"--claim", "internal/features/vcs", "--apply"})
			if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal = exit %d, %v\n%s", clikit.ExitCode(err), err, output)
			}
		})
	}
}

func seedWorktreeRun(t *testing.T, w *workspace.Workspace, wt, runID, task string, pid int, outcome string) (string, string) {
	t.Helper()
	child, _, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}, "fixer", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worktree.txt"), []byte(wt+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Child: child, Task: task, Role: "fixer", PID: pid, PGID: pid, Started: time.Now().Add(-time.Minute), Claims: []string{"claimed.txt"}, Outcome: outcome}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	return runID, child
}

func commitCtx(dir string) (*clikit.Ctx, *bytes.Buffer) {
	var output bytes.Buffer
	return &clikit.Ctx{Stdout: &output, Stderr: &output, Cwd: dir}, &output
}
