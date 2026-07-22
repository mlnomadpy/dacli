package spm

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"sort"
)

// DepType is a task dependency type.
//
// Recording the type matters because SS is what makes two tasks genuinely
// safe to run in parallel, and that distinction is invisible in a plain
// "blocked_by" list — which is why agents so often serialize work that could
// have been fanned out, or fan out work that could not.
type DepType string

const (
	FS DepType = "FS" // finish-start: A must finish before B starts (most common)
	SS DepType = "SS" // start-start: A must start before B starts (may overlap)
	FF DepType = "FF" // finish-finish: A must finish before B can finish
	SF DepType = "SF" // start-finish: A must start before B can finish
)

// Node is one task in the network.
type Node struct {
	ID       string
	Duration float64
}

// Edge is a dependency from one task to another.
type Edge struct {
	From string
	To   string
	Type DepType
}

// Schedule is the computed CPM result for one node.
type Schedule struct {
	ID       string
	Duration float64

	EarlyStart  float64
	EarlyFinish float64
	LateStart   float64
	LateFinish  float64

	// Slack is how much this task can slip without delaying the project.
	// Zero slack means it is on the critical path.
	Slack float64

	// Critical is true when Slack is zero (within float tolerance).
	Critical bool
}

// Network is the CPM result for a whole project.
type Network struct {
	Duration  float64
	Schedules map[string]Schedule

	// CriticalPath is the zero-slack chain in topological order.
	//
	// For a human team this says where a delay hurts. For a parent agent it
	// says which tasks to spawn subagents on FIRST: tasks with slack can
	// wait, tasks with zero slack are gating completion. Fanning out onto
	// slack tasks while the critical path sits idle is wasted concurrency,
	// and it is the default agent behavior because agents fan out on
	// whatever decomposed most cleanly, not on what is blocking.
	CriticalPath []string
}

// ErrCycle is returned when the dependency graph is not a DAG. A cycle means
// the decomposition is wrong, not that the scheduler failed.
var ErrCycle = errors.New("dependency cycle: tasks cannot all be ordered")

const eps = 1e-9

// ComputeCPM runs the forward and backward passes and returns the schedule.
func ComputeCPM(nodes []Node, edges []Edge) (*Network, error) {
	dur := make(map[string]float64, len(nodes))
	order := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if _, dup := dur[n.ID]; dup {
			return nil, fmt.Errorf("duplicate task %q", n.ID)
		}
		if n.Duration < 0 {
			return nil, fmt.Errorf("task %q has negative duration", n.ID)
		}
		dur[n.ID] = n.Duration
		order = append(order, n.ID)
	}

	in := map[string][]Edge{}
	out := map[string][]Edge{}
	indeg := map[string]int{}
	for _, id := range order {
		indeg[id] = 0
	}
	for _, e := range edges {
		if _, ok := dur[e.From]; !ok {
			return nil, fmt.Errorf("edge references unknown task %q", e.From)
		}
		if _, ok := dur[e.To]; !ok {
			return nil, fmt.Errorf("edge references unknown task %q", e.To)
		}
		if e.Type == "" {
			e.Type = FS
		}
		in[e.To] = append(in[e.To], e)
		out[e.From] = append(out[e.From], e)
		indeg[e.To]++
	}

	topo, err := kahn(order, out, indeg)
	if err != nil {
		return nil, err
	}

	es := map[string]float64{}
	ef := map[string]float64{}

	// Forward pass. Each dependency type bounds a different endpoint, so FF
	// and SF are converted back to a bound on ES via the node's duration.
	for _, id := range topo {
		start := 0.0
		for _, e := range in[id] {
			var bound float64
			switch e.Type {
			case FS:
				bound = ef[e.From]
			case SS:
				bound = es[e.From]
			case FF:
				bound = ef[e.From] - dur[id]
			case SF:
				bound = es[e.From] - dur[id]
			default:
				return nil, fmt.Errorf("unknown dependency type %q", e.Type)
			}
			start = math.Max(start, bound)
		}
		es[id] = start
		ef[id] = start + dur[id]
	}

	project := 0.0
	for _, id := range topo {
		project = math.Max(project, ef[id])
	}

	lf := map[string]float64{}
	ls := map[string]float64{}

	// Backward pass, in reverse topological order.
	for i := len(topo) - 1; i >= 0; i-- {
		id := topo[i]
		finish := project
		for _, e := range out[id] {
			var bound float64
			switch e.Type {
			case FS:
				bound = ls[e.To]
			case SS:
				bound = ls[e.To] + dur[id]
			case FF:
				bound = lf[e.To]
			case SF:
				bound = lf[e.To] + dur[id]
			}
			finish = math.Min(finish, bound)
		}
		lf[id] = finish
		ls[id] = finish - dur[id]
	}

	net := &Network{Duration: project, Schedules: make(map[string]Schedule, len(topo))}
	for _, id := range topo {
		slack := ls[id] - es[id]
		if math.Abs(slack) < eps {
			slack = 0
		}
		net.Schedules[id] = Schedule{
			ID:          id,
			Duration:    dur[id],
			EarlyStart:  es[id],
			EarlyFinish: ef[id],
			LateStart:   ls[id],
			LateFinish:  lf[id],
			Slack:       slack,
			Critical:    slack == 0,
		}
	}
	for _, id := range topo {
		if net.Schedules[id].Critical {
			net.CriticalPath = append(net.CriticalPath, id)
		}
	}
	return net, nil
}

// kahn topologically sorts the graph, breaking ties by the caller's node
// order so results are deterministic across runs.
//
// The ready frontier is a min-heap keyed on the caller's node position, so
// each pop is O(log V) and the whole sort is O(V log V) — the previous code
// re-sorted the entire frontier on every pop, which is O(V^2 log V).
func kahn(order []string, out map[string][]Edge, indeg map[string]int) ([]string, error) {
	pos := make(map[string]int, len(order))
	for i, id := range order {
		pos[id] = i
	}

	ready := &posHeap{pos: pos}
	for _, id := range order {
		if indeg[id] == 0 {
			ready.ids = append(ready.ids, id)
		}
	}
	heap.Init(ready)

	topo := make([]string, 0, len(order))
	for ready.Len() > 0 {
		id := heap.Pop(ready).(string)
		topo = append(topo, id)
		for _, e := range out[id] {
			indeg[e.To]--
			if indeg[e.To] == 0 {
				heap.Push(ready, e.To)
			}
		}
	}
	if len(topo) != len(order) {
		return nil, ErrCycle
	}
	return topo, nil
}

// posHeap is a min-heap of node ids ordered by their position in the caller's
// node order, giving Kahn's algorithm a deterministic, order-preserving
// frontier without re-sorting on every pop.
type posHeap struct {
	ids []string
	pos map[string]int
}

func (h posHeap) Len() int            { return len(h.ids) }
func (h posHeap) Less(i, j int) bool  { return h.pos[h.ids[i]] < h.pos[h.ids[j]] }
func (h posHeap) Swap(i, j int)       { h.ids[i], h.ids[j] = h.ids[j], h.ids[i] }
func (h *posHeap) Push(x any)         { h.ids = append(h.ids, x.(string)) }
func (h *posHeap) Pop() any {
	old := h.ids
	n := len(old)
	x := old[n-1]
	h.ids = old[:n-1]
	return x
}

// Parallelizable returns up to n tasks worth spawning subagents on right now:
// tasks whose dependencies are already satisfied, ordered critical-path
// first, then by least slack.
//
// This is the scheduling primitive a parent agent actually needs. It answers
// "what should my next N children work on" rather than "what is the status."
func (net *Network) Parallelizable(done map[string]bool, n int) []string {
	type cand struct {
		id    string
		slack float64
	}
	var cs []cand
	for id, s := range net.Schedules {
		if done[id] {
			continue
		}
		cs = append(cs, cand{id, s.Slack})
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].slack != cs[j].slack {
			return cs[i].slack < cs[j].slack
		}
		return cs[i].id < cs[j].id
	})
	if n > 0 && len(cs) > n {
		cs = cs[:n]
	}
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.id
	}
	return out
}
