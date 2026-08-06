package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// git runs a git subcommand in dir, failing the test on error.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// A token minted for a --worktree spawn is written to the SHARED main
// workspace; the child runs dacli from inside the linked worktree, whose
// own git-tracked .dacli snapshot went stale the moment the branch was cut.
// The child must still resolve its own identity — workspace.Find redirects a
// linked worktree to the main root. This is the whole point of task 296:
// without it, an agent in a worktree cannot use dacli at all.
func TestWorktreeChildResolvesOwnIdentity(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitRepo(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "a")

	// Commit .dacli so the branch carries a snapshot, THEN cut the worktree from
	// it. The snapshot is frozen here — it does NOT contain the child minted
	// below, which is exactly the stale shadow the redirect must see past.
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "snapshot .dacli")
	wt := filepath.Join(dir, ".dacli", "worktrees", "p-001-t-one")
	gitIn(t, dir, "worktree", "add", "-q", "-b", "dacli/001-t-one", wt, "HEAD")

	// Mint the child in the MAIN workspace AFTER the worktree exists — this is
	// what spawn --worktree does: agentid.Spawn writes agents/<id>.md under the
	// resolved (main) workspace, uncommitted. It is therefore ABSENT from the
	// worktree's frozen .dacli snapshot, so only a redirect to the shared root
	// can resolve it. (Committing it first would let the test pass even with the
	// redirect broken, since the shadow would carry the child.)
	spawnOut := run(t, dir, 0, "agent", "spawn", "--grant", "rw")
	token := strings.TrimSpace(strings.Split(spawnOut, "\n")[0])

	// From INSIDE the worktree, the child's token must resolve to its identity.
	// It resolves to the shadow .dacli/agents snapshot without the redirect and
	// the whole lifecycle (commit, task check, note) is dead on arrival.
	t.Setenv("DACLI_AGENT", token)
	who := run(t, wt, 0, "whoami")
	if !strings.Contains(who, "grant: rw") {
		t.Fatalf("worktree child token does not resolve to its identity from inside the worktree:\n%s", who)
	}
	t.Setenv("DACLI_AGENT", "")
}
