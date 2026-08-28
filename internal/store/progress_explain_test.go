package store

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func explainFixture(t *testing.T) (*workspace.Workspace, *Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "explain")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = w.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	first, err := CreateTask(w, "a-root", "core", "foundation", TaskOpts{Accept: []string{"done"}, Estimate: "1,2,3", Claims: []string{"internal/store"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTask(w, "a-root", "core", "dependent", TaskOpts{Accept: []string{"done"}, Estimate: "1,2,3", DependsOn: []string{first.ID}}); err != nil {
		t.Fatal(err)
	}
	for _, role := range []team.Role{
		{Name: "builder", Kind: "implementer", Grant: "rw", Runtime: "codex-rw", Scope: []string{"internal/**"}, Profile: team.ModelProfile{ID: "small", CostTier: 1}},
		{Name: "review-only", Kind: "reviewer", Grant: "ro", Runtime: "codex", Scope: []string{"**"}, Profile: team.ModelProfile{ID: "large", CostTier: 3}},
	} {
		if err := CreateRole(w, "a-root", role); err != nil {
			t.Fatal(err)
		}
	}
	return w, first
}

func TestBuildProgressExplainRetainsRejectedRolesAndCanonicalLanding(t *testing.T) {
	w, first := explainFixture(t)
	at := time.Unix(100, 0).UTC()
	delivery := DeliveryProjection{Schema: DeliverySchemaVersion, Version: 1, Project: "core", ObservedAt: at, Reconciled: true, Findings: []DeliveryFinding{{
		ID: "canonical-pr:" + first.ID, Classification: "canonical-pr", ObjectID: first.ID, Source: "github", ObservedAt: at,
		Confidence: DeliveryKnown, Detail: "pr=#9", NextAction: "wait for required checks", RelatedRefs: []DeliveryRef{{Kind: "pull_request", ID: "#9"}},
	}}}
	p, err := BuildProgressExplain(w, "core", delivery, at)
	if err != nil {
		t.Fatal(err)
	}
	if p.Schema != ProgressExplainSchema || len(p.Tasks) != 2 {
		t.Fatalf("projection envelope/tasks = %+v", p)
	}
	var got *TaskExplain
	for i := range p.Tasks {
		if p.Tasks[i].ID.Value == first.ID {
			got = &p.Tasks[i]
		}
	}
	if got == nil {
		t.Fatal("foundation task absent")
	}
	if got.Landing.Value.Classification != "canonical-pr" || got.Landing.Source != "github" || got.NextAction.Value == "" {
		t.Fatalf("canonical landing/next action lost: %+v", got)
	}
	if got.Rank.Value != 1 || got.Slack.Value == nil || len(got.Claims.Value) != 1 {
		t.Fatalf("rank/slack/claims incomplete: %+v", got)
	}
	rejected := got.RoleRouting.Value.Candidate("review-only")
	if rejected == nil || rejected.Eligible || len(rejected.Exclusions) == 0 || !strings.Contains(strings.Join(rejected.Exclusions, " "), "kind") {
		t.Fatalf("rejected reviewer role was dropped or unexplained: %+v", got.RoleRouting.Value)
	}
	if selected := got.RoleRouting.Value.Selected.Role; selected != "builder" {
		t.Fatalf("selected role = %q, want builder", selected)
	}
	for _, fact := range []struct {
		name   string
		source string
		at     time.Time
	}{
		{"id", got.ID.Source, got.ID.ObservedAt}, {"title", got.Title.Source, got.Title.ObservedAt}, {"status", got.Status.Source, got.Status.ObservedAt},
		{"rank", got.Rank.Source, got.Rank.ObservedAt}, {"slack", got.Slack.Source, got.Slack.ObservedAt}, {"blockers", got.Blockers.Source, got.Blockers.ObservedAt},
		{"claims", got.Claims.Source, got.Claims.ObservedAt}, {"parent", got.Parent.Source, got.Parent.ObservedAt}, {"aggregate", got.Aggregate.Source, got.Aggregate.ObservedAt},
		{"landing", got.Landing.Source, got.Landing.ObservedAt}, {"routing", got.RoleRouting.Source, got.RoleRouting.ObservedAt}, {"next", got.NextAction.Source, got.NextAction.ObservedAt},
	} {
		if fact.source == "" || fact.at.IsZero() {
			t.Errorf("%s is missing source/observed time", fact.name)
		}
	}
}

func TestExplainCacheMarksEveryFallbackFactStaleAndIsBounded(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	fact := func(value string) Observed[string] { return observed(value, "fixture", at, false) }
	projection := ProgressExplain{Schema: ProgressExplainSchema, Version: 1, Project: "core", ObservedAt: at, Tasks: []TaskExplain{{
		ID: fact("t-1"), Title: fact("title"), Status: fact("open"), Rank: observed(1, "fixture", at, false), Slack: observed[*float64](nil, "fixture", at, false),
		Blockers: observed([]string{}, "fixture", at, false), Claims: observed([]string{}, "fixture", at, false), Parent: fact(""), Aggregate: observed(AggregateExplain{}, "fixture", at, false),
		Landing: observed(LandingExplain{}, "fixture", at, false), RoleRouting: observed(team.Explanation{Candidates: []team.CandidateExplanation{{Role: "rejected", Eligible: false, Exclusions: []string{"wrong kind"}}}}, "fixture", at, false), NextAction: fact("assign"),
	}}, Workers: []WorkerExplain{{RunID: fact("run"), AgentID: fact("agent"), TaskID: fact("t-1"), Role: fact("builder"), Runtime: fact("codex"), State: fact("live"), Claims: observed([]string{"internal"}, "fixture", at, false), NextAction: fact("observe")}}}
	cache := NewExplainCache(1, time.Second, time.Minute)
	if _, err := cache.Get("core", at, func() (ProgressExplain, error) { return projection, nil }); err != nil {
		t.Fatal(err)
	}
	got, err := cache.Get("core", at.Add(2*time.Second), func() (ProgressExplain, error) { return ProgressExplain{}, errors.New("fixture outage") })
	if err != nil {
		t.Fatal(err)
	}
	if !got.Stale || got.CacheState != "stale-fallback" || !strings.Contains(got.Warning, "fixture outage") {
		t.Fatalf("fallback freshness = stale:%t cache:%s warning:%q", got.Stale, got.CacheState, got.Warning)
	}
	task, worker := got.Tasks[0], got.Workers[0]
	if !task.ID.Stale || !task.Status.Stale || !task.RoleRouting.Stale || !task.Landing.Stale || !task.NextAction.Stale || !worker.Role.Stale || !worker.State.Stale || !worker.NextAction.Stale {
		t.Fatalf("stale cache escaped as current: task=%+v worker=%+v", task, worker)
	}
	if candidate := task.RoleRouting.Value.Candidate("rejected"); candidate == nil || candidate.Eligible {
		t.Fatalf("stale fallback lost rejected role evidence: %+v", task.RoleRouting.Value)
	}
	if _, err := cache.Get("core", at.Add(2*time.Minute), func() (ProgressExplain, error) { return ProgressExplain{}, errors.New("still unavailable") }); err == nil {
		t.Fatal("cache served evidence beyond its maximum stale age")
	}
	if _, err := cache.Get("other", at.Add(3*time.Second), func() (ProgressExplain, error) { p := projection; p.Project = "other"; return p, nil }); err != nil {
		t.Fatal(err)
	}
	if len(cache.entries) != 1 || cache.entries["other"].projection.Project != "other" {
		t.Fatalf("bounded cache did not evict LRU entry: %+v", cache.entries)
	}
}
