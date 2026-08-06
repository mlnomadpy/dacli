package shortcuts

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/shortcut"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	if v, ok := os.LookupEnv(agentid.EnvVar); ok {
		t.Setenv(agentid.EnvVar, v)
		_ = os.Unsetenv(agentid.EnvVar)
	}
	w, err := workspace.Init(t.TempDir(), "shortcuts-test")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func newCtx(cwd string) (*clikit.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd}, &out, &errb
}

func becomeChild(t *testing.T, w *workspace.Workspace, role string, grant model.Grant) string {
	t.Helper()
	id, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, role, grant)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)
	return id
}

func TestCmdAddUsageAndRejects(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAdd(ctx, nil)); code != 2 {
		t.Error("shortcut add with no name/command must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAdd(ctx2, []string{"lint", "--command", "go vet", "--bogus", "x"})); code != 2 {
		t.Error("a typo'd flag must be a usage error, not silently dropped")
	}
}

// Defining a shortcut is a write to the capability surface — a read-only
// agent must not be able to define one and then run it as the operator.
func TestCmdAddNeedsRW(t *testing.T) {
	w := newWS(t)
	becomeChild(t, w, "auditor", model.GrantRO)
	ctx, _, _ := newCtx(w.Root)
	err := cmdAdd(ctx, []string{"lint", "--command", "go vet ./...", "--effect", "read"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro shortcut add: exit %d, want 3 (err %v)", code, err)
	}
}

func TestCmdAddCreatesAShortcut(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	err := cmdAdd(ctx, []string{"lint", "--command", "go vet ./...", "--effect", "read",
		"--summary", "runs vet", "--param", "pkg=./...", "--role", "backend", "--why", "saves typing"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `shortcut "lint" defined`) {
		t.Errorf("must confirm the shortcut was defined: %q", out)
	}
	sc, err := store.LoadShortcut(w, "lint")
	if err != nil {
		t.Fatal(err)
	}
	if sc.Command != "go vet ./..." || sc.Effect != shortcut.EffectRead {
		t.Errorf("shortcut was not written correctly: %+v", sc)
	}
}

func TestCmdPromoteUsageAndNotFound(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdPromote(ctx, nil)); code != 2 {
		t.Error("shortcut promote with no name/from-event must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdPromote(ctx2, []string{"name", "--from-event", "01NOPE", "--effect", "read"})); code != 4 {
		t.Error("promoting from an unknown event must be a not-found")
	}
}

// Promotion is refused when the event names an already-named shortcut run,
// not an ad-hoc one — there is nothing to promote.
func TestCmdPromoteRefusesAlreadyNamedShortcut(t *testing.T) {
	w := newWS(t)
	ev, err := eventlog.Append(w, agentid.RootID, model.EventRun, "lint", "", "go vet\nexit 0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	err = cmdPromote(ctx, []string{"promoted", "--from-event", ev.ID, "--effect", "read"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("promoting an already-named run: exit %d, want 3 (err %v)", code, err)
	}
}

// A one-off ad-hoc command is not "repeated" — promotion requires at least
// two runs of the identical command.
func TestCmdPromoteRefusesASingleRun(t *testing.T) {
	w := newWS(t)
	cmdStr := "echo hi"
	ev, err := eventlog.Append(w, agentid.RootID, model.EventRun, adhocKey(cmdStr), "", cmdStr+"\nexit 0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	err = cmdPromote(ctx, []string{"echoer", "--from-event", ev.ID, "--effect", "read"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("promoting a single run: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "run once") {
		t.Errorf("refusal must explain it ran once: %v", err)
	}
}

func TestCmdPromoteSucceedsAfterTwoRuns(t *testing.T) {
	w := newWS(t)
	cmdStr := "echo hi"
	key := adhocKey(cmdStr)
	if _, err := eventlog.Append(w, agentid.RootID, model.EventRun, key, "", cmdStr+"\nexit 0"); err != nil {
		t.Fatal(err)
	}
	ev2, err := eventlog.Append(w, agentid.RootID, model.EventRun, key, "", cmdStr+"\nexit 0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdPromote(ctx, []string{"echoer", "--from-event", ev2.ID, "--effect", "read", "--summary", "says hi"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "promoted ad-hoc command (2 runs)") {
		t.Errorf("must report the repeat count: %q", out)
	}
	sc, err := store.LoadShortcut(w, "echoer")
	if err != nil {
		t.Fatal(err)
	}
	if sc.Command != cmdStr {
		t.Errorf("promoted shortcut command = %q, want %q", sc.Command, cmdStr)
	}
}

func TestCmdRunUsage(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRun(ctx, nil)); code != 2 {
		t.Error("run with nothing must be a usage error")
	}
}

func TestCmdRunUnknownShortcut(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRun(ctx, []string{"nope"})); code != 4 {
		t.Error("running an unknown shortcut must be a not-found")
	}
}

func TestCmdRunList(t *testing.T) {
	w := newWS(t)
	if err := store.CreateShortcut(w, agentid.RootID, "lint", "runs vet", "go vet ./...", "read", nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRun(ctx, []string{"--list"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "lint") {
		t.Errorf("--list must show the shortcut: %q", out)
	}
}

func TestCmdRunDryRunExpandsWithoutExecuting(t *testing.T) {
	w := newWS(t)
	if err := store.CreateShortcut(w, agentid.RootID, "greet", "", "echo {{name}}", "write", []string{"name"}, nil, ""); err != nil {
		t.Fatal(err)
	}
	becomeChild(t, w, "auditor", model.GrantRO)

	ctx, out, _ := newCtx(w.Root)
	if err := cmdRun(ctx, []string{"greet", "--name", "world", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "echo world" {
		t.Errorf("dry-run must print the expanded command: %q", out)
	}
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
	if len(events) != 0 {
		t.Error("dry-run must not record a run event")
	}
}

// The effect gate blocks a write shortcut for a read-only agent even outside
// dry-run.
func TestCmdRunGuardBlocksWriteForReadOnlyAgent(t *testing.T) {
	w := newWS(t)
	if err := store.CreateShortcut(w, agentid.RootID, "touch", "", "true", "write", nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	becomeChild(t, w, "auditor", model.GrantRO)

	ctx, _, _ := newCtx(w.Root)
	err := cmdRun(ctx, []string{"touch"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro running a write shortcut: exit %d, want 3 (err %v)", code, err)
	}
}

// A destructive shortcut needs both rw AND --confirm — never one token away
// from a read shortcut.
func TestCmdRunGuardRequiresConfirmForDestructive(t *testing.T) {
	w := newWS(t)
	if err := store.CreateShortcut(w, agentid.RootID, "wipe", "", "true", "destructive", nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	err := cmdRun(ctx, []string{"wipe"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("unconfirmed destructive run: exit %d, want 3 (err %v)", code, err)
	}

	ctx2, _, _ := newCtx(w.Root)
	if err := cmdRun(ctx2, []string{"wipe", "--confirm"}); err != nil {
		t.Fatalf("confirmed destructive run should succeed: %v", err)
	}
}

// A successful run executes, records an attributed event, and running it
// again shows up in --list's use count via FillUses.
func TestCmdRunExecutesAndRecordsEventAndUses(t *testing.T) {
	w := newWS(t)
	if err := store.CreateShortcut(w, agentid.RootID, "hello", "", "echo hi", "read", nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRun(ctx, []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hi") {
		t.Errorf("must actually execute the command: %q", out)
	}
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
	if len(events) != 1 || events[0].About != "hello" {
		t.Fatalf("want 1 run event about %q, got %+v", "hello", events)
	}

	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdRun(ctx2, []string{"--list"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "hello") {
		t.Errorf("used shortcut must appear in the catalog: %q", out2)
	}
}

// A failing command still records its event (with the error status), and the
// command's own failure is returned as an error.
func TestCmdRunRecordsEventOnFailure(t *testing.T) {
	w := newWS(t)
	if err := store.CreateShortcut(w, agentid.RootID, "fail", "", "exit 7", "read", nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	err := cmdRun(ctx, []string{"fail"})
	if err == nil {
		t.Fatal("a failing command must surface as an error")
	}
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
	if len(events) != 1 || strings.Contains(events[0].Body, "exit 0") {
		t.Fatalf("failure event must record the actual (non-zero) status: %+v", events)
	}
}

func TestCmdRunAdhocPath(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRun(ctx, []string{"--cmd", "echo adhoc-out"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "adhoc-out") {
		t.Errorf("ad-hoc command must actually run: %q", out)
	}
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
	if len(events) != 1 || !strings.HasPrefix(events[0].About, adhocPrefix) {
		t.Fatalf("ad-hoc run event must be keyed under the adhoc prefix: %+v", events)
	}
}

func TestCmdRunAdhocDryRun(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdRun(ctx, []string{"--cmd", "echo hi", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "echo hi" {
		t.Errorf("ad-hoc dry-run must print the literal command: %q", out)
	}
	events, _ := eventlog.List(w, eventlog.Query{})
	if len(events) != 0 {
		t.Error("ad-hoc dry-run must not record an event")
	}
}

// An ad-hoc command has no declared effect to gate on, so it never runs
// read-only, unattended.
func TestCmdRunAdhocNeedsRW(t *testing.T) {
	w := newWS(t)
	becomeChild(t, w, "auditor", model.GrantRO)
	ctx, _, _ := newCtx(w.Root)
	err := cmdRun(ctx, []string{"--cmd", "echo hi"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro ad-hoc run: exit %d, want 3 (err %v)", code, err)
	}
}

func TestFillUsesCountsRunEvents(t *testing.T) {
	w := newWS(t)
	if _, err := eventlog.Append(w, agentid.RootID, model.EventRun, "lint", "", "go vet\nexit 0"); err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, agentid.RootID, model.EventRun, "lint", "", "go vet\nexit 0"); err != nil {
		t.Fatal(err)
	}
	scs := []shortcut.Shortcut{{Name: "lint"}, {Name: "unused"}}
	FillUses(w, scs)
	if scs[0].Uses != 2 {
		t.Errorf("lint uses = %d, want 2", scs[0].Uses)
	}
	if scs[1].Uses != 0 {
		t.Errorf("unused uses = %d, want 0", scs[1].Uses)
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	want := map[string]bool{"shortcut add": false, "shortcut promote": false, "run": false}
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
