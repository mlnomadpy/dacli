package cli

import (
	"os/exec"
	"strings"
	"testing"
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
	t.Setenv("DACLI_AGENT", "")

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
	t.Setenv("DACLI_AGENT", "")

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
	t.Setenv("DACLI_AGENT", "")

	// A read-only reviewer files a finding against the junior. Being ro, this is
	// stored as an EventFinding (not a note directly).
	roTok := strings.TrimSpace(strings.Split(run(t, dir, 0, "agent", "spawn", "--grant", "ro"), "\n")[0])
	t.Setenv("DACLI_AGENT", roTok)
	run(t, dir, 0, "note", "add", "finding", "widget lacks error handling",
		"--project", "p", "--about", "001", "--severity", "moderate", "--against", childID)
	t.Setenv("DACLI_AGENT", "")

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
		cmd.CombinedOutput()
	}
	run(t, dir, 0, "init", "--name", "x")
	writeFile(t, dir, "f.txt", "x\n")
	out := run(t, dir, 3, "commit", "on main")
	if !strings.Contains(out, "refusing to commit on main") || !strings.Contains(out, "branch first") {
		t.Errorf("default-branch guard wrong:\n%s", out)
	}
}
