package queues

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	t.Setenv(agentid.EnvVar, "") // act as root (rw) unless a test says otherwise
	w, err := workspace.Init(t.TempDir(), "queues-test")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func newCtx(cwd string) (*clikit.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd}, &out, &errb
}

// becomeReadOnlyChild mints a read-only agent and makes it the acting identity
// for the rest of the test.
func becomeReadOnlyChild(t *testing.T, w *workspace.Workspace) string {
	t.Helper()
	id, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "junior", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)
	return id
}

func becomeOtherRWChild(t *testing.T, w *workspace.Workspace) string {
	t.Helper()
	id, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "peer", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)
	return id
}

// A queue with no steps is not a queue. dacli never executes a step, so the
// steps are the entire content — creating an empty one would produce an object
// that can only ever report "complete".
func TestQueueAddRequiresSlugAndSteps(t *testing.T) {
	w := newWS(t)
	cases := [][]string{
		nil,
		{"deploy"},                      // slug but no steps
		{"--step", "run the migration"}, // steps but no slug
	}
	for _, args := range cases {
		ctx, _, _ := newCtx(w.Root)
		err := cmdAdd(ctx, args)
		if code := clikit.ExitCode(err); code != 2 {
			t.Errorf("cmdAdd(%v) exit %d, want 2 (err %v)", args, code, err)
		}
	}
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAdd(ctx, []string{"deploy", "--stepp", "x"})); code != 2 {
		t.Error("a typo'd --step must be a usage error, not a queue with zero steps")
	}
	if qs, _ := store.ListQueues(w); len(qs) != 0 {
		t.Errorf("a refused add created %d queue(s)", len(qs))
	}
}

func TestQueueAddAndAdvanceWalksTheCursor(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"deploy", "--title", "Ship it", "--step", "build", "--step", "test", "--step", "release"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "3 steps") {
		t.Errorf("add report = %q, want the step count", out)
	}

	// `queue next` reports the CURRENT step, 1-indexed for humans, and does not
	// move the cursor — reading is not advancing.
	for i := 0; i < 2; i++ {
		ctx, out, _ := newCtx(w.Root)
		if err := cmdNext(ctx, []string{"deploy"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "step 1/3: build") {
			t.Fatalf("cmdNext = %q, want 'step 1/3: build' (reading must not advance)", out)
		}
	}

	want := []string{"next → step 2/3: test", "next → step 3/3: release", "queue complete"}
	for _, w2 := range want {
		ctx, out, _ := newCtx(w.Root)
		if err := cmdAdvance(ctx, []string{"deploy"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), w2) {
			t.Fatalf("advance printed %q, want %q", out, w2)
		}
	}
	// A completed queue keeps saying complete rather than wrapping around.
	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdNext(ctx2, []string{"deploy"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "queue complete") {
		t.Errorf("a finished queue reported %q", out2)
	}
}

// --fail halts the queue. A halted queue must REFUSE to hand out its next step:
// continuing past a failed step is exactly what the halt exists to prevent.
func TestQueueHaltStopsTheCursor(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"deploy", "--step", "build", "--step", "release"}); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdAdvance(ctx, []string{"deploy", "--fail", "build broke"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "queue halted: build broke") {
		t.Errorf("advance --fail printed %q", out)
	}

	ctx2, _, _ := newCtx(w.Root)
	err := cmdNext(ctx2, []string{"deploy"})
	if err == nil {
		t.Fatal("a halted queue must not hand out its next step")
	}
	if !strings.Contains(err.Error(), "build broke") {
		t.Errorf("the halt refusal %q does not carry the reason", err)
	}

	ctx3, out3, _ := newCtx(w.Root)
	if err := cmdList(ctx3, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out3.String(), "HALTED: build broke") {
		t.Errorf("queue list must surface the halt: %q", out3)
	}
}

// The cursor is mutable state with exactly ONE writer. A different agent — even
// a read-write one — must be refused, or two agents racing the same checklist
// silently skip steps.
func TestQueueAdvanceHasExactlyOneWriter(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"deploy", "--step", "build", "--step", "release"}); err != nil {
		t.Fatal(err)
	}

	other := becomeOtherRWChild(t, w)
	ctx2, _, _ := newCtx(w.Root)
	err := cmdAdvance(ctx2, []string{"deploy"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("a non-owner rw agent advancing: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), agentid.RootID) {
		t.Errorf("the refusal %q must name the owner so %s knows who to ask", err, other)
	}
	// And the cursor did not move.
	q, err := store.LoadQueue(w, "deploy")
	if err != nil {
		t.Fatal(err)
	}
	if q.Cursor != 0 {
		t.Errorf("a refused advance moved the cursor to %d", q.Cursor)
	}
}

// Advancing rewrites the cursor, so it needs an rw grant even from the queue's
// own owner: a read-only agent may report, never rewrite.
func TestQueueAdvanceRequiresRWEvenForTheOwner(t *testing.T) {
	w := newWS(t)
	child := becomeReadOnlyChild(t, w)

	// The read-only child may CREATE a queue (an append), and owns it.
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"deploy", "--step", "build", "--step", "release"}); err != nil {
		t.Fatal(err)
	}
	q, err := store.LoadQueue(w, "deploy")
	if err != nil {
		t.Fatal(err)
	}
	if q.Owner != child {
		t.Fatalf("queue owner = %q, want the creating child %q", q.Owner, child)
	}

	ctx2, _, _ := newCtx(w.Root)
	err = cmdAdvance(ctx2, []string{"deploy"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("read-only owner advancing: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "rw grant") {
		t.Errorf("the refusal %q must name the grant, not the ownership", err)
	}
	if q2, _ := store.LoadQueue(w, "deploy"); q2.Cursor != 0 {
		t.Errorf("a refused advance moved the cursor to %d", q2.Cursor)
	}
}

func TestQueueCommandsOnMissingQueue(t *testing.T) {
	w := newWS(t)
	for name, run := range map[string]func(*clikit.Ctx, []string) error{"next": cmdNext, "advance": cmdAdvance} {
		ctx, _, _ := newCtx(w.Root)
		if code := clikit.ExitCode(run(ctx, []string{"no-such-queue"})); code != 4 {
			t.Errorf("%s on a missing queue: exit %d, want 4", name, code)
		}
		ctx2, _, _ := newCtx(w.Root)
		if code := clikit.ExitCode(run(ctx2, nil)); code != 2 {
			t.Errorf("%s with no slug: exit %d, want 2", name, code)
		}
	}
}

// Every queue command is registered under the path the CLI dispatches on; a
// renamed path silently removes the command from the surface.
func TestCommandsAreRegistered(t *testing.T) {
	want := map[string]bool{"queue add": false, "queue list": false, "queue next": false, "queue advance": false}
	for _, c := range Commands {
		if _, ok := want[c.Path]; !ok {
			t.Errorf("unexpected command path %q", c.Path)
			continue
		}
		want[c.Path] = true
		if c.Run == nil || c.Brief == "" {
			t.Errorf("command %q is missing a Run or Brief", c.Path)
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("command %q is no longer registered", path)
		}
	}
}
