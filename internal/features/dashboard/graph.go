package dashboard

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
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
// buildGraph first assembles the complete DAG. The typed /api/graph endpoint
// then applies a bounded projection and discloses its exact scope/counts;
// legacy /api/state retains the complete compatibility graph. The
// critical-path overlay is best-effort: Scheduled is true only when every open
// task carries a PERT estimate and the open subgraph is acyclic (the same gate
// cmdCriticalPath enforces, except this degrades with a Note rather than
// refusing, so the DAG still draws).
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
	// Projection describes the bounded server-side view. Legacy /api/state
	// graphs leave this at its zero value; /api/graph always populates it.
	Projection graphProjection `json:"projection"`
}

const (
	operationalGraphLimit = 120
	focusedGraphLimit     = 80
	historyGraphLimit     = 100
)

type graphProjection struct {
	Mode          string   `json:"mode"`
	Rule          string   `json:"rule"`
	Focus         string   `json:"focus,omitempty"`
	Statuses      []string `json:"statuses"`
	Page          int      `json:"page"`
	Limit         int      `json:"limit"`
	TotalNodes    int      `json:"total_nodes"`
	VisibleNodes  int      `json:"visible_nodes"`
	HiddenNodes   int      `json:"hidden_nodes"`
	TotalEdges    int      `json:"total_edges"`
	VisibleEdges  int      `json:"visible_edges"`
	HiddenEdges   int      `json:"hidden_edges"`
	CriticalTotal int      `json:"critical_total"`
	HasMore       bool     `json:"has_more"`
}

type graphOptions struct {
	Mode     string
	Focus    string
	Statuses map[string]bool
	Page     int
}

func parseGraphOptions(mode, focus, statuses, page string) (graphOptions, error) {
	opts := graphOptions{Mode: mode, Focus: focus, Page: 1}
	if opts.Mode == "" {
		opts.Mode = "operational"
	}
	if opts.Mode != "operational" && opts.Mode != "history" {
		return opts, fmt.Errorf("invalid graph mode %q", opts.Mode)
	}
	if focus != "" && strings.ContainsAny(focus, "/\\") {
		return opts, fmt.Errorf("invalid graph focus")
	}
	if page != "" {
		n, err := strconv.Atoi(page)
		if err != nil || n < 1 {
			return opts, fmt.Errorf("invalid graph page %q", page)
		}
		opts.Page = n
	}
	if statuses != "" {
		opts.Statuses = map[string]bool{}
		for _, status := range strings.Split(statuses, ",") {
			if status != "open" && status != "active" && status != "blocked" {
				return opts, fmt.Errorf("invalid graph status %q", status)
			}
			opts.Statuses[status] = true
		}
	}
	return opts, nil
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

// graphResponse is GET /api/graph: a bounded graphView with its own `generated`
// stamp so the surface can be polled independently. Query parameters select an
// operational view, exact focus neighborhood, or completed-history page.
type graphResponse struct {
	Generated string `json:"generated"`
	graphView
}

func buildGraphResponse(w *workspace.Workspace, project string) (graphResponse, error) {
	gv, err := buildGraph(w, project)
	if err != nil {
		return graphResponse{}, err
	}
	return graphResponse{Generated: nowStamp(), graphView: projectGraph(gv, graphOptions{})}, nil
}

func buildGraphResponseWithOptions(w *workspace.Workspace, project string, opts graphOptions) (graphResponse, error) {
	if opts.Focus != "" {
		task, err := store.FindTask(w, opts.Focus)
		if err != nil {
			return graphResponse{}, taskError(opts.Focus, err)
		}
		if project != "" && task.Project != project {
			return graphResponse{}, notFound("task in project "+project, opts.Focus)
		}
		opts.Focus = task.ID
	}
	gv, err := buildGraph(w, project)
	if err != nil {
		return graphResponse{}, err
	}
	projected := projectGraph(gv, opts)
	if opts.Focus != "" && projected.Projection.Focus == "" {
		return graphResponse{}, fmt.Errorf("task %q is not present in project %q", opts.Focus, project)
	}
	return graphResponse{Generated: nowStamp(), graphView: projected}, nil
}

// projectGraph applies the API's bounded, deterministic read model after CPM
// has been calculated from the complete graph. It never changes scheduling
// truth: visible critical nodes retain their full-graph markings and the
// projection metadata says exactly what was omitted.
func projectGraph(full graphView, opts graphOptions) graphView {
	totalNodes, totalEdges, totalCritical := len(full.Nodes), len(full.Edges), len(full.CriticalPath)
	mode := opts.Mode
	if mode == "" {
		mode = "operational"
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	byID := make(map[string]graphNode, len(full.Nodes))
	byRef := make(map[string]string, len(full.Nodes)*3)
	preds := map[string][]string{}
	succs := map[string][]string{}
	for _, n := range full.Nodes {
		byID[n.ID] = n
		for _, ref := range []string{n.ID, strings.TrimPrefix(n.ID, "t-"), n.Slug, fmt.Sprintf("%03d", n.Seq), strconv.Itoa(n.Seq)} {
			byRef[ref] = n.ID
		}
	}
	for _, e := range full.Edges {
		preds[e.To] = append(preds[e.To], e.From)
		succs[e.From] = append(succs[e.From], e.To)
	}

	selected := map[string]bool{}
	hasMore := false
	rule := "unfinished tasks matching the status filter, then completed ancestors; capped at 120 nodes"
	limit := operationalGraphLimit
	focusID := ""
	if opts.Focus != "" {
		focusID = byRef[opts.Focus]
		if focusID != "" {
			selected[focusID] = true
			frontier := []string{focusID}
			for depth := 0; depth < 2 && len(frontier) > 0; depth++ {
				next := []string{}
				for _, id := range frontier {
					neighbors := append(append([]string{}, preds[id]...), succs[id]...)
					sort.Strings(neighbors)
					for _, neighbor := range neighbors {
						if !selected[neighbor] && len(selected) < focusedGraphLimit {
							selected[neighbor] = true
							next = append(next, neighbor)
						}
					}
				}
				frontier = next
			}
		}
		mode, limit = "focus", focusedGraphLimit
		rule = "exact task plus two predecessor/successor hops; capped at 80 nodes"
	} else if mode == "history" {
		limit = historyGraphLimit
		rule = "completed tasks ordered by sequence, one page of 100 nodes"
		done := []graphNode{}
		for _, n := range full.Nodes {
			if n.Status == string(model.StatusDone) {
				done = append(done, n)
			}
		}
		sort.Slice(done, func(i, j int) bool { return done[i].Seq < done[j].Seq })
		start := (opts.Page - 1) * limit
		if start > len(done) {
			start = len(done)
		}
		end := start + limit
		if end > len(done) {
			end = len(done)
		}
		for _, n := range done[start:end] {
			selected[n.ID] = true
		}
		hasMore = end < len(done)
	} else {
		seeds := []graphNode{}
		for _, n := range full.Nodes {
			if n.Status == string(model.StatusDone) || (len(opts.Statuses) > 0 && !opts.Statuses[n.Status]) {
				continue
			}
			seeds = append(seeds, n)
		}
		sort.SliceStable(seeds, func(i, j int) bool {
			if seeds[i].Critical != seeds[j].Critical {
				return seeds[i].Critical
			}
			return seeds[i].Seq < seeds[j].Seq
		})
		for _, n := range seeds {
			if len(selected) >= limit {
				break
			}
			selected[n.ID] = true
		}
		queue := append([]graphNode{}, seeds...)
		for len(queue) > 0 && len(selected) < limit {
			id := queue[0].ID
			queue = queue[1:]
			parents := append([]string{}, preds[id]...)
			sort.Strings(parents)
			for _, parent := range parents {
				if selected[parent] {
					continue
				}
				selected[parent] = true
				queue = append(queue, byID[parent])
				if len(selected) >= limit {
					break
				}
			}
		}
		hasMore = len(selected) < totalNodes
	}

	visible := make([]graphNode, 0, len(selected))
	for _, n := range full.Nodes {
		if selected[n.ID] {
			visible = append(visible, n)
		}
	}
	edges := make([]graphEdge, 0)
	for _, e := range full.Edges {
		if selected[e.From] && selected[e.To] {
			edges = append(edges, e)
		}
	}
	cp := make([]string, 0, len(full.CriticalPath))
	for _, id := range full.CriticalPath {
		if selected[id] {
			cp = append(cp, id)
		}
	}
	statuses := make([]string, 0, len(opts.Statuses))
	for status := range opts.Statuses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	full.Nodes, full.Edges, full.CriticalPath = visible, edges, cp
	full.Projection = graphProjection{
		Mode: mode, Rule: rule, Focus: focusID, Statuses: statuses, Page: opts.Page, Limit: limit,
		TotalNodes: totalNodes, VisibleNodes: len(visible), HiddenNodes: totalNodes - len(visible),
		TotalEdges: totalEdges, VisibleEdges: len(edges), HiddenEdges: totalEdges - len(edges),
		CriticalTotal: totalCritical, HasMore: hasMore,
	}
	return full
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
