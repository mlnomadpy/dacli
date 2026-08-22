package teamops

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// unsetAgentEnv clears DACLI_AGENT for the test, restoring whatever the
// process started with. t.Setenv cannot unset a variable, and since dacli 288
// a present-but-empty DACLI_AGENT is a lost token that fails closed rather
// than resolving to root — so a test wanting the root identity must remove
// the variable entirely, not blank it.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv(agentid.EnvVar); ok {
		t.Setenv(agentid.EnvVar, v)
		_ = os.Unsetenv(agentid.EnvVar)
	}
}

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	unsetAgentEnv(t)
	w, err := workspace.Init(t.TempDir(), "teamops-test")
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

// writeRun lays down the run record a real `dacli spawn` writes, so the
// traceability join (id → run → task) is exercised against the actual format
// rather than a stub.
func writeRun(t *testing.T, w *workspace.Workspace, runID, child, task, role string) {
	t.Helper()
	dir := w.RunDir(runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Child: child, Task: task, Role: role, Runtime: "claude-code", PID: 1, PGID: 1, Started: time.Now()}
	if err := procmon.WriteRecord(filepath.Join(dir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
}

func writeLiveRun(t *testing.T, w *workspace.Workspace, runID, child, role string) {
	t.Helper()
	pid := os.Getpid()
	start, _ := procmon.ProcStart(pid)
	rec := procmon.Record{RunID: runID, Child: child, Role: role, PID: pid, PGID: pid, PIDStart: start, Started: time.Now()}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
}

func mustRole(t *testing.T, w *workspace.Workspace, r team.Role) {
	t.Helper()
	if err := store.CreateRole(w, agentid.RootID, r); err != nil {
		t.Fatal(err)
	}
}

// The token goes to stdout ALONE so `TOKEN=$(dacli agent spawn ...)` captures
// exactly it; everything human-facing goes to stderr. Any stray stdout byte
// corrupts the captured credential.
func TestAgentSpawnPutsOnlyTheTokenOnStdout(t *testing.T) {
	w := newWS(t)
	mustRole(t, w, team.Role{Name: "reviewer", Skills: []string{"code-review"}, Shortcuts: []string{"lint"}})

	ctx, out, errb := newCtx(w.Root)
	if err := cmdAgentSpawn(ctx, []string{"--role", "reviewer"}); err != nil {
		t.Fatal(err)
	}
	token := strings.TrimSpace(out.String())
	if token == "" || strings.ContainsAny(token, " \t") || strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("stdout must carry the bare token and nothing else; got %q", out)
	}
	// The token resolves back to a real identity in this workspace.
	t.Setenv(agentid.EnvVar, token)
	id, err := agentid.Resolve(w)
	if err != nil {
		t.Fatalf("the printed token does not resolve: %v", err)
	}
	if id.Role != "reviewer" {
		t.Errorf("resolved role = %q, want reviewer", id.Role)
	}
	// The role's mechanical bundle is reported — on stderr.
	for _, want := range []string{"code-review", "lint", "spawned "} {
		if !strings.Contains(errb.String(), want) {
			t.Errorf("stderr missing %q:\n%s", want, errb)
		}
	}
}

// A role's grant is a ceiling REQUEST; attenuation against the PARENT still
// wins. A read-only agent must not be able to escalate by naming an rw role.
func TestAgentSpawnAttenuationBeatsTheRoleGrant(t *testing.T) {
	w := newWS(t)
	mustRole(t, w, team.Role{Name: "writer", Grant: "rw"})
	becomeChild(t, w, "junior", model.GrantRO)

	ctx, out, _ := newCtx(w.Root)
	err := cmdAgentSpawn(ctx, []string{"--role", "writer"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("a ro agent spawning into an rw role: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "your grant is ro") {
		t.Errorf("refusal %q must name the caller's own grant", err)
	}
	if out.Len() != 0 {
		t.Errorf("a refused spawn printed %q to stdout — a caller capturing it would get garbage", out)
	}
}

// WIP is preventable, not merely detectable: the refusal happens before the
// next child exists. A process finishing frees the slot again.
func TestAgentSpawnWIPLimit(t *testing.T) {
	w := newWS(t)
	mustRole(t, w, team.Role{Name: "junior", WIP: 2})

	for i := 0; i < 2; i++ {
		ctx, _, _ := newCtx(w.Root)
		if err := cmdAgentSpawn(ctx, []string{"--role", "junior"}); err != nil {
			t.Fatalf("spawn %d: %v", i, err)
		}
	}
	agents, _ := store.ListAgents(w)
	for i, a := range agents {
		if a.Role == "junior" {
			writeLiveRun(t, w, fmt.Sprintf("RUN-WIP-%d", i), a.ID, "junior")
		}
	}
	if got, err := store.ActiveInRole(w, "junior"); err != nil || got != 2 {
		t.Fatalf("active in role = (%d, %v), want (2, nil)", got, err)
	}

	ctx, out, _ := newCtx(w.Root)
	err := cmdAgentSpawn(ctx, []string{"--role", "junior"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("over-WIP spawn: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "WIP limit (2/2)") {
		t.Errorf("refusal %q must report the actual counts", err)
	}
	if out.Len() != 0 {
		t.Errorf("a refused spawn still minted and printed a token: %q", out)
	}
	if got, err := store.ActiveInRole(w, "junior"); err != nil || got != 2 {
		t.Errorf("a refused spawn created a third agent (active=%d, err=%v)", got, err)
	}

	// A terminal run frees the slot. Identity retirement alone must not make a
	// still-running process disappear from the capacity read model.
	agents, _ = store.ListAgents(w)
	var victim string
	for _, a := range agents {
		if a.Role == "junior" {
			victim = a.ID
			break
		}
	}
	ctx2, _, _ := newCtx(w.Root)
	if err := cmdAgentRetire(ctx2, []string{victim}); err != nil {
		t.Fatal(err)
	}
	for i, a := range agents {
		if a.ID == victim {
			runID := fmt.Sprintf("RUN-WIP-%d", i)
			if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{RunID: runID, Child: a.ID, Role: "junior", PID: 0}); err != nil {
				t.Fatal(err)
			}
		}
	}
	ctx3, _, _ := newCtx(w.Root)
	if err := cmdAgentSpawn(ctx3, []string{"--role", "junior"}); err != nil {
		t.Errorf("finished run did not free the WIP slot: %v", err)
	}
}

// Retiring rewrites an agent file, so it needs an rw grant; and lineage
// survives retirement — attribution outlives the agent.
func TestAgentRetireRefusals(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAgentRetire(ctx, nil)); code != 2 {
		t.Error("retire with no agent id must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAgentRetire(ctx2, []string{"a-nope"})); code != 4 {
		t.Error("retiring an unknown agent must be a not-found")
	}

	target := becomeChild(t, w, "junior", model.GrantRO)
	writeLiveRun(t, w, "RUN-RETIRE-REFUSAL", target, "junior")
	ctx3, _, _ := newCtx(w.Root)
	err := cmdAgentRetire(ctx3, []string{target})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("a ro agent retiring: exit %d, want 3 (err %v)", code, err)
	}
	if got, err := store.ActiveInRole(w, "junior"); err != nil || got != 1 {
		t.Errorf("a refused retire still freed the slot (active=%d, err=%v)", got, err)
	}
}

// Role removal is a policy refusal while a live child still holds the
// capability. The public command must preserve exit 3 and name both handles an
// operator can inspect or stop (issue #690).
func TestRoleRmRefusesLiveHolderWithChildAndRun(t *testing.T) {
	w := newWS(t)
	mustRole(t, w, team.Role{Name: "ephemeral", Grant: "rw"})
	child := becomeChild(t, w, "ephemeral", model.GrantRW)
	unsetAgentEnv(t)
	runID := "01RUNLIVEHOLDER00000000000"
	writeRun(t, w, runID, child, "t-live", "ephemeral")

	ctx, _, _ := newCtx(w.Root)
	err := cmdRoleRm(ctx, []string{"ephemeral"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("role rm with live holder exit = %d, want 3 (err %v)", code, err)
	}
	if msg := err.Error(); !strings.Contains(msg, child) || !strings.Contains(msg, runID) {
		t.Fatalf("live-holder refusal must name child and run: %v", err)
	}
	if _, ok := store.LoadRole(w, "ephemeral"); !ok {
		t.Fatal("refused role rm deleted the role")
	}
}

// agent tree renders lineage with write attribution. A child must appear
// INDENTED under its parent — a flat list loses the delegation structure the
// tree exists to show.
func TestAgentTreeShowsLineage(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAgentSpawn(ctx, []string{"--role", "lead", "--grant", "rw"}); err != nil {
		t.Fatal(err)
	}
	unsetAgentEnv(t) // back to root

	ctx2, out, _ := newCtx(w.Root)
	if err := cmdAgentTree(ctx2, nil); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected root + one child, got:\n%s", out)
	}
	if !strings.HasPrefix(lines[0], agentid.RootID) {
		t.Errorf("root must head the tree; got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "  a-") {
		t.Errorf("the child must be indented under its parent; got %q", lines[1])
	}
	if !strings.Contains(lines[1], "lead") || !strings.Contains(lines[1], "rw") {
		t.Errorf("the child line must carry its role and grant; got %q", lines[1])
	}
}

// Traceability (dacli 225): the tree must join the run records, so an operator
// reading it learns what an agent is working on and where its work is recorded
// — without opening a run dir by hand.
func TestAgentTreeShowsTaskAndRun(t *testing.T) {
	w := newWS(t)
	child := becomeChild(t, w, "go-auditor", model.GrantRO)
	unsetAgentEnv(t)
	writeRun(t, w, "01RUNAAAAAAAAAAAAAAAAAAAAA", child, "t-42", "go-auditor")

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgentTree(ctx, nil); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "task t-42") {
		t.Errorf("tree must name the task the agent was spawned for:\n%s", got)
	}
	if !strings.Contains(got, "run 01RUNAAAAAAAAAAAAAAAAAAAAA") {
		t.Errorf("tree must name the run that holds the agent's work:\n%s", got)
	}
	if !strings.Contains(got, "go-auditor") {
		t.Errorf("tree must carry the role:\n%s", got)
	}
}

// agent show is the "I read an unfamiliar id in git log" command: one id in,
// role + lineage + run + task out. Every one of those must be present, or the
// operator is back to grepping .dacli by hand.
func TestAgentShowResolvesRoleLineageAndRun(t *testing.T) {
	w := newWS(t)
	child := becomeChild(t, w, "go-auditor", model.GrantRO)
	unsetAgentEnv(t)
	writeRun(t, w, "01RUNBBBBBBBBBBBBBBBBBBBBB", child, "t-7", "go-auditor")

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgentShow(ctx, []string{child}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{child, "go-auditor", "ro", agentid.RootID + " → " + child, "01RUNBBBBBBBBBBBBBBBBBBBBB", "task t-7"} {
		if !strings.Contains(got, want) {
			t.Errorf("agent show missing %q:\n%s", want, got)
		}
	}

	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAgentShow(ctx2, nil)); code != 2 {
		t.Error("agent show with no id must be a usage error")
	}
	// An id with no file still yields what the id itself says — a commit trailer
	// can name an agent whose file is not in this checkout.
	ctx3, _, _ := newCtx(w.Root)
	err := cmdAgentShow(ctx3, []string{"a-fixer-7k3q"})
	if clikit.ExitCode(err) != 4 {
		t.Fatalf("unknown agent must be a not-found; got %v", err)
	}
	if !strings.Contains(err.Error(), "fixer") {
		t.Errorf("a missing agent file must still report the role the id carries: %v", err)
	}
}

// An OLD-format id (minted before dacli 225) must be shown exactly as well as a
// new one: existing workspaces are full of them, and `agent show` is the tool
// an operator reaches for precisely when the id is unfamiliar.
func TestAgentShowHandlesOldFormatIDs(t *testing.T) {
	w := newWS(t)
	old := "a-4w4dtttpe8"
	d := &mdstore.Doc{}
	d.Front.Set("id", old)
	d.Front.Set("kind", string(model.KindAgent))
	d.Front.Set("parent", "[["+agentid.RootID+"]]")
	d.Front.Set("grant", "rw")
	d.Front.Set("role", "frontend-engineer")
	if err := mdstore.WriteFile(w.AgentPath(old), d); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgentShow(ctx, []string{old}); err != nil {
		t.Fatalf("agent show refused an old-format id: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "frontend-engineer") || !strings.Contains(got, agentid.RootID+" → "+old) {
		t.Errorf("old-format id lost its role or lineage:\n%s", got)
	}

	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdAgentTree(ctx2, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), old) {
		t.Errorf("agent tree dropped an old-format id:\n%s", out2)
	}
}

// A role must change what an agent can DO, not just what it calls itself. A
// name-only role is a costume: warn loudly (it can be filled in later) but
// still create it.
func TestRoleAddWarnsAboutACostumeRole(t *testing.T) {
	w := newWS(t)
	ctx, _, errb := newCtx(w.Root)
	if err := cmdRoleAdd(ctx, []string{"architect", "--summary", "Thinks big thoughts"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errb.String(), "costume, not a role") {
		t.Errorf("a mechanically-empty role was accepted silently: %q", errb)
	}
	if _, ok := store.LoadRole(w, "architect"); !ok {
		t.Error("the warning must not block creation")
	}

	// A role with ANY mechanical field must not draw the warning. Enumerated:
	// the condition is a long conjunction and one dropped clause makes the
	// warning fire (or stop firing) for a whole class of real roles.
	for i, args := range [][]string{
		{"r1", "--skill", "code-review"},
		{"r2", "--scope", "internal/**"},
		{"r3", "--shortcut", "lint"},
		{"r4", "--escalate-to", "architect"},
		{"r5", "--grant", "rw"},
		{"r6", "--wip", "3"},
		{"r7", "--model", "opus"},
		{"r8", "--runtime", "claude-code"},
		{"r9", "--max-points", "5"},
		{"r10", "--kind", "reviewer"},
	} {
		ctx, _, errb := newCtx(w.Root)
		if err := cmdRoleAdd(ctx, args); err != nil {
			t.Fatalf("case %d (%v): %v", i, args, err)
		}
		if strings.Contains(errb.String(), "costume") {
			t.Errorf("case %d (%v) is mechanically real but drew the costume warning", i, args)
		}
	}
}

func TestRoleAddRejectsUnknownFlags(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRoleAdd(ctx, []string{"r", "--skils", "x"})); code != 2 {
		t.Error("a typo'd --skill must be a usage error, not a silently skill-less role")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRoleAdd(ctx2, nil)); code != 2 {
		t.Error("role add with no name must be a usage error")
	}
}

func TestRoleAddDeclaresProviderNeutralModelProfile(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	err := cmdRoleAdd(ctx, []string{"profiled", "--kind", "implementer", "--runtime", "generic-cli",
		"--model-id", "frontier-medium", "--cost-tier", "12", "--max-task-points", "8",
		"--context-limit", "200000", "--capability-tag", "code", "--capability-tag", "vision"})
	if err != nil {
		t.Fatal(err)
	}
	r, ok := store.LoadRole(w, "profiled")
	if !ok {
		t.Fatal("created role was not readable")
	}
	if r.Runtime != "generic-cli" || r.Profile.ID != "frontier-medium" || r.Profile.CostTier != 12 ||
		r.Profile.MaxTaskPoints != 8 || r.Profile.ContextLimit != 200000 || len(r.Profile.CapabilityTags) != 2 {
		t.Fatalf("profile did not round-trip: runtime=%q profile=%+v", r.Runtime, r.Profile)
	}
}

// Bumping a role version rewrites its file — rw only — and an unknown role is a
// not-found rather than a silently created v2.
func TestRoleBumpRefusals(t *testing.T) {
	w := newWS(t)
	mustRole(t, w, team.Role{Name: "reviewer", Skills: []string{"code-review"}})

	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRoleBump(ctx, []string{"nope"})); code != 4 {
		t.Error("bumping an unknown role must be a not-found")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRoleBump(ctx2, nil)); code != 2 {
		t.Error("bump with no name must be a usage error")
	}

	becomeChild(t, w, "junior", model.GrantRO)
	ctx3, _, _ := newCtx(w.Root)
	err := cmdRoleBump(ctx3, []string{"reviewer"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("a ro agent bumping: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "rw grant") {
		t.Errorf("refusal %q must name the missing grant", err)
	}
}

func TestRoleShowUnknownRole(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRoleShow(ctx, []string{"nope"})); code != 4 {
		t.Error("showing an unknown role must be a not-found")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdRoleShow(ctx2, nil)); code != 2 {
		t.Error("role show with no name must be a usage error")
	}
}

// team route answers "who owns this path, and how do I reach them". The G8
// rule: an owner that EXISTS but is unreachable is a missing escalation edge,
// not a dead end — and the message has to say which, or the caller cannot tell
// "nobody owns this" from "you can't get there from here".
func TestTeamRouteDistinguishesUnownedFromUnreachable(t *testing.T) {
	w := newWS(t)

	ctx, _, _ := newCtx(w.Root)
	if err := cmdTeamRoute(ctx, []string{"internal/store"}); err == nil {
		t.Error("routing with no roles defined must error, not print an empty answer")
	}
	ctx0, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdTeamRoute(ctx0, nil)); code != 2 {
		t.Error("team route with no path must be a usage error")
	}

	mustRole(t, w, team.Role{Name: "storekeeper", Scope: []string{"internal/store/**"}})
	// A scoped role with NO escalate_to: it cannot reach storekeeper. (An
	// empty scope would be permissive and make it own everything.)
	mustRole(t, w, team.Role{Name: "loner", Scope: []string{"cmd/**"}})

	// Nothing covers this path at all.
	ctx1, out1, _ := newCtx(w.Root)
	if err := cmdTeamRoute(ctx1, []string{"docs/index.md"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out1.String(), "no role covers") {
		t.Errorf("unowned path reported %q", out1)
	}

	// Owned, and reachable from the owner itself.
	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdTeamRoute(ctx2, []string{"internal/store/roles.go", "--from", "storekeeper"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "owners (most specific first): storekeeper") {
		t.Errorf("owner not reported: %q", out2)
	}
	if !strings.Contains(out2.String(), "chain from storekeeper") {
		t.Errorf("chain not reported: %q", out2)
	}

	// Owned, but no escalation edge leads there: a missing edge, named as such.
	ctx3, _, _ := newCtx(w.Root)
	err := cmdTeamRoute(ctx3, []string{"internal/store/roles.go", "--from", "loner"})
	if err == nil {
		t.Fatal("an unreachable owner must be reported, not silently omitted")
	}
	if !strings.Contains(err.Error(), "storekeeper owns this but is not reachable") {
		t.Errorf("error %q does not distinguish unreachable from unowned", err)
	}
	if !strings.Contains(err.Error(), "escalate_to") {
		t.Errorf("error %q does not name the remedy (the missing edge)", err)
	}
}

// `dacli team` is the roster: WIP headroom per role, and an explicit count of
// agents with no role — unroled agents are invisible to every WIP limit, so
// they have to be surfaced somewhere.
func TestTeamRosterReportsHeadroomAndUnroledAgents(t *testing.T) {
	w := newWS(t)
	mustRole(t, w, team.Role{Name: "junior", WIP: 3, Summary: "Small tasks"})
	mustRole(t, w, team.Role{Name: "senior", Summary: "Anything"})

	var liveChild string
	for i := 0; i < 2; i++ {
		child, _, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "junior", model.GrantRO)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			liveChild = child
		}
	}
	if _, _, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "", model.GrantRO); err != nil {
		t.Fatal(err)
	}

	pid := os.Getpid()
	start, _ := procmon.ProcStart(pid)
	if err := procmon.WriteRecord(filepath.Join(w.RunDir("RUN-LIVE"), "proc.txt"), procmon.Record{RunID: "RUN-LIVE", Child: liveChild, Role: "junior", PID: pid, PGID: pid, PIDStart: start}); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdTeam(ctx, nil); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "junior         occupancy:1 headroom:2") {
		t.Errorf("junior headroom wrong:\n%s", got)
	}
	if !strings.Contains(got, "senior         occupancy:0 headroom:∞") {
		t.Errorf("an uncapped role must report unbounded headroom:\n%s", got)
	}
	if !strings.Contains(got, "(plus 1 agents with no role)") {
		t.Errorf("unroled agents must be surfaced — they escape every WIP limit:\n%s", got)
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	want := map[string]bool{
		"agent spawn": false, "agent tree": false, "agent show": false, "agent retire": false,
		"role add": false, "role rm": false, "role list": false, "role show": false, "role bump": false,
		"team": false, "team route": false, "team assign": false,
	}
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

// Two identical write-tests tasks, one word apart, must route to the same kind.
// inferKind scanned every word and returned the first keyword found anywhere,
// so an incidental noun overrode the verb that states the intent: "Write the
// tests the suite audit calls for" matched "audit" and routed pure code-writing
// to a role whose charter is "never implements" (task 318, hit live on 315).
func TestKindComesFromTheLeadingVerbNotAnIncidentalNoun(t *testing.T) {
	w := teamopsWS(t)

	mention := mustTask(t, w, "Write the tests the suite audit calls for")
	control := mustTask(t, w, "Write the unit tests the suite requires")

	kMention, _ := inferKind(w, mention)
	kControl, _ := inferKind(w, control)
	if kMention != kControl {
		t.Errorf("titles differing only by an incidental noun routed differently: %q vs %q", kMention, kControl)
	}
	if kMention != "implementer" {
		t.Errorf("writing tests is implementer work, got %q", kMention)
	}

	// A title whose LEADING verb really is a review verb still classifies.
	audit := mustTask(t, w, "Audit the system for swallowed errors")
	if k, _ := inferKind(w, audit); k != "reviewer" {
		t.Errorf("a leading review verb must still classify as reviewer, got %q", k)
	}

	// And a modifier before the verb is tolerated.
	full := mustTask(t, w, "Full audit of the event log")
	if k, _ := inferKind(w, full); k != "reviewer" {
		t.Errorf("a modifier before the verb must not hide it, got %q", k)
	}
}

func TestLeadingImplementationIntentBlocksLaterReviewerVerb(t *testing.T) {
	w := teamopsWS(t)

	for _, leading := range []string{"Test", "Check", "Improve", "Cover"} {
		for _, reviewer := range []string{"verify", "audit", "review"} {
			title := leading + " " + reviewer + " behavior"
			if got, src := inferKind(w, mustTask(t, w, title)); got != "implementer" {
				t.Errorf("%q routed to %q (via %s), want implementer", title, got, src)
			}
		}
	}

	for _, title := range []string{"Full verify behavior", "Full audit behavior", "Full review behavior"} {
		if got, src := inferKind(w, mustTask(t, w, title)); got != "reviewer" {
			t.Errorf("%q routed to %q (via %s), want reviewer", title, got, src)
		}
	}
}

func teamopsWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

func mustTask(t *testing.T, w *workspace.Workspace, title string) *store.Task {
	t.Helper()
	task, err := store.CreateTask(w, "a-root", "core", title,
		store.TaskOpts{Accept: []string{"done"}, Estimate: "1,2,3"})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

// Every verb in the table routes to the kind it declares, and — the half that
// matters — a verb an IMPLEMENTATION task would plausibly lead with still
// falls through to implementer. Task 318 fixed review work leaking to
// implementers; the opposite leak is just as expensive, because the reviewer
// roles are charted "never implements" and would refuse the work outright.
//
// The additions came from two live misroutes: 324 ("Falsify the
// safety-property suites…") and 325 ("Trace one user-invoked verb end to
// end…") are pure audit work that matched nothing and routed to `fixer`
// (task 326).
func TestEveryKindVerbRoutesAndImplementationVerbsStillFallThrough(t *testing.T) {
	w := teamopsWS(t)

	for verb, want := range kindVerbs {
		// Leading position, which is where inferKind looks.
		task := mustTask(t, w, strings.ToUpper(verb[:1])+verb[1:]+" the event log for drift")
		if got, src := inferKind(w, task); got != want {
			t.Errorf("title verb %q routed to %q (via %s), want %q", verb, got, src, want)
		}
	}

	// The two titles that actually misrouted, verbatim.
	for _, title := range []string{
		"Falsify the safety-property suites: name a surviving mutation or report none exists",
		"Trace one user-invoked verb end to end across slice seams and name where the report diverges from the effect",
	} {
		if got, _ := inferKind(w, mustTask(t, w, title)); got != "reviewer" {
			t.Errorf("%q routed to %q; this is audit work whose whole output is a filed finding", title, got)
		}
	}

	// The table must stay small and high-signal. These lead real
	// implementation tasks, so admitting them would send code-writing to a
	// role that refuses to write code — task 318 in reverse.
	for _, title := range []string{
		"Test the retry path against a flapping remote",
		"Check the token is refreshed before every call",
		"Fix the swallowed error in the gate reader",
		"Improve the brief assembly for worktree agents",
		"Cover the accept arc with an integration test",
	} {
		if got, src := inferKind(w, mustTask(t, w, title)); got != "implementer" {
			t.Errorf("%q routed to %q (via %s); this is code to write, not a review", title, got, src)
		}
	}
}

// `agent spawn`'s WIP check is the SIBLING of gateRoleWIP, and it was left
// discarding ActiveInRole's error when task 341 widened the signature — the
// "rule applied in four places and missed in a fifth" pattern that produced
// every capability bug in this codebase. It refuses a spawn, so it must fail
// closed: a WIP cap that cannot be read is not a WIP cap of zero.
func TestAgentSpawnFailsClosedWhenTheWIPCountCannotBeRead(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read a 0000 directory")
	}
	w := teamopsWS(t)
	if err := store.CreateRole(w, agentid.RootID, team.Role{Name: "capped", Grant: "ro", WIP: 1}); err != nil {
		t.Fatal(err)
	}
	// A spawn with the agents dir readable establishes the baseline: it works.
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdAgentSpawn(ctx, []string{"--role", "capped", "--grant", "ro"}); err != nil {
		t.Fatalf("baseline spawn should succeed: %v", err)
	}

	// Now make the count unreadable. The gate must REFUSE, not read it as
	// "nobody holds this role" and wave the spawn through.
	if err := os.Chmod(w.AgentsDir(), 0o000); err != nil {
		t.Skipf("cannot make the agents dir unreadable: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(w.AgentsDir(), 0o755) })

	ctx2 := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	err := cmdAgentSpawn(ctx2, []string{"--role", "capped", "--grant", "ro"})
	if err == nil {
		t.Fatal("the spawn was allowed while the WIP count was unreadable — the cap failed OPEN")
	}
	if got := clikit.ExitCode(err); got != 3 {
		t.Errorf("exit %d, want 3 (policy refusal): %v", got, err)
	}
	if !strings.Contains(err.Error(), "WIP") {
		t.Errorf("the refusal must name what it could not check: %v", err)
	}
}
