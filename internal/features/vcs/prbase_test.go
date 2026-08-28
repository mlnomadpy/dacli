package vcs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func stubRepositoryDefaultBranch(t *testing.T, branch string, err error) {
	t.Helper()
	original := queryRepositoryDefaultBranch
	queryRepositoryDefaultBranch = func(string) (string, error) { return branch, err }
	t.Cleanup(func() { queryRepositoryDefaultBranch = original })
}

func TestPRUsesLinkedRepositoryDefaultMaster(t *testing.T) {
	dir, _, task := prIntegrateEnv(t)
	stubRepositoryDefaultBranch(t, "master", nil)
	calls := stubGH(t, func(_ string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "view" {
			return "", errNoPR
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/30", nil
		}
		return "", nil
	})
	ctx, out := prCtx(dir)
	if err := cmdPR(ctx, []string{"--task", task.ID}); err != nil {
		t.Fatal(err)
	}
	create := findCreate(*calls)
	if create == nil || !strings.Contains(strings.Join(create, " "), "--base master") {
		t.Fatalf("PR creation did not use the linked repository default master: %v", *calls)
	}
	if !strings.Contains(out.String(), "landing base: master (repository default)") {
		t.Fatalf("real execution did not report resolved base and source:\n%s", out.String())
	}
}

func TestPRDryRunReportsConfiguredAndExplicitBasePrecedence(t *testing.T) {
	dir, w, task := prIntegrateEnv(t)
	project, err := store.LoadProject(w, task.Project)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConfigureProjectLanding(project, model.LandingPolicy{Mode: model.LandingPR, Base: "dev"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	stubRepositoryDefaultBranch(t, "", fmt.Errorf("repository lookup must not run when policy supplies a base"))
	calls := stubGH(t, func(_ string, args ...string) (string, error) {
		return "", fmt.Errorf("dry-run must not call GitHub: %v", args)
	})

	ctx, configured := prCtx(dir)
	if err := cmdPR(ctx, []string{"--task", task.ID, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(configured.String(), "landing base: dev (project policy)") {
		t.Fatalf("configured dry-run output = %q", configured.String())
	}

	ctx, explicit := prCtx(dir)
	if err := cmdPR(ctx, []string{"--task", task.ID, "--base", "release", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(explicit.String(), "landing base: release (explicit --base)") {
		t.Fatalf("explicit dry-run output = %q", explicit.String())
	}
	if len(*calls) != 0 {
		t.Fatalf("dry-run performed outward GitHub calls: %v", *calls)
	}
}

func TestPRDryRunReportsStaleReusedBodyWithoutMutation(t *testing.T) {
	w, task := prEnv(t)
	task.Doc.Front.SetBlock("github", "  issue: 841\n  repo: acme/widgets")
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-child", task.Project, model.NoteFinding, "stale verdict disclosure",
		store.NoteOpts{About: task.ID, Severity: "major", Body: "internal verification detail"}); err != nil {
		t.Fatal(err)
	}
	project, err := store.LoadProject(w, task.Project)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConfigureProjectLanding(project, model.LandingPolicy{Mode: model.LandingPR, Base: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(prBody(w, task, true), "Implements dacli task 001-enrich-pr.\n", "Implements dacli task 001-enrich-pr.\n\nFixes #841\n", 1)
	calls := stubGH(t, func(_ string, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "--json url,state"):
			return "OPEN https://github.com/acme/widgets/pull/9", nil
		case strings.Contains(joined, "--json body"):
			return stale, nil
		default:
			return "", fmt.Errorf("dry-run attempted unexpected GitHub operation: %v", args)
		}
	})
	ctx, out := prCtx(w.Root)
	if err := cmdPR(ctx, []string{"--task", task.ID, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "requires removal of a stale closing keyword") ||
		!strings.Contains(out.String(), "stale generated trust-verdict section") ||
		!strings.Contains(out.String(), "dry-run made no GitHub mutation") {
		t.Fatalf("dry-run omitted body reconciliation plan: %s", out.String())
	}
	for _, call := range *calls {
		if len(call) >= 2 && call[0] == "pr" && (call[1] == "edit" || call[1] == "create") {
			t.Fatalf("dry-run mutated GitHub: %v", call)
		}
	}
}

func TestPRFailsClosedBeforeCreateWhenDefaultBranchUnknown(t *testing.T) {
	dir, _, task := prIntegrateEnv(t)
	stubRepositoryDefaultBranch(t, "", fmt.Errorf("GitHub unavailable"))
	calls := stubGH(t, func(_ string, args ...string) (string, error) {
		return "", fmt.Errorf("PR creation must be unreachable: %v", args)
	})
	ctx, _ := prCtx(dir)
	err := cmdPR(ctx, []string{"--task", task.ID})
	if err == nil || !strings.Contains(err.Error(), "--base") || !strings.Contains(err.Error(), "project show") {
		t.Fatalf("unknown default branch did not fail closed with recovery action: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("GitHub PR mutation happened before base resolution: %v", *calls)
	}
}
