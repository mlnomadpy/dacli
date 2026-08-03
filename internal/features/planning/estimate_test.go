package planning

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func estEnv(t *testing.T) (*clikit.Ctx, *workspace.Workspace) {
	t.Helper()
	t.Setenv("DACLI_AGENT", "")
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "P", "p", "", ""); err != nil {
		t.Fatal(err)
	}
	return &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}, w
}

// An estimate could only be set at creation, so a backlog filed without one was
// permanently unsizable — and every scheduling command that needs estimates
// (critical-path, next's slack ordering, next --parallel) silently degraded to
// MoSCoW-then-sequence, the one ordering that cannot express what runs
// concurrently (dacli 228).
func TestTaskEstimateSizesAnExistingTask(t *testing.T) {
	ctx, w := estEnv(t)
	tk, err := store.CreateTask(w, agentid.RootID, "p", "unsized work", store.TaskOpts{Accept: []string{"ok"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tk.Estimate(); ok {
		t.Fatal("precondition: the task should start unsized")
	}

	if err := cmdTaskEstimate(ctx, []string{tk.Slug, "--estimate", "2,5,14"}); err != nil {
		t.Fatalf("task estimate: %v", err)
	}

	// It must survive a reload — an estimate that only exists in memory is not
	// an estimate.
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	tp, ok := got.Estimate()
	if !ok {
		t.Fatal("the estimate did not survive a reload")
	}
	if tp.Optimistic != 2 || tp.Probable != 5 || tp.Pessimistic != 14 {
		t.Errorf("estimate = (%g,%g,%g), want (2,5,14)", tp.Optimistic, tp.Probable, tp.Pessimistic)
	}
	// PERT: (o + 4m + p) / 6 = (2 + 20 + 14)/6 = 6
	if e := tp.Expected(); e < 5.99 || e > 6.01 {
		t.Errorf("Te = %g, want 6", e)
	}
}

// Re-sizing is the normal case — you learn the shape of work by looking at it.
func TestTaskEstimateOverwritesAnExistingSize(t *testing.T) {
	ctx, w := estEnv(t)
	tk, err := store.CreateTask(w, agentid.RootID, "p", "resized work", store.TaskOpts{Accept: []string{"ok"}, Estimate: "1,2,3"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdTaskEstimate(ctx, []string{tk.Slug, "--estimate", "10,20,30"}); err != nil {
		t.Fatal(err)
	}
	got, _ := store.FindTask(w, tk.Slug)
	tp, _ := got.Estimate()
	if tp.Probable != 20 {
		t.Errorf("probable = %g, want 20 (a re-size must replace, not append)", tp.Probable)
	}
}

// A scalar estimate hides the risk the third point exists to state, so it is a
// usage error rather than a silently-accepted guess.
func TestTaskEstimateRejectsNonThreePoint(t *testing.T) {
	ctx, w := estEnv(t)
	tk, _ := store.CreateTask(w, agentid.RootID, "p", "bad size", store.TaskOpts{Accept: []string{"ok"}})

	for _, bad := range []string{"5", "1,2", "1,2,3,4", "1,,3", " , , "} {
		err := cmdTaskEstimate(ctx, []string{tk.Slug, "--estimate", bad})
		if err == nil {
			t.Errorf("estimate %q was accepted; a malformed estimate must be refused", bad)
			continue
		}
		if clikit.ExitCode(err) != 2 {
			t.Errorf("estimate %q: exit %d, want 2 (usage)", bad, clikit.ExitCode(err))
		}
	}
	// And none of those attempts may have written anything.
	got, _ := store.FindTask(w, tk.Slug)
	if _, ok := got.Estimate(); ok {
		t.Error("a refused estimate must not have been written")
	}
}

func TestTaskEstimateUsageAndNotFound(t *testing.T) {
	ctx, w := estEnv(t)
	tk, _ := store.CreateTask(w, agentid.RootID, "p", "some work", store.TaskOpts{Accept: []string{"ok"}})

	if err := cmdTaskEstimate(ctx, []string{tk.Slug}); clikit.ExitCode(err) != 2 {
		t.Errorf("missing --estimate: exit %d, want 2", clikit.ExitCode(err))
	}
	if err := cmdTaskEstimate(ctx, []string{"999", "--estimate", "1,2,3"}); clikit.ExitCode(err) != 4 {
		t.Errorf("unknown ref: exit %d, want 4 (not found)", clikit.ExitCode(err))
	}
	if err := cmdTaskEstimate(ctx, []string{tk.Slug, "--estimat", "1,2,3"}); clikit.ExitCode(err) != 2 {
		t.Errorf("typo'd flag: exit %d, want 2", clikit.ExitCode(err))
	}
	_ = strings.TrimSpace("")
}
