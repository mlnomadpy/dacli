package orchestration

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func readyFrontier(w *workspace.Workspace, project string) (store.Frontier, error) {
	tasks, err := store.ListTasks(w, "", "")
	if err != nil {
		return store.Frontier{}, err
	}
	return store.ReadyFrontierForProject(tasks, project), nil
}

// readyTasks returns the workable frontier the loop draws from.
func readyTasks(w *workspace.Workspace, project string) ([]*store.Task, error) {
	fr, err := readyFrontier(w, project)
	if err != nil {
		return nil, err
	}
	return fr.Ready, nil
}

type waveClaimCollision struct {
	Task         *store.Task
	Theirs, Mine string
}

type compatibleWave struct {
	Tasks      []*store.Task
	Claims     map[int][]string
	Collisions []waveClaimCollision
}

// selectClaimCompatibleWave fills a bounded wave from an already-ranked ready
// frontier. Claim compatibility is part of selection, not a launch-time
// afterthought: truncating first let two zero-slack tasks that claimed the same
// slice consume both width slots, then rejected the second while independent
// ready work sat unused (issue #838). Scanning until width is full also gives
// start preview and loop execution one deterministic selection predicate.
func selectClaimCompatibleWave(root string, ready []*store.Task, width int) compatibleWave {
	wave := compatibleWave{Claims: make(map[int][]string)}
	if width <= 0 {
		return wave
	}
	var claimed []string
	for _, task := range ready {
		if len(wave.Tasks) == width {
			break
		}
		claims := store.ClaimHints(root, task)
		if theirs, mine, overlap := procmon.PathsOverlap(claimed, claims); overlap {
			wave.Collisions = append(wave.Collisions, waveClaimCollision{Task: task, Theirs: theirs, Mine: mine})
			continue
		}
		wave.Tasks = append(wave.Tasks, task)
		wave.Claims[task.Seq] = claims
		claimed = append(claimed, claims...)
	}
	return wave
}

// readyTasks (driver method) is the frontier the BUILD phase draws from, and
// the one place that REPORTS the data faults holding tasks back. A dependency
// ref naming no task blocks its task forever; the loop refusing to run it is
// correct, but refusing in silence is what made 240 a mystery instead of a
// one-line fix. Called once per cycle at the decision point — the frontier
// re-reads elsewhere in the loop stay quiet so the note appears once.
func (d *driver) readyTasks() ([]*store.Task, error) {
	fr, err := readyFrontier(d.w, d.cfg.project)
	if err != nil {
		return nil, err
	}
	for _, line := range fr.ProblemLines() {
		d.logf("  note: %s — fix the ref; this task cannot be scheduled until it resolves", line)
	}
	return fr.Ready, nil
}

// rankByPriority orders the ready frontier by MoSCoW priority rank, then
// critical-path slack when a CPM schedule can be computed, then Seq as the
// final tiebreak — mirroring cmdNext's selection (insight.go cmdNext) so the
// loop's BUILD phase and `dacli next` agree on what to work on first. Without
// this, a low-seq could/should would be built ahead of a higher-seq must and
// the critical path would be ignored, contradicting the loop's own
// MoSCoW/critical-path-first charter. Sorts in place.
func rankByPriority(w *workspace.Workspace, project string, ready []*store.Task) {
	if len(ready) < 2 {
		return
	}
	slack, haveCPM := criticalPathSlack(w, project)
	sort.SliceStable(ready, func(i, j int) bool {
		pi, pj := model.Priority(ready[i].Priority()).Rank(), model.Priority(ready[j].Priority()).Rank()
		if pi != pj {
			return pi < pj
		}
		if haveCPM && slack[ready[i].ID] != slack[ready[j].ID] {
			return slack[ready[i].ID] < slack[ready[j].ID]
		}
		return ready[i].Seq < ready[j].Seq
	})
}

// criticalPathSlack computes CPM slack for every open (non-done, non-blocked)
// task in the project. Duplicated from insight.cmdNext's CPM block rather
// than imported — the feature-slice isolation rule (TestFeatureSlicesAreIsolated)
// forbids orchestration importing a sibling feature. Degrades to
// haveCPM=false when any open task is missing an estimate, same as cmdNext.
func criticalPathSlack(w *workspace.Workspace, project string) (map[string]float64, bool) {
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return nil, false
	}
	byRef := map[string]*store.Task{}
	openIDs := map[string]bool{}
	var open []*store.Task
	for _, t := range tasks {
		for _, ref := range []string{t.ID, strings.TrimPrefix(t.ID, "t-"), t.Slug, fmt.Sprintf("%03d", t.Seq)} {
			byRef[ref] = t
		}
		// Exclude the loop anchor, exactly as cmdNext does (insight.go:168).
		// It is a standing review-phase prompt, never implementer work — and
		// it is created UNSIZED (ensureImproveTask passes no Estimate) and
		// never sized, because sizeUnestimated only sizes the wave batch and
		// readiness filters anchors out of that. Including it here meant
		// t.Estimate() failed on it every cycle, so haveCPM went false and the
		// BUILD phase silently fell back to MoSCoW+seq while `dacli next`
		// showed the operator critical-path order. The two are documented to
		// agree; they did not.
		if t.Status != model.StatusDone && t.Status != model.StatusBlocked && !t.IsLoopAnchor() {
			open = append(open, t)
			openIDs[t.ID] = true
		}
	}

	var nodes []spm.Node
	var edges []spm.Edge
	for _, t := range open {
		est, ok := t.Estimate()
		if !ok {
			return nil, false
		}
		nodes = append(nodes, spm.Node{ID: t.ID, Duration: est.Expected()})
		for _, d := range t.Deps() {
			if dep, ok := byRef[d.Ref]; ok && openIDs[dep.ID] {
				edges = append(edges, spm.Edge{From: dep.ID, To: t.ID, Type: spm.DepType(d.Type)})
			}
		}
	}
	net, err := spm.ComputeCPM(nodes, edges)
	if err != nil {
		return nil, false
	}
	slack := map[string]float64{}
	for id, s := range net.Schedules {
		slack[id] = s.Slack
	}
	return slack, true
}

// --- small helpers ---

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// resolveLoopHarnessPolicy fixes the eligible CLI family before any costly
// phase starts. Model selection remains free inside that boundary; crossing it
// requires an explicit hybrid policy (issue #833).
func resolveLoopHarnessPolicy(w *workspace.Workspace, cfg *loopCfg, requested []string, hybrid bool) error {
	allowed := uniqueStrings(requested)
	mode := "single"
	if len(allowed) == 0 {
		if p, err := loadProfile(w, cfg.project); err == nil {
			allowed = append([]string{}, p.Routing.AllowedHarnesses...)
			mode = p.Routing.HarnessMode
		}
	}
	if mode == "" {
		mode = "single"
	}
	if len(allowed) == 0 {
		role, ok := store.LoadRole(w, cfg.implRole)
		if ok {
			if rt, err := store.LoadRuntime(w, role.Runtime); err == nil {
				allowed = []string{rt.Harness}
			}
		}
	}
	if len(requested) > 0 {
		if hybrid {
			mode = "hybrid"
		} else {
			mode = "single"
		}
	} else if hybrid {
		return clikit.Usagef("--hybrid requires at least two --harness values")
	}
	if mode == "single" && len(allowed) > 1 {
		return clikit.Usagef("single-harness mode needs exactly one --harness value; pass --hybrid to authorize multiple")
	}
	if mode == "hybrid" && len(allowed) < 2 {
		return clikit.Usagef("--hybrid requires at least two --harness values")
	}
	cfg.harnessMode, cfg.allowedHarnesses = mode, allowed
	if len(allowed) == 0 {
		return nil // legacy fixtures/workspaces with no runtime declaration
	}

	for _, spec := range []struct {
		name     *string
		explicit bool
		phase    string
	}{
		{&cfg.implRole, cfg.implRoleExplicit, "implementation"},
		{&cfg.reviewRole, cfg.reviewRoleExplicit, "review"},
	} {
		role, ok := store.LoadRole(w, *spec.name)
		if ok && roleAllowedByHarness(w, role, allowed) {
			continue
		}
		if spec.explicit {
			return clikit.Refusedf("explicit %s role %s is outside harness policy %s:%s; choose a compatible role or explicitly authorize a hybrid harness set", spec.phase, *spec.name, mode, strings.Join(allowed, ","))
		}
		replacement, found := cheapestAllowedRole(w, role.Kind, allowed, spec.phase == "implementation")
		if !found {
			return clikit.Refusedf("no %s role is available inside harness policy %s:%s; configure one or explicitly authorize a hybrid harness set", spec.phase, mode, strings.Join(allowed, ","))
		}
		*spec.name = replacement.Name
	}
	return nil
}

func (d *driver) roleAllowedByHarness(role team.Role) bool {
	if len(d.cfg.allowedHarnesses) == 0 {
		return true
	}
	return roleAllowedByHarness(d.w, role, d.cfg.allowedHarnesses)
}

func roleAllowedByHarness(w *workspace.Workspace, role team.Role, allowed []string) bool {
	rt, err := store.LoadRuntime(w, role.Runtime)
	return err == nil && slices.Contains(allowed, rt.Harness)
}

func cheapestAllowedRole(w *workspace.Workspace, kind string, allowed []string, writer bool) (team.Role, bool) {
	roles, _ := store.LoadRoles(w)
	var candidates []team.Role
	for _, role := range roles {
		if !strings.EqualFold(role.Kind, kind) || !roleAllowedByHarness(w, role, allowed) {
			continue
		}
		if writer {
			rt, err := store.LoadRuntime(w, role.Runtime)
			if role.Grant == "ro" || err != nil || !store.RuntimeWritable(rt) {
				continue
			}
		}
		candidates = append(candidates, role)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return team.ModelTier(candidates[i].Profile.CostTier) < team.ModelTier(candidates[j].Profile.CostTier)
	})
	if len(candidates) == 0 {
		return team.Role{}, false
	}
	return candidates[0], true
}
