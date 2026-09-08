package orchestration

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/verifyroute"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var errInjectedLoopRestart = errors.New("injected loop process interruption")

// codexRestartRunner is a deterministic local stand-in for the Codex adapter.
// It exercises the same dacli-native command boundary as the loop while making
// each external effect observable and countable. The implementation worker is
// intentionally unable to commit: it leaves a structured root handoff that
// wait recovers through the parent-mediated commit transaction.
type codexRestartRunner struct {
	w              *workspace.Workspace
	task           *store.Task
	review         *reviewTransactionRunner
	calls          [][]string
	baseOID        string
	worktreeBase   string
	implementation int
	corrections    int
	pushes         int
	prs            int
	accepts        int
	runID          string
	handoff        store.RootHandoff
}

func (r *codexRestartRunner) runResult(label string, target any, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	return r.review.runResult(label, target, args...)
}

func (r *codexRestartRunner) run(label string, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if len(args) == 0 {
		return "", nil
	}
	switch {
	case args[0] == "spawn" && contains(args, "--detach"):
		r.implementation++
		worktree := r.w.WorktreePath(r.task.Project, r.task.Seq, r.task.Slug)
		if _, err := gitx.AddWorktreeFrom(r.w.Root, worktree, taskBranch(r.task), "refs/remotes/origin/release"); err != nil {
			return err.Error(), err
		}
		r.worktreeBase, _ = gitx.Run(worktree, "rev-parse", "HEAD")
		if err := os.WriteFile(filepath.Join(worktree, "source.go"), []byte("package source\n\nconst Implemented = true\n"), 0o644); err != nil {
			return err.Error(), err
		}
		r.runID = "01CODEXRESTARTFIXTURE000001"
		rec := procmon.Record{RunID: r.runID, Child: "a-codex-builder", Task: r.task.ID, Role: "fixer", Runtime: "codex", Claims: []string{"source.go"}, Started: time.Unix(10, 0), Outcome: "success"}
		if err := procmon.WriteRecord(filepath.Join(r.w.RunDir(r.runID), "proc.txt"), rec); err != nil {
			return err.Error(), err
		}
		handoff, ok, err := store.CaptureRootHandoff(r.w, r.runID, r.task.ID, rec.Child, worktree, store.RootHandoffRequest{
			Schema: store.RootHandoffSchema, CommitMessage: "Implement claimed source",
			Verification:    []store.RootHandoffVerification{{Command: "go test ./...", ExitCode: 0, Result: "pass"}},
			FailedOperation: "git index lock", FailureClass: "filesystem_sandbox_refusal", NextAction: "parent applies exact commit",
		}, time.Unix(20, 0))
		if err != nil || !ok {
			failure := fmt.Errorf("capture parent handoff ok=%t: %w", ok, err)
			return failure.Error(), failure
		}
		r.handoff = handoff
	case args[0] == "wait":
		if r.handoff.RunID != "" {
			if _, err := os.Stat(store.ParentCommitReceiptPath(r.w, r.runID)); os.IsNotExist(err) {
				if _, err := store.ApplyParentCommit(r.w, r.handoff, time.Unix(30, 0)); err != nil {
					return "", err
				}
			}
		}
	case label == "review-correction":
		r.corrections++
		worktree := r.w.WorktreePath(r.task.Project, r.task.Seq, r.task.Slug)
		if err := os.WriteFile(filepath.Join(worktree, "source.go"), []byte("package source\n\nconst Corrected = true\n"), 0o644); err != nil {
			return err.Error(), err
		}
		for _, gitArgs := range [][]string{{"add", "--", "source.go"}, {"commit", "-m", "review correction"}} {
			if out, err := gitx.Run(worktree, gitArgs...); err != nil {
				return out, err
			}
		}
	case args[0] == "push":
		r.pushes++
		if _, err := gitx.Run(r.w.Root, "push", "origin", taskBranch(r.task)); err != nil {
			return "", err
		}
	case args[0] == "pr":
		r.prs++
	case args[0] == "accept":
		r.accepts++
		fresh, err := store.FindTask(r.w, r.task.ID)
		if err != nil {
			return "", err
		}
		store.CheckAllAcceptance(fresh)
		if err := store.SaveTask(fresh); err != nil {
			return "", err
		}
		return "", store.MoveTask(r.w, fresh, model.StatusDone)
	}
	return "", nil
}

// TestCodexLoopComposesEveryRestartBoundary closes the final composed-test gap
// from #1016. In one fixture it crosses the boundaries previously covered only
// in isolation: configured remote-base worktree creation (#1018), exact Codex
// review contracts (#1019), parent-mediated commits (#1020), claimed edits
// (#1021), task-branch publication (#1022), durable phase/recovery state
// (#1023/#1024), and exact bounded run/task/branch provenance (#1025).
func TestCodexLoopComposesEveryRestartBoundary(t *testing.T) {
	w := loopEnv(t)
	if err := os.WriteFile(filepath.Join(w.Root, "source.go"), []byte("package source\n\nconst Corrected = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitTo(t, w.Root, "base.txt")

	bare := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare origin: %v\n%s", err, out)
	}
	for _, args := range [][]string{{"remote", "add", "origin", bare}, {"push", "origin", "main:release"}, {"fetch", "origin"}} {
		if out, err := gitx.Run(w.Root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	baseOID, err := gitx.Run(w.Root, "rev-parse", "refs/remotes/origin/release")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "branch", "release", strings.TrimSpace(baseOID)); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "checkout", "-b", "operator-feature"); err != nil {
		t.Fatal(err)
	}
	commitTo(t, w.Root, "unrelated-operator.txt")
	operatorOID, _ := gitx.Run(w.Root, "rev-parse", "HEAD")

	task, err := store.CreateTask(w, "a-root", "p", "Correct claimed source", store.TaskOpts{Accept: []string{"source.go is independently reviewed"}, Claims: []string{"source.go"}})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := defaultProfile("p", "task")
	if err != nil {
		t.Fatal(err)
	}
	profile.Verification.Rules = []verifyroute.Rule{{ID: "source-test", Include: []string{"source.go"}, Cwd: ".", Argv: []string{"/usr/bin/true"}, Gate: "test", Required: true}}
	profile.Verification.IndependentReviews = 1
	profile.Verification.CorrectionTurns = 1
	profile.Landing = LandingPolicy{Mode: "pr", ChecksRequired: true, ReviewsRequired: 1, AutoMerge: true, ProtectedBranch: "release"}
	if err := saveProfile(w, profile); err != nil {
		t.Fatal(err)
	}

	runner := &codexRestartRunner{w: w, task: task, baseOID: strings.TrimSpace(baseOID)}
	runner.review = &reviewTransactionRunner{w: w, task: task}
	configure := func(d *driver) {
		d.cfg.project, d.cfg.width = "p", 1
		d.cfg.implRole, d.cfg.implRoleExplicit = "fixer", true
		d.cfg.reviewRole, d.cfg.reviewRoleExplicit = "reviewer", true
		d.cfg.allowedHarnesses = []string{"codex"}
		d.cfg.harnessMode = "single"
		d.cfg.pr, d.cfg.autoMerge = true, true
		d.cfg.landing = model.LandingPolicy{Mode: model.LandingPR, Base: "release"}
		d.trunkBranch = "release"
		d.now = func() time.Time { return time.Unix(100, 0) }
	}

	boundaries := []cyclePhase{phaseSpawned, phaseWaited, phaseCommitted, phaseVerified, phaseReviewPending, phaseCorrectionPending, phaseRereviewPending, phaseReviewed, phasePushed, phasePRCreated}
	for _, boundary := range boundaries {
		journal, err := readPhaseJournal(w, "p")
		if err != nil {
			t.Fatal(err)
		}
		d := newDriver(w, runner, &Governor{})
		d.phases = journal
		configure(d)
		tripped := false
		d.afterPhase = func(got cyclePhase) error {
			if got == boundary && !tripped {
				tripped = true
				return errInjectedLoopRestart
			}
			return nil
		}
		d.runCycle([]*store.Task{task})
		if !tripped || !errors.Is(d.phaseErr, errInjectedLoopRestart) {
			t.Fatalf("restart boundary %s was not reached: tripped=%t err=%v calls=%v log=%s", boundary, tripped, d.phaseErr, runner.calls, d.ctx.Stdout)
		}
		checkpoint, ok := d.taskPhase(task)
		if !ok || checkpoint.Phase != boundary || checkpoint.TaskID != task.ID || checkpoint.Branch != taskBranch(task) || checkpoint.RunID != runner.runID {
			t.Fatalf("boundary %s lost exact provenance: %+v", boundary, checkpoint)
		}
	}

	if strings.TrimSpace(runner.worktreeBase) != runner.baseOID || strings.TrimSpace(operatorOID) == runner.baseOID {
		t.Fatalf("task worktree base=%s, want origin/release=%s; operator=%s", runner.worktreeBase, runner.baseOID, operatorOID)
	}
	if _, err := os.Stat(filepath.Join(w.RunDir(runner.runID), store.RootHandoffConsumedFile)); err != nil {
		t.Fatalf("parent-mediated commit was not consumed: %v", err)
	}
	if runner.implementation != 1 || runner.review.reviews != 2 || runner.corrections != 1 || runner.pushes != 1 || runner.prs != 1 {
		t.Fatalf("restart duplicated work: implementation=%d reviews=%d corrections=%d pushes=%d prs=%d calls=%v", runner.implementation, runner.review.reviews, runner.corrections, runner.pushes, runner.prs, runner.calls)
	}

	journal, err := readPhaseJournal(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	final := newDriver(w, runner, &Governor{})
	final.phases = journal
	configure(final)
	final.restoreLandingPhases()
	stubOrchestrationGH(t, func(string, ...string) (string, error) {
		return `[{"state":"OPEN","autoMergeRequest":{"enabledAt":"2026-01-01T00:00:00Z"}}]`, nil
	})
	if got := final.prLandStatus(taskBranch(task)); got != "landing" {
		t.Fatalf("simulated CI-pending PR state=%s", got)
	}
	if !final.checkpointTaskPhase(task, phaseCIPending) {
		t.Fatal(final.phaseErr)
	}

	journal, err = readPhaseJournal(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	merged := newDriver(w, runner, &Governor{})
	merged.phases = journal
	configure(merged)
	merged.restoreLandingPhases()
	stubOrchestrationGH(t, func(string, ...string) (string, error) { return `[{"state":"MERGED"}]`, nil })
	rollup := merged.reconcilePendingAccepts()
	if rollup.Landed != 1 || runner.accepts != 1 || len(merged.pendingAccept) != 0 {
		t.Fatalf("merged reconciliation=%+v accepts=%d pending=%v", rollup, runner.accepts, merged.pendingAccept)
	}
	merged.reconcilePendingAccepts()
	if runner.accepts != 1 {
		t.Fatalf("accepted task replayed after restart: %d calls", runner.accepts)
	}
	closed, err := store.FindTask(w, task.ID)
	if err != nil || closed.Status != model.StatusDone {
		t.Fatalf("final task state=%v err=%v", closed.Status, err)
	}
	for _, call := range runner.calls {
		if len(call) > 0 && !map[string]bool{"spawn": true, "wait": true, "sync": true, "push": true, "pr": true, "ship": true, "lint": true, "retro": true, "doctor": true, "accept": true}[call[0]] {
			t.Fatalf("fixture required a non-dacli lifecycle command: %v", call)
		}
	}
}

var _ resultRunner = (*codexRestartRunner)(nil)
var _ runner = (*codexRestartRunner)(nil)
