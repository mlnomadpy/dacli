package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// status drives one request and returns just the code — the shared primitive for
// the detail endpoints' 400/404 contract.
func status(t *testing.T, h http.Handler, target string) int {
	t.Helper()
	return do(t, h, "GET", target, "localhost")
}

// bodyOf returns a response's raw bytes as a string, for assertions that must
// inspect what actually went over the wire rather than what decoded off it.
func bodyOf(t *testing.T, h http.Handler, target string) string {
	t.Helper()
	req := httptest.NewRequest("GET", target, nil)
	req.Host = "localhost"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("GET %s = %d: %s", target, rw.Code, rw.Body.String())
	}
	return rw.Body.String()
}

// TestAPITaskDetail is the task-detail contract (dacli 227): the summary row a
// board already knows, plus everything the board deliberately omits — the three
// points behind the estimate, the acceptance boxes with their checked state, the
// resolved dependency edges, and the Log.
func TestAPITaskDetail(t *testing.T) {
	w := dashboardEnv(t)

	// A third task that depends on the open one and on a ref that names nothing,
	// so both the resolved and the dangling edge are exercised.
	open, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatalf("find open task: %v", err)
	}
	dependent, err := store.CreateTask(w, "a-root", "core", "Depends on work", store.TaskOpts{
		Accept: []string{"first box", "second box"}, Estimate: "2,4,9",
		SoThat: "the chain is visible", Context: "filed by the detail test",
		DependsOn: []string{open.ID + ":SS", "t-01NOSUCHTASK00000000000AA"},
	})
	if err != nil {
		t.Fatalf("create dependent task: %v", err)
	}
	// Check one acceptance box and stamp the Log, so "partially done" is a real
	// state the response has to report rather than an all-or-nothing.
	dependent.Doc.SetSection("Acceptance", "- [x] first box\n- [ ] second box\n")
	store.AppendLog(dependent, "claimed by a-root")
	if err := store.SaveTask(dependent); err != nil {
		t.Fatalf("save: %v", err)
	}

	h := newHandler(w)
	var resp taskDetailResponse
	getJSON(t, h, "/api/task?ref="+dependent.ID, &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	got := resp.Task
	if got.ID != dependent.ID || got.Title != "Depends on work" || got.Status != "open" {
		t.Errorf("summary = %+v", got.taskView)
	}
	if got.Owner != "a-root" || got.Project != "core" {
		t.Errorf("owner/project = %q/%q, want a-root/core", got.Owner, got.Project)
	}
	if !got.Estimated || got.Points <= 0 {
		t.Errorf("estimated=%v points=%v, want a positive PERT expected", got.Estimated, got.Points)
	}
	if got.Estimate == nil {
		t.Fatalf("estimate is null for an estimated task")
	}
	if got.Estimate.Optimistic != 2 || got.Estimate.Probable != 4 || got.Estimate.Pessimistic != 9 {
		t.Errorf("estimate = %+v, want 2/4/9", *got.Estimate)
	}
	if got.Estimate.Expected != got.Points {
		t.Errorf("estimate.expected = %v, points = %v — they must agree", got.Estimate.Expected, got.Points)
	}
	if got.SoThat != "the chain is visible" || got.Context != "filed by the detail test" {
		t.Errorf("narrative = %q / %q", got.SoThat, got.Context)
	}

	if got.AcceptanceTotal != 2 || got.AcceptanceDone != 1 {
		t.Errorf("acceptance counts = %d/%d, want 1/2", got.AcceptanceDone, got.AcceptanceTotal)
	}
	if len(got.Acceptance) != 2 || !got.Acceptance[0].Done || got.Acceptance[1].Done {
		t.Errorf("acceptance boxes = %+v, want first checked and second not", got.Acceptance)
	}
	if got.Acceptance[0].Text != "first box" {
		t.Errorf("acceptance text = %q", got.Acceptance[0].Text)
	}

	if len(got.Deps) != 2 {
		t.Fatalf("deps = %d, want 2", len(got.Deps))
	}
	resolved, dangling := got.Deps[0], got.Deps[1]
	if !resolved.Resolved || resolved.Type != "SS" || resolved.ID != open.ID {
		t.Errorf("resolved dep = %+v, want the open task with type SS", resolved)
	}
	if resolved.Title != open.Title || resolved.Status != "open" {
		t.Errorf("resolved dep detail = %+v, want title/status from the target task", resolved)
	}
	// A dangling edge is REPORTED, not dropped: a dependency that silently
	// vanishes is a schedule that silently lies.
	if dangling.Resolved || dangling.ID != "" || dangling.Type != "FS" {
		t.Errorf("dangling dep = %+v, want unresolved with the default FS type", dangling)
	}

	if len(got.Log) != 1 {
		t.Fatalf("log = %+v, want the one claim line", got.Log)
	}
	if got.Log[0].Text != "claimed by a-root" {
		t.Errorf("log text = %q, want the stamp stripped off", got.Log[0].Text)
	}
	if _, err := time.Parse(time.RFC3339, got.Log[0].At); err != nil {
		t.Errorf("log at = %q, want an RFC3339 stamp: %v", got.Log[0].At, err)
	}
}

// TestAPITaskDetailUnestimatedIsNullNotZero proves an unestimated task reports a
// null estimate rather than a fabricated 0/0/0 — the difference between "not
// sized" and "sized at nothing".
func TestAPITaskDetailUnestimatedIsNullNotZero(t *testing.T) {
	w := dashboardEnv(t)
	bare, err := store.CreateTask(w, "a-root", "core", "Unsized", store.TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	h := newHandler(w)

	var resp taskDetailResponse
	getJSON(t, h, "/api/task?ref="+bare.ID, &resp)
	if resp.Task.Estimate != nil {
		t.Errorf("estimate = %+v, want null for an unestimated task", *resp.Task.Estimate)
	}
	if resp.Task.Estimated || resp.Task.Points != 0 {
		t.Errorf("estimated=%v points=%v, want false/0", resp.Task.Estimated, resp.Task.Points)
	}
	// Empty lists, never null: the shape is a contract.
	if resp.Task.Acceptance == nil || resp.Task.Deps == nil || resp.Task.Log == nil {
		t.Errorf("a bare task emitted a null list: %+v", resp.Task)
	}
}

// TestAPITaskRefContract covers the three ways a ref can fail: unusable (400,
// before the store is touched), absent (404 — a dead link must read as dead, not
// as an outage), and ambiguous (400 — the store refuses to guess, and so does
// this).
func TestAPITaskRefContract(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	for _, bad := range []string{"", "../../etc/passwd", "a/b", "..", "/etc/passwd"} {
		if code := status(t, h, "/api/task?ref="+bad); code != http.StatusBadRequest {
			t.Errorf("GET /api/task?ref=%q = %d, want 400", bad, code)
		}
	}
	if code := status(t, h, "/api/task?ref=t-01NOSUCHTASK00000000000AA"); code != http.StatusNotFound {
		t.Errorf("unknown task = %d, want 404", code)
	}

	// Two projects each holding a task numbered 001 makes "001" ambiguous.
	if _, err := store.CreateProject(w, "a-root", "Other", "other", "goal", "build"); err != nil {
		t.Fatalf("project: %v", err)
	}
	if _, err := store.CreateTask(w, "a-root", "other", "Ship the thing", store.TaskOpts{}); err != nil {
		t.Fatalf("task: %v", err)
	}
	if code := status(t, h, "/api/task?ref=001"); code != http.StatusBadRequest {
		t.Errorf("ambiguous ref = %d, want 400", code)
	}
}

// eventByKind picks the one event of a given kind out of a page.
func eventByKind(t *testing.T, events []eventView, kind string) eventView {
	t.Helper()
	for _, e := range events {
		if e.Kind == kind {
			return e
		}
	}
	t.Fatalf("no %s event in %+v", kind, events)
	return eventView{}
}

// TestAPIEvents is the event-history contract (dacli 227): kind, actor, about
// and a real timestamp, newest first, with the ?task= filter resolving a ref to
// the id events actually record.
func TestAPIEvents(t *testing.T) {
	w := dashboardEnv(t)
	open, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if _, err := eventlog.Append(w, "a-child1", model.EventClaim, open.ID, "", ""); err != nil {
		t.Fatalf("claim event: %v", err)
	}
	if _, err := eventlog.AppendFinding(w, "a-reviewer", model.EventFinding, open.ID,
		"file:internal/x.go:12", "a-child1", "the retry loop never terminates"); err != nil {
		t.Fatalf("finding event: %v", err)
	}
	h := newHandler(w)

	// The whole log: the env's project comment plus the two task events.
	var all eventsResponse
	getJSON(t, h, "/api/events", &all)
	if all.Generated == "" {
		t.Errorf("generated is empty")
	}
	if all.Task != "" {
		t.Errorf("task = %q, want empty for the whole-log query", all.Task)
	}
	if all.Limit != eventsDefaultLimit || all.Truncated {
		t.Errorf("limit = %d truncated = %v, want the default and no truncation", all.Limit, all.Truncated)
	}
	if all.Filters.Range != "all" || all.Filters.State != "all" {
		t.Errorf("default filters = %+v, want backward-compatible all/all", all.Filters)
	}
	if len(all.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(all.Events))
	}
	// Newest first. Asserted as descending ULID order rather than by naming the
	// last-appended event: three Appends can land in the same millisecond, where
	// the random half of the ULID decides — so pinning an index would be a flake,
	// while the ordering CONTRACT holds either way.
	for i := 1; i < len(all.Events); i++ {
		if all.Events[i-1].ID <= all.Events[i].ID {
			t.Errorf("events not newest-first: %s before %s", all.Events[i-1].ID, all.Events[i].ID)
		}
	}
	for _, e := range all.Events {
		if e.ID == "" || e.Actor == "" {
			t.Errorf("event missing identity: %+v", e)
		}
		if _, err := time.Parse(time.RFC3339, e.At); err != nil {
			t.Errorf("event at = %q, want an RFC3339 stamp decoded from the ULID: %v", e.At, err)
		}
		if e.Applied {
			t.Errorf("event %s applied = true, want false (nothing has synced)", e.ID)
		}
	}

	// The finding's review/taint fields survive the round trip.
	f := eventByKind(t, all.Events, string(model.EventFinding))
	if f.Against != "a-child1" || f.Origin != "file:<withheld-local-path>" {
		t.Errorf("finding fields = against %q origin %q", f.Against, f.Origin)
	}
	if f.Body != "the retry loop never terminates" {
		t.Errorf("finding body = %q", f.Body)
	}

	// ?task= resolves the ref to the task id events record, so a human-typed
	// "002" finds events filed against the ULID.
	var scoped eventsResponse
	getJSON(t, h, "/api/events?task=002", &scoped)
	if scoped.Task != open.ID {
		t.Errorf("resolved task = %q, want %q", scoped.Task, open.ID)
	}
	if len(scoped.Events) != 2 {
		t.Fatalf("scoped events = %d, want 2", len(scoped.Events))
	}
	for _, e := range scoped.Events {
		if e.About != open.ID {
			t.Errorf("scoped event about = %q, want %q", e.About, open.ID)
		}
	}
}

func TestAPIEventsCursorFiltersAndSafePartialProjection(t *testing.T) {
	w := dashboardEnv(t)
	open, err := store.FindTask(w, "002")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		body := fmt.Sprintf("finding %d", i)
		if i == 0 {
			body = `<img src="https://attacker.invalid/pixel"> token=ghp_123456789 /Users/operator/private.txt`
		}
		if _, err := eventlog.AppendFinding(w, "a-reviewer", model.EventFinding, open.ID, "file:/Users/operator/private.go:12", "a-worker", body); err != nil {
			t.Fatal(err)
		}
	}
	// A malformed durable record is reported as a partial page, not silently
	// presented as a complete log.
	badDir := filepath.Join(w.EventsDir(), "2026", "09", "01")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "not-an-event.md"), []byte("---\nbroken"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newHandler(w)

	var first eventsResponse
	getJSON(t, h, "/api/events?project=core&kind=finding&actor=a-reviewer&state=pending&range=24h&limit=2", &first)
	if len(first.Events) != 2 || !first.Truncated || first.NextCursor == "" {
		t.Fatalf("first page = %+v", first)
	}
	if !first.Partial || first.UnreadableRecords != 1 {
		t.Fatalf("malformed record disappeared: partial=%v unreadable=%d", first.Partial, first.UnreadableRecords)
	}
	if first.Filters.Project != "core" || first.Filters.Kind != "finding" || first.Filters.State != "pending" {
		t.Fatalf("filters = %+v", first.Filters)
	}

	var second eventsResponse
	getJSON(t, h, "/api/events?project=core&kind=finding&actor=a-reviewer&state=pending&range=24h&limit=2&cursor="+url.QueryEscape(first.NextCursor), &second)
	seen := map[string]bool{}
	for _, event := range append(first.Events, second.Events...) {
		if seen[event.ID] {
			t.Fatalf("cursor duplicated event %s", event.ID)
		}
		seen[event.ID] = true
		if event.Label != "Review finding" || event.Category != "finding" || event.RelatedTask != open.ID || event.RelatedAgent != "a-reviewer" {
			t.Errorf("typed event = %+v", event)
		}
	}
	if len(seen) != 4 {
		t.Fatalf("two pages contain %d unique events, want 4", len(seen))
	}

	var all eventsResponse
	getJSON(t, h, "/api/events?project=core&kind=finding&actor=a-reviewer&state=pending&range=24h&limit=10", &all)
	wire, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	text := string(wire)
	if strings.Contains(text, "ghp_123456789") || strings.Contains(text, "/Users/operator") {
		t.Fatalf("event projection leaked a secret or local path: %s", text)
	}
	maliciousBody := ""
	for _, event := range all.Events {
		if strings.Contains(event.Body, "attacker.invalid") {
			maliciousBody = event.Body
		}
	}
	if !strings.Contains(maliciousBody, `<img src="https://attacker.invalid/pixel">`) {
		t.Fatalf("untrusted markup was not retained as inert text fixture: %q", maliciousBody)
	}

	for _, target := range []string{
		"/api/events?state=maybe", "/api/events?range=forever", "/api/events?kind=unknown",
		"/api/events?actor=../worker", "/api/events?cursor=../event",
	} {
		if got := status(t, h, target); got != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400", target, got)
		}
	}
}

func TestSafeEventBodyBoundsValidUTF8(t *testing.T) {
	body := safeEventBody(strings.Repeat("界", eventBodyLimit))
	if !utf8.ValidString(body) {
		t.Fatalf("bounded body is invalid UTF-8: %q", body[len(body)-20:])
	}
	if !strings.HasSuffix(body, "[body truncated]") {
		t.Fatalf("bounded body did not disclose truncation")
	}
}

func TestEventPresentationUsesTypedTextLabels(t *testing.T) {
	tests := []struct {
		event    eventlog.Event
		label    string
		category string
	}{
		{eventlog.Event{Kind: model.EventBlock, Body: "policy refusal"}, "Policy refusal", "refusal"},
		{eventlog.Event{Kind: model.EventFinding}, "Review finding", "finding"},
		{eventlog.Event{Kind: model.EventHelp}, "Owner ask", "ask"},
		{eventlog.Event{Kind: model.EventReview}, "Review verdict", "review"},
		{eventlog.Event{Kind: model.EventDismissal}, "Reconciliation", "reconciliation"},
		{eventlog.Event{Kind: model.EventComment, Body: "root handoff required"}, "Owner handoff", "handoff"},
		{eventlog.Event{Kind: model.EventCommit}, "Delivery event", "delivery"},
	}
	for _, test := range tests {
		label, category := eventPresentation(&test.event)
		if label != test.label || category != test.category {
			t.Errorf("%s = %q/%q, want %q/%q", test.event.Kind, label, category, test.label, test.category)
		}
	}
}

// TestAPIEventsLimit proves the page ceiling truncates AND says so — a page that
// stops short must not look like the end of history — and that a limit the
// caller got wrong is refused rather than silently clamped.
func TestAPIEventsLimit(t *testing.T) {
	w := dashboardEnv(t)
	for i := 0; i < 4; i++ {
		if _, err := eventlog.Append(w, "a-child1", model.EventComment, "core", "", "note"); err != nil {
			t.Fatalf("event: %v", err)
		}
	}
	h := newHandler(w)

	var page eventsResponse
	getJSON(t, h, "/api/events?limit=2", &page)
	if len(page.Events) != 2 || page.Limit != 2 || !page.Truncated {
		t.Errorf("limit=2 page = %d events, limit %d, truncated %v", len(page.Events), page.Limit, page.Truncated)
	}

	// A page that exactly exhausts the log is NOT truncated.
	var whole eventsResponse
	getJSON(t, h, "/api/events?limit=5", &whole)
	if len(whole.Events) != 5 || whole.Truncated {
		t.Errorf("limit=5 page = %d events, truncated %v, want 5 and false", len(whole.Events), whole.Truncated)
	}

	for _, bad := range []string{"0", "-1", "abc", "9999"} {
		if code := status(t, h, "/api/events?limit="+bad); code != http.StatusBadRequest {
			t.Errorf("GET /api/events?limit=%s = %d, want 400", bad, code)
		}
	}
	for _, bad := range []string{"../../etc", "a/b", ".."} {
		if code := status(t, h, "/api/events?task="+bad); code != http.StatusBadRequest {
			t.Errorf("GET /api/events?task=%s = %d, want 400", bad, code)
		}
	}
	if code := status(t, h, "/api/events?task=t-01NOSUCHTASK00000000000AA"); code != http.StatusNotFound {
		t.Errorf("unknown task filter = %d, want 404", code)
	}
}

// TestAPIAgentDetail is the agent-detail contract (dacli 227): identity, lineage,
// the tasks it owns, and the runs attributable to it — dead ones included, since
// a finished run is the evidence of what the agent actually did.
func TestAPIAgentDetail(t *testing.T) {
	w := dashboardEnv(t)
	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}
	worker, _, err := agentid.Spawn(w, root, "maintainer", model.GrantRW)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	child, _, err := agentid.Spawn(w, &agentid.Identity{ID: worker, Grant: model.GrantRW}, "builder", model.GrantRO)
	if err != nil {
		t.Fatalf("spawn child: %v", err)
	}
	// A task this agent owns (CreateTask stamps the actor as owner).
	owned, err := store.CreateTask(w, worker, "core", "Owned by the worker", store.TaskOpts{Estimate: "1,2,3"})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	// A finished run led by this agent: PID 0 can never be alive, so it exercises
	// the dead-but-recorded path.
	deadRun := "01RUNIDTESTWORKERDEAD00000"
	if err := os.MkdirAll(w.RunDir(deadRun), 0o755); err != nil {
		t.Fatalf("rundir: %v", err)
	}
	rec := procmon.Record{
		RunID: deadRun, Child: worker, Task: owned.ID, Role: "maintainer",
		Runtime: "claude", PID: 0, PGID: 0, Started: time.Now().Add(-time.Hour),
	}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(deadRun), "proc.txt"), rec); err != nil {
		t.Fatalf("proc.txt: %v", err)
	}

	h := newHandler(w)
	var resp agentDetailResponse
	getJSON(t, h, "/api/agent?id="+worker, &resp)

	a := resp.Agent
	if a.ID != worker || a.Role != "maintainer" || a.Grant != "rw" || a.Retired {
		t.Errorf("agent identity = %+v", a)
	}
	if a.Parent != agentid.RootID {
		t.Errorf("parent = %q, want %q (brackets stripped)", a.Parent, agentid.RootID)
	}
	if len(a.Children) != 1 || a.Children[0] != child {
		t.Errorf("children = %v, want [%s]", a.Children, child)
	}
	if len(a.Tasks) != 1 || a.Tasks[0].ID != owned.ID {
		t.Errorf("tasks = %+v, want the one owned task", a.Tasks)
	}
	if len(a.Runs) != 1 {
		t.Fatalf("runs = %d, want 1 (the finished run)", len(a.Runs))
	}
	r := a.Runs[0]
	if r.RunID != deadRun || r.Task != owned.ID || r.Role != "maintainer" {
		t.Errorf("run = %+v", r)
	}
	if r.Live {
		t.Errorf("run live = true for PID 0")
	}
	if r.TranscriptURL == "" || r.DiffURL == "" {
		t.Errorf("run detail links missing: %+v", r)
	}

	// The env's live run belongs to a-child1, not this agent — attribution must
	// not leak another agent's work in.
	for _, run := range a.Runs {
		if run.RunID == "01RUNIDTESTLIVEAGENT00000" {
			t.Errorf("another agent's run leaked into %s's history", worker)
		}
	}
}

// TestAPIAgentNeverServesTokenHash is the secrets guard: an agent file carries a
// sha256 token hash, and no byte of it may reach the wire. Asserted against the
// RAW response rather than the decoded struct, so a future field that happens to
// carry the hash cannot pass unnoticed.
func TestAPIAgentNeverServesTokenHash(t *testing.T) {
	w := dashboardEnv(t)
	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}
	id, token, err := agentid.Spawn(w, root, "builder", model.GrantRW)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	// Prove the hash really is on disk, so a passing test cannot be vacuous.
	raw, err := os.ReadFile(w.AgentPath(id))
	if err != nil {
		t.Fatalf("read agent file: %v", err)
	}
	if !strings.Contains(string(raw), "token_hash:") {
		t.Fatalf("fixture has no token_hash — the guard would be vacuous")
	}

	h := newHandler(w)
	for _, target := range []string{"/api/agent?id=" + id, "/api/roles", "/api/state"} {
		body := bodyOf(t, h, target)
		for _, secret := range []string{"token_hash", "sha256:", token} {
			if strings.Contains(body, secret) {
				t.Errorf("GET %s leaked agent credentials (%q):\n%s", target, secret, body)
			}
		}
	}
}

// TestAPIAgentIDContract: an unusable id is 400 before the store is touched, an
// absent one is 404.
func TestAPIAgentIDContract(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	for _, bad := range []string{"", "../../etc/passwd", "a/b", "..", "/etc/passwd"} {
		if code := status(t, h, "/api/agent?id="+bad); code != http.StatusBadRequest {
			t.Errorf("GET /api/agent?id=%q = %d, want 400", bad, code)
		}
	}
	if code := status(t, h, "/api/agent?id=a-nobody"); code != http.StatusNotFound {
		t.Errorf("unknown agent = %d, want 404", code)
	}
}

// TestDetailEndpointsOnEmptyWorkspace proves the detail surfaces are zero-safe:
// a fresh workspace answers the unfiltered log with an empty list and reports a
// missing task/agent as 404, never a 500.
func TestDetailEndpointsOnEmptyWorkspace(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	h := newHandler(w)

	var resp eventsResponse
	getJSON(t, h, "/api/events", &resp)
	if len(resp.Events) != 0 {
		t.Errorf("events = %d on an empty workspace, want 0", len(resp.Events))
	}
	if code := status(t, h, "/api/task?ref=001"); code != http.StatusNotFound {
		t.Errorf("task on empty workspace = %d, want 404", code)
	}
	if code := status(t, h, "/api/agent?id=a-nobody"); code != http.StatusNotFound {
		t.Errorf("unknown agent on empty workspace = %d, want 404", code)
	}
	// `dacli init` mints the root agent, which therefore HAS a detail page — one
	// with nothing in it, which must still be empty lists rather than nulls.
	var root agentDetailResponse
	getJSON(t, h, "/api/agent?id="+agentid.RootID, &root)
	if root.Agent.ID != agentid.RootID {
		t.Errorf("root agent = %+v", root.Agent)
	}
	if root.Agent.Children == nil || root.Agent.Tasks == nil || root.Agent.Runs == nil {
		t.Errorf("root agent emitted a null list: %+v", root.Agent)
	}
}

// TestEventTimeRejectsNonULID pins the timestamp decoder's honest fallback: an
// id that is not a ULID yields "" so a client renders "unknown" rather than the
// epoch.
func TestEventTimeRejectsNonULID(t *testing.T) {
	for _, bad := range []string{"", "short", "!!!!!!!!!!"} {
		if got := eventTime(bad); got != "" {
			t.Errorf("eventTime(%q) = %q, want empty", bad, got)
		}
	}
	at := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	if got := eventTime(ulid.At(at)); got != at.Format(time.RFC3339) {
		t.Errorf("eventTime round trip = %q, want %q", got, at.Format(time.RFC3339))
	}
}
