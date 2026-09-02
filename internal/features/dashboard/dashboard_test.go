package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestBurndownDownsamplingIsBoundedAndPreservesExtrema(t *testing.T) {
	points := make([]burndownDay, 180)
	for i := range points {
		points[i] = burndownDay{Day: fmt.Sprintf("day-%03d", i), Points: float64(10 + i%5), TaskIDs: []string{fmt.Sprintf("task-%03d", i)}}
	}
	points[31].Points = 1
	points[151].Points = 999
	got, hidden := downsampleBurndown(points, 90)
	if len(got) != 90 || hidden != 90 {
		t.Fatalf("points=%d hidden=%d", len(got), hidden)
	}
	want := map[string]bool{"day-000": false, "day-031": false, "day-151": false, "day-179": false}
	for _, point := range got {
		if _, ok := want[point.Day]; ok {
			want[point.Day] = true
		}
	}
	for day, found := range want {
		if !found {
			t.Fatalf("required edge/extreme %s hidden", day)
		}
	}
}

// seedRoster adds the team half of the fixture (dacli 226): a capped role at
// its WIP limit and an uncapped one it escalates to, plus two agents in the
// capped role — one live, one retired — so the roster's active count can prove
// it applies store.ActiveInRole's rule (retired frees the slot) rather than
// counting agent files.
func seedRoster(t *testing.T, w *workspace.Workspace) string {
	t.Helper()
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "builder", Summary: "writes the code", Kind: "implementer",
		Grant: "rw", Runtime: "claude", Model: "sonnet", WIP: 1, MaxPoints: 5,
		Skills: []string{"go"}, Scope: []string{"internal/**"},
		OutOfScope: []string{"internal/agentid/**"},
		Shortcuts:  []string{"test"}, EscalateTo: []string{"maintainer"},
	}); err != nil {
		t.Fatalf("role builder: %v", err)
	}
	if err := store.CreateRole(w, "a-root", team.Role{
		Name: "maintainer", Summary: "owns the whole tree",
	}); err != nil {
		t.Fatalf("role maintainer: %v", err)
	}

	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}
	live := "a-child1"
	d := &mdstore.Doc{}
	d.Front.Set("id", live)
	d.Front.Set("kind", "agent")
	d.Front.Set("role", "builder")
	d.Front.Set("grant", "rw")
	if err := mdstore.WriteFile(w.AgentPath(live), d); err != nil {
		t.Fatalf("write live builder identity: %v", err)
	}
	retired, _, err := agentid.Spawn(w, root, "builder", model.GrantRO)
	if err != nil {
		t.Fatalf("spawn second builder: %v", err)
	}
	if err := store.RetireAgent(w, retired); err != nil {
		t.Fatalf("retire: %v", err)
	}
	return live
}

// dashboardEnv builds a workspace with one project holding a done and an
// open estimated task, the two-role roster seedRoster describes, plus one live
// agent record — a run whose leader process is this very test binary, so
// procmon.AliveRecord finds it alive without needing to spawn a real child.
func dashboardEnv(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatalf("project: %v", err)
	}
	done, err := store.CreateTask(w, "a-root", "core", "Ship the thing", store.TaskOpts{
		Accept: []string{"it ships"}, Estimate: "1,2,3",
	})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if err := store.CloseTask(w, done, "a-root"); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := store.CreateTask(w, "a-root", "core", "Still open work", store.TaskOpts{
		Accept: []string{"x"}, Estimate: "1,2,3",
	}); err != nil {
		t.Fatalf("task: %v", err)
	}
	if _, err := eventlog.Append(w, "a-child1", model.EventComment, "core", "", "a note from a child"); err != nil {
		t.Fatalf("event: %v", err)
	}
	liveChild := seedRoster(t, w)

	pid := os.Getpid()
	start, _ := procmon.ProcStart(pid)
	runID := "01RUNIDTESTLIVEAGENT00000"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("rundir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "transcript.log"), []byte("thinking...\n"), 0o644); err != nil {
		t.Fatalf("transcript: %v", err)
	}
	rec := procmon.Record{
		RunID: runID, Child: liveChild, Task: "core/001", Role: "builder",
		Runtime: "claude", PID: pid, PGID: pid, PIDStart: start, Started: time.Now().Add(-90 * time.Second),
	}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatalf("proc.txt: %v", err)
	}
	return w
}

func TestIndexServesEmbeddedPage(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	req := httptest.NewRequest("GET", "/", nil)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	if rw.Code != 200 {
		t.Fatalf("GET / = %d", rw.Code)
	}
	body := rw.Body.String()
	// "/" serves whichever index page is embedded: the built SPA bundle when a
	// frontend build produced ui/dist/index.html, else the legacy dashboard.
	// Assert the served bytes ARE that resolved page rather than pinning to one,
	// so the test passes whether or not `npm run build` ran before `go test`.
	if body != string(indexPage()) {
		t.Errorf("GET / did not serve the resolved index page")
	}
	if !strings.Contains(body, "<title>dacli dashboard</title>") {
		t.Errorf("index page missing title, got:\n%s", body)
	}
	if ct := rw.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type = %q, want text/html", ct)
	}
}

func TestUnknownPathIs404(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	req := httptest.NewRequest("GET", "/nope", nil)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	if rw.Code != 404 {
		t.Fatalf("GET /nope = %d, want 404", rw.Code)
	}
}

func TestAPIStateReportsProjectsAndLiveAgent(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	req := httptest.NewRequest("GET", "/api/state", nil)
	req.Host = "localhost"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	if rw.Code != 200 {
		t.Fatalf("GET /api/state = %d: %s", rw.Code, rw.Body.String())
	}
	var state dashboardState
	if err := json.Unmarshal(rw.Body.Bytes(), &state); err != nil {
		t.Fatalf("bad json: %v\n%s", err, rw.Body.String())
	}

	if len(state.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(state.Projects))
	}
	p := state.Projects[0]
	if p.Slug != "core" || p.Total != 2 {
		t.Errorf("project view = %+v", p)
	}
	if p.Counts["done"] != 1 || p.Counts["open"] != 1 {
		t.Errorf("counts = %+v, want done:1 open:1", p.Counts)
	}
	if p.Burndown.DonePoints <= 0 {
		t.Errorf("burndown done points = %v, want > 0", p.Burndown.DonePoints)
	}
	if p.Burndown.RemainingPoints <= 0 {
		t.Errorf("burndown remaining points = %v, want > 0", p.Burndown.RemainingPoints)
	}
	if len(p.Burndown.PerDay) == 0 {
		t.Errorf("burndown per-day is empty, want the done task's completion day")
	}
	if len(p.Graph.Nodes) != 2 {
		t.Errorf("legacy /api/state graph nodes = %d, want 2", len(p.Graph.Nodes))
	}

	if len(state.Agents) != 1 {
		t.Fatalf("agents = %d, want 1 (the live one)", len(state.Agents))
	}
	a := state.Agents[0]
	if a.Child != "a-child1" || a.Role != "builder" || a.Runtime != "claude" || a.PID != os.Getpid() {
		t.Errorf("agent view = %+v", a)
	}
	if a.RuntimeSecs < 80 {
		t.Errorf("runtime_secs = %d, want >= ~90", a.RuntimeSecs)
	}
	if a.LastActivity == "" {
		t.Errorf("last_activity is empty")
	}
	if state.PendingEvents != 1 {
		t.Errorf("pending_events = %d, want 1 (the child's unsynced comment)", state.PendingEvents)
	}
	if ct := rw.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}
}

// getJSON drives the handler for one path and decodes the body into v, asserting
// a 200 and a JSON content type — the shared preamble for the typed-endpoint tests.
func getJSON(t *testing.T, h http.Handler, path string, v any) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Host = "localhost"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != 200 {
		t.Fatalf("GET %s = %d: %s", path, rw.Code, rw.Body.String())
	}
	if ct := rw.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("GET %s content-type = %q, want application/json", path, ct)
	}
	if err := json.Unmarshal(rw.Body.Bytes(), v); err != nil {
		t.Fatalf("GET %s bad json: %v\n%s", path, err, rw.Body.String())
	}
}

func TestAPIOverview(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	var resp overviewResponse
	getJSON(t, h, "/api/overview", &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	if resp.ProjectCount != 1 {
		t.Errorf("project_count = %d, want 1", resp.ProjectCount)
	}
	if resp.TaskCount != 2 {
		t.Errorf("task_count = %d, want 2", resp.TaskCount)
	}
	if resp.Counts["done"] != 1 || resp.Counts["open"] != 1 {
		t.Errorf("counts = %+v, want done:1 open:1", resp.Counts)
	}
	if resp.PendingEvents != 1 {
		t.Errorf("pending_events = %d, want 1", resp.PendingEvents)
	}
	if resp.LiveAgents != 1 {
		t.Errorf("live_agents = %d, want 1", resp.LiveAgents)
	}
}

func TestAPIProjects(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	var resp projectsResponse
	getJSON(t, h, "/api/projects", &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	if len(resp.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(resp.Projects))
	}
	p := resp.Projects[0]
	if p.Slug != "core" || p.Total != 2 {
		t.Errorf("project view = %+v", p)
	}
	if p.Counts["done"] != 1 || p.Counts["open"] != 1 {
		t.Errorf("counts = %+v, want done:1 open:1", p.Counts)
	}
	if p.Burndown.DonePoints <= 0 || p.Burndown.RemainingPoints <= 0 {
		t.Errorf("burndown = %+v, want positive done/remaining points", p.Burndown)
	}
	if len(p.Burndown.PerDay) == 0 {
		t.Errorf("burndown per-day is empty, want the done task's completion day")
	}

	// The Vue SPA loads only the selected graph from /api/graph. Assert on the
	// wire shape, not the Go response type, so accidentally adding `graph` back
	// to project summaries makes this performance regression red (issue #932).
	var raw struct {
		Projects []map[string]any `json:"projects"`
	}
	getJSON(t, h, "/api/projects", &raw)
	if _, ok := raw.Projects[0]["graph"]; ok {
		t.Errorf("/api/projects unexpectedly embeds graph: %+v", raw.Projects[0]["graph"])
	}
}

func TestAPITasks(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	var resp tasksResponse
	getJSON(t, h, "/api/tasks", &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	if len(resp.Tasks) != 2 {
		t.Fatalf("tasks = %d, want 2 (one done, one open)", len(resp.Tasks))
	}
	byStatus := map[string]taskView{}
	for _, tk := range resp.Tasks {
		byStatus[tk.Status] = tk
		if tk.Project != "core" {
			t.Errorf("task %s project = %q, want core", tk.ID, tk.Project)
		}
		if tk.ID == "" || tk.Seq == 0 || tk.Title == "" {
			t.Errorf("task row missing identity fields: %+v", tk)
		}
		// Both env tasks carry a 1,2,3 estimate, so points must be positive.
		if !tk.Estimated || tk.Points <= 0 {
			t.Errorf("task %s estimated=%v points=%v, want estimated with positive points", tk.ID, tk.Estimated, tk.Points)
		}
	}
	if _, ok := byStatus["done"]; !ok {
		t.Errorf("no done task in %+v", resp.Tasks)
	}
	if _, ok := byStatus["open"]; !ok {
		t.Errorf("no open task in %+v", resp.Tasks)
	}

	// The ?project= filter is honored: an unknown slug yields no rows.
	var empty tasksResponse
	getJSON(t, h, "/api/tasks?project=nope", &empty)
	if len(empty.Tasks) != 0 {
		t.Errorf("tasks for unknown project = %d, want 0", len(empty.Tasks))
	}
}

func TestAPIAgents(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	var resp agentsResponse
	getJSON(t, h, "/api/agents", &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	if len(resp.Agents) != 1 {
		t.Fatalf("agents = %d, want 1 (the live one)", len(resp.Agents))
	}
	a := resp.Agents[0]
	if a.Child != "a-child1" || a.Role != "builder" || a.Runtime != "claude" || a.PID != os.Getpid() {
		t.Errorf("agent view = %+v", a)
	}
	if a.RuntimeSecs < 80 {
		t.Errorf("runtime_secs = %d, want >= ~90", a.RuntimeSecs)
	}
	if a.LastActivity == "" {
		t.Errorf("last_activity is empty")
	}
	// The freshly-written "thinking...\n" transcript is plain prose with a current
	// mtime → the honest state is "thinking", and both detail links are exposed
	// and carry this run's id (they must resolve to the same run).
	if a.State != "thinking" {
		t.Errorf("state = %q, want thinking (fresh prose transcript)", a.State)
	}
	if !strings.Contains(a.TranscriptURL, a.RunID) {
		t.Errorf("transcript_url = %q, want a link carrying run id %s", a.TranscriptURL, a.RunID)
	}
	if !strings.Contains(a.DiffURL, a.RunID) {
		t.Errorf("diff_url = %q, want a link carrying run id %s", a.DiffURL, a.RunID)
	}
}

// setTranscript overwrites the live run's transcript.log and back-dates its
// mtime by age (0 = leave it current), so a test can drive deriveAgentState
// across the fresh/stalled boundary deterministically.
func setTranscript(t *testing.T, w *workspace.Workspace, body string, age time.Duration) {
	t.Helper()
	path := filepath.Join(w.RunDir("01RUNIDTESTLIVEAGENT00000"), "transcript.log")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("transcript: %v", err)
	}
	if age > 0 {
		mt := time.Now().Add(-age)
		if err := os.Chtimes(path, mt, mt); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
}

// firstAgent drives /api/agents and returns the single live agent.
func firstAgent(t *testing.T, h http.Handler) agentView {
	t.Helper()
	var resp agentsResponse
	getJSON(t, h, "/api/agents", &resp)
	if len(resp.Agents) != 1 {
		t.Fatalf("agents = %d, want 1", len(resp.Agents))
	}
	return resp.Agents[0]
}

func TestAgentStateDerivation(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	// A [tool: X] marker as the last rendered line → acting.
	setTranscript(t, w, "Looking at the file.\n[tool: Read]\n", 0)
	if got := firstAgent(t, h).State; got != "acting" {
		t.Errorf("tool-marker transcript: state = %q, want acting", got)
	}

	// A raw stream-json tool_use event decodes to a [tool: X] marker → acting,
	// proving the dashboard reads a detached run's transcript, not just teed text.
	setTranscript(t, w, `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash"}]}}`+"\n", 0)
	if got := firstAgent(t, h).State; got != "acting" {
		t.Errorf("stream-json tool_use: state = %q, want acting", got)
	}

	// Assistant prose that has not moved for longer than the stall window → stalled.
	setTranscript(t, w, "Still reasoning about the approach.\n", 5*time.Minute)
	if got := firstAgent(t, h).State; got != "stalled" {
		t.Errorf("frozen prose transcript: state = %q, want stalled", got)
	}

	// An empty transcript on a stream runtime that has been quiet past the window
	// → stalled (it produced nothing and has gone silent).
	setTranscript(t, w, "", 5*time.Minute)
	if got := firstAgent(t, h).State; got != "stalled" {
		t.Errorf("empty frozen stream transcript: state = %q, want stalled", got)
	}

	// An empty transcript that is still fresh → waiting (just spawned).
	setTranscript(t, w, "", 0)
	if got := firstAgent(t, h).State; got != "waiting" {
		t.Errorf("empty fresh transcript: state = %q, want waiting", got)
	}
}

// setAgentTask rewrites the live fixture agent's proc.txt to name a
// different task ref, so a test can drive agentstate.Derive's blocked check
// (which reads the record's Task) deterministically.
func setAgentTask(t *testing.T, w *workspace.Workspace, ref string) {
	t.Helper()
	path := filepath.Join(w.RunDir("01RUNIDTESTLIVEAGENT00000"), "proc.txt")
	rec, err := procmon.ReadRecord(path)
	if err != nil {
		t.Fatalf("proc.txt: %v", err)
	}
	rec.Task = ref
	if err := procmon.WriteRecord(path, rec); err != nil {
		t.Fatalf("proc.txt: %v", err)
	}
}

// A live agent whose task is blocked on an outstanding `dacli ask` must
// report "blocked" — overriding whatever the transcript alone would say —
// the state `dacli agents` shows through the same agentstate.Derive call.
func TestAgentStateDerivationBlockedOnOutstandingAsk(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	task, err := store.CreateTask(w, "a-root", "core", "Needs an answer", store.TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if err := store.MoveTask(w, task, model.StatusBlocked); err != nil {
		t.Fatalf("move: %v", err)
	}
	setAgentTask(t, w, task.Slug)

	// Even a fresh, mid-tool-call transcript is reported blocked.
	setTranscript(t, w, "Looking at the file.\n[tool: Read]\n", 0)
	if got := firstAgent(t, h).State; got != "blocked" {
		t.Errorf("state = %q, want blocked (task has an outstanding ask)", got)
	}
}

func TestAgentTranscriptEndpoint(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)
	// A stream-json assistant event must be decoded to readable text on read.
	setTranscript(t, w, `{"type":"assistant","message":{"content":[{"type":"text","text":"hello from the agent"},{"type":"tool_use","name":"Read"}]}}`+"\n", 0)

	req := httptest.NewRequest("GET", "/api/agents/transcript?run=01RUNIDTESTLIVEAGENT00000", nil)
	req.Host = "localhost"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != 200 {
		t.Fatalf("GET transcript = %d: %s", rw.Code, rw.Body.String())
	}
	if ct := rw.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
	body := rw.Body.String()
	if !strings.Contains(body, "hello from the agent") || !strings.Contains(body, "[tool: Read]") {
		t.Errorf("transcript body did not render the stream event:\n%s", body)
	}

	// An unknown run is a 404, not an empty 200 (a dead link must read as dead).
	req = httptest.NewRequest("GET", "/api/agents/transcript?run=01NOSUCHRUN0000000000000", nil)
	req.Host = "localhost"
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != 404 {
		t.Errorf("unknown run transcript = %d, want 404", rw.Code)
	}

	// A path-traversal id is rejected before it can escape the runs dir.
	req = httptest.NewRequest("GET", "/api/agents/transcript?run=../../etc", nil)
	req.Host = "localhost"
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != 404 {
		t.Errorf("traversal run transcript = %d, want 404", rw.Code)
	}
}

func TestAgentDiffEndpoint(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	// The env is not a git repo, so `git diff` cannot run — the endpoint must
	// still answer 200 text/plain with an honest note, never a fake diff or a 500.
	req := httptest.NewRequest("GET", "/api/agents/diff?run=01RUNIDTESTLIVEAGENT00000", nil)
	req.Host = "localhost"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != 200 {
		t.Fatalf("GET diff = %d: %s", rw.Code, rw.Body.String())
	}
	if ct := rw.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
	if strings.TrimSpace(rw.Body.String()) == "" {
		t.Errorf("diff body is empty, want an honest note")
	}

	// An unknown run is a 404.
	req = httptest.NewRequest("GET", "/api/agents/diff?run=01NOSUCHRUN0000000000000", nil)
	req.Host = "localhost"
	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != 404 {
		t.Errorf("unknown run diff = %d, want 404", rw.Code)
	}
}

func TestAPIStateOmitsDeadAgent(t *testing.T) {
	w := dashboardEnv(t)

	// A stale run dir whose PID cannot possibly be alive (0 is never a real
	// process) must not appear as a live agent.
	runID := "01RUNIDTESTDEADAGENT000000"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("rundir: %v", err)
	}
	rec := procmon.Record{RunID: runID, Child: "a-dead", PID: 0, PGID: 0, Started: time.Now()}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatalf("proc.txt: %v", err)
	}

	h := newHandler(w)
	req := httptest.NewRequest("GET", "/api/state", nil)
	req.Host = "localhost"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	var state dashboardState
	if err := json.Unmarshal(rw.Body.Bytes(), &state); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	for _, a := range state.Agents {
		if a.Child == "a-dead" {
			t.Errorf("dead agent leaked into live state: %+v", a)
		}
	}
	if len(state.Agents) != 1 {
		t.Errorf("agents = %d, want 1 (only the live one)", len(state.Agents))
	}
}
