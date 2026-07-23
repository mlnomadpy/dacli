package store

import (
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// TestTitleSimilarityCatchesRealNearDuplicates reproduces the two pairs of
// near-duplicate titles reported in dacli task 116 (a real review-phase
// churn incident) and asserts they score at or above the dedup threshold.
func TestTitleSimilarityCatchesRealNearDuplicates(t *testing.T) {
	cases := []struct{ a, b string }{
		{
			"charge idle-cycle review spawns to the token window",
			"charge idle-cycle reviewer tokens to the --window-tokens budget",
		},
		{
			"bound the three remaining unbounded git/gh subprocesses",
			"give the last three unbounded git and gh subprocesses deadlines",
		},
	}
	for _, c := range cases {
		got := TitleSimilarity(c.a, c.b)
		if got < DuplicateTitleThreshold {
			t.Errorf("TitleSimilarity(%q, %q) = %.2f, want >= %.2f", c.a, c.b, got, DuplicateTitleThreshold)
		}
	}
}

// TestTitleSimilarityLeavesUnrelatedTitlesAlone guards against the dedup
// guard being so aggressive it blocks ordinary, unrelated backlog work.
func TestTitleSimilarityLeavesUnrelatedTitlesAlone(t *testing.T) {
	cases := []struct{ a, b string }{
		{"fix flaky retry timer in the spawn watchdog", "document the SPM glossary term for slack"},
		{"add color to the dashboard header", "migrate goreleaser brews to homebrew_casks"},
	}
	for _, c := range cases {
		if got := TitleSimilarity(c.a, c.b); got >= DuplicateTitleThreshold {
			t.Errorf("TitleSimilarity(%q, %q) = %.2f, want < %.2f", c.a, c.b, got, DuplicateTitleThreshold)
		}
	}
}

func TestFindNearDuplicateTaskMatchesOpenBacklog(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	existing, err := CreateTask(w, "a-root", "core", "charge idle-cycle review spawns to the token window", TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}

	dup, score, err := FindNearDuplicateTask(w, "core", "charge idle-cycle reviewer tokens to the --window-tokens budget")
	if err != nil {
		t.Fatalf("FindNearDuplicateTask: %v", err)
	}
	if dup == nil || dup.ID != existing.ID {
		t.Fatalf("FindNearDuplicateTask = %v, want match on %s", dup, existing.ID)
	}
	if score < DuplicateTitleThreshold {
		t.Errorf("score = %.2f, want >= %.2f", score, DuplicateTitleThreshold)
	}

	if dup, _, err := FindNearDuplicateTask(w, "core", "rewrite the onboarding walkthrough for new agents"); err != nil {
		t.Fatalf("FindNearDuplicateTask: %v", err)
	} else if dup != nil {
		t.Errorf("unrelated title matched %v", dup)
	}
}

// TestFindNearDuplicateTaskIgnoresShortLookAlikes guards against the
// worktree-parallelism fixture shape "Feature A" / "Feature B": two short,
// deliberately distinct sibling titles that share only one generic word and
// must never collide, even though their Jaccard ratio alone clears the
// threshold.
func TestFindNearDuplicateTaskIgnoresShortLookAlikes(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	if _, err := CreateTask(w, "a-root", "core", "Feature A", TaskOpts{}); err != nil {
		t.Fatalf("task: %v", err)
	}

	dup, score, err := FindNearDuplicateTask(w, "core", "Feature B")
	if err != nil {
		t.Fatalf("FindNearDuplicateTask: %v", err)
	}
	if dup != nil {
		t.Errorf("FindNearDuplicateTask(\"Feature B\") = %v (score %.2f), want no match", dup, score)
	}
}

// TestFindNearDuplicateTaskIgnoresDoneTasks confirms the guard only checks
// the live backlog (open/active) — a title matching already-shipped work is
// not blocked, since that is arguably legitimate follow-up, not churn.
func TestFindNearDuplicateTaskIgnoresDoneTasks(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	done, err := CreateTask(w, "a-root", "core", "charge idle-cycle review spawns to the token window", TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if err := MoveTask(w, done, model.StatusDone); err != nil {
		t.Fatalf("move: %v", err)
	}

	dup, _, err := FindNearDuplicateTask(w, "core", "charge idle-cycle reviewer tokens to the --window-tokens budget")
	if err != nil {
		t.Fatalf("FindNearDuplicateTask: %v", err)
	}
	if dup != nil {
		t.Errorf("matched a done task %v, want no match", dup)
	}
}
