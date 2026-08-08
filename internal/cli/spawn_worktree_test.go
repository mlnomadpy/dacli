package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A --worktree child's cwd (cmd.Dir) is the worktree and the brief tells it
// so — but that is an instruction, not an enforced boundary: nothing stops a
// child (or a runtime whose own path allowlist resolves back to the main
// checkout's absolute path, dacli-267) from writing outside it. A real
// incident did exactly this: a spawned child wrote a source file straight
// into the main checkout under an absolute path while its own worktree never
// had the file, and it went unnoticed because the file was untracked and
// every existing check scans with --untracked-files=no.
//
// This drives that same shape: a mock child writes one file the COOPERATIVE
// way (relative path, lands in the worktree) and one file the ESCAPING way
// (absolute path into the main checkout). It asserts the worktree write
// survives, and that spawn --worktree leaves the main checkout exactly as it
// found it — the escaped write is detected and undone before spawn returns,
// which is what makes "cannot modify the main checkout" true as an
// observable postcondition of running this command.
func TestSpawnWorktreeReclaimsMainCheckoutEscape(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitAt(t, dir, "init", "-q")
	gitAt(t, dir, "config", "user.email", "x@x")
	gitAt(t, dir, "config", "user.name", "x")
	gitAt(t, dir, "checkout", "-q", "-b", "main")
	writeAt(t, dir, "base.txt", "shared base\n")
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-q", "-m", "base")

	run(t, dir, 0, "init", "--name", "x")
	// init now writes a .gitignore (it keeps the workspace off trunk). Commit
	// it, as a real user would, so the escape assertion below stays strict:
	// this file is setup written BEFORE spawn ran, not something the child
	// touched, and tolerating it by name would weaken the check.
	gitAt(t, dir, "add", "-A", "--", ".gitignore")
	gitAt(t, dir, "commit", "-q", "-m", "gitignore the dacli workspace")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Fix the batch job", "--project", "p", "--accept", "a")

	leaked := filepath.Join(dir, "internal", "store", "preflight.go")
	mockRuntime(t, dir, "escapee", strings.Join([]string{
		"cat > /dev/null", // drain the stdin brief like the other mock scripts
		"echo cooperative work > innocuous.txt",
		"mkdir -p " + filepath.Dir(leaked),
		"echo leaked > " + leaked,
	}, "\n"))

	out := run(t, dir, 1, "spawn", "--task", "001", "--runtime", "escapee", "--grant", "rw", "--worktree")

	// Named: the mechanism that caught and undid the escape is stated, not
	// silent — an operator (or a future maintainer) needs to know this ran.
	if !strings.Contains(out, "main checkout") || !strings.Contains(out, "internal/store/preflight.go") {
		t.Fatalf("worktree escape not named in output:\n%s", out)
	}

	// The main checkout has no trace of the escaped file...
	if _, err := os.Stat(leaked); !os.IsNotExist(err) {
		t.Errorf("escaped file still present in main checkout (stat err %v)", err)
	}
	// ...and git agrees the main checkout is exactly as it was (only the
	// always-expected .dacli churn is tolerated).
	status := gitAt(t, dir, "status", "--porcelain")
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimSpace(line)
		// .dacli churns by design; escapee.sh is the mock runtime's own
		// fixture script, written before spawn ran — neither is part of what
		// the child touched.
		if line == "" || strings.Contains(line, ".dacli") || strings.Contains(line, "escapee.sh") {
			continue
		}
		t.Errorf("main checkout left dirty after spawn --worktree: %q\nfull status:\n%s", line, status)
	}

	// The cooperative write — relative path, honoring cwd — landed in the
	// worktree and was left alone.
	wt := filepath.Join(dir, ".dacli", "worktrees", "p-001-fix-the-batch-job")
	if _, err := os.Stat(filepath.Join(wt, "innocuous.txt")); err != nil {
		t.Errorf("cooperative worktree write missing: %v", err)
	}
}
