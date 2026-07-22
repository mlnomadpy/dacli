package orchestration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// fakeRunner records the subcommands the driver would run, executing none.
type fakeRunner struct{ calls [][]string }

func (r *fakeRunner) run(label string, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	return "", nil
}

func (r *fakeRunner) firstArgs() []string {
	var out []string
	for _, c := range r.calls {
		if len(c) > 0 {
			out = append(out, c[0])
		}
	}
	return out
}

func loopEnv(t *testing.T) *workspace.Workspace {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("DACLI_AGENT", "")
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "x@x"}, {"config", "user.name", "x"},
		{"checkout", "-q", "-b", "main"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	w, err := workspace.Init(dir, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

func newDriver(w *workspace.Workspace, r runner, gov *Governor) *driver {
	return &driver{
		ctx:   &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root},
		w:     w,
		cfg:   loopCfg{project: "p", implRole: "fixer", reviewRole: "go-auditor", width: 2, pr: true},
		gov:   gov,
		run:   r,
		sleep: func(time.Duration) {},
		now:   func() time.Time { return time.Unix(1_000_000, 0) },
	}
}

func TestDriverRunsSprintPhasesInOrder(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	fr := &fakeRunner{}
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3})
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}

	got := strings.Join(fr.firstArgs(), ",")
	// build spawn → wait → ship → review spawn → retro
	for _, want := range []string{"spawn", "wait", "ship", "retro"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected phase %q in sequence, got: %s", want, got)
		}
	}
	// The build spawn must target the ready task with the implementer role + PR.
	var buildSpawn []string
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "fixer") {
			buildSpawn = c
		}
	}
	if buildSpawn == nil {
		t.Fatal("no build spawn with the impl role")
	}
	for _, need := range []string{"--task", "--role", "fixer", "--detach", "--pr"} {
		if !contains(buildSpawn, need) {
			t.Fatalf("build spawn missing %q: %v", need, buildSpawn)
		}
	}
	// The review phase must spawn the reviewer role.
	sawReview := false
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "go-auditor") {
			sawReview = true
		}
	}
	if !sawReview {
		t.Fatal("review phase did not spawn the reviewer role")
	}
}

func TestDriverIdlesWhenBacklogEmpty(t *testing.T) {
	w := loopEnv(t) // no ready tasks
	fr := &fakeRunner{}
	// Idle path with dry-run stops after one pass so the test terminates.
	d := newDriver(w, fr, &Governor{MaxCycles: 1, NoProgressHalt: 3, Idle: time.Millisecond})
	d.cfg.dryRun = true
	if err := d.loop(); err != nil {
		t.Fatal(err)
	}
	// No build spawn should have happened; only the review-regeneration spawn.
	for _, c := range fr.calls {
		if len(c) > 0 && c[0] == "spawn" && contains(c, "fixer") {
			t.Fatalf("idle cycle must not spawn implementers, got: %v", c)
		}
	}
}

func commitTo(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, a := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", name}} {
		c := exec.Command("git", a...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", a, err, out)
		}
	}
}

// TestTrunkMarkerReflectsTrunkAdvance is the regression for the thrash guard's
// progress signal: `landed` must track commits that actually reach trunk, not a
// task-status delta (which counts a proposed-but-unmerged PR as progress and
// lets a --pr --auto loop that never lands anything dodge NoProgressHalt). A
// real commit on trunk moves the marker by exactly one; a cycle that merges
// nothing leaves it flat.
func TestTrunkMarkerReflectsTrunkAdvance(t *testing.T) {
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt") // ensure a born trunk branch
	d := newDriver(w, &fakeRunner{}, &Governor{})
	d.trunkBranch = d.resolveTrunkBranch()

	before := d.trunkMarker()
	if flat := d.trunkMarker(); flat != before {
		t.Fatalf("marker must be stable when trunk does not move: %d vs %d", before, flat)
	}
	commitTo(t, w.Root, "landed.txt")
	if after := d.trunkMarker(); after != before+1 {
		t.Fatalf("marker delta want +1 after a trunk commit, got before=%d after=%d", before, after)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
