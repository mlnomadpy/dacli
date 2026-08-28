package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// gitInit makes dir a real git repo on a feature branch (dacli refuses to
// commit on main), configured so the fallback identity is stable.
func gitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "fallback@x"},
		{"config", "user.name", "fallback"},
		{"checkout", "-q", "-b", "feature"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	cmd := exec.Command("sh", "-c", "cat > "+name)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

// Agents commit as themselves with their role; git blame and dacli blame
// read it back; contrib rolls it up. The whole self-evolving-team loop.
func TestCommitAttributionAndBlame(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitRepo(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Build the widget", "--project", "p", "--accept", "a")

	// A read-only agent may not commit — writing to the repo needs rw.
	tokRO := strings.TrimSpace(strings.Split(run(t, dir, 0, "agent", "spawn", "--grant", "ro"), "\n")[0])
	t.Setenv("DACLI_AGENT", tokRO)
	writeFile(t, dir, "a.txt", "read-only cannot commit\n")
	run(t, dir, 3, "commit", "should be refused")
	_ = os.Unsetenv("DACLI_AGENT")

	// An rw agent with a role commits — author carries the role.
	run(t, dir, 0, "role", "add", "junior", "--grant", "rw")
	out := run(t, dir, 0, "agent", "spawn", "--role", "junior", "--grant", "rw")
	tok := strings.TrimSpace(strings.Split(out, "\n")[0])
	// Find the child's id from the tree.
	var childID string
	for _, l := range strings.Split(run(t, dir, 0, "agent", "tree"), "\n") {
		if strings.Contains(l, "junior") {
			childID = strings.Fields(strings.TrimSpace(l))[0]
		}
	}

	t.Setenv("DACLI_AGENT", tok)
	writeFile(t, dir, "widget.go", "package widget\n\nfunc New() {}\n")
	commitOut := run(t, dir, 0, "commit", "001: add the widget", "--task", "001")
	if !strings.Contains(commitOut, "committed") || !strings.Contains(commitOut, "junior") {
		t.Fatalf("commit not attributed to the role:\n%s", commitOut)
	}
	_ = os.Unsetenv("DACLI_AGENT")

	// git itself sees the attribution: author name carries id+role, trailers
	// carry machine-parseable provenance.
	logCmd := exec.Command("git", "log", "-1", "--format=%an|%ae|%(trailers:key=Dacli-Role,valueonly)")
	logCmd.Dir = dir
	gitLog, _ := logCmd.CombinedOutput()
	if !strings.Contains(string(gitLog), childID) || !strings.Contains(string(gitLog), "junior") {
		t.Errorf("git log missing agent/role attribution: %s", gitLog)
	}
	if !strings.Contains(string(gitLog), "@agent.dacli") {
		t.Errorf("author email not the agent's: %s", gitLog)
	}

	// dacli blame: who wrote this file, in what role.
	blame := run(t, dir, 0, "blame", "widget.go")
	if !strings.Contains(blame, "junior") || !strings.Contains(blame, "* ") || !strings.Contains(blame, "agent(s) touched") {
		t.Errorf("blame did not attribute the file:\n%s", blame)
	}

	// A reviewer files a finding AGAINST the junior's work (the loop the
	// prompts now instruct). contrib joins it: junior gets a defect rate.
	run(t, dir, 0, "note", "add", "finding", "widget lacks error handling",
		"--project", "p", "--severity", "moderate", "--against", childID)
	contrib := run(t, dir, 0, "contrib")
	if !strings.Contains(contrib, "by role") || !strings.Contains(contrib, "junior") {
		t.Errorf("contrib rollup wrong:\n%s", contrib)
	}
	if !strings.Contains(contrib, "1 commit(s) · 1 finding(s)-against") {
		t.Errorf("findings-against not joined to the agent:\n%s", contrib)
	}
	if !strings.Contains(contrib, "per commit") {
		t.Errorf("defect rate not computed:\n%s", contrib)
	}

	// The commit is a first-class workspace event, so the read surface sees it.
	if events := run(t, dir, 0, "events", "tail"); !strings.Contains(events, "commit") {
		t.Errorf("commit not recorded as an event:\n%s", events)
	}
}

// A read-only reviewer's finding-against is stored as an event and, on sync,
// promoted to a note. contrib must count that ONE finding once, not twice
// (once as the applied event, again as its synced note).
func TestContribDoesNotDoubleCountSyncedFinding(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitRepo(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Build the widget", "--project", "p", "--accept", "a")

	// An rw junior commits work on task 001.
	run(t, dir, 0, "role", "add", "junior", "--grant", "rw")
	junTok := strings.TrimSpace(strings.Split(run(t, dir, 0, "agent", "spawn", "--role", "junior", "--grant", "rw"), "\n")[0])
	var childID string
	for _, l := range strings.Split(run(t, dir, 0, "agent", "tree"), "\n") {
		if strings.Contains(l, "junior") {
			childID = strings.Fields(strings.TrimSpace(l))[0]
		}
	}
	t.Setenv("DACLI_AGENT", junTok)
	writeFile(t, dir, "widget.go", "package widget\n\nfunc New() {}\n")
	run(t, dir, 0, "commit", "001: add the widget", "--task", "001")
	_ = os.Unsetenv("DACLI_AGENT")

	// A read-only reviewer files a finding against the junior. Being ro, this is
	// stored as an EventFinding (not a note directly).
	roTok := strings.TrimSpace(strings.Split(run(t, dir, 0, "agent", "spawn", "--grant", "ro"), "\n")[0])
	t.Setenv("DACLI_AGENT", roTok)
	run(t, dir, 0, "note", "add", "finding", "widget lacks error handling",
		"--project", "p", "--about", "001", "--severity", "moderate", "--against", childID)
	_ = os.Unsetenv("DACLI_AGENT")

	// The owner syncs: the event is promoted to a durable NoteFinding. Now the
	// SAME finding exists as both an applied event AND a note.
	run(t, dir, 0, "sync")

	// contrib must count it once, not twice.
	contrib := run(t, dir, 0, "contrib")
	if !strings.Contains(contrib, "1 finding(s)-against") {
		t.Errorf("synced finding double-counted (expected 1 finding(s)-against):\n%s", contrib)
	}
	if strings.Contains(contrib, "2 finding(s)-against") {
		t.Errorf("finding counted twice — event and its synced note both counted:\n%s", contrib)
	}
}

// Opening a PR is an outward-facing GitHub write (and, with --with-verdicts,
// leaks internal findings/verdicts). A read-only agent must be refused before
// any gh call — like push/merge/integrate.
func TestPRRefusesReadOnlyGrant(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitRepo(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Build the widget", "--project", "p", "--accept", "a")

	roTok := strings.TrimSpace(strings.Split(run(t, dir, 0, "agent", "spawn", "--grant", "ro"), "\n")[0])
	t.Setenv("DACLI_AGENT", roTok)
	out := run(t, dir, 3, "pr", "--task", "001")
	if !strings.Contains(out, "rw grant") {
		t.Errorf("ro agent should be refused a PR for lacking an rw grant:\n%s", out)
	}
}

// dacli commit refuses on the default branch — the git-discipline rule,
// enforced not just prompted.
func TestCommitRefusesDefaultBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	// A repo left on main.
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "x@x"}, {"config", "user.name", "x"}, {"checkout", "-q", "-b", "main"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		_, _ = cmd.CombinedOutput()
	}
	run(t, dir, 0, "init", "--name", "x")
	writeFile(t, dir, "f.txt", "x\n")
	out := run(t, dir, 3, "commit", "on main")
	if !strings.Contains(out, "refusing to commit on main") || !strings.Contains(out, "branch first") {
		t.Errorf("default-branch guard wrong:\n%s", out)
	}
}

// Task 494 reproduced root's task-493 commit being fenced by task 492's
// completed recovery transfer merely because that historical transfer was the
// newest record owned by a-root. Exercise the public commands: the unrelated
// transfer remains durable, but only the transfer for this checkout governs.
func TestCommitScopesRootRecoveryTransferToCurrentWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitRepo(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Repair execution store", "--project", "p", "--accept", "a")
	run(t, dir, 0, "task", "add", "Scope commit authorization", "--project", "p", "--accept", "a")
	writeFile(t, dir, "base.txt", "base\n")
	gitRun(t, dir, "add", "base.txt")
	gitRun(t, dir, "commit", "-q", "-m", "base")

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	task492, err := store.FindTask(w, "001")
	if err != nil {
		t.Fatal(err)
	}
	task493, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatal(err)
	}
	wt492 := filepath.Join(dir, "wt-492")
	wt493 := filepath.Join(dir, "wt-493")
	gitRun(t, dir, "worktree", "add", "-q", "-b", "dacli/492", wt492, "HEAD")
	gitRun(t, dir, "worktree", "add", "-q", "-b", "dacli/493", wt493, "HEAD")

	seedRootTransfer(t, w, "01KZCURRENTTRANSFER0000001", task493.ID, wt493, "dacli/493", "internal/features/vcs,internal/cli")
	// Deliberately newer: the pre-fix owner-only lookup selected this unrelated
	// task-492 scope first and refused the exact task-493 path below.
	seedRootTransfer(t, w, "01KZSTALETRANSFER00000002", task492.ID, wt492, "dacli/492", "internal/store,internal/features/execution")

	writeFile(t, wt493, "current.txt", "current recovery\n")
	gitRun(t, wt493, "add", "current.txt")
	out := run(t, wt493, 3, "commit", "task 493 recovery", "--task", "002", "--no-add")
	if !strings.Contains(out, "current recovery transfer run 01KZCURRENTTRANSFER0000001") ||
		!strings.Contains(out, "current.txt") || strings.Contains(out, "--force") || strings.Contains(out, "01KZSTALETRANSFER") {
		t.Fatalf("scope refusal did not name only the current transfer provenance:\n%s", out)
	}

	gitRun(t, wt493, "restore", "--staged", "current.txt")
	path := filepath.Join(wt493, "internal", "cli", "current.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("current recovery\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, wt493, "add", "internal/cli/current.txt")
	out = run(t, wt493, 0, "commit", "task 493 recovery", "--task", "002", "--no-add")
	if !strings.Contains(out, "committed") {
		t.Fatalf("current exact recovery claim did not commit:\n%s", out)
	}
}

// Issue #811: after root recovered an old task, a later root correction run
// retained that task's two websocket claims. Claiming a new task and creating
// its canonical worktree did not publish a run of its own, so commit's
// identity-only fallback selected the old claim and refused every correct file.
// Drive the public task/worktree/commit path and prove both halves of the fix:
// stale authority is ignored, while the new task's inferred scope still
// refuses an unrelated file instead of silently broadening root authority.
func TestClaimedTaskWorktreeReplacesStaleTransferredRootClaim(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	gitRun(t, dir, "config", "user.email", "x@x")
	gitRun(t, dir, "config", "user.name", "x")
	gitRun(t, dir, "checkout", "-q", "-b", "main")
	for _, path := range []string{"websocket/client.go", "internal/cli/base.go"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, path), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, dir, "add", "-A")
	gitRun(t, dir, "commit", "-q", "-m", "base")

	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Repair websocket/client.go", "--project", "p", "--accept", "websocket/client.go is repaired")
	run(t, dir, 0, "task", "add", "Fix internal/cli/fresh.go", "--project", "p", "--accept", "internal/cli/fresh.go is covered")
	run(t, dir, 0, "worktree", "add", "--task", "001")

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	oldTask, err := store.FindTask(w, "001")
	if err != nil {
		t.Fatal(err)
	}
	newTask, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatal(err)
	}
	oldWT := w.WorktreePath(oldTask.Project, oldTask.Seq, oldTask.Slug)
	seedRootTransfer(t, w, "01KZ811TRANSFER0000000001", oldTask.ID, oldWT, store.TaskBranch(oldTask), "websocket/client.go,websocket/client_test.go")
	rootRun := "01KZ811ROOTFOLLOWUP0000002"
	if err := os.MkdirAll(w.RunDir(rootRun), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir(rootRun), "worktree.txt"), []byte(oldWT+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(rootRun), "proc.txt"), procmon.Record{
		RunID: rootRun, Child: agentid.RootID, Task: oldTask.ID, Role: "root", Started: time.Now(), Outcome: "done",
		Claims: []string{"websocket/client.go", "websocket/client_test.go"},
	}); err != nil {
		t.Fatal(err)
	}

	run(t, dir, 0, "task", "claim", "002")
	run(t, dir, 0, "worktree", "add", "--task", "002")
	newWT := w.WorktreePath(newTask.Project, newTask.Seq, newTask.Slug)
	if err := os.WriteFile(filepath.Join(newWT, "internal", "cli", "fresh.go"), []byte("package cli\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newWT, "unrelated.txt"), []byte("outside task scope\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, newWT, "add", "-A")
	out := run(t, newWT, 3, "commit", "new task work", "--task", "002", "--no-add")
	for _, want := range []string{"inferred claim from task 002-", "internal/cli", "unrelated.txt"} {
		if !strings.Contains(out, want) {
			t.Fatalf("new task refusal missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "websocket/client.go") {
		t.Fatalf("stale transferred root claim governed the new task:\n%s", out)
	}

	gitRun(t, newWT, "restore", "--staged", "unrelated.txt")
	out = run(t, newWT, 0, "commit", "new task work", "--task", "002", "--no-add")
	if !strings.Contains(out, "committed") {
		t.Fatalf("task-scoped change was not committed:\n%s", out)
	}
}

func seedRootTransfer(t *testing.T, w *workspace.Workspace, runID, task, worktree, branch, claims string) {
	t.Helper()
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "worktree.txt"), []byte(worktree+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
		RunID: runID, Child: "a-failed-" + runID, Task: task, Role: "fixer", Started: time.Now(), Outcome: "failed",
	}); err != nil {
		t.Fatal(err)
	}
	body := "version: 1\nworktree: " + worktree + "\nbranch: " + branch + "\nprior_run: " + runID +
		"\nprior_owner: a-failed\nnew_owner: " + agentid.RootID + "\nclaims: " + claims + "\ntransferred_at: " + time.Now().UTC().Format(time.RFC3339Nano) + "\n"
	if err := os.WriteFile(filepath.Join(runDir, "worktree-transfer.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
