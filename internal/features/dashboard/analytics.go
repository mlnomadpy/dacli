package dashboard

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const outcomeAnalyticsSchema = "outcome-analytics/v1"
const analyticsEvidenceLimit = 100
const analyticsCacheLimit = 8
const analyticsCacheTTL = 5 * time.Second

type analyticsWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Days  int    `json:"days"`
}

type analyticsEvidence struct {
	Tasks     []string `json:"tasks"`
	Runs      []string `json:"runs"`
	Truncated bool     `json:"truncated"`
}

type analyticsMeasure struct {
	Key        string            `json:"key"`
	Label      string            `json:"label"`
	Value      *float64          `json:"value"`
	Unit       string            `json:"unit"`
	Samples    int               `json:"sample_size"`
	Eligible   int               `json:"eligible"`
	Coverage   float64           `json:"coverage"`
	State      string            `json:"state"`
	Provenance string            `json:"provenance"`
	Caveat     string            `json:"caveat,omitempty"`
	Evidence   analyticsEvidence `json:"evidence"`
}

type analyticsMetric struct {
	Key      string           `json:"key"`
	Label    string           `json:"label"`
	Current  analyticsMeasure `json:"current"`
	Previous analyticsMeasure `json:"previous"`
	Change   *float64         `json:"change"`
	Trend    string           `json:"trend"`
}

type analyticsBreakdown struct {
	Dimension  string            `json:"dimension"`
	Key        string            `json:"key"`
	SizeBand   string            `json:"size_band,omitempty"`
	Current    analyticsMeasure  `json:"current"`
	Previous   analyticsMeasure  `json:"previous"`
	Comparable bool              `json:"comparable"`
	Caveat     string            `json:"caveat,omitempty"`
	Evidence   analyticsEvidence `json:"evidence"`
}

type outcomeAnalyticsResponse struct {
	Schema      string               `json:"schema"`
	Generated   string               `json:"generated"`
	Project     string               `json:"project"`
	Current     analyticsWindow      `json:"current_window"`
	Previous    analyticsWindow      `json:"previous_window"`
	Metrics     []analyticsMetric    `json:"metrics"`
	Breakdowns  []analyticsBreakdown `json:"breakdowns"`
	Series      []analyticsDay       `json:"series"`
	Performance analyticsPerformance `json:"performance"`
	Notes       []string             `json:"notes"`
}

type analyticsDay struct {
	Day       string            `json:"day"`
	Completed int               `json:"completed"`
	Runs      int               `json:"runs"`
	Tokens    int64             `json:"tokens"`
	Evidence  analyticsEvidence `json:"evidence"`
}

type analyticsPerformance struct {
	TasksScanned int    `json:"tasks_scanned"`
	RunsScanned  int    `json:"runs_scanned"`
	SeriesPoints int    `json:"series_points"`
	BuildMS      int64  `json:"build_ms"`
	EvidenceCap  int    `json:"evidence_cap"`
	Cache        string `json:"cache"`
	CacheEntries int    `json:"cache_entries"`
}

type analyticsRun struct {
	id, task, role, runtime, model, outcome string
	started, ended                          time.Time
	tokens                                  int64
	cost                                    float64
	costKnown, usageKnown                   bool
}

type analyticsTask struct {
	task                 *store.Task
	created, generation  time.Time
	completed            time.Time
	runs                 []analyticsRun
	size                 string
	verificationCurrent  bool
	verificationContract string
	reviewKnown          bool
	corrections          int
	generationNumber     int
}

type outcomeCacheEntry struct {
	created time.Time
	value   outcomeAnalyticsResponse
}

type outcomeCache struct {
	mu      sync.Mutex
	entries map[string]outcomeCacheEntry
}

func newOutcomeCache() *outcomeCache { return &outcomeCache{entries: map[string]outcomeCacheEntry{}} }

func (c *outcomeCache) build(w *workspace.Workspace, project string, days int, now time.Time) (outcomeAnalyticsResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%s/%d", project, days)
	if hit, ok := c.entries[key]; ok && now.Sub(hit.created) < analyticsCacheTTL {
		value := hit.value
		value.Performance.Cache = "bounded-ttl-hit"
		value.Performance.CacheEntries = len(c.entries)
		return value, nil
	}
	value, err := buildOutcomeAnalytics(w, project, days, now)
	if err != nil {
		return value, err
	}
	for len(c.entries) >= analyticsCacheLimit {
		oldestKey, oldest := "", now
		for candidate, entry := range c.entries {
			if oldestKey == "" || entry.created.Before(oldest) {
				oldestKey, oldest = candidate, entry.created
			}
		}
		delete(c.entries, oldestKey)
	}
	value.Performance.Cache = "fresh-index"
	c.entries[key] = outcomeCacheEntry{created: now, value: value}
	value.Performance.CacheEntries = len(c.entries)
	return value, nil
}

func buildOutcomeAnalytics(w *workspace.Workspace, project string, days int, now time.Time) (outcomeAnalyticsResponse, error) {
	started := time.Now()
	if days != 7 && days != 30 && days != 90 {
		return outcomeAnalyticsResponse{}, fmt.Errorf("range must be 7d, 30d, or 90d")
	}
	now = now.UTC()
	currentStart, previousStart := now.AddDate(0, 0, -days), now.AddDate(0, 0, -2*days)
	resp := outcomeAnalyticsResponse{
		Schema: outcomeAnalyticsSchema, Generated: now.Format(time.RFC3339Nano), Project: project,
		Current:  analyticsWindow{Start: currentStart.Format(time.RFC3339), End: now.Format(time.RFC3339), Days: days},
		Previous: analyticsWindow{Start: previousStart.Format(time.RFC3339), End: currentStart.Format(time.RFC3339), Days: days},
		Notes: []string{
			"Comparisons are descriptive, not causal; model and route slices require comparable task sizes and adequate samples.",
			"Cost is provider-reported usage, not a billing statement; missing cost is excluded, never counted as zero.",
			"Ready-to-merged remains unknown until dacli durably timestamps the ready transition.",
		},
	}
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return resp, err
	}
	runs := scanAnalyticsRuns(w)
	byTask := map[string][]analyticsRun{}
	for _, run := range runs {
		byTask[run.task] = append(byTask[run.task], run)
	}
	indexed := make([]analyticsTask, 0, len(tasks))
	projectRuns := 0
	for _, task := range tasks {
		at := analyticsTask{task: task, created: taskULIDTime(task.ID), generation: generationStart(task), completed: lastLogStamp(task, "completed by"), size: taskSize(task), generationNumber: task.Generation()}
		for _, run := range byTask[task.ID] {
			if at.generation.IsZero() || !run.started.Before(at.generation) {
				at.runs = append(at.runs, run)
			}
		}
		evidence := store.VerificationEvidenceRecords(task)
		var latestVerification store.VerificationEvidence
		for i := len(evidence) - 1; i >= 0; i-- {
			if evidence[i].Legacy == "" && evidence[i].ExitCode == 0 && evidence[i].Clean && evidence[i].CommitSHA != "" && evidence[i].TreeSHA != "" {
				latestVerification = evidence[i]
				at.verificationContract = strings.Join(evidence[i].Argv, " ")
				break
			}
		}
		if tx, readErr := store.ReadReviewTransaction(w, project, task.ID); readErr == nil {
			at.reviewKnown, at.corrections = !tx.UpdatedAt.Before(at.generation), tx.CorrectionTurns
			at.verificationCurrent = latestVerification.TreeSHA != "" && tx.State == store.ReviewApproved && tx.CurrentTree == latestVerification.TreeSHA && !tx.UpdatedAt.Before(at.generation)
		} else if task.Generation() == 0 {
			at.verificationCurrent = latestVerification.TreeSHA != ""
		}
		projectRuns += len(at.runs)
		indexed = append(indexed, at)
	}
	resp.Metrics = buildAnalyticsMetrics(indexed, previousStart, currentStart, now)
	resp.Breakdowns = buildAnalyticsBreakdowns(indexed, previousStart, currentStart, now)
	resp.Series = buildAnalyticsSeries(indexed, currentStart, now)
	resp.Performance = analyticsPerformance{TasksScanned: len(tasks), RunsScanned: projectRuns, SeriesPoints: len(resp.Series), BuildMS: time.Since(started).Milliseconds(), EvidenceCap: analyticsEvidenceLimit}
	return resp, nil
}

func scanAnalyticsRuns(w *workspace.Workspace) []analyticsRun {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return nil
	}
	out := make([]analyticsRun, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := w.RunDir(entry.Name())
		rec, err := procmon.ReadRecord(filepath.Join(dir, "proc.txt"))
		if err != nil || rec.RunID != entry.Name() || rec.Task == "" {
			continue
		}
		r := analyticsRun{id: rec.RunID, task: rec.Task, role: rec.Role, runtime: rec.Runtime, outcome: rec.Outcome, started: rec.Started}
		if r.started.IsZero() {
			r.started, _ = ulidTime(entry.Name())
		}
		for _, name := range []string{"outcome.md", "result.txt", "usage.txt"} {
			if fi, statErr := os.Stat(filepath.Join(dir, name)); statErr == nil && fi.ModTime().After(r.ended) {
				r.ended = fi.ModTime().UTC()
			}
		}
		readAnalyticsInvocation(dir, &r)
		readAnalyticsUsage(dir, &r)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].started.Before(out[j].started) })
	return out
}

func readAnalyticsInvocation(dir string, r *analyticsRun) {
	f, err := os.Open(filepath.Join(dir, "invocation.txt"))
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		k, v, ok := strings.Cut(s.Text(), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "role":
			r.role = strings.TrimSpace(v)
		case "runtime":
			r.runtime = strings.TrimSpace(v)
		case "model":
			r.model = strings.TrimSpace(v)
		}
	}
}

func readAnalyticsUsage(dir string, r *analyticsRun) {
	f, err := os.Open(filepath.Join(dir, "usage.txt"))
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		k, v, ok := strings.Cut(s.Text(), ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "input_tokens", "output_tokens":
			n, e := strconv.ParseInt(v, 10, 64)
			if e == nil {
				r.tokens += n
				r.usageKnown = true
			}
		case "cost_usd":
			n, e := strconv.ParseFloat(v, 64)
			if e == nil {
				r.cost = n
				r.costKnown = true
			}
		}
	}
}

func taskULIDTime(id string) time.Time { t, _ := ulidTime(id); return t.UTC() }
func generationStart(t *store.Task) time.Time {
	if t.Generation() == 0 {
		return taskULIDTime(t.ID)
	}
	return lastLogStamp(t, "reopened by")
}
func lastLogStamp(t *store.Task, prefix string) time.Time {
	s, ok := t.Doc.Section("Log")
	if !ok {
		return time.Time{}
	}
	var result time.Time
	for _, line := range strings.Split(s.Content, "\n") {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if len(fields) < 2 || !strings.HasPrefix(strings.Join(fields[1:], " "), prefix) {
			continue
		}
		if ts, e := time.Parse(time.RFC3339, fields[0]); e == nil {
			result = ts.UTC()
		}
	}
	return result
}
func taskSize(t *store.Task) string {
	est, ok := t.Estimate()
	if !ok {
		return "unknown"
	}
	x := est.Expected()
	if x <= 3 {
		return "small"
	}
	if x <= 8 {
		return "medium"
	}
	return "large"
}

type metricSamples struct {
	values      []float64
	tasks, runs []string
	eligible    int
}

func (s metricSamples) measure(key, label, unit, provenance, caveat string) analyticsMeasure {
	m := analyticsMeasure{Key: key, Label: label, Unit: unit, Samples: len(s.values), Eligible: s.eligible, State: "unknown", Provenance: provenance, Caveat: caveat, Evidence: boundedEvidence(s.tasks, s.runs)}
	if s.eligible > 0 {
		m.Coverage = float64(len(s.values)) / float64(s.eligible)
	}
	if len(s.values) > 0 {
		var total float64
		for _, v := range s.values {
			total += v
		}
		value := total / float64(len(s.values))
		m.Value = &value
		if len(s.values) == s.eligible {
			m.State = "complete"
		} else {
			m.State = "partial"
		}
	}
	return m
}
func boundedEvidence(tasks, runs []string) analyticsEvidence {
	sort.Strings(tasks)
	sort.Strings(runs)
	tasks = dedupeStrings(tasks)
	runs = dedupeStrings(runs)
	e := analyticsEvidence{}
	if len(tasks) > analyticsEvidenceLimit {
		e.Truncated = true
		tasks = tasks[:analyticsEvidenceLimit]
	}
	if len(runs) > analyticsEvidenceLimit {
		e.Truncated = true
		runs = runs[:analyticsEvidenceLimit]
	}
	e.Tasks = tasks
	e.Runs = runs
	return e
}
func dedupeStrings(in []string) []string {
	out := in[:0]
	for _, v := range in {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

func buildAnalyticsMetrics(tasks []analyticsTask, previousStart, currentStart, now time.Time) []analyticsMetric {
	type builder func(time.Time, time.Time) metricSamples
	builds := []struct {
		key, label, unit, prov, caveat string
		fn                             builder
	}{
		{"throughput", "Accepted throughput", "tasks", "task Log current-generation completed stamp", "", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				if inWindow(t.completed, a, b) {
					s.eligible++
					s.values = append(s.values, 1)
					s.tasks = append(s.tasks, t.task.ID)
				}
			}
			return s
		}},
		{"queue_time", "Queue time", "hours", "task ULID/reopen stamp → first current-generation run", "Generation zero uses task creation because no separate ready stamp exists.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				if !inWindow(t.created, a, b) {
					continue
				}
				s.eligible++
				if len(t.runs) > 0 && !t.generation.IsZero() {
					s.values = append(s.values, t.runs[0].started.Sub(t.generation).Hours())
					s.tasks = append(s.tasks, t.task.ID)
					s.runs = append(s.runs, t.runs[0].id)
				}
			}
			return s
		}},
		{"execution_time", "Execution time", "hours", "proc started → terminal evidence mtime", "Durations are wall-clock run spans, not billed provider time.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				for _, r := range t.runs {
					if !inWindow(r.started, a, b) {
						continue
					}
					s.eligible++
					if !r.ended.IsZero() && !r.ended.Before(r.started) {
						s.values = append(s.values, r.ended.Sub(r.started).Hours())
						s.tasks = append(s.tasks, t.task.ID)
						s.runs = append(s.runs, r.id)
					}
				}
			}
			return s
		}},
		{"current_tree_acceptance", "Current-tree acceptance", "percent", "done task + structured clean exact-tree verification", "Verification evidence has no observation timestamp; this is a current-state cohort, not historical success.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				if !inWindow(t.completed, a, b) {
					continue
				}
				s.eligible++
				if t.verificationCurrent {
					s.values = append(s.values, 100)
					s.tasks = append(s.tasks, t.task.ID)
				} else {
					s.values = append(s.values, 0)
				}
			}
			return s
		}},
		{"first_pass_review", "First-pass review", "percent", "independent review transaction correction_turns", "Missing review transactions are excluded, not failed.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				if !inWindow(t.completed, a, b) {
					continue
				}
				s.eligible++
				if t.reviewKnown {
					v := float64(0)
					if t.corrections == 0 {
						v = 100
					}
					s.values = append(s.values, v)
					s.tasks = append(s.tasks, t.task.ID)
				}
			}
			return s
		}},
		{"retry_rate", "Retry rate", "retries/task", "current-generation proc records", "Retries count additional coding runs in the current generation.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				if !inWindow(t.completed, a, b) {
					continue
				}
				s.eligible++
				if len(t.runs) > 0 {
					s.values = append(s.values, float64(max(0, len(t.runs)-1)))
					s.tasks = append(s.tasks, t.task.ID)
					for _, r := range t.runs {
						s.runs = append(s.runs, r.id)
					}
				}
			}
			return s
		}},
		{"review_corrections", "Review corrections", "corrections/task", "independent review transaction correction_turns", "Missing review transactions are excluded rather than counted as zero corrections.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				if !inWindow(t.completed, a, b) {
					continue
				}
				s.eligible++
				if t.reviewKnown {
					s.values = append(s.values, float64(t.corrections))
					s.tasks = append(s.tasks, t.task.ID)
				}
			}
			return s
		}},
		{"reopen_rate", "Reopen/regression rate", "percent", "task generation + reopened Log stamp", "A reopen is a regression signal, not proof of a model-caused defect.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				s.eligible++
				if t.generationNumber > 0 && inWindow(t.generation, a, b) {
					s.values = append(s.values, 100)
					s.tasks = append(s.tasks, t.task.ID)
				} else {
					s.values = append(s.values, 0)
				}
			}
			return s
		}},
		{"tokens", "Tokens per run", "tokens", "provider usage.txt input_tokens + output_tokens", "Provider-reported usage; missing usage is excluded.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				for _, r := range t.runs {
					if !inWindow(r.started, a, b) {
						continue
					}
					s.eligible++
					if r.usageKnown {
						s.values = append(s.values, float64(r.tokens))
						s.tasks = append(s.tasks, t.task.ID)
						s.runs = append(s.runs, r.id)
					}
				}
			}
			return s
		}},
		{"cost", "Cost per run", "USD", "provider usage.txt cost_usd presence", "Provider-reported, not billing; absent cost is unknown rather than zero.", func(a, b time.Time) metricSamples {
			s := metricSamples{}
			for _, t := range tasks {
				for _, r := range t.runs {
					if !inWindow(r.started, a, b) {
						continue
					}
					s.eligible++
					if r.costKnown {
						s.values = append(s.values, r.cost)
						s.tasks = append(s.tasks, t.task.ID)
						s.runs = append(s.runs, r.id)
					}
				}
			}
			return s
		}},
	}
	out := make([]analyticsMetric, 0, len(builds)+1)
	for _, b := range builds {
		cur, prev := b.fn(currentStart, now).measure(b.key, b.label, b.unit, b.prov, b.caveat), b.fn(previousStart, currentStart).measure(b.key, b.label, b.unit, b.prov, b.caveat)
		if b.key == "throughput" {
			if cur.Samples > 0 {
				value := float64(cur.Samples)
				cur.Value = &value
			}
			if prev.Samples > 0 {
				value := float64(prev.Samples)
				prev.Value = &value
			}
		}
		if b.key == "cost" {
			if cur.Value != nil {
				cur.State = "advisory"
			}
			if prev.Value != nil {
				prev.State = "advisory"
			}
		}
		m := analyticsMetric{Key: b.key, Label: b.label, Current: cur, Previous: prev, Trend: "not-comparable"}
		if cur.Value != nil && prev.Value != nil && cur.Samples >= 3 && prev.Samples >= 3 {
			change := *cur.Value - *prev.Value
			m.Change = &change
			if change > 0 {
				m.Trend = "up"
			} else if change < 0 {
				m.Trend = "down"
			} else {
				m.Trend = "flat"
			}
		}
		out = append(out, m)
	}
	unknown := analyticsMeasure{Key: "ready_to_merged", Label: "Ready to merged", Unit: "hours", State: "unknown", Provenance: "no durable ready transition timestamp", Caveat: "Not computed from creation or first-run proxies; add a typed ready event before comparing this duration.", Evidence: analyticsEvidence{Tasks: []string{}, Runs: []string{}}}
	out = append(out, analyticsMetric{Key: "ready_to_merged", Label: "Ready to merged", Current: unknown, Previous: unknown, Trend: "not-comparable"})
	for _, unsupported := range []struct{ key, label, provenance, caveat string }{
		{"first_pass_verification", "First-pass verification", "verification evidence stores final exact-tree proof, not every attempt", "Attempt history must be durably recorded before a first-pass rate is defensible."},
		{"first_pass_landing", "First-pass landing", "current PR identity does not preserve every landing attempt", "Superseded PR history is not complete enough to classify first-pass landing."},
	} {
		measure := analyticsMeasure{Key: unsupported.key, Label: unsupported.label, Unit: "percent", State: "unknown", Provenance: unsupported.provenance, Caveat: unsupported.caveat, Evidence: analyticsEvidence{Tasks: []string{}, Runs: []string{}}}
		out = append(out, analyticsMetric{Key: unsupported.key, Label: unsupported.label, Current: measure, Previous: measure, Trend: "not-comparable"})
	}
	return out
}

func buildAnalyticsBreakdowns(tasks []analyticsTask, previousStart, currentStart, now time.Time) []analyticsBreakdown {
	type cohort struct {
		dimension, key, size string
		tasks                []analyticsTask
		runs                 map[string]map[string]bool
	}
	cohorts := map[string]*cohort{}
	for _, t := range tasks {
		add := func(k [3]string, runID string) {
			if k[1] == "" {
				return
			}
			id := k[0] + "\x00" + k[1] + "\x00" + k[2]
			if cohorts[id] == nil {
				cohorts[id] = &cohort{dimension: k[0], key: k[1], size: k[2], runs: map[string]map[string]bool{}}
			}
			cohorts[id].tasks = append(cohorts[id].tasks, t)
			if runID != "" {
				if cohorts[id].runs[t.task.ID] == nil {
					cohorts[id].runs[t.task.ID] = map[string]bool{}
				}
				cohorts[id].runs[t.task.ID][runID] = true
			}
		}
		add([3]string{"project", t.task.Project, t.size}, "")
		add([3]string{"task_size", t.size, t.size}, "")
		if t.verificationContract != "" {
			add([3]string{"verification_contract", t.verificationContract, t.size}, "")
		}
		for _, r := range t.runs {
			add([3]string{"route", strings.Trim(strings.Join([]string{r.role, r.runtime, r.model}, "/"), "/"), t.size}, r.id)
			add([3]string{"role", r.role, t.size}, r.id)
			add([3]string{"runtime", r.runtime, t.size}, r.id)
			add([3]string{"model", r.model, t.size}, r.id)
			add([3]string{"failure_taxonomy", analyticsOutcomeClass(r.outcome), t.size}, r.id)
		}
	}
	out := make([]analyticsBreakdown, 0, len(cohorts))
	for _, c := range cohorts {
		sample := func(a, b time.Time) metricSamples {
			s := metricSamples{}
			seen := map[string]bool{}
			for _, t := range c.tasks {
				if seen[t.task.ID] || !inWindow(t.completed, a, b) {
					continue
				}
				seen[t.task.ID] = true
				s.eligible++
				s.values = append(s.values, 1)
				s.tasks = append(s.tasks, t.task.ID)
				for _, r := range t.runs {
					if selected := c.runs[t.task.ID]; len(selected) == 0 || selected[r.id] {
						s.runs = append(s.runs, r.id)
					}
				}
			}
			return s
		}
		cs, ps := sample(currentStart, now), sample(previousStart, currentStart)
		cm, pm := cs.measure("throughput", "Accepted throughput", "tasks", "bounded cohort index", ""), ps.measure("throughput", "Accepted throughput", "tasks", "bounded cohort index", "")
		comparable := cm.Samples >= 3 && pm.Samples >= 3 && c.size != "unknown"
		caveat := ""
		if !comparable {
			caveat = "Descriptive only: both windows need at least 3 tasks with a known comparable size band."
		}
		out = append(out, analyticsBreakdown{Dimension: c.dimension, Key: c.key, SizeBand: c.size, Current: cm, Previous: pm, Comparable: comparable, Caveat: caveat, Evidence: boundedEvidence(append(cs.tasks, ps.tasks...), append(cs.runs, ps.runs...))})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dimension != out[j].Dimension {
			return out[i].Dimension < out[j].Dimension
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func analyticsOutcomeClass(outcome string) string {
	value := strings.ToLower(strings.TrimSpace(outcome))
	switch {
	case successfulRunOutcome(value):
		return "success"
	case strings.Contains(value, "timeout"):
		return "timeout"
	case strings.Contains(value, "kill") || strings.Contains(value, "cancel"):
		return "cancelled"
	case value == "":
		return "unknown"
	default:
		return "failed"
	}
}

func buildAnalyticsSeries(tasks []analyticsTask, start, end time.Time) []analyticsDay {
	by := map[string]*analyticsDay{}
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		by[key] = &analyticsDay{Day: key}
	}
	for _, t := range tasks {
		if inWindow(t.completed, start, end) {
			key := t.completed.Format("2006-01-02")
			if by[key] == nil {
				by[key] = &analyticsDay{Day: key}
			}
			by[key].Completed++
			by[key].Evidence.Tasks = append(by[key].Evidence.Tasks, t.task.ID)
		}
		for _, r := range t.runs {
			if inWindow(r.started, start, end) {
				key := r.started.Format("2006-01-02")
				p := by[key]
				if p == nil {
					p = &analyticsDay{Day: key}
					by[key] = p
				}
				p.Runs++
				if r.usageKnown {
					p.Tokens += r.tokens
				}
				p.Evidence.Tasks = append(p.Evidence.Tasks, t.task.ID)
				p.Evidence.Runs = append(p.Evidence.Runs, r.id)
			}
		}
	}
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]analyticsDay, 0, len(keys))
	for _, k := range keys {
		by[k].Evidence = boundedEvidence(by[k].Evidence.Tasks, by[k].Evidence.Runs)
		out = append(out, *by[k])
	}
	return out
}
func inWindow(t, start, end time.Time) bool { return !t.IsZero() && !t.Before(start) && t.Before(end) }
