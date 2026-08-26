package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// TestProjectShowLandingFlagsPersistBeforeRendering drives the documented
// public command, then inspects both the record and a freshly loaded policy.
// A rendered override alone is not configuration: ship and integrate reload
// the project record in later processes (issue #762).
func TestProjectShowLandingFlagsPersistBeforeRendering(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "preserve this body")

	first := dacliRun(t, bin, dir, "project", "show", "p", "--landing-mode", "pr", "--landing-base", "master")
	if !strings.Contains(first, "Landing configured: mode=pr base=master") || !strings.Contains(first, "Landing effective: mode=pr base=master (override: false)") {
		t.Fatalf("project show did not render its persisted policy:\n%s", first)
	}

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".dacli", "projects", "p", "project.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "landing.mode: pr") || !strings.Contains(string(raw), "landing.base: master") || !strings.Contains(string(raw), "preserve this body") {
		t.Fatalf("project record did not retain landing policy and body:\n%s", raw)
	}
	project, err := store.LoadProject(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if project.Landing != (model.LandingPolicy{Mode: model.LandingPR, Base: "master"}) {
		t.Fatalf("reloaded landing policy = %+v", project.Landing)
	}
	for _, args := range [][]string{{"ship", "--project", "p"}, {"integrate", "--project", "p"}} {
		out := run(t, dir, 3, args...)
		if !strings.Contains(out, "project landing policy requires the PR path") {
			t.Fatalf("%v did not use the policy persisted by project show:\n%s", args, out)
		}
	}

	dacliRun(t, bin, dir, "project", "show", "p", "--landing-base", "release")
	project, err = store.LoadProject(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if project.Landing != (model.LandingPolicy{Mode: model.LandingPR, Base: "release"}) {
		t.Fatalf("updating base did not preserve mode: %+v", project.Landing)
	}
}

func TestProjectShowRejectsInvalidOrConflictingLandingFlagsWithoutChangingRecord(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--landing-mode", "pr", "--landing-base", "main")
	path := filepath.Join(dir, ".dacli", "projects", "p", "project.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"project", "show", "p", "--landing-mode", "merge"},
		{"project", "show", "p", "--landing-base", "  "},
		{"project", "show", "p", "--landing-base", "main..evil"},
		{"project", "show", "p", "--landing-base", "foo/.hidden"},
		{"project", "show", "p", "--landing-base", "HEAD"},
		{"project", "show", "p", "--landing-mode", "pr", "--landing-mode", "local"},
		{"project", "show", "p", "--landing-base", "main", "--landing-base", "master"},
	} {
		out := run(t, dir, 2, args...)
		if !strings.Contains(out, "landing") && !strings.Contains(out, "conflicting") {
			t.Fatalf("%v did not explain the rejected landing policy:\n%s", args, out)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatalf("%v partially changed the project record:\n%s", args, after)
		}
	}
}

func TestProjectShowReadOnlyInspectionDoesNotRequireWriteGrant(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--landing-mode", "pr", "--landing-base", "main")
	out := run(t, dir, 0, "agent", "spawn", "--role", "junior", "--grant", "ro")
	token := strings.TrimSpace(strings.Split(strings.TrimSpace(out), "\n")[0])
	t.Setenv("DACLI_AGENT", token)

	run(t, dir, 0, "project", "show", "p")
	cmd, rest := match([]string{"project", "show", "p"})
	var stdout, stderr bytes.Buffer
	if err := invoke(&Ctx{Stdout: &stdout, Stderr: &stderr, Cwd: dir, JSON: true}, cmd, rest); err != nil {
		t.Fatalf("read-only JSON project show failed: %v\n%s", err, stderr.String())
	}
	if got := run(t, dir, 3, "project", "show", "p", "--landing-base", "release"); !strings.Contains(strings.ToLower(got), "grant") {
		t.Fatalf("mutating project show did not report its grant refusal: %s", got)
	}
}

func TestConcurrentOneFlagProjectShowUpdatesSerialize(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--landing-mode", "local", "--landing-base", "main")

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	locked := make(chan struct{})
	release := make(chan struct{})
	lockDone := make(chan error, 1)
	go func() {
		lockDone <- store.WithFileLock(filepath.Join(w.ProjectDir("p"), ".project.lock"), func() error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	commands := []*exec.Cmd{
		exec.Command(bin, "project", "show", "p", "--landing-mode", "pr"),
		exec.Command(bin, "project", "show", "p", "--landing-base", "release"),
	}
	done := make(chan error, len(commands))
	for _, command := range commands {
		command.Dir = dir
		if err := command.Start(); err != nil {
			close(release)
			t.Fatal(err)
		}
		go func(cmd *exec.Cmd) { done <- cmd.Wait() }(command)
	}
	select {
	case err := <-done:
		close(release)
		t.Fatalf("project show bypassed the project transaction lock: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	close(release)
	if err := <-lockDone; err != nil {
		t.Fatal(err)
	}
	for range commands {
		if err := <-done; err != nil {
			t.Fatalf("concurrent project show failed: %v", err)
		}
	}

	project, err := store.LoadProject(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if project.Landing != (model.LandingPolicy{Mode: model.LandingPR, Base: "release"}) {
		t.Fatalf("concurrent one-flag updates lost a successful value: %+v", project.Landing)
	}
}
