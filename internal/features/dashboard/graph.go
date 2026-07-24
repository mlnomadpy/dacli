package dashboard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// graphView is the /api/graph payload: the task dependency DAG plus, when the
// open tasks are schedulable, the CPM critical path drawn over them. It exposes
// what internal/spm/criticalpath.go already computes (the same subset `dacli
// next` and `dacli critical-path` schedule) so the operator can SEE the chain
// instead of reconstructing it by hand.
//
// The DAG itself (Nodes + Edges) is always present — every task in scope is a
// node and every resolvable depends_on is an edge, regardless of status or
// estimate. The critical-path overlay is best-effort: Scheduled is true only
// when every open task carries a PERT estimate and the open subgraph is acyclic
// (the same gate cmdCriticalPath enforces, except this degrades with a Note
// rather than refusing, so the DAG still draws).
type graphView struct {
	// Project is the slug this graph covers, "" when it spans every project.
	Project string `json:"project"`
	// Nodes is one entry per task in scope, in ListTasks order (project then seq).
	Nodes []graphNode `json:"nodes"`
	// Edges is a dependency A→B for every depends_on B declares whose ref
	// resolves to a task in Nodes. Type is FS|SS|FF|SF (FS when unspecified).
	Edges []graphEdge `json:"edges"`
	// CriticalPath is the zero-slack chain in topological order (task IDs), empty
	// when Scheduled is false.
	CriticalPath []string `json:"critical_path"`
	// Duration is the project duration in Te units, 0 when Scheduled is false.
	Duration float64 `json:"duration"`
	// Scheduled reports whether the CPM overlay ran: all open tasks estimated and
	// the open subgraph acyclic. When false, the DAG still renders but no node is
	// marked critical and Note explains why.
	Scheduled bool `json:"scheduled"`
	// Note is a human-readable reason the critical path is absent (unestimated
	// open tasks, a cycle, or no open tasks), empty when Scheduled is true.
	Note string `json:"note"`
}

// graphNode is one task in the DAG. Critical/Slack/EarlyStart are meaningful
// only when the graph is Scheduled AND the node is in the scheduled subset
// (open, non-blocked); otherwise Slack is -1 (n/a) and Critical is false.
type graphNode struct {
	ID         string  `json:"id"`
	Seq        int     `json:"seq"`
	Slug       string  `json:"slug"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Points     float64 `json:"points"`      // PERT expected duration; 0 when unestimated
	Estimated  bool    `json:"estimated"`   // whether a valid three-point estimate exists
	Critical   bool    `json:"critical"`    // on the zero-slack critical path
	Slack      float64 `json:"slack"`       // -1 when unscheduled (done, blocked, or Scheduled==false)
	EarlyStart float64 `json:"early_start"` // CPM early start; 0 when unscheduled
}

// graphEdge is a dependency: From must satisfy its type before To. Both ends are
// always node IDs present in graphView.Nodes.
type graphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // FS | SS | FF | SF
}

// graphResponse is GET /api/graph: the graphView with its own `generated` stamp
// so the surface can be polled independently. An optional ?project=<slug> query
// scopes it to one project; absent, the DAG spans every project's tasks.
type graphResponse struct {
	Generated string `json:"generated"`
	graphView
}

func buildGraphResponse(w *workspace.Workspace, project string) (graphResponse, error) {
	gv, err := buildGraph(w, project)
	if err != nil {
		return graphResponse{}, err
	}
	return graphResponse{Generated: nowStamp(), graphView: gv}, nil
}

// buildGraph assembles the dependency DAG fresh on every call (no cache, the
// same honesty rule buildState follows). It never mutates the workspace.
//
// Two passes: first every task becomes a node and every resolvable depends_on
// becomes an edge (the DAG, always drawn); then — only if the open subset is
// schedulable — spm.ComputeCPM runs over exactly the subset dacli next uses
// (open, non-blocked, estimated) and its slack/critical/early-start is copied
// back onto the matching nodes. The overlay degrades to a Note rather than an
// error so the DAG survives an unestimated task or a cycle.
func buildGraph(w *workspace.Workspace, project string) (graphView, error) {
	gv := graphView{
		Project:      project,
		Nodes:        []graphNode{},
		Edges:        []graphEdge{},
		CriticalPath: []string{},
	}
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return gv, err
	}

	// byRef resolves a depends_on ref to a task the same four ways cmdCriticalPath
	// does (full ID, ID without the t- prefix, slug, zero-padded seq), so an edge
	// lands whichever form the author wrote.
	byRef := map[string]*store.Task{}
	for _, t := range tasks {
		for _, ref := range []string{t.ID, strings.TrimPrefix(t.ID, "t-"), t.Slug, fmt.Sprintf("%03d", t.Seq)} {
			byRef[ref] = t
		}
	}

	// Pass 1 — the DAG: a node per task, an edge per resolvable dependency.
	nodeIdx := map[string]int{}
	for _, t := range tasks {
		n := graphNode{
			ID: t.ID, Seq: t.Seq, Slug: t.Slug, Title: t.Title,
			Status: string(t.Status), Slack: -1,
		}
		if tp, ok := t.Estimate(); ok {
			n.Points = tp.Expected()
			n.Estimated = true
		}
		nodeIdx[t.ID] = len(gv.Nodes)
		gv.Nodes = append(gv.Nodes, n)
		for _, d := range t.Deps() {
			if dep, ok := byRef[d.Ref]; ok {
				typ := d.Type
				if typ == "" {
					typ = string(spm.FS)
				}
				gv.Edges = append(gv.Edges, graphEdge{From: dep.ID, To: t.ID, Type: typ})
			}
		}
	}

	// Pass 2 — the CPM overlay over the schedulable subset. Blocked tasks are
	// excluded exactly as `dacli next` and `dacli critical-path` exclude them, so
	// all three readouts agree on what is runnable.
	openIDs := map[string]bool{}
	var open []*store.Task
	for _, t := range tasks {
		if t.Status == model.StatusDone || t.Status == model.StatusBlocked {
			continue
		}
		open = append(open, t)
		openIDs[t.ID] = true
	}

	var unestimated int
	for _, t := range open {
		if _, ok := t.Estimate(); !ok {
			unestimated++
		}
	}
	switch {
	case len(open) == 0:
		gv.Note = "no open tasks to schedule — DAG shown without a critical path"
		return gv, nil
	case unestimated > 0:
		gv.Note = fmt.Sprintf("%d open task(s) lack a PERT estimate — DAG shown without the critical path", unestimated)
		return gv, nil
	}

	nodes := make([]spm.Node, 0, len(open))
	var edges []spm.Edge
	for _, t := range open {
		est, _ := t.Estimate()
		nodes = append(nodes, spm.Node{ID: t.ID, Duration: est.Expected()})
		for _, d := range t.Deps() {
			// Only edges between two scheduled nodes, so a done/blocked predecessor
			// never triggers "edge references unknown task".
			if dep, ok := byRef[d.Ref]; ok && openIDs[dep.ID] {
				edges = append(edges, spm.Edge{From: dep.ID, To: t.ID, Type: spm.DepType(d.Type)})
			}
		}
	}
	net, err := spm.ComputeCPM(nodes, edges)
	if err != nil {
		if errors.Is(err, spm.ErrCycle) {
			gv.Note = "dependency cycle among open tasks — DAG shown without the critical path"
		} else {
			gv.Note = "cannot schedule the critical path — DAG shown without it"
		}
		return gv, nil
	}

	gv.Scheduled = true
	gv.Duration = net.Duration
	gv.CriticalPath = net.CriticalPath
	for id, s := range net.Schedules {
		if i, ok := nodeIdx[id]; ok {
			gv.Nodes[i].Critical = s.Critical
			gv.Nodes[i].Slack = s.Slack
			gv.Nodes[i].EarlyStart = s.EarlyStart
		}
	}
	return gv, nil
}
