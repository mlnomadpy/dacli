package execution

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestReviewCodexHelperProcess(t *testing.T) {
	if os.Getenv("DACLI_REVIEW_HELPER") != "1" {
		return
	}
	prompt, _ := io.ReadAll(os.Stdin)
	if strings.HasPrefix(string(prompt), "Return no content.") {
		fmt.Println(`{"type":"turn.started"}`)
		return
	}
	const identityPrefix = "You are agent "
	start := strings.Index(string(prompt), identityPrefix)
	if start < 0 {
		os.Exit(21)
	}
	fields := strings.Fields(string(prompt)[start+len(identityPrefix):])
	if len(fields) == 0 {
		os.Exit(22)
	}
	branch := os.Getenv("DACLI_REVIEW_BRANCH")
	commitCmd := exec.Command(os.Getenv("DACLI_REVIEW_GIT"), "rev-parse", "--verify", branch)
	commitCmd.Dir, _ = os.Getwd()
	commitRaw, err := commitCmd.Output()
	if err != nil {
		os.Exit(23)
	}
	commit := strings.TrimSpace(string(commitRaw))
	treeCmd := exec.Command(os.Getenv("DACLI_REVIEW_GIT"), "rev-parse", "--verify", commit+"^{tree}")
	treeCmd.Dir = commitCmd.Dir
	treeRaw, err := treeCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tree lookup failed: branch=%q commit=%q dir=%q err=%v\n", branch, commit, treeCmd.Dir, err)
		os.Exit(24)
	}
	result := store.IndependentReviewResult{
		Schema: store.ReviewResultSchema, Verdict: store.ReviewApprove,
		ReviewerID: fields[0], ReviewerRole: "codex-independent", Runtime: "codex-review-fixture", Model: "gpt-review", Grant: "ro",
		IndependentOf: []string{"a-builder"}, CommitSHA: commit, TreeSHA: strings.TrimSpace(string(treeRaw)), ObservedAt: time.Now().UTC(),
	}
	raw, _ := json.Marshal(result)
	fmt.Printf("%s%s\n", store.ReviewOutputMarker, raw)
}

func reviewContractFixture(t *testing.T) (*store.Task, string, *clikit.Ctx) {
	t.Helper()
	w := newExecWS(t)
	task := mustTask(t, w, "Bound review contract", store.TaskOpts{})
	binary := fakeBinary(t)
	runtime := store.Runtime{
		Name: "codex-review-contract", Harness: "codex", Binary: binary, Mode: "stdin",
		SandboxRO: []string{"--sandbox", "read-only"},
	}
	mustRuntime(t, w, runtime)
	loaded, err := store.LoadRuntime(w, runtime.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRuntimeROProbe(w, loaded, binary, store.RuntimeROVerified, "fixture verified"); err != nil {
		t.Fatal(err)
	}
	mustRole(t, w, team.Role{Name: "review-contract", Kind: "reviewer", Grant: "ro", Runtime: runtime.Name, Model: "gpt-review"})
	ctx, _, _ := newCtx(w.Root)
	return task, w.Root, ctx
}

func TestPreflightEmitsExactVersionedIndependentReviewContract(t *testing.T) {
	_, root, ctx := reviewContractFixture(t)
	_, out, _ := newCtx(root)
	ctx.Stdout = out
	if err := cmdPreflight(ctx, []string{"--role", "review-contract", "--grant", "ro", "--structured-review-result"}); err != nil {
		t.Fatal(err)
	}
	contract, err := store.ParseRuntimeLaunchContract(out.String())
	if err != nil {
		t.Fatal(err)
	}
	if contract.Schema != store.RuntimeLaunchContractSchema || contract.Harness != "codex" || contract.Runtime != "codex-review-contract" || contract.Model != "gpt-review" || contract.Grant != "ro" || contract.ResultChannel != store.IndependentReviewChannel || strings.Join(contract.SandboxFlags, " ") != "--sandbox read-only" {
		t.Fatalf("preflight contract = %+v", contract)
	}
}

func TestSpawnRefusesChangedPreflightFingerprintBeforeRecords(t *testing.T) {
	task, root, ctx := reviewContractFixture(t)
	_, out, _ := newCtx(root)
	ctx.Stdout = out
	if err := cmdPreflight(ctx, []string{"--role", "review-contract", "--grant", "ro", "--structured-review-result"}); err != nil {
		t.Fatal(err)
	}
	contract, err := store.ParseRuntimeLaunchContract(out.String())
	if err != nil {
		t.Fatal(err)
	}
	w, err := workspace.Find(root)
	if err != nil {
		t.Fatal(err)
	}
	agentsBefore, runsBefore := countAgents(t, w), countRuns(t, w)
	err = cmdSpawn(ctx, []string{"--task", task.ID, "--role", "review-contract", "--grant", "ro", "--model", "different-model", "--review", "--structured-review-result", "--preflight-fingerprint", contract.Fingerprint})
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "fingerprint changed") || !strings.Contains(err.Error(), contract.Fingerprint) {
		t.Fatalf("changed launch fingerprint = exit %d, %v", clikit.ExitCode(err), err)
	}
	if countAgents(t, w) != agentsBefore || countRuns(t, w) != runsBefore {
		t.Fatal("fingerprint mismatch created an agent or run")
	}
}

func TestIndependentReviewRefusesCooperativeOrWriteGrantBeforeRecords(t *testing.T) {
	task, root, ctx := reviewContractFixture(t)
	w, err := workspace.Find(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"cooperative-ro", []string{"--task", task.ID, "--role", "review-contract", "--grant", "ro", "--review", "--structured-review-result", "--cooperative"}},
		{"rw", []string{"--task", task.ID, "--role", "review-contract", "--grant", "rw", "--review", "--structured-review-result"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agentsBefore, runsBefore := countAgents(t, w), countRuns(t, w)
			err := cmdSpawn(ctx, tc.args)
			if clikit.ExitCode(err) != 3 {
				t.Fatalf("refusal exit=%d err=%v", clikit.ExitCode(err), err)
			}
			if countAgents(t, w) != agentsBefore || countRuns(t, w) != runsBefore {
				t.Fatal("invalid independent review created an agent or run")
			}
		})
	}
}

func TestCodexIndependentReviewCompletesThroughReadOnlyResultChannel(t *testing.T) {
	w := newExecWS(t)
	initExecGitRepo(t, w.Root)
	task := mustTask(t, w, "Codex read-only review", store.TaskOpts{})
	branch := taskBranch(task)
	if _, err := gitx.Run(w.Root, "branch", branch); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DACLI_REVIEW_HELPER", "1")
	t.Setenv("DACLI_REVIEW_BRANCH", branch)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DACLI_REVIEW_GIT", gitPath)
	runtime := store.Runtime{
		Name: "codex-review-fixture", Harness: "codex", Binary: os.Args[0], Mode: "stdin",
		GlobalArgs: []string{"-test.run=TestReviewCodexHelperProcess", "--"}, SandboxRO: []string{"--sandbox", "read-only"},
		Env: []string{"DACLI_REVIEW_HELPER", "DACLI_REVIEW_BRANCH", "DACLI_REVIEW_GIT"}, ModelFlag: "--model",
		BehavioralPreflight: store.BehavioralPreflightCodexExecJSONV2,
	}
	mustRuntime(t, w, runtime)
	loaded, err := store.LoadRuntime(w, runtime.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRuntimeROProbe(w, loaded, os.Args[0], store.RuntimeROVerified, "fixture verified"); err != nil {
		t.Fatal(err)
	}
	mustRole(t, w, team.Role{Name: "codex-independent", Kind: "reviewer", Grant: "ro", Runtime: runtime.Name, Model: "gpt-review"})

	ctx, out, _ := newCtx(w.Root)
	if err := cmdPreflight(ctx, []string{"--role", "codex-independent", "--grant", "ro", "--structured-review-result"}); err != nil {
		t.Fatal(err)
	}
	contract, err := store.ParseRuntimeLaunchContract(out.String())
	if err != nil {
		t.Fatal(err)
	}
	before, err := gitx.Run(w.Root, "status", "--porcelain", "--", ".", ":(exclude).dacli")
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdSpawn(ctx, []string{"--task", task.ID, "--role", "codex-independent", "--grant", "ro", "--review", "--structured-review-result", "--preflight-fingerprint", contract.Fingerprint}); err != nil {
		t.Fatal(err)
	}
	after, err := gitx.Run(w.Root, "status", "--porcelain", "--", ".", ":(exclude).dacli")
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("read-only reviewer changed repository files: before=%q after=%q", before, after)
	}
	events, err := eventlog.List(w, eventlog.Query{About: task.ID, Kinds: []model.EventKind{model.EventReview}})
	if err != nil || len(events) != 1 {
		t.Fatalf("review events=%d err=%v", len(events), err)
	}
	var result store.IndependentReviewResult
	if err := json.Unmarshal([]byte(events[0].Body), &result); err != nil || result.Verdict != store.ReviewApprove || result.Grant != "ro" || result.Runtime != runtime.Name {
		t.Fatalf("materialized review=%+v err=%v", result, err)
	}
}
