package acceptance

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func evidenceEnv(t *testing.T) (*clikit.Ctx, *workspace.Workspace, *agentid.Identity) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
		if out, err := gitx.Run(w.Root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(w.Root+"/.gitignore", []byte(".dacli/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", ".gitignore"}, {"commit", "-m", "seed"}} {
		if out, err := gitx.Run(w.Root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if _, err := store.CreateProject(w, agentid.RootID, "Core", "core", "", ""); err != nil {
		t.Fatal(err)
	}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	return ctx, w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
}

// taskLog returns the task's "## Log" section text.
func taskLog(t *testing.T, tk *store.Task) string {
	t.Helper()
	s, ok := tk.Doc.Section("Log")
	if !ok {
		return ""
	}
	return s.Content
}

func mkTask(t *testing.T, w *workspace.Workspace, title string) *store.Task {
	t.Helper()
	tk, err := store.CreateTask(w, agentid.RootID, "core", title, store.TaskOpts{Accept: []string{"it works"}})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

// A close must record WHAT certified it. Before this, an unverified close was
// indistinguishable from a verified one in the record — which is what made
// every `done` label an unverified assertion (dacli 184).
func TestCloseRecordsVerificationEvidence(t *testing.T) {
	ctx, w, root := evidenceEnv(t)

	unverified := mkTask(t, w, "closed with no check")
	if err := acceptOne(ctx, w, root, unverified, "", false, false, false, false, false, ""); err != nil {
		t.Fatal(err)
	}
	got, err := store.FindTask(w, unverified.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if body := taskLog(t, got); !strings.Contains(body, "WITHOUT verification") {
		t.Errorf("an unverified close must say so in the task log; got:\n%s", body)
	}

	verified := mkTask(t, w, "closed with a real check")
	if err := acceptOne(ctx, w, root, verified, "true", false, false, false, false, false, ""); err != nil {
		t.Fatal(err)
	}
	got2, err := store.FindTask(w, verified.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if body := taskLog(t, got2); !strings.Contains(body, "verified by") {
		t.Errorf("a verified close must record the command; got:\n%s", body)
	}
	records := store.VerificationEvidenceRecords(got2)
	if len(records) != 1 {
		t.Fatalf("structured verification records = %#v, want one", records)
	}
	ev := records[0]
	if ev.Command != "true" || ev.ExitCode != 0 || ev.DurationMS < 0 || ev.ArtifactHash == "" || ev.Verifier != root.ID {
		t.Fatalf("incomplete structured verification evidence: %#v", ev)
	}
}

func TestAcceptanceConsumesConfiguredExactHeadExternalEvidence(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	project, err := store.LoadProject(w, "core")
	if err != nil {
		t.Fatal(err)
	}
	project.Doc.Front.Set("github_repo", "owner/repo")
	project.Doc.Front.SetList("required_checks", []string{"ci"})
	project.Doc.Front.SetList("required_artifacts", []string{"binary"})
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	old := store.ObserveGitHubExternalVerification
	oldPolicy := store.ObserveGitHubRequiredCheckPolicy
	t.Cleanup(func() {
		store.ObserveGitHubExternalVerification = old
		store.ObserveGitHubRequiredCheckPolicy = oldPolicy
	})
	store.ObserveGitHubRequiredCheckPolicy = func(_, repo, branch string, _ []string, at time.Time) (store.GitHubRequiredCheckPolicy, error) {
		return store.GitHubRequiredCheckPolicy{Schema: store.GitHubRequiredCheckPolicySchema, Repository: repo, Branch: branch, ObservedAt: at, State: "observed", Requirements: []store.RequiredCheckRequirement{{Name: "ci", Sources: []store.RequiredCheckSource{{Kind: "configured"}, {Kind: "ruleset", RulesetID: 8}}}}}, nil
	}
	stale := true
	store.ObserveGitHubExternalVerification = func(_, _, commit string, at time.Time) ([]store.ExternalVerificationEvidence, error) {
		head := commit
		if stale {
			head = "stale-head"
		}
		return []store.ExternalVerificationEvidence{{Provider: "github-actions", WorkflowRunID: "22", HeadSHA: head, Name: "ci", ObservedAt: at, State: "observed", Conclusion: "success", Artifacts: []store.ExternalArtifactEvidence{{ID: "31", Name: "binary", Digest: "sha256:abc"}}}}, nil
	}
	task := mkTask(t, w, "external evidence")
	if err := acceptOne(ctx, w, root, task, "true", false, false, false, false, false, ""); clikit.ExitCode(err) != 3 {
		t.Fatalf("stale external head = exit %d err=%v, want refusal", clikit.ExitCode(err), err)
	}
	got, _ := store.FindTask(w, task.ID)
	if got.Status == model.StatusDone {
		t.Fatal("stale external evidence closed task")
	}
	stale = false
	ctx.Stderr.(*bytes.Buffer).Reset()
	if err := acceptOne(ctx, w, root, got, "true", false, false, false, false, false, ""); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"required check ci", "configured", "ruleset 8"} {
		if output := ctx.Stderr.(*bytes.Buffer).String(); !strings.Contains(output, want) {
			t.Fatalf("human policy output missing %q:\n%s", want, output)
		}
	}
	got, _ = store.FindTask(w, task.ID)
	records := store.VerificationEvidenceRecords(got)
	if len(records) != 1 || len(records[0].External) != 1 || records[0].External[0].Artifacts[0].Digest != "sha256:abc" {
		t.Fatalf("external evidence was not persisted with local evidence: %+v", records)
	}
}

func TestAcceptanceRulesetPolicyFailsClosedAndExposesProvenance(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	ctx.JSON = true
	project, err := store.LoadProject(w, "core")
	if err != nil {
		t.Fatal(err)
	}
	project.Doc.Front.Set("github_repo", "owner/repo")
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	oldPolicy, oldEvidence := store.ObserveGitHubRequiredCheckPolicy, store.ObserveGitHubExternalVerification
	t.Cleanup(func() {
		store.ObserveGitHubRequiredCheckPolicy, store.ObserveGitHubExternalVerification = oldPolicy, oldEvidence
	})
	store.ObserveGitHubRequiredCheckPolicy = func(_, repo, branch string, _ []string, at time.Time) (store.GitHubRequiredCheckPolicy, error) {
		return store.GitHubRequiredCheckPolicy{Schema: store.GitHubRequiredCheckPolicySchema, Repository: repo, Branch: branch, ObservedAt: at, State: "observed", Requirements: []store.RequiredCheckRequirement{{Name: "ruleset-ci", Sources: []store.RequiredCheckSource{{Kind: "ruleset", RulesetID: 42, RulesetSourceType: "Organization", RulesetSource: "owner"}}}}}, nil
	}
	green := false
	store.ObserveGitHubExternalVerification = func(_, _, commit string, at time.Time) ([]store.ExternalVerificationEvidence, error) {
		conclusion := "failure"
		if green {
			conclusion = "success"
		}
		return []store.ExternalVerificationEvidence{{Provider: "github-check", CheckRunID: "1", HeadSHA: commit, Name: "ruleset-ci", ObservedAt: at, State: "observed", Conclusion: conclusion}}, nil
	}
	task := mkTask(t, w, "ruleset evidence")
	if err := acceptOneForTreePolicy(ctx, w, root, task, "true", false, false, false, false, false, false, "", "", ""); clikit.ExitCode(err) != 3 {
		t.Fatalf("red ruleset check exit=%d err=%v", clikit.ExitCode(err), err)
	}
	green = true
	ctx.Stdout.(*bytes.Buffer).Reset()
	if err := acceptOneForTreePolicy(ctx, w, root, task, "true", false, false, false, false, false, false, "", "", ""); err != nil {
		t.Fatal(err)
	}
	output := ctx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{`"external_policy"`, `"name": "ruleset-ci"`, `"kind": "ruleset"`, `"ruleset_id": 42`} {
		if !strings.Contains(output, want) {
			t.Fatalf("JSON result missing %s:\n%s", want, output)
		}
	}
	got, _ := store.FindTask(w, task.ID)
	records := store.VerificationEvidenceRecords(got)
	if len(records) != 1 || records[0].ExternalPolicy == nil || records[0].ExternalPolicy.Requirements[0].Name != "ruleset-ci" {
		t.Fatalf("persisted policy = %+v", records)
	}
}

func TestAcceptanceUnobservablePolicyRequiresAuditedOverride(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	project, _ := store.LoadProject(w, "core")
	project.Doc.Front.Set("github_repo", "owner/repo")
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	oldPolicy, oldEvidence := store.ObserveGitHubRequiredCheckPolicy, store.ObserveGitHubExternalVerification
	t.Cleanup(func() {
		store.ObserveGitHubRequiredCheckPolicy, store.ObserveGitHubExternalVerification = oldPolicy, oldEvidence
	})
	store.ObserveGitHubRequiredCheckPolicy = func(_, repo, branch string, _ []string, at time.Time) (store.GitHubRequiredCheckPolicy, error) {
		return store.GitHubRequiredCheckPolicy{Schema: store.GitHubRequiredCheckPolicySchema, Repository: repo, Branch: branch, ObservedAt: at, State: "unobservable", Error: "HTTP 403"}, errors.New("HTTP 403")
	}
	store.ObserveGitHubExternalVerification = func(_, _, _ string, _ time.Time) ([]store.ExternalVerificationEvidence, error) { return nil, nil }
	task := mkTask(t, w, "unobservable policy")
	if err := acceptOneForTreePolicy(ctx, w, root, task, "true", false, false, false, false, false, false, "", "", ""); clikit.ExitCode(err) != 3 {
		t.Fatalf("unobservable policy exit=%d err=%v", clikit.ExitCode(err), err)
	}
	if err := acceptOneForTreePolicy(ctx, w, root, task, "true", false, false, false, false, true, false, "", "", ""); err != nil {
		t.Fatal(err)
	}
	got, _ := store.FindTask(w, task.ID)
	if body := taskLog(t, got); !strings.Contains(body, "required-check policy override by a-root") || !strings.Contains(body, "HTTP 403") {
		t.Fatalf("override was not audited:\n%s", body)
	}
}

func TestAcceptAllEnforcesRulesetChecksPerTask(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	project, _ := store.LoadProject(w, "core")
	project.Doc.Front.Set("github_repo", "owner/repo")
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	oldPolicy, oldEvidence := store.ObserveGitHubRequiredCheckPolicy, store.ObserveGitHubExternalVerification
	t.Cleanup(func() {
		store.ObserveGitHubRequiredCheckPolicy, store.ObserveGitHubExternalVerification = oldPolicy, oldEvidence
	})
	store.ObserveGitHubRequiredCheckPolicy = func(_, repo, branch string, _ []string, at time.Time) (store.GitHubRequiredCheckPolicy, error) {
		return store.GitHubRequiredCheckPolicy{Schema: store.GitHubRequiredCheckPolicySchema, Repository: repo, Branch: branch, ObservedAt: at, State: "observed", Requirements: []store.RequiredCheckRequirement{{Name: "batch-ci", Sources: []store.RequiredCheckSource{{Kind: "ruleset", RulesetID: 77}}}}}, nil
	}
	store.ObserveGitHubExternalVerification = func(_, _, commit string, at time.Time) ([]store.ExternalVerificationEvidence, error) {
		return []store.ExternalVerificationEvidence{{Provider: "github-check", CheckRunID: "2", HeadSHA: commit, Name: "batch-ci", ObservedAt: at, State: "observed", Conclusion: "failure"}}, nil
	}
	task := mkTask(t, w, "batch ruleset evidence")
	if err := propose(ctx, w, root, task); err != nil {
		t.Fatal(err)
	}
	err := acceptAllForTreePolicy(ctx, w, root, "true", false, false, false, false, true, false, false, "", "", "")
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("batch ruleset check exit=%d err=%v", clikit.ExitCode(err), err)
	}
	got, _ := store.FindTask(w, task.ID)
	if got.Status == model.StatusDone {
		t.Fatal("batch acceptance closed task over red ruleset check")
	}
}

func TestAcceptRefusesCommandCriterionWithoutProvenance(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk, err := store.CreateTask(w, agentid.RootID, "core", "command checked close", store.TaskOpts{Accept: []string{"`go test ./...` exits zero"}})
	if err != nil {
		t.Fatal(err)
	}

	err = acceptOne(ctx, w, root, tk, "", false, false, false, false, false, "")
	if clikit.ExitCode(err) != 3 {
		t.Fatalf("accept exit = %d, want refusal for missing command evidence: %v", clikit.ExitCode(err), err)
	}
	got, findErr := store.FindTask(w, tk.Slug)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if got.Status == model.StatusDone || got.Acceptance()[0].Done {
		t.Fatal("command criterion was accepted without provenance")
	}
}

// --require-verify makes an unverified close impossible rather than merely
// visible: the strict mode for generating repos whose record is the product.
func TestRequireVerifyRefusesUnverifiedClose(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk := mkTask(t, w, "must not close unverified")

	err := acceptOne(ctx, w, root, tk, "", true, false, false, false, false, "")
	if err == nil {
		t.Fatal("acceptOne with requireVerify and no command must refuse")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refused by policy)", clikit.ExitCode(err))
	}
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Error("a refused close must leave the task open")
	}
}

// The implementer marking its own work complete is the failure the role split
// exists to prevent, and nothing enforced it: the claimant owned the task and
// accepted it (dacli 188).
func TestRequireIndependentBlocksSelfCertification(t *testing.T) {
	ctx, w, _ := evidenceEnv(t)
	tk := mkTask(t, w, "self certified")

	// The worker claims the task, then tries to certify its own work.
	worker := &agentid.Identity{ID: "a-worker", Grant: model.GrantRW, Role: "fixer"}
	store.AppendLog(tk, "claimed by "+worker.ID)
	tk.Doc.Front.Set("owner", worker.ID)
	if err := store.SaveTask(tk); err != nil {
		t.Fatal(err)
	}

	err := acceptOne(ctx, w, worker, tk, "true", false, true, false, false, false, "")
	if err == nil {
		t.Fatal("the claimant must not be able to certify its own task under --require-independent")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refused by policy)", clikit.ExitCode(err))
	}

	// A DIFFERENT agent certifying the same task is fine.
	reviewer := &agentid.Identity{ID: "a-reviewer", Grant: model.GrantRW, Role: "reviewer"}
	if err := acceptOne(ctx, w, reviewer, tk, "true", false, true, false, false, false, ""); err != nil {
		t.Fatalf("an independent certifier must be allowed: %v", err)
	}
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Errorf("status = %s, want done after independent certification", got.Status)
	}
}

// A failing verification must never close the task.
func TestFailedVerificationLeavesTaskOpen(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk := mkTask(t, w, "verification fails")

	if err := acceptOne(ctx, w, root, tk, "exit 1", false, false, false, false, false, ""); err == nil {
		t.Fatal("a non-zero verify command must refuse the close")
	}
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Error("a task whose verification failed must stay open")
	}
}

func TestAcceptanceGradeVerificationRefusesDirtyTreeWithoutClosing(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk := mkTask(t, w, "dirty tree cannot certify")
	if err := os.WriteFile(w.Root+"/.gitignore", []byte(".dacli/\nchanged\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := acceptOne(ctx, w, root, tk, "true", true, false, false, false, false, "")
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "working tree is dirty") {
		t.Fatalf("dirty acceptance = exit %d, %v; want policy refusal naming dirty tree", clikit.ExitCode(err), err)
	}
	got, findErr := store.FindTask(w, tk.Slug)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if got.Status == model.StatusDone || len(store.VerificationEvidenceRecords(got)) != 0 {
		t.Fatalf("dirty verification closed or recorded success: status=%s evidence=%#v", got.Status, store.VerificationEvidenceRecords(got))
	}
}

func TestAcceptanceGradeVerificationRejectsMutationWithoutRecordingSuccess(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk := mkTask(t, w, "mutation cannot certify")

	err := acceptOne(ctx, w, root, tk, "printf mutation >> .gitignore", true, false, false, false, false, "")
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "changed during execution") {
		t.Fatalf("mutating acceptance = exit %d, %v; want policy refusal naming before/after change", clikit.ExitCode(err), err)
	}
	got, findErr := store.FindTask(w, tk.Slug)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if got.Status == model.StatusDone || len(store.VerificationEvidenceRecords(got)) != 0 {
		t.Fatalf("mutating verification closed or recorded success: status=%s evidence=%#v", got.Status, store.VerificationEvidenceRecords(got))
	}
}

func TestAcceptanceRefusesEvidenceForDifferentReviewedLandingTree(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk := mkTask(t, w, "stale tree cannot certify")
	commit, err := gitx.Run(w.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	err = acceptOneForTree(ctx, w, root, tk, "true", true, false, false, false, false, "", strings.TrimSpace(commit), "different-reviewed-tree")
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "rerun verification on the reviewed head") {
		t.Fatalf("stale final-tree acceptance = exit %d, %v; want actionable policy refusal", clikit.ExitCode(err), err)
	}
	got, findErr := store.FindTask(w, tk.Slug)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if got.Status == model.StatusDone || len(store.VerificationEvidenceRecords(got)) != 0 {
		t.Fatalf("stale final-tree evidence closed or persisted: status=%s evidence=%#v", got.Status, store.VerificationEvidenceRecords(got))
	}
}
