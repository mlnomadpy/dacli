package collab

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const testProject = "proj"

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
	w, err := workspace.Init(t.TempDir(), "collab-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "Test project", testProject, "Ship it.", ""); err != nil {
		t.Fatal(err)
	}
	return w
}

func newCtx(cwd string) (*clikit.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd}, &out, &errb
}

func mustTask(t *testing.T, w *workspace.Workspace, title string) *store.Task {
	t.Helper()
	task, err := store.CreateTask(w, agentid.RootID, testProject, title, store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	return task
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

// A question you can proceed without was a comment, not an ask: `dacli ask`
// BLOCKS the asking task. If it did not, an agent would keep working on a
// premise it just admitted it does not have.
func TestAskBlocksTheTask(t *testing.T) {
	w := newWS(t)
	task := mustTask(t, w, "Audit the write paths")

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAsk(ctx, []string{"Which ledger is authoritative?", "--about", "001", "--need", "docs/ledger.md"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "blocked until answered") {
		t.Errorf("ask reported %q", out)
	}

	got, err := store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusBlocked {
		t.Errorf("task status = %q, want blocked", got.Status)
	}

	events, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one help event, got %d", len(events))
	}
	if !strings.Contains(events[0].Body, "Which ledger is authoritative?") {
		t.Errorf("question body lost: %q", events[0].Body)
	}
	// --need is carried into the question, not dropped: it is what the asker
	// says would unblock them.
	if !strings.Contains(events[0].Body, "need: docs/ledger.md") {
		t.Errorf("--need was dropped from the question: %q", events[0].Body)
	}
}

func TestAskUsageErrors(t *testing.T) {
	w := newWS(t)
	mustTask(t, w, "a task")
	cases := [][]string{
		nil,                                      // no question, no --about
		{"Why?"},                                 // question but no --about
		{"--about", "001"},                       // --about but no question
		{"Why?", "--about", "001", "--abt", "x"}, // typo'd flag
	}
	for _, args := range cases {
		ctx, _, _ := newCtx(w.Root)
		if code := clikit.ExitCode(cmdAsk(ctx, args)); code != 2 {
			t.Errorf("cmdAsk(%v) exit %d, want 2", args, code)
		}
	}
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAsk(ctx, []string{"Why?", "--about", "999"})); code != 4 {
		t.Error("asking about an unknown task must be a not-found")
	}
}

// The question is transient; the ANSWER is permanent. It lands as a durable
// note that enters every future brief in scope, the question is marked applied
// so it stops showing as open, and the task unblocks.
func TestAnswerCreatesADurableNoteAndUnblocks(t *testing.T) {
	w := newWS(t)
	task := mustTask(t, w, "Audit the write paths")

	ctx, _, _ := newCtx(w.Root)
	if err := cmdAsk(ctx, []string{"Which ledger is authoritative?", "--about", "001"}); err != nil {
		t.Fatal(err)
	}
	pending, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}, Pending: true})
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending questions = %d (%v)", len(pending), err)
	}
	qid := pending[0].ID

	ctx2, out, _ := newCtx(w.Root)
	if err := cmdAnswer(ctx2, []string{qid[:10], "The", "new", "ledger", "is", "authoritative."}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "recorded as a") {
		t.Errorf("answer reported %q", out)
	}

	// A durable note carrying BOTH sides of the exchange.
	notes, err := store.ListNotes(w, testProject, model.NoteFinding)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected one note, got %d", len(notes))
	}
	body := notes[0].Sections[len(notes[0].Sections)-1].Content
	joined := body
	for _, s := range notes[0].Sections {
		joined += s.Content
	}
	if !strings.Contains(joined, "Which ledger is authoritative?") || !strings.Contains(joined, "The new ledger is authoritative.") {
		t.Errorf("the note must carry the question AND the answer:\n%s", joined)
	}

	// The question is no longer open.
	stillPending, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}, Pending: true})
	if len(stillPending) != 0 {
		t.Errorf("the answered question is still listed as pending")
	}
	got, err := store.FindTask(w, task.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusBlocked {
		t.Error("the task stayed blocked after being answered")
	}
}

func TestAnswerRefusals(t *testing.T) {
	w := newWS(t)
	mustTask(t, w, "a task")
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAnswer(ctx, []string{"onlyone"})); code != 2 {
		t.Error("answer needs both an id and an answer")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAnswer(ctx2, []string{"01NOPE", "some", "answer"})); code != 4 {
		t.Error("answering a question that is not open must be a not-found")
	}
}

// `dacli threads` attributes each answer to the QUESTION it resolved, not to
// the task. Two questions on one task answered by different agents must each
// name their own answerer — keying by task would collapse them to one.
func TestThreadsAttributesPerQuestion(t *testing.T) {
	w := newWS(t)
	mustTask(t, w, "a task")

	ctx, _, _ := newCtx(w.Root)
	if err := cmdAsk(ctx, []string{"First question?", "--about", "001"}); err != nil {
		t.Fatal(err)
	}
	ctx1, _, _ := newCtx(w.Root)
	if err := cmdAsk(ctx1, []string{"Second question?", "--about", "001"}); err != nil {
		t.Fatal(err)
	}
	pending, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}, Pending: true})
	if len(pending) != 2 {
		t.Fatalf("expected two open questions, got %d", len(pending))
	}
	byBody := map[string]string{}
	for _, e := range pending {
		byBody[strings.SplitN(e.Body, "\n", 2)[0]] = e.ID
	}

	// Root answers the first...
	ctxA, _, _ := newCtx(w.Root)
	if err := cmdAnswer(ctxA, []string{byBody["First question?"][:10], "root", "says", "yes"}); err != nil {
		t.Fatal(err)
	}
	// ...a different agent answers the second.
	other := becomeChild(t, w, "reviewer", model.GrantRW)
	ctxB, _, _ := newCtx(w.Root)
	if err := cmdAnswer(ctxB, []string{byBody["Second question?"][:10], "reviewer", "says", "no"}); err != nil {
		t.Fatal(err)
	}

	unsetAgentEnv(t)
	ctxT, out, _ := newCtx(w.Root)
	if err := cmdThreads(ctxT, nil); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two thread lines:\n%s", out)
	}
	answerers := map[string]string{}
	for _, ln := range lines {
		switch {
		case strings.Contains(ln, "First question?"):
			answerers["first"] = ln
		case strings.Contains(ln, "Second question?"):
			answerers["second"] = ln
		}
	}
	if !strings.Contains(answerers["first"], "answered by "+agentid.RootID) {
		t.Errorf("first question mis-attributed: %q", answerers["first"])
	}
	if !strings.Contains(answerers["second"], "answered by "+other) {
		t.Errorf("second question mis-attributed (want %s): %q", other, answerers["second"])
	}
}

func TestThreadsEmptyState(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdThreads(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no questions asked yet") {
		t.Errorf("threads printed %q, want an explicit empty state", out)
	}
}

// Escalation is the terminal hop: it leaves the tree. The LOCAL escalation is
// open to any agent — that is the point — so it must work under a read-only
// grant and still record a durable, answerable event.
func TestEscalateIsOpenToReadOnlyAgents(t *testing.T) {
	w := newWS(t)
	mustTask(t, w, "a task")
	becomeChild(t, w, "junior", model.GrantRO)

	ctx, out, _ := newCtx(w.Root)
	if err := cmdEscalate(ctx, []string{"Nobody", "here", "owns", "the", "billing", "contract", "--about", "001"}); err != nil {
		t.Fatalf("a read-only agent must be able to escalate: %v", err)
	}
	if !strings.Contains(out.String(), "a human does now") {
		t.Errorf("escalate reported %q", out)
	}
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}})
	if len(events) != 1 || !strings.Contains(events[0].Body, "[escalation to human]") {
		t.Fatalf("escalation event = %+v", events)
	}
}

// Filing a public GitHub issue is a REMOTE write, so --github is rw-only. The
// refusal must land before any `gh` invocation — this test never touches the
// network and would fail (or hang) if the gate moved after the exec.
func TestEscalateGithubRequiresRW(t *testing.T) {
	w := newWS(t)
	mustTask(t, w, "a task")
	becomeChild(t, w, "junior", model.GrantRO)

	ctx, _, _ := newCtx(w.Root)
	err := cmdEscalate(ctx, []string{"Escalating", "upward", "--github"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("--github as a read-only agent: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "escalating with --github") {
		t.Errorf("refusal %q does not name the action", err)
	}
	// The local escalation still stands — the remote mirror failing must not
	// erase the fact that someone escalated.
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}})
	if len(events) != 1 {
		t.Errorf("the local escalation event was lost when --github was refused (%d events)", len(events))
	}
}

func TestEscalateUsageErrors(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdEscalate(ctx, nil)); code != 2 {
		t.Error("escalate with no summary must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdEscalate(ctx2, []string{"x", "--githb"})); code != 2 {
		t.Error("a typo'd --github must be a usage error, not a silently local-only escalation")
	}
}

// `sync` applies pending child events to objects the caller owns and reports
// both halves of the split: what it applied and what is still pending. A silent
// zero would be indistinguishable from a broken sync.
func TestSyncReportsAppliedAndPending(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdSync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "sync: 0 applied, 0 left pending") {
		t.Errorf("sync on an empty log reported %q", out)
	}
}

func TestEventsDismissAuthorizationAuditAndTaskCleanup(t *testing.T) {
	w := newWS(t)
	owner := becomeChild(t, w, "maintainer", model.GrantRW)
	task, err := store.CreateTask(w, owner, testProject, "obsolete orphan", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	claim, err := eventlog.Append(w, agentid.RootID, model.EventClaim, task.ID, "", "diagnostic claim")
	if err != nil {
		t.Fatal(err)
	}
	block, err := eventlog.Append(w, agentid.RootID, model.EventBlock, task.ID, "", "diagnostic block")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := eventlog.Append(w, "a-retired-diagnostic-author", model.EventBlock, task.ID, "", "foreign diagnostic block")
	if err != nil {
		t.Fatal(err)
	}

	// An unrelated sibling, including rw, cannot reject another actor's event.
	becomeChild(t, w, "reviewer", model.GrantRW)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdEventsDismiss(ctx, []string{foreign.ID, "--reason", "not mine"})); code != 3 {
		t.Fatalf("unrelated sibling dismissal exit = %d, want refusal 3", code)
	}

	// Root's exceptional authority is restricted to a known RETIRED owner.
	unsetAgentEnv(t)
	ctx, _, _ = newCtx(w.Root)
	if code := clikit.ExitCode(cmdEventsDismiss(ctx, []string{foreign.ID, "--reason", "owner still live"})); code != 3 {
		t.Fatalf("root rejected live child-owned event with exit %d", code)
	}
	if err := store.RetireAgent(w, owner); err != nil {
		t.Fatal(err)
	}
	ctxList, listed, _ := newCtx(w.Root)
	if err := cmdEventsPending(ctxList, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{claim.ID, block.ID, "actor=" + agentid.RootID, "action=claim", "action=block", "target=" + task.ID, "authorization=reject-retired-owner-orphan"} {
		if !strings.Contains(listed.String(), want) {
			t.Errorf("pending listing missing %q:\n%s", want, listed)
		}
	}

	ctxDismiss, out, _ := newCtx(w.Root)
	if err := cmdEventsDismiss(ctxDismiss, []string{claim.ID, "--reason", "superseded by orphan recovery"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdEventsDismiss(ctxDismiss, []string{block.ID, "--reason", "superseded by orphan recovery"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdEventsDismiss(ctxDismiss, []string{foreign.ID, "--reason", "retired owner cannot resolve it"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdEventsDismiss(ctxDismiss, []string{claim.ID, "--reason", "repeated operator request"}); err != nil {
		t.Fatalf("repeated command dismissal was not idempotent: %v", err)
	}
	if !strings.Contains(out.String(), "superseded by orphan recovery") {
		t.Fatalf("dismissal output omitted reason: %s", out)
	}
	if err := store.RemoveTask(w, task); err != nil {
		t.Fatalf("dismissed pending reference still blocked RemoveTask: %v", err)
	}

	all, err := eventlog.List(w, eventlog.Query{})
	if err != nil {
		t.Fatal(err)
	}
	dismissals := 0
	for _, event := range all {
		if event.Kind == model.EventDismissal && (event.About == claim.ID || event.About == block.ID || event.About == foreign.ID) {
			dismissals++
			if event.Actor != agentid.RootID || strings.TrimSpace(event.Body) == "" {
				t.Errorf("incomplete audit record: %+v", event)
			}
		}
	}
	if dismissals != 3 {
		t.Fatalf("dismissal audit records = %d, want exactly one for each proposal", dismissals)
	}
}

// The loop anchor is owned by the synthetic "loop" identity, rather than by
// the reviewer it repeatedly spawns. Once that reviewer's run is finished,
// neither it nor the synthetic owner can consume a proposal, so root must be
// able to leave an audited disposition instead of an impossible pending item.
func TestRootDismissesFinishedUnretiredProposalOnLoopAnchor(t *testing.T) {
	w := newWS(t)
	anchor, err := store.CreateTask(w, "loop", testProject,
		store.ContinuousImprovementMarker+": file the next evidence-based change", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	author := becomeChild(t, w, "loop-auditor", model.GrantRO)
	proposal, err := eventlog.Append(w, author, model.EventProposeStatus, anchor.ID, "", "propose: done")
	if err != nil {
		t.Fatal(err)
	}
	runDir := w.RunDir("01FINISHEDLOOPAUDITOR0000")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
		RunID: "01FINISHEDLOOPAUDITOR0000", Child: author, Task: anchor.ID,
		PID: 999999, PGID: 999999, Outcome: "ok",
	}); err != nil {
		t.Fatal(err)
	}

	unsetAgentEnv(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdEventsDismiss(ctx, []string{proposal.ID, "--reason", "stale loop review"}); err != nil {
		t.Fatalf("root could not dismiss retired loop proposal: %v", err)
	}
	if !strings.Contains(out.String(), "stale loop review") {
		t.Errorf("dismissal output omitted reason: %s", out)
	}

	original, err := eventlog.Find(w, proposal.ID)
	if err != nil {
		t.Fatalf("dismissal removed original proposal: %v", err)
	}
	if !original.Dismissed || original.Pending {
		t.Errorf("original proposal = %+v, want retained, dismissed, and no longer pending", original)
	}
	dismissals, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventDismissal}})
	if err != nil || len(dismissals) != 1 {
		t.Fatalf("dismissal audit records = %d, err=%v", len(dismissals), err)
	}
	if got := dismissals[0]; got.Actor != agentid.RootID || got.About != proposal.ID || strings.TrimSpace(got.Body) != "stale loop review" {
		t.Errorf("dismissal audit record = %+v", got)
	}
	pending, err := eventlog.List(w, eventlog.Query{Pending: true})
	if err != nil || len(pending) != 0 {
		t.Fatalf("dismissed loop proposal still counts as pending: %+v, err=%v", pending, err)
	}
	ctxPending, listed, _ := newCtx(w.Root)
	if err := cmdEventsPending(ctxPending, nil); err != nil || !strings.Contains(listed.String(), "no pending events") {
		t.Fatalf("events pending after dismissal = %q, err=%v", listed, err)
	}
}

func TestRootCannotDismissLiveOrUnknownActorProposalOnLoopAnchor(t *testing.T) {
	w := newWS(t)
	anchor, err := store.CreateTask(w, "loop", testProject,
		store.ContinuousImprovementMarker+": file the next evidence-based change", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	live := becomeChild(t, w, "loop-auditor", model.GrantRO)
	liveProposal, err := eventlog.Append(w, live, model.EventProposeStatus, anchor.ID, "", "propose: done")
	if err != nil {
		t.Fatal(err)
	}
	self := os.Getpid()
	start, _ := procmon.ProcStart(self)
	runDir := w.RunDir("01LIVELOOPAUDITOR00000000")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
		RunID: "01LIVELOOPAUDITOR00000000", Child: live, Task: anchor.ID,
		PID: self, PGID: self, PIDStart: start,
	}); err != nil {
		t.Fatal(err)
	}
	unknown, err := eventlog.Append(w, "a-unknown-loop-auditor", model.EventProposeStatus, anchor.ID, "", "propose: done")
	if err != nil {
		t.Fatal(err)
	}
	neverRan := becomeChild(t, w, "loop-auditor", model.GrantRO)
	neverRanProposal, err := eventlog.Append(w, neverRan, model.EventProposeStatus, anchor.ID, "", "propose: done")
	if err != nil {
		t.Fatal(err)
	}

	unsetAgentEnv(t)
	for _, proposal := range []*eventlog.Event{liveProposal, unknown, neverRanProposal} {
		ctx, _, _ := newCtx(w.Root)
		err := cmdEventsDismiss(ctx, []string{proposal.ID, "--reason", "must remain pending"})
		if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "refused-unrelated") {
			t.Errorf("root dismissed non-terminal loop actor %s: %v (exit %d)", proposal.Actor, err, clikit.ExitCode(err))
		}
	}
}

func TestLoopAnchorRecoveryFailsClosedForUnresolvedOrCorruptTargets(t *testing.T) {
	w := newWS(t)
	author := becomeChild(t, w, "loop-auditor", model.GrantRO)
	if err := store.RetireAgent(w, author); err != nil {
		t.Fatal(err)
	}
	missing, err := eventlog.Append(w, author, model.EventProposeStatus, "t-missing", "", "propose: done")
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := store.CreateTask(w, "loop", testProject,
		store.ContinuousImprovementMarker+": file the next evidence-based change", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	corrupt, err := eventlog.Append(w, author, model.EventProposeStatus, anchor.ID, "", "propose: done")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(anchor.Path, []byte("---\nunterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	unsetAgentEnv(t)
	for _, event := range []*eventlog.Event{missing, corrupt} {
		ctx, _, _ := newCtx(w.Root)
		err := cmdEventsDismiss(ctx, []string{event.ID, "--reason", "must remain pending"})
		if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "refused-unresolved-target") {
			t.Errorf("root dismissed unresolved/corrupt target %s: %v (exit %d)", event.ID, err, clikit.ExitCode(err))
		}
	}
}

func TestReadOnlyAuthorCanWithdrawOwnEventButNotAnother(t *testing.T) {
	w := newWS(t)
	task := mustTask(t, w, "proposal target")
	author := becomeChild(t, w, "reviewer", model.GrantRO)
	owned, err := eventlog.Append(w, author, model.EventBlock, task.ID, "", "my proposal")
	if err != nil {
		t.Fatal(err)
	}
	other, err := eventlog.Append(w, agentid.RootID, model.EventClaim, task.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	if err := cmdEventsDismiss(ctx, []string{owned.ID, "--reason", "withdrawn after review"}); err != nil {
		t.Fatalf("ro author could not withdraw own event: %v", err)
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdEventsDismiss(ctx2, []string{other.ID, "--reason", "not allowed"})); code != 3 {
		t.Fatalf("ro author dismissed another event with exit %d", code)
	}
}

func TestEventsDismissRefusesAppliedEventWithCompensatingWorkflow(t *testing.T) {
	w := newWS(t)
	task := mustTask(t, w, "applied proposal target")
	event, err := eventlog.Append(w, agentid.RootID, model.EventBlock, task.ID, "", "already applied")
	if err != nil {
		t.Fatal(err)
	}
	if err := eventlog.MarkApplied(event.Path); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	err = cmdEventsDismiss(ctx, []string{event.ID, "--reason", "erase history"})
	if err == nil || clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "append a compensating event") {
		t.Fatalf("applied dismissal = %v (exit %d), want refusal naming compensating event", err, clikit.ExitCode(err))
	}
}

func TestCorruptDismissalDoesNotUnblockTaskRemoval(t *testing.T) {
	w := newWS(t)
	task := mustTask(t, w, "proposal target")
	proposal, err := eventlog.Append(w, "a-reviewer", model.EventBlock, task.ID, "", "still relevant")
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	if err := cmdEventsDismiss(ctx, []string{proposal.ID, "--reason", "initially valid"}); err != nil {
		t.Fatal(err)
	}
	dispositions, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventDismissal}})
	if err != nil || len(dispositions) != 1 {
		t.Fatalf("dismissal record: events=%d err=%v", len(dispositions), err)
	}
	doc, err := mdstore.ReadFile(dispositions[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	doc.Sections[0].Content = "tampered reason\n"
	if err := mdstore.WriteFile(dispositions[0].Path, doc); err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveTask(w, task); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("corrupt dismissal unblocked canonical removal: %v", err)
	}
}

func TestEventsTailRespectsLimit(t *testing.T) {
	w := newWS(t)
	task := mustTask(t, w, "a task")
	for i := 0; i < 5; i++ {
		if _, err := eventlog.Append(w, agentid.RootID, model.EventFinding, task.ID, "", "finding number "+string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
	}
	ctx, out, _ := newCtx(w.Root)
	if err := cmdEventsTail(ctx, []string{"--limit", "2"}); err != nil {
		t.Fatal(err)
	}
	if got := len(strings.Split(strings.TrimSpace(out.String()), "\n")); got != 2 {
		t.Errorf("--limit 2 printed %d lines:\n%s", got, out)
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdEventsTail(ctx2, []string{"--limitt", "2"})); code != 2 {
		t.Error("a typo'd --limit must be a usage error")
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	want := map[string]bool{
		"sync": false, "events tail": false, "events pending": false, "events dismiss": false, "ask": false,
		"answer": false, "threads": false, "escalate": false,
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
