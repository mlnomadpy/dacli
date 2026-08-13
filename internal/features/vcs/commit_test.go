package vcs

import (
	"bytes"
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
	log := gitAt(t, wt, "log", "-1", "--format=%s%n%(trailers:key=Dacli-Agent,valueonly)%n%(trailers:key=Dacli-Role,valueonly)%n%(trailers:key=Dacli-Task,valueonly)")
	for _, want := range []string{"requested subject", child, "fixer", "001-protect-child-staging"} {
		if !strings.Contains(log, want) {
			t.Fatalf("child commit missing %q:\n%s", want, log)
		}
	}
}

func commitCtx(dir string) (*clikit.Ctx, *bytes.Buffer) {
	var output bytes.Buffer
	return &clikit.Ctx{Stdout: &output, Stderr: &output, Cwd: dir}, &output
}
