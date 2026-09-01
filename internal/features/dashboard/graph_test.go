package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// graphEnv builds a workspace with one project holding a linear dependency
// chain A→B→C (each estimated, so the whole chain is the critical path) plus a
// standalone done task — enough to exercise the DAG (four nodes, two edges),
// the CPM overlay (all three open tasks zero-slack critical), and status-aware
// node rendering (the done task is a node but not on the path).
func graphEnv(t *testing.T) (*workspace.Workspace, [4]string) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatalf("project: %v", err)
	}
	// A (seq 1), B (seq 2, depends on 001), C (seq 3, depends on 002).
	a, err := store.CreateTask(w, "a-root", "core", "Design the thing", store.TaskOpts{
		Accept: []string{"x"}, Estimate: "1,2,3",
	})
	if err != nil {
		t.Fatalf("task A: %v", err)
	}
	b, err := store.CreateTask(w, "a-root", "core", "Build the thing", store.TaskOpts{
		Accept: []string{"x"}, Estimate: "1,2,3", DependsOn: []string{"001"},
	})
	if err != nil {
		t.Fatalf("task B: %v", err)
	}
	c, err := store.CreateTask(w, "a-root", "core", "Ship the thing", store.TaskOpts{
		Accept: []string{"x"}, Estimate: "1,2,3", DependsOn: []string{"002"},
	})
	if err != nil {
		t.Fatalf("task C: %v", err)
	}
	// A standalone done task — a node with status "done", never on the path.
	d, err := store.CreateTask(w, "a-root", "core", "Old finished work", store.TaskOpts{
		Accept: []string{"x"}, Estimate: "1,2,3",
	})
	if err != nil {
		t.Fatalf("task D: %v", err)
	}
	if err := store.CloseTask(w, d, "a-root"); err != nil {
		t.Fatalf("close D: %v", err)
	}
	return w, [4]string{a.ID, b.ID, c.ID, d.ID}
}

func TestAPIGraph(t *testing.T) {
	w, ids := graphEnv(t)
	a, b, c, d := ids[0], ids[1], ids[2], ids[3]
	h := newHandler(w)

	var resp graphResponse
	getJSON(t, h, "/api/graph?project=core", &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	if resp.Project != "core" {
		t.Errorf("project = %q, want core", resp.Project)
	}

	// The operational projection excludes unrelated completed history.
	if len(resp.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3 operational tasks\n%+v", len(resp.Nodes), resp.Nodes)
	}
	byID := map[string]graphNode{}
	for _, n := range resp.Nodes {
		byID[n.ID] = n
	}
	if _, ok := byID[d]; ok {
		t.Errorf("unrelated done task %s leaked into operational graph", d)
	}
	if resp.Projection.VisibleNodes != 3 || resp.Projection.HiddenNodes != 1 || resp.Projection.TotalNodes != 4 {
		t.Errorf("projection counts = %+v, want visible=3 hidden=1 total=4", resp.Projection)
	}
	if !byID[a].Estimated || byID[a].Points <= 0 {
		t.Errorf("task A estimated=%v points=%v, want estimated with positive points", byID[a].Estimated, byID[a].Points)
	}

	// Two edges: A→B and B→C, both FS (default).
	if len(resp.Edges) != 2 {
		t.Fatalf("edges = %d, want 2\n%+v", len(resp.Edges), resp.Edges)
	}
	edge := map[string]string{} // to -> from
	for _, e := range resp.Edges {
		edge[e.To] = e.From
		if e.Type != "FS" {
			t.Errorf("edge %s→%s type = %q, want FS", e.From, e.To, e.Type)
		}
	}
	if edge[b] != a {
		t.Errorf("expected edge A→B (%s→%s), got from %q", a, b, edge[b])
	}
	if edge[c] != b {
		t.Errorf("expected edge B→C (%s→%s), got from %q", b, c, edge[c])
	}

	// CPM overlay: the linear chain is entirely zero-slack, so all three open
	// tasks are critical and the path is A,B,C in topological order.
	if !resp.Scheduled {
		t.Fatalf("scheduled = false, want true; note=%q", resp.Note)
	}
	if resp.Duration <= 0 {
		t.Errorf("duration = %v, want > 0", resp.Duration)
	}
	if len(resp.CriticalPath) != 3 {
		t.Fatalf("critical_path = %d, want 3: %+v", len(resp.CriticalPath), resp.CriticalPath)
	}
	if resp.CriticalPath[0] != a || resp.CriticalPath[1] != b || resp.CriticalPath[2] != c {
		t.Errorf("critical_path = %v, want [%s %s %s] (topological)", resp.CriticalPath, a, b, c)
	}
	for _, id := range []string{a, b, c} {
		if !byID[id].Critical {
			t.Errorf("open chain task %s should be critical: %+v", id, byID[id])
		}
		if byID[id].Slack != 0 {
			t.Errorf("open chain task %s slack = %v, want 0", id, byID[id].Slack)
		}
	}
}

func TestAPIGraphFocusAndHistoryAreBoundedAndExact(t *testing.T) {
	w, ids := graphEnv(t)
	h := newHandler(w)
	var focused graphResponse
	getJSON(t, h, "/api/graph?project=core&focus="+ids[1], &focused)
	if focused.Projection.Mode != "focus" || focused.Projection.Focus != ids[1] {
		t.Fatalf("focus projection = %+v", focused.Projection)
	}
	if len(focused.Nodes) != 3 { // A <- B -> C within two hops; isolated D omitted.
		t.Fatalf("focused nodes = %d, want 3", len(focused.Nodes))
	}
	var history graphResponse
	getJSON(t, h, "/api/graph?project=core&mode=history", &history)
	if history.Projection.Mode != "history" || len(history.Nodes) != 1 || history.Nodes[0].ID != ids[3] {
		t.Fatalf("history projection = %+v nodes=%+v", history.Projection, history.Nodes)
	}
	if history.Projection.HasMore {
		t.Fatalf("single-page history unexpectedly reports more: %+v", history.Projection)
	}
}

func TestAPIGraphRejectsInvalidOrMissingFocus(t *testing.T) {
	w, _ := graphEnv(t)
	h := newHandler(w)
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/api/graph?project=core&mode=everything", 400},
		{"/api/graph?project=core&status=done", 400},
		{"/api/graph?project=core&page=0", 400},
		{"/api/graph?project=core&focus=missing", 404},
	} {
		req := httptest.NewRequest("GET", tc.path, nil)
		req.Host = "127.0.0.1"
		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, req)
		if rw.Code != tc.want {
			t.Errorf("GET %s = %d (%s), want %d", tc.path, rw.Code, rw.Body.String(), tc.want)
		}
	}
}

func TestLargeOperationalGraphIsBoundedWithTruthfulCost(t *testing.T) {
	if operationalGraphLimit != 120 {
		t.Fatalf("operational graph contract limit = %d, want 120", operationalGraphLimit)
	}
	full := graphView{Project: "core", Nodes: []graphNode{}, Edges: []graphEdge{}, CriticalPath: []string{}}
	for i := 1; i <= 553; i++ {
		status := "done"
		if i > 500 {
			status = "open"
		}
		id := fmt.Sprintf("t-%03d", i)
		full.Nodes = append(full.Nodes, graphNode{ID: id, Seq: i, Slug: id, Title: "Representative task", Status: status, Slack: -1})
		if i > 1 {
			full.Edges = append(full.Edges, graphEdge{From: fmt.Sprintf("t-%03d", i-1), To: id, Type: "FS"})
		}
	}
	started := time.Now()
	bounded := projectGraph(full, graphOptions{})
	encoded, err := json.Marshal(bounded)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(bounded.Nodes); got > operationalGraphLimit {
		t.Fatalf("default rendered nodes = %d, want <= %d", got, operationalGraphLimit)
	}
	if bounded.Projection.TotalNodes != 553 || bounded.Projection.HiddenNodes != 553-len(bounded.Nodes) {
		t.Fatalf("large projection counts = %+v", bounded.Projection)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("projection took %s, want <=100ms", elapsed)
	}
	if len(encoded) > 128*1024 {
		t.Fatalf("bounded response = %d bytes, want <= 128 KiB", len(encoded))
	}
}

// The graph embedded per-project in /api/state must match the standalone
// /api/graph surface — the two contracts can never drift.
func TestAPIStateEmbedsGraph(t *testing.T) {
	w, _ := graphEnv(t)
	h := newHandler(w)

	var state dashboardState
	getJSON(t, h, "/api/state", &state)
	if len(state.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(state.Projects))
	}
	g := state.Projects[0].Graph
	if len(g.Nodes) != 4 || len(g.Edges) != 2 {
		t.Errorf("embedded graph nodes/edges = %d/%d, want 4/2", len(g.Nodes), len(g.Edges))
	}
	if !g.Scheduled || len(g.CriticalPath) != 3 {
		t.Errorf("embedded graph scheduled=%v critical_path=%d, want true/3", g.Scheduled, len(g.CriticalPath))
	}
}

// An open task with no PERT estimate makes the open subset unschedulable: the
// DAG must still render every node and edge, but with no critical path and an
// honest note rather than a refusal.
func TestAPIGraphDegradesWhenUnestimated(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", "build"); err != nil {
		t.Fatalf("project: %v", err)
	}
	if _, err := store.CreateTask(w, "a-root", "core", "Estimated", store.TaskOpts{
		Accept: []string{"x"}, Estimate: "1,2,3",
	}); err != nil {
		t.Fatalf("task: %v", err)
	}
	// A second open task with NO estimate → the schedule cannot be computed.
	if _, err := store.CreateTask(w, "a-root", "core", "Unestimated", store.TaskOpts{
		Accept: []string{"x"}, DependsOn: []string{"001"},
	}); err != nil {
		t.Fatalf("task: %v", err)
	}
	h := newHandler(w)

	var resp graphResponse
	getJSON(t, h, "/api/graph?project=core", &resp)

	if len(resp.Nodes) != 2 || len(resp.Edges) != 1 {
		t.Errorf("nodes/edges = %d/%d, want 2/1 (DAG still drawn)", len(resp.Nodes), len(resp.Edges))
	}
	if resp.Scheduled {
		t.Errorf("scheduled = true, want false (an open task has no estimate)")
	}
	if resp.Note == "" {
		t.Errorf("note is empty, want an explanation of why the path is absent")
	}
	if len(resp.CriticalPath) != 0 {
		t.Errorf("critical_path = %v, want empty when unscheduled", resp.CriticalPath)
	}
}

// An empty workspace degrades to an empty-but-non-null graph: no nodes, no
// edges, not scheduled — never a null slice or a fabricated path.
func TestGraphEmptyWorkspaceIsZeroSafe(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	g, err := buildGraph(w, "")
	if err != nil {
		t.Fatalf("buildGraph: %v", err)
	}
	if g.Nodes == nil || g.Edges == nil || g.CriticalPath == nil {
		t.Errorf("empty graph has a nil slice (must marshal as [], not null): %+v", g)
	}
	if len(g.Nodes) != 0 || g.Scheduled {
		t.Errorf("empty workspace graph not empty: %+v", g)
	}
}
