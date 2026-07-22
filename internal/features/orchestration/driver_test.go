package orchestration

import (
	"bytes"
	"os/exec"
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

// shipOutputRunner behaves like fakeRunner but returns a scripted stdout for
// the "ship" phase, so a test can feed it real ship/integrate output (e.g.
// the --auto path, which queues GitHub auto-merge but lands nothing in-cycle)
// and assert what the driver derives as `landed` from it.
type shipOutputRunner struct {
	fakeRunner
	shipOut string
}

func (r *shipOutputRunner) run(label string, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	if label == "ship" {
		return r.shipOut, nil
	}
	return "", nil
}

// TestRunCycleLandedReflectsShipMergeCount is the driver-level regression for
// the thrash guard's "no net progress" signal: under the default --pr --auto
// path, ship queues GitHub auto-merge (lifecycle.go prIntegrateTask returns
// landed=false) and merges nothing in-cycle, even though `accept --all`
// already closed the tasks as StatusDone. `landed` must come from ship's
// reported merge count — never a done-task-count delta, which would count
// queued-but-unmerged work as trunk progress and let a --auto loop that never
// actually lands anything dodge the NoProgressHalt guard forever.
func TestRunCycleLandedReflectsShipMergeCount(t *testing.T) {
	w := loopEnv(t)
	if _, err := store.CreateTask(w, "a-root", "p", "Feature A", store.TaskOpts{Accept: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	ready, err := readyTasks(w, "p")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		shipOut string
		want    int
	}{
		{
			name:    "--auto queues PRs but merges nothing this cycle",
			shipOut: "queued 1 PR(s) for auto-merge targeting main — GitHub merges each when CI passes (hands-off)\n",
			want:    0,
		},
		{
			name:    "--no-merge opens PRs for human review, merges nothing",
			shipOut: "opened 1 PR(s) targeting main, none merged (--no-merge) — review and merge them yourself\n",
			want:    0,
		},
		{
			name:    "gated/local path actually merges into trunk",
			shipOut: "integrated 1 branch(es) into main, no conflicts\n",
			want:    1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &shipOutputRunner{shipOut: tc.shipOut}
			d := newDriver(w, r, &Governor{MaxCycles: 1, NoProgressHalt: 3})
			landed, _ := d.runCycle(ready)
			if landed != tc.want {
				t.Fatalf("landed = %d, want %d (ship reported: %q)", landed, tc.want, tc.shipOut)
			}
		})
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
