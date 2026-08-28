package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestSemanticBackendFailureIsObservableButRemainsOptional(t *testing.T) {
	t.Setenv("DACLI_TEST_SECRET", "SEMANTIC_TOKEN")
	cmd := `printf 'setup line\nSEMANTIC_TOKEN at /private/operator/model\nsemantic backend unavailable decisively\n' >&2; exit 29`
	if score, ok := runSemanticCmd(cmd, "safe title a", "safe title b"); ok || score != 0 {
		t.Fatalf("optional wrapper = (%v, %v), want no opinion", score, ok)
	}
	score, ok, err := runSemanticCmdChecked(cmd, "safe title a", "safe title b")
	if ok || score != 0 || err == nil {
		t.Fatalf("checked wrapper = (%v, %v, %v), want typed no-opinion failure", score, ok, err)
	}
	d, present := commandresult.AsDiagnostic(err)
	if !present || d.Operation != "score semantic similarity" || d.Executable != "sh" || d.ExitCode == nil || *d.ExitCode != 29 {
		t.Fatalf("semantic diagnostic = %#v, present=%v, error=%v", d, present, err)
	}
	if strings.Contains(d.StderrTail, "SEMANTIC_TOKEN") || strings.Contains(d.StderrTail, "/private/operator") {
		t.Fatalf("semantic diagnostic leaked protected material: %#v", d)
	}
	if !strings.Contains(d.StderrTail, "<redacted>") || !strings.Contains(d.StderrTail, "semantic backend unavailable decisively") {
		t.Fatalf("semantic diagnostic lost actionable redacted detail: %#v", d)
	}
	var exit interface{ ExitCode() int }
	if !errors.As(err, &exit) || exit.ExitCode() != 29 {
		t.Fatalf("semantic diagnostic lost process cause: %T %v", err, err)
	}
}

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

func TestFindNearDuplicateTaskContentCatchesInFlightGeneratedRefDuplicate(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	existing, err := CreateTask(w, "a-root", "core", "Use task ULIDs for generated mutating commands", TaskOpts{
		Context: "Generated execution prompts use sequence 001 in both alpha and beta projects, so a mutation can target the wrong task in internal/features/execution/execution.go.",
		Accept:  []string{"Generated mutating commands contain the stable task ULID", "A cross-project alpha/001 and beta/001 mutation regression passes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := MoveTask(w, existing, model.StatusActive); err != nil {
		t.Fatal(err)
	}

	candidate := TaskSimilarityInput{
		Title:      "Fix generated worker prompts using ambiguous numeric refs for mutations",
		Problem:    "In internal/features/execution/execution.go alpha task 001 and beta task 001 can make a generated mutation target the wrong task.",
		Acceptance: []string{"Generated mutation commands use the stable task ULID", "The alpha/001 and beta/001 cross-project mutation regression passes"},
	}
	dup, _, err := FindNearDuplicateTaskContent(w, "core", candidate)
	if err != nil {
		t.Fatal(err)
	}
	if dup == nil || dup.ID != existing.ID {
		t.Fatalf("duplicate = %#v, want active task %s", dup, existing.ID)
	}
}

func TestFindNearDuplicateTaskContentLeavesDistinctGeneratedRefDefect(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTask(w, "a-root", "core", "Use task ULIDs for generated mutating commands", TaskOpts{
		Context: "Generated execution prompts can target the wrong cross-project task in internal/features/execution/execution.go.",
		Accept:  []string{"Generated mutating commands contain the stable task ULID"},
	}); err != nil {
		t.Fatal(err)
	}

	candidate := TaskSimilarityInput{
		Title:      "Quote generated task refs containing shell metacharacters",
		Problem:    "A generated command in internal/features/execution/execution.go is split by the shell when a human task ref contains whitespace.",
		Acceptance: []string{"Generated commands shell-quote whitespace and metacharacters", "A task ref containing a single quote reaches the intended command unchanged"},
	}
	dup, _, err := FindNearDuplicateTaskContent(w, "core", candidate)
	if err != nil {
		t.Fatal(err)
	}
	if dup != nil {
		t.Fatalf("distinct generated-reference defect matched %#v", dup)
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

// paraphrasePairA and B mean the same thing — "keep the watchdog from killing
// live agents" — but share no content word after stopword-stripping and
// stemming, so no token-overlap score can ever see them as the same work. They
// are the fixture for dacli task 249's paraphrase-detection gap.
const (
	paraphrasePairA = "stop the watchdog from reaping healthy agents"
	paraphrasePairB = "prevent the supervisor killing live workers"
)

// TestFindNearDuplicateTaskCatchesParaphraseViaSemanticBackend is task 249's
// acceptance: two tasks with the same meaning and NO shared words are detected
// once a semantic backend is installed. Before the fix the near-duplicate scan
// skipped any pair below the shared-token floor outright, so no backend could
// ever rescue a zero-word-overlap paraphrase.
func TestFindNearDuplicateTaskCatchesParaphraseViaSemanticBackend(t *testing.T) {
	// Guard the premise: these titles really do share no words, so a match can
	// only come from the semantic backend, never the lexical score.
	if lex := TitleSimilarity(paraphrasePairA, paraphrasePairB); lex != 0 {
		t.Fatalf("premise broken: TitleSimilarity(%q,%q)=%.2f, want 0 (no shared words)", paraphrasePairA, paraphrasePairB, lex)
	}

	// Install a stub backend that recognizes just this one paraphrase pair.
	prev := SemanticSimilarity
	t.Cleanup(func() { SemanticSimilarity = prev })
	SemanticSimilarity = func(a, b string) (float64, bool) {
		if (a == paraphrasePairA && b == paraphrasePairB) || (a == paraphrasePairB && b == paraphrasePairA) {
			return 0.95, true
		}
		return 0, false
	}

	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	existing, err := CreateTask(w, "a-root", "core", paraphrasePairA, TaskOpts{})
	if err != nil {
		t.Fatalf("task: %v", err)
	}

	dup, score, err := FindNearDuplicateTask(w, "core", paraphrasePairB)
	if err != nil {
		t.Fatalf("FindNearDuplicateTask: %v", err)
	}
	if dup == nil || dup.ID != existing.ID {
		t.Fatalf("FindNearDuplicateTask(%q) = %v, want match on %s (semantic backend)", paraphrasePairB, dup, existing.ID)
	}
	if score < DuplicateTitleThreshold {
		t.Errorf("score = %.2f, want >= %.2f", score, DuplicateTitleThreshold)
	}
}

// TestFindNearDuplicateTaskParaphraseInvisibleWithoutBackend is the other half
// of task 249's acceptance: the semantic backend is OPTIONAL. With no hook and
// no $DACLI_SEMANTIC_CMD, dacli stays purely lexical — the zero-dependency
// property holds — so the same paraphrase pair is (correctly, for the default
// build) NOT matched.
func TestFindNearDuplicateTaskParaphraseInvisibleWithoutBackend(t *testing.T) {
	prev := SemanticSimilarity
	t.Cleanup(func() { SemanticSimilarity = prev })
	SemanticSimilarity = nil
	t.Setenv("DACLI_SEMANTIC_CMD", "")
	if activeSemanticScorer() != nil {
		t.Fatal("activeSemanticScorer() != nil with no hook and no $DACLI_SEMANTIC_CMD — zero-dependency default broken")
	}

	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	if _, err := CreateTask(w, "a-root", "core", paraphrasePairA, TaskOpts{}); err != nil {
		t.Fatalf("task: %v", err)
	}
	dup, _, err := FindNearDuplicateTask(w, "core", paraphrasePairB)
	if err != nil {
		t.Fatalf("FindNearDuplicateTask: %v", err)
	}
	if dup != nil {
		t.Errorf("paraphrase matched %v with no backend, want no match (lexical-only default)", dup)
	}
}

// TestEnvSemanticBackend exercises the real external-command seam end to end:
// $DACLI_SEMANTIC_CMD is invoked with the two titles as positional args and its
// printed float becomes the score. This is what makes the optional backend
// actually usable, not just a test hook.
func TestEnvSemanticBackend(t *testing.T) {
	prev := SemanticSimilarity
	t.Cleanup(func() { SemanticSimilarity = prev })
	SemanticSimilarity = nil

	// A scorer that returns 0.9 only when both titles are non-empty, reading
	// them from "$1"/"$2" — proving the titles arrive as args, not spliced into
	// the shell string.
	t.Setenv("DACLI_SEMANTIC_CMD", `if [ -n "$1" ] && [ -n "$2" ]; then echo 0.9; else echo 0; fi`)
	sem := activeSemanticScorer()
	if sem == nil {
		t.Fatal("activeSemanticScorer() == nil with $DACLI_SEMANTIC_CMD set")
	}
	if score, ok := sem("a title", "another title"); !ok || score != 0.9 {
		t.Fatalf("sem() = %.2f, %v; want 0.90, true", score, ok)
	}

	// A backend that prints garbage is treated as "no opinion", never a score.
	t.Setenv("DACLI_SEMANTIC_CMD", "echo not-a-number")
	if score, ok := activeSemanticScorer()("x", "y"); ok {
		t.Errorf("garbage backend returned ok=true (score %.2f), want ok=false", score)
	}

	// So is one that exits non-zero.
	t.Setenv("DACLI_SEMANTIC_CMD", "exit 1")
	if _, ok := activeSemanticScorer()("x", "y"); ok {
		t.Errorf("failing backend returned ok=true, want ok=false")
	}

	// And an out-of-range score is rejected rather than blindly trusted.
	t.Setenv("DACLI_SEMANTIC_CMD", "echo 4.2")
	if _, ok := activeSemanticScorer()("x", "y"); ok {
		t.Errorf("out-of-range score returned ok=true, want ok=false")
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
