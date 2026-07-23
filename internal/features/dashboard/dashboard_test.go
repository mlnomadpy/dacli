package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// dashboardEnv builds a workspace with one project holding a done and an
// open estimated task, plus one live agent record — a run whose leader
// process is this very test binary, so procmon.AliveRecord finds it alive
// without needing to spawn a real child.
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
		RunID: runID, Child: "a-child1", Task: "core/001", Role: "builder",
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
	if !strings.Contains(body, "<title>dacli dashboard</title>") {
		t.Errorf("index page missing title, got:\n%s", body)
	}
	if !strings.Contains(body, "/api/state") {
		t.Errorf("index page does not poll /api/state:\n%s", body)
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
