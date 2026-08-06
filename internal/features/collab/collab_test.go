package collab

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
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
		"sync": false, "events tail": false, "ask": false,
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
