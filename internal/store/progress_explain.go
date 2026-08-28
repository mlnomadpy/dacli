// Progress explain projects task and worker state from the same reconciliation
// evidence used by `reconcile`. It is an observed read model, never authority:
// every value carries its source, observation time, and staleness (issue #869).
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const ProgressExplainSchema = "progress-explain/v1"

// Observed is deliberately attached to each field rather than only to the
// envelope. A mixed-source view (task files, proc records, GitHub) must not let
// a fresh local read make an older remote value look current.
type Observed[T any] struct {
	Value      T         `json:"value"`
	Source     string    `json:"source"`
	ObservedAt time.Time `json:"observed_at"`
	Stale      bool      `json:"stale"`
}

type LandingExplain struct {
	Classification string             `json:"classification"`
	Confidence     DeliveryConfidence `json:"confidence"`
	Detail         string             `json:"detail,omitempty"`
	RelatedRefs    []DeliveryRef      `json:"related_refs,omitempty"`
}

type AggregateExplain struct {
	Kind         string `json:"kind"`
	RequiredDone int    `json:"required_done"`
	Required     int    `json:"required"`
	ReadyToClose bool   `json:"ready_to_close"`
}

type TaskExplain struct {
	ID          Observed[string]           `json:"id"`
	Title       Observed[string]           `json:"title"`
	Status      Observed[string]           `json:"status"`
	Rank        Observed[int]              `json:"rank"`
	Slack       Observed[*float64]         `json:"slack"`
	Blockers    Observed[[]string]         `json:"blockers"`
	Claims      Observed[[]string]         `json:"claims"`
	Parent      Observed[string]           `json:"parent"`
	Aggregate   Observed[AggregateExplain] `json:"aggregate"`
	Landing     Observed[LandingExplain]   `json:"landing"`
	RoleRouting Observed[team.Explanation] `json:"role_routing"`
	NextAction  Observed[string]           `json:"next_action"`
}

type WorkerExplain struct {
	RunID      Observed[string]   `json:"run_id"`
	AgentID    Observed[string]   `json:"agent_id"`
	TaskID     Observed[string]   `json:"task_id"`
	Role       Observed[string]   `json:"role"`
	Runtime    Observed[string]   `json:"runtime"`
	State      Observed[string]   `json:"state"`
	Claims     Observed[[]string] `json:"claims"`
	NextAction Observed[string]   `json:"next_action"`
}

type ProgressExplain struct {
	Schema         string             `json:"schema"`
	Version        int                `json:"version"`
	Project        string             `json:"project"`
	ObservedAt     time.Time          `json:"observed_at"`
	Stale          bool               `json:"stale"`
	CacheState     string             `json:"cache_state"`
	Warning        string             `json:"warning,omitempty"`
	Reconciliation DeliveryProjection `json:"reconciliation"`
	Tasks          []TaskExplain      `json:"tasks"`
	Workers        []WorkerExplain    `json:"workers"`
}

func observed[T any](value T, source string, at time.Time, stale bool) Observed[T] {
	return Observed[T]{Value: value, Source: source, ObservedAt: at.UTC(), Stale: stale}
}

// BuildProgressExplain consumes a canonical DeliveryProjection. Landing and
// next-action facts therefore retain reconciliation's classifications instead
// of reinterpreting PR/check state in a second model.
func BuildProgressExplain(w *workspace.Workspace, project string, delivery DeliveryProjection, now time.Time) (ProgressExplain, error) {
	p := ProgressExplain{Schema: ProgressExplainSchema, Version: 1, Project: project, ObservedAt: now.UTC(), CacheState: "observed", Reconciliation: delivery, Tasks: []TaskExplain{}, Workers: []WorkerExplain{}}
	tasks, err := ListTasks(w, project, "")
	if err != nil {
		return p, err
	}
	idx := NewTaskIndex(tasks)
	frontier := ReadyFrontier(tasks)
	ready := map[string]bool{}
	for _, task := range frontier.Ready {
		ready[task.ID] = true
	}
	problemByTask := map[string][]string{}
	for _, problem := range frontier.Problems {
		problemByTask[problem.Task.ID] = append(problemByTask[problem.Task.ID], problem.String())
	}

	slack := explainSlack(tasks)
	rank := explainRank(tasks, frontier.Ready, slack)
	roles, _ := LoadRoles(w)
	occupancy, occupancyErr := LiveOccupancyByRole(w)
	if occupancyErr != nil {
		occupancy = map[string]int{}
	}

	findings := map[string]DeliveryFinding{}
	for _, finding := range delivery.Findings {
		if finding.ObjectID != "" {
			findings[finding.ObjectID] = finding
		}
	}

	for _, task := range tasks {
		blockers := append([]string(nil), problemByTask[task.ID]...)
		for _, dep := range task.Deps() {
			if !DepBlocksStart(dep) {
				continue
			}
			other, findErr := idx.Find(dep.Ref)
			if findErr == nil && other.Status != model.StatusDone {
				blockers = append(blockers, fmt.Sprintf("dependency %s is %s", other.ID, other.Status))
			}
		}
		if task.Status == model.StatusBlocked {
			blockers = append(blockers, "task is recorded blocked")
		}
		sort.Strings(blockers)

		agg := AggregateExplain{Kind: task.TaskKind(), ReadyToClose: !task.IsAggregate()}
		if task.IsAggregate() {
			if progress, progressErr := AggregateProgressFor(w, task); progressErr == nil {
				agg.RequiredDone, agg.Required, agg.ReadyToClose = progress.RequiredDone, progress.Required, progress.ReadyToClose
				blockers = append(blockers, progress.Blockers...)
			}
		}

		landing := LandingExplain{Classification: "unobserved", Confidence: DeliveryUnknown}
		landingSource, landingAt := "delivery-reconciliation", delivery.ObservedAt
		if finding, ok := findings[task.ID]; ok {
			landing = LandingExplain{Classification: finding.Classification, Confidence: finding.Confidence, Detail: finding.Detail, RelatedRefs: append([]DeliveryRef(nil), finding.RelatedRefs...)}
			landingSource, landingAt = finding.Source, finding.ObservedAt
		}

		points := 0.0
		if estimate, ok := task.Estimate(); ok {
			points = estimate.Expected()
		}
		candidates := make([]team.RouteCandidate, 0, len(roles))
		for _, role := range roles {
			copyRole := role
			grantEnforced := copyRole.Grant != "ro"
			if copyRole.Grant == "" {
				copyRole.Grant = "rw"
			}
			remaining := 1
			if copyRole.WIP > 0 {
				remaining = copyRole.WIP - occupancy[copyRole.Name]
			}
			candidates = append(candidates, team.RouteCandidate{Role: copyRole, GrantEnforced: grantEnforced, ContextLimit: copyRole.Profile.ContextLimit, CapacityRemaining: remaining})
		}
		routing := (team.Strategy{}).Select(team.RouteRequirements{Kind: "implementer", Grant: "rw", Title: task.Title, Paths: ClaimHints(w.Root, task), TaskPoints: points}, candidates)
		claims := task.Claims()
		if len(claims) == 0 {
			claims = ClaimHints(w.Root, task)
		}

		next := "observe recorded state"
		switch {
		case task.Status == model.StatusDone:
			next = "none — task is done"
		case len(blockers) > 0:
			next = "resolve: " + blockers[0]
		case landing.Classification != "unobserved" && landing.Classification != "canonical-pr":
			next = findings[task.ID].NextAction
		case task.Status == model.StatusActive:
			next = "observe the assigned worker and landing evidence"
		case ready[task.ID] && routing.Selected.Role != "":
			next = "assign " + routing.Selected.Role
		case ready[task.ID]:
			next = "repair role eligibility; no implementation role is currently eligible"
		}

		var slackValue *float64
		if value, ok := slack[task.ID]; ok {
			valueCopy := value
			slackValue = &valueCopy
		}
		p.Tasks = append(p.Tasks, TaskExplain{
			ID: observed(task.ID, "task-file", now, false), Title: observed(task.Title, "task-file", now, false),
			Status: observed(string(task.Status), "task-folder", now, false), Rank: observed(rank[task.ID], "ready-frontier/cpm", now, false),
			Slack: observed(slackValue, "cpm", now, false), Blockers: observed(blockers, "ready-frontier/aggregate", now, false),
			Claims: observed(claims, "task-file", now, false), Parent: observed(task.ParentID(), "task-file", now, false),
			Aggregate: observed(agg, "aggregate-progress", now, false), Landing: observed(landing, landingSource, landingAt, false),
			RoleRouting: observed(routing, "team-routing/live-occupancy", now, false), NextAction: observed(next, "canonical-progress-explain", now, false),
		})
	}

	entries, readErr := os.ReadDir(w.RunsDir())
	if readErr != nil && !os.IsNotExist(readErr) {
		return p, fmt.Errorf("read runs: %w", readErr)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rec, recErr := procmon.ReadRecord(filepath.Join(w.RunDir(entry.Name()), "proc.txt"))
		if recErr != nil || rec.Task == "" {
			continue
		}
		task, findErr := idx.Find(rec.Task)
		if findErr != nil || task.Project != project {
			continue
		}
		state, next := rec.Outcome, "inspect terminal run evidence"
		if state == "" && procmon.AliveRecord(rec) {
			state, next = "live", "observe worker progress"
		} else if state == "" {
			state, next = "finished-unfinalized", "finalize the run before trusting its outcome"
		}
		if RootHandoffRequested(w, rec.RunID) {
			state, next = "handoff-required", "root re-observes and consumes the structured handoff"
		}
		p.Workers = append(p.Workers, WorkerExplain{
			RunID: observed(rec.RunID, "run/proc", now, false), AgentID: observed(rec.Child, "run/proc", now, false),
			TaskID: observed(task.ID, "run/proc", now, false), Role: observed(rec.Role, "run/proc", now, false), Runtime: observed(rec.Runtime, "run/proc", now, false),
			State: observed(state, "run/proc+liveness", now, false), Claims: observed(append([]string(nil), rec.Claims...), "run/proc", now, false),
			NextAction: observed(next, "canonical-progress-explain", now, false),
		})
	}
	sort.Slice(p.Tasks, func(i, j int) bool { return p.Tasks[i].ID.Value < p.Tasks[j].ID.Value })
	sort.Slice(p.Workers, func(i, j int) bool { return p.Workers[i].RunID.Value < p.Workers[j].RunID.Value })
	return p, nil
}

func explainSlack(tasks []*Task) map[string]float64 {
	byRef, openIDs := map[string]*Task{}, map[string]bool{}
	var open []*Task
	for _, task := range tasks {
		for _, ref := range []string{task.ID, strings.TrimPrefix(task.ID, "t-"), task.Slug, fmt.Sprintf("%03d", task.Seq)} {
			byRef[ref] = task
		}
		if task.Status != model.StatusDone && task.Status != model.StatusBlocked && !task.IsLoopAnchor() && !task.IsAggregate() {
			open, openIDs[task.ID] = append(open, task), true
		}
	}
	var nodes []spm.Node
	var edges []spm.Edge
	for _, task := range open {
		estimate, ok := task.Estimate()
		if !ok {
			return map[string]float64{}
		}
		nodes = append(nodes, spm.Node{ID: task.ID, Duration: estimate.Expected()})
		for _, dep := range task.Deps() {
			if other, ok := byRef[dep.Ref]; ok && openIDs[other.ID] {
				edges = append(edges, spm.Edge{From: other.ID, To: task.ID, Type: spm.DepType(dep.Type)})
			}
		}
	}
	net, err := spm.ComputeCPM(nodes, edges)
	if err != nil {
		return map[string]float64{}
	}
	out := map[string]float64{}
	for id, schedule := range net.Schedules {
		out[id] = schedule.Slack
	}
	return out
}

func explainRank(tasks, ready []*Task, slack map[string]float64) map[string]int {
	ordered := append([]*Task(nil), ready...)
	sort.SliceStable(ordered, func(i, j int) bool {
		pi, pj := model.Priority(ordered[i].Priority()).Rank(), model.Priority(ordered[j].Priority()).Rank()
		if pi != pj {
			return pi < pj
		}
		si, iok := slack[ordered[i].ID]
		sj, jok := slack[ordered[j].ID]
		if iok && jok && si != sj {
			return si < sj
		}
		return ordered[i].Seq < ordered[j].Seq
	})
	out := map[string]int{}
	for i, task := range ordered {
		out[task.ID] = i + 1
	}
	return out
}

type explainCacheEntry struct {
	projection ProgressExplain
	usedAt     time.Time
}

// ExplainCache is a bounded process-local observation cache. Fresh values are
// reused briefly; an observer outage may fall back only within maxStale, and
// every fact in that fallback is relabelled stale before it escapes.
type ExplainCache struct {
	mu         sync.Mutex
	maxEntries int
	freshFor   time.Duration
	maxStale   time.Duration
	entries    map[string]explainCacheEntry
}

func NewExplainCache(maxEntries int, freshFor, maxStale time.Duration) *ExplainCache {
	if maxEntries < 1 {
		maxEntries = 1
	}
	return &ExplainCache{maxEntries: maxEntries, freshFor: freshFor, maxStale: maxStale, entries: map[string]explainCacheEntry{}}
}

func (c *ExplainCache) Get(key string, now time.Time, observe func() (ProgressExplain, error)) (ProgressExplain, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[key]; ok && now.Sub(entry.projection.ObservedAt) <= c.freshFor {
		entry.usedAt = now
		entry.projection.CacheState = "fresh-cache"
		c.entries[key] = entry
		return entry.projection, nil
	}
	p, err := observe()
	if err == nil {
		p.CacheState = "observed"
		c.entries[key] = explainCacheEntry{projection: p, usedAt: now}
		c.evict()
		return p, nil
	}
	if entry, ok := c.entries[key]; ok && now.Sub(entry.projection.ObservedAt) <= c.maxStale {
		p = entry.projection
		p.CacheState, p.Warning = "stale-fallback", err.Error()
		markProgressExplainStale(&p)
		entry.usedAt = now
		c.entries[key] = entry
		return p, nil
	}
	return p, err
}

func (c *ExplainCache) evict() {
	for len(c.entries) > c.maxEntries {
		oldestKey, oldest := "", time.Time{}
		for key, entry := range c.entries {
			if oldestKey == "" || entry.usedAt.Before(oldest) {
				oldestKey, oldest = key, entry.usedAt
			}
		}
		delete(c.entries, oldestKey)
	}
}

func markProgressExplainStale(p *ProgressExplain) {
	p.Stale = true
	for i := range p.Tasks {
		t := &p.Tasks[i]
		t.ID.Stale, t.Title.Stale, t.Status.Stale, t.Rank.Stale, t.Slack.Stale = true, true, true, true, true
		t.Blockers.Stale, t.Claims.Stale, t.Parent.Stale, t.Aggregate.Stale = true, true, true, true
		t.Landing.Stale, t.RoleRouting.Stale, t.NextAction.Stale = true, true, true
	}
	for i := range p.Workers {
		w := &p.Workers[i]
		w.RunID.Stale, w.AgentID.Stale, w.TaskID.Stale, w.Role.Stale, w.Runtime.Stale = true, true, true, true, true
		w.State.Stale, w.Claims.Stale, w.NextAction.Stale = true, true, true
	}
}

var SharedProgressExplainCache = NewExplainCache(32, 15*time.Second, 5*time.Minute)

func ExplainProject(w *workspace.Workspace, project string, now time.Time) (ProgressExplain, error) {
	key := w.Root + "\x00" + project
	p, err := SharedProgressExplainCache.Get(key, now, func() (ProgressExplain, error) {
		delivery, observeErr := ReconcileDelivery(w, project, now)
		projection, buildErr := BuildProgressExplain(w, project, delivery, now)
		if buildErr != nil {
			return projection, buildErr
		}
		return projection, observeErr
	})
	// Explain is advisory. On the first GitHub outage there is no cache to fall
	// back to, but ReconcileDelivery still returned current local facts plus a
	// typed github-state-unknown finding. Preserve that useful projection and
	// warning instead of making agent status disappear behind the outage.
	if err != nil && p.Schema == ProgressExplainSchema {
		p.CacheState = "partial-observation"
		p.Warning = err.Error()
		return p, nil
	}
	return p, err
}
