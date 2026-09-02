package store

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func cleanupFixture(t *testing.T) (*workspace.Workspace, *Task, string, func()) {
	t.Helper()
	root := t.TempDir()
	worktreeGit(t, root, "init", "-q", "-b", "main")
	worktreeGit(t, root, "config", "user.email", "test@example.test")
	worktreeGit(t, root, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(root, "product"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, root, "add", "product")
	worktreeGit(t, root, "commit", "-qm", "base")
	w, err := workspace.Init(root, "cleanup")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := CreateTask(w, "a-root", "core", "landed change", TaskOpts{Accept: []string{"landed"}})
	if err != nil {
		t.Fatal(err)
	}
	branch := TaskBranch(task)
	worktreeGit(t, root, "checkout", "-qb", branch)
	if err := os.WriteFile(filepath.Join(root, "product"), []byte("landed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, root, "commit", "-qam", "land task")
	branchTip, err := exec.Command("git", "-C", root, "rev-parse", branch).Output()
	if err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, root, "checkout", "-q", "main")
	worktreeGit(t, root, "merge", "--no-ff", "-qm", "merge task", branch)
	worktreeGit(t, root, "update-ref", "refs/remotes/origin/"+branch, strings.TrimSpace(string(branchTip)))
	checkout := w.WorktreePath(task.Project, task.Seq, task.Slug)
	if err := os.MkdirAll(filepath.Dir(checkout), 0o755); err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, root, "worktree", "add", "-q", checkout, branch)
	if err := MoveTask(w, task, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	runID := "01M14CLEANUP0000000000001"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{RunID: runID, Task: task.ID, Child: "a-worker", Outcome: "completed"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir(runID), "transcript.log"), []byte("durable evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir(runID), "capture.tmp"), []byte("generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir(runID), "verification.tmp"), []byte("durable verification\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := ObserveDeliveryPRs
	ObserveDeliveryPRs = func(string) ([]DeliveryPR, error) {
		merge := &struct {
			OID string `json:"oid"`
		}{OID: "merge"}
		return []DeliveryPR{{Number: 1, DeliveryConfidence: "CLOSED", HeadRefName: branch, HeadRefOid: strings.TrimSpace(string(branchTip)), MergeCommit: merge}}, nil
	}
	return w, task, checkout, func() { ObserveDeliveryPRs = old }
}

func addDetachedCleanupWorktree(t *testing.T, w *workspace.Workspace, name, ref string) string {
	t.Helper()
	path := filepath.Join(w.WorktreesDir(), name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	worktreeGit(t, w.Root, "worktree", "add", "-q", "--detach", path, ref)
	return path
}

func cleanupItemAt(t *testing.T, plan CleanupPlan, path string) CleanupItem {
	t.Helper()
	for _, item := range plan.Items {
		if cleanPath(item.Worktree) == cleanPath(path) {
			return item
		}
	}
	t.Fatalf("cleanup plan has no item for %s: %+v", path, plan.Items)
	return CleanupItem{}
}

func TestCleanupPlansAndAppliesContainedDetachedWorktree(t *testing.T) {
	w, _, ordinary, restore := cleanupFixture(t)
	defer restore()
	detached := addDetachedCleanupWorktree(t, w, "accept-detached", "main")
	wantCommit := strings.TrimSpace(runGitOutput(t, detached, "rev-parse", "HEAD"))

	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(10, 0), ordinary)
	if err != nil {
		t.Fatal(err)
	}
	item := cleanupItemAt(t, plan, detached)
	if item.Branch != "" || item.Commit != wantCommit || !item.Eligible || item.PRState != "not-applicable" || len(item.Operations) != 1 || len(item.Recovery) != 1 {
		t.Fatalf("detached classification = %+v", item)
	}
	if strings.Contains(strings.Join(item.Reasons, " "), "ambiguous argument") {
		t.Fatalf("detached HEAD was resolved through an empty branch: %+v", item)
	}

	audit, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(11, 0), ordinary)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Removed) != 1 || audit.Removed[0].Worktree != item.Worktree || audit.Removed[0].Commit != wantCommit {
		t.Fatalf("detached cleanup audit = %+v", audit)
	}
	if _, err := os.Stat(detached); !os.IsNotExist(err) {
		t.Fatalf("eligible detached worktree remains: %v", err)
	}
	if _, err := os.Stat(ordinary); err != nil {
		t.Fatalf("protected ordinary worktree changed: %v", err)
	}
	if raw, err := os.ReadFile(filepath.Join(w.RunDir("01M14CLEANUP0000000000001"), "transcript.log")); err != nil || string(raw) != "durable evidence\n" {
		t.Fatalf("cleanup changed run evidence: err=%v raw=%q", err, raw)
	}
}

func TestCleanupProtectsUnsafeDetachedWorktrees(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *workspace.Workspace, *Task, string)
		plan   func(*workspace.Workspace, string) ([]string, time.Time)
		check  func(CleanupItem) bool
	}{
		{"dirty", func(t *testing.T, _ *workspace.Workspace, _ *Task, path string) {
			if err := os.WriteFile(filepath.Join(path, "scratch"), []byte("keep\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, nil, func(item CleanupItem) bool { return item.Dirty }},
		{"base-uncontained", func(t *testing.T, _ *workspace.Workspace, _ *Task, path string) {
			if err := os.WriteFile(filepath.Join(path, "detached-change"), []byte("keep\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			worktreeGit(t, path, "add", "detached-change")
			worktreeGit(t, path, "commit", "-qm", "detached work")
		}, nil, func(item CleanupItem) bool { return item.Unpushed }},
		{"live-owned", func(t *testing.T, w *workspace.Workspace, task *Task, path string) {
			runID := "01M14DETACHEDLIVE000000001"
			if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{RunID: runID, Task: task.ID, Child: "a-live", Claims: []string{"internal/store"}}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(w.RunDir(runID), "worktree.txt"), []byte(path+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, nil, func(item CleanupItem) bool { return len(item.Runs) == 1 && item.Runs[0].State == "live" }},
		{"explicitly-protected", func(*testing.T, *workspace.Workspace, *Task, string) {}, func(_ *workspace.Workspace, path string) ([]string, time.Time) {
			return []string{path}, time.Unix(20, 0)
		}, func(item CleanupItem) bool { return item.Protected }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, task, ordinary, restore := cleanupFixture(t)
			defer restore()
			path := addDetachedCleanupWorktree(t, w, "accept-protected", "main")
			test.mutate(t, w, task, path)
			protect, observedAt := []string{ordinary}, time.Unix(20, 0)
			if test.plan != nil {
				extra, at := test.plan(w, path)
				protect, observedAt = append(protect, extra...), at
			}
			plan, err := PlanRepositoryCleanup(w, "core", observedAt, protect...)
			if err != nil {
				t.Fatal(err)
			}
			item := cleanupItemAt(t, plan, path)
			if item.Eligible || !test.check(item) {
				t.Fatalf("unsafe detached worktree became eligible: %+v", item)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("planning changed protected worktree: %v", err)
			}
		})
	}
}

func TestCleanupRefusesStaleDetachedPlanBeforeRemoval(t *testing.T) {
	w, _, ordinary, restore := cleanupFixture(t)
	defer restore()
	detached := addDetachedCleanupWorktree(t, w, "accept-stale", "main")
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(30, 0), ordinary)
	if err != nil {
		t.Fatal(err)
	}
	if !cleanupItemAt(t, plan, detached).Eligible {
		t.Fatal("safe detached fixture was not eligible")
	}
	if err := os.WriteFile(filepath.Join(detached, "late-scratch"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(31, 0), ordinary); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("changed detached plan apply = %v, want stale refusal", err)
	}
	if _, err := os.Stat(filepath.Join(detached, "late-scratch")); err != nil {
		t.Fatalf("stale apply changed detached worktree: %v", err)
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestCleanupPlanAndApplyUseSameImmutableIdentity(t *testing.T) {
	w, _, checkout, restore := cleanupFixture(t)
	defer restore()
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Schema != CleanupPlanSchema || len(plan.ID) != 64 || len(plan.Items) != 1 || !plan.Items[0].Eligible {
		t.Fatalf("safe fixture not eligible: %+v", plan)
	}
	if plan.Items[0].Owner != "a-root" || len(plan.Items[0].Runs) != 1 || plan.Items[0].Runs[0].State != "terminal" {
		t.Fatalf("task/run/claim ownership classification missing: %+v", plan.Items[0])
	}
	classes := map[string]bool{}
	var generated CleanupArtifact
	for _, artifact := range plan.Artifacts {
		classes[artifact.Classification] = true
		if artifact.Pruneable {
			generated = artifact
		}
	}
	if !classes["generated-run-artifact"] || !classes["durable-evidence"] {
		t.Fatalf("run artifact classifications incomplete: %+v", plan.Artifacts)
	}
	if len(generated.Identity) != 64 || !strings.HasPrefix(generated.Digest, "sha256:") || generated.Quarantine == "" || generated.Operation == "" || !strings.Contains(generated.Recovery, plan.ID) {
		t.Fatalf("generated artifact omitted immutable quarantine/recovery identity: %+v", generated)
	}
	if _, err := os.Stat(filepath.Join(w.Root, workspace.Dir, "audit", "cleanup", plan.ID+".json")); !os.IsNotExist(err) {
		t.Fatalf("planning wrote audit state: %v", err)
	}
	audit, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Planned) != 1 || len(audit.Removed) != 1 || audit.Removed[0].Commit == "" || len(audit.Removed[0].Recovery) != 2 {
		t.Fatalf("incomplete cleanup audit: %+v", audit)
	}
	if len(audit.Quarantined) != 1 || audit.Quarantined[0].Identity != generated.Identity {
		t.Fatalf("quarantine move missing from cleanup audit: %+v", audit)
	}
	if _, err := os.Stat(generated.Path); !os.IsNotExist(err) {
		t.Fatalf("generated artifact source remains after quarantine: %v", err)
	}
	if raw, err := os.ReadFile(generated.Quarantine); err != nil || string(raw) != "generated\n" {
		t.Fatalf("quarantine did not preserve exact generated artifact: err=%v raw=%q", err, raw)
	}
	if raw, err := os.ReadFile(filepath.Join(w.RunDir(generated.RunID), "transcript.log")); err != nil || string(raw) != "durable evidence\n" {
		t.Fatalf("durable transcript was not preserved: err=%v raw=%q", err, raw)
	}
	if raw, err := os.ReadFile(filepath.Join(w.RunDir(generated.RunID), "verification.tmp")); err != nil || string(raw) != "durable verification\n" {
		t.Fatalf("durable verification evidence was not preserved: err=%v raw=%q", err, raw)
	}
	if _, err := os.Stat(checkout); !os.IsNotExist(err) {
		t.Fatalf("worktree remains after safe apply: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(w.Root, workspace.Dir, "audit", "cleanup", plan.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted CleanupAudit
	if json.Unmarshal(raw, &persisted) != nil || persisted.PlanID != plan.ID || len(persisted.Removed) != 1 || len(persisted.Quarantined) != 1 {
		t.Fatalf("audit does not preserve exact plan/removal: %s", raw)
	}
}

func TestCleanupArtifactStateChangesRefuseBeforeMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, *workspace.Workspace, *Task, string)
	}{
		{"artifact bytes", func(t *testing.T, _ *workspace.Workspace, _ *Task, source string) {
			if err := os.WriteFile(source, []byte("GENERATED\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"live claim", func(t *testing.T, w *workspace.Workspace, task *Task, _ string) {
			if err := procmon.WriteRecord(filepath.Join(w.RunDir("01M14CLEANUP0000000000001"), "proc.txt"), procmon.Record{RunID: "01M14CLEANUP0000000000001", Task: task.ID, Child: "a-worker", Claims: []string{"internal/store"}}); err != nil {
				t.Fatal(err)
			}
		}},
		{"malformed proc", func(t *testing.T, w *workspace.Workspace, _ *Task, _ string) {
			if err := os.WriteFile(filepath.Join(w.RunDir("01M14CLEANUP0000000000001"), "proc.txt"), []byte("malformed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"changed task", func(t *testing.T, w *workspace.Workspace, task *Task, _ string) {
			fresh, err := FindTask(w, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := MoveTask(w, fresh, model.StatusOpen); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, task, checkout, restore := cleanupFixture(t)
			defer restore()
			plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
			if err != nil {
				t.Fatal(err)
			}
			var source string
			for _, artifact := range plan.Artifacts {
				if artifact.Pruneable {
					source = artifact.Path
				}
			}
			tc.mutate(t, w, task, source)
			if _, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0)); err == nil || !strings.Contains(err.Error(), "stale") {
				t.Fatalf("changed evidence apply = %v, want stale refusal", err)
			}
			if _, err := os.Stat(source); err != nil {
				t.Fatalf("stale apply moved generated artifact: %v", err)
			}
			if _, err := os.Stat(checkout); err != nil {
				t.Fatalf("stale apply removed worktree: %v", err)
			}
		})
	}
}

func TestCleanupRestoreIsExactAndNeverOverwrites(t *testing.T) {
	for _, existingSource := range []bool{false, true} {
		t.Run(map[bool]string{false: "restore", true: "no-overwrite"}[existingSource], func(t *testing.T) {
			w, _, _, release := cleanupFixture(t)
			defer release()
			plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
			if err != nil {
				t.Fatal(err)
			}
			var artifact CleanupArtifact
			for _, candidate := range plan.Artifacts {
				if candidate.Pruneable {
					artifact = candidate
				}
			}
			if _, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0)); err != nil {
				t.Fatal(err)
			}
			if existingSource {
				if err := os.WriteFile(artifact.Path, []byte("new occupant\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			audit, err := RestoreRepositoryCleanupArtifact(w, "core", plan.ID, artifact.Identity, time.Unix(3, 0))
			if existingSource {
				if err == nil || !strings.Contains(err.Error(), "overwrite") {
					t.Fatalf("restore over existing source = %v", err)
				}
				raw, _ := os.ReadFile(artifact.Path)
				if string(raw) != "new occupant\n" {
					t.Fatalf("restore overwrote source: %q", raw)
				}
				if _, err := os.Stat(artifact.Quarantine); err != nil {
					t.Fatalf("refused restore lost quarantine: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(audit.Restored) != 1 || audit.Restored[0].Identity != artifact.Identity {
				t.Fatalf("restore missing from audit: %+v", audit)
			}
			raw, readErr := os.ReadFile(artifact.Path)
			if readErr != nil || string(raw) != "generated\n" {
				t.Fatalf("restored bytes = %q, err=%v", raw, readErr)
			}
			if _, err := os.Stat(artifact.Quarantine); !os.IsNotExist(err) {
				t.Fatalf("quarantine remains after restore: %v", err)
			}
		})
	}
}

func TestCleanupRestoreRefusesChangedQuarantineIdentity(t *testing.T) {
	w, _, _, release := cleanupFixture(t)
	defer release()
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	var artifact CleanupArtifact
	for _, candidate := range plan.Artifacts {
		if candidate.Pruneable {
			artifact = candidate
		}
	}
	if _, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact.Quarantine, []byte("GENERATED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreRepositoryCleanupArtifact(w, "core", plan.ID, artifact.Identity, time.Unix(3, 0)); err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatalf("restore changed quarantine = %v, want identity refusal", err)
	}
	if _, err := os.Stat(artifact.Path); !os.IsNotExist(err) {
		t.Fatalf("changed quarantine was restored: %v", err)
	}
	if _, err := os.Stat(artifact.Quarantine); err != nil {
		t.Fatalf("refused restore lost quarantined artifact: %v", err)
	}
}

func TestCleanupArtifactParentsRefuseSymlinkEscape(t *testing.T) {
	t.Run("quarantine parent", func(t *testing.T) {
		w, _, checkout, release := cleanupFixture(t)
		defer release()
		plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
		if err != nil {
			t.Fatal(err)
		}
		var artifact CleanupArtifact
		for _, candidate := range plan.Artifacts {
			if candidate.Pruneable {
				artifact = candidate
			}
		}
		external := t.TempDir()
		link := filepath.Join(w.Root, workspace.Dir, "quarantine")
		if err := os.Symlink(external, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0)); err == nil || !strings.Contains(err.Error(), "symlinked parent") {
			t.Fatalf("symlinked quarantine apply = %v, want canonical-parent refusal", err)
		}
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Fatalf("symlink refusal moved source: %v", err)
		}
		if _, err := os.Stat(checkout); err != nil {
			t.Fatalf("symlink refusal removed worktree: %v", err)
		}
		entries, err := os.ReadDir(external)
		if err != nil || len(entries) != 0 {
			t.Fatalf("symlink escape wrote outside workspace: entries=%v err=%v", entries, err)
		}
	})

	t.Run("run parent", func(t *testing.T) {
		w, _, _, release := cleanupFixture(t)
		defer release()
		plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
		if err != nil {
			t.Fatal(err)
		}
		var artifact CleanupArtifact
		for _, candidate := range plan.Artifacts {
			if candidate.Pruneable {
				artifact = candidate
			}
		}
		external := t.TempDir()
		runDir := w.RunDir(artifact.RunID)
		if err := os.RemoveAll(runDir); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(external, filepath.Base(artifact.Path)), []byte("generated\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(external, runDir); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if err := validateCleanupArtifactPaths(w, artifact, false); err == nil || !strings.Contains(err.Error(), "symlinked parent") {
			t.Fatalf("symlinked run parent validation = %v, want refusal", err)
		}
	})
}

func TestCleanupPreservesDirtyUnpushedAndNonterminalMaterial(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, *workspace.Workspace, *Task, string)
		check  func(CleanupItem) bool
	}{
		{"dirty", func(t *testing.T, _ *workspace.Workspace, _ *Task, checkout string) {
			if err := os.WriteFile(filepath.Join(checkout, "scratch"), []byte("keep"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, func(i CleanupItem) bool { return i.Dirty }},
		{"unpushed", func(t *testing.T, _ *workspace.Workspace, _ *Task, checkout string) {
			if err := os.WriteFile(filepath.Join(checkout, "product"), []byte("new local commit\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			worktreeGit(t, checkout, "commit", "-qam", "not pushed")
		}, func(i CleanupItem) bool { return i.Unpushed }},
		{"nonterminal", func(t *testing.T, w *workspace.Workspace, task *Task, _ string) {
			current, err := FindTask(w, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := MoveTask(w, current, model.StatusOpen); err != nil {
				t.Fatal(err)
			}
		}, func(i CleanupItem) bool { return i.TaskStatus != string(model.StatusDone) }},
		{"live-claims", func(t *testing.T, w *workspace.Workspace, task *Task, _ string) {
			if err := procmon.WriteRecord(filepath.Join(w.RunDir("01M14CLEANUP0000000000001"), "proc.txt"), procmon.Record{RunID: "01M14CLEANUP0000000000001", Task: task.ID, Child: "a-worker", Claims: []string{"internal/store"}}); err != nil {
				t.Fatal(err)
			}
		}, func(i CleanupItem) bool {
			return len(i.Runs) == 1 && i.Runs[0].State == "live" && len(i.Runs[0].Claims) == 1
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, task, checkout, restore := cleanupFixture(t)
			defer restore()
			tc.mutate(t, w, task, checkout)
			p, err := PlanRepositoryCleanup(w, "core", time.Unix(3, 0))
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Items) != 1 || p.Items[0].Eligible || !tc.check(p.Items[0]) {
				t.Fatalf("unsafe %s material was not preserved: %+v", tc.name, p.Items)
			}
		})
	}
}

func TestCleanupUnreadableRunEvidenceFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		corrupt func(*testing.T, *workspace.Workspace)
	}{
		{"malformed process record", func(t *testing.T, w *workspace.Workspace) {
			procPath := filepath.Join(w.RunDir("01M14CLEANUP0000000000001"), "proc.txt")
			if err := os.WriteFile(procPath, []byte("not a process record\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"unreadable runs directory", func(t *testing.T, w *workspace.Workspace) {
			if err := os.RemoveAll(w.RunsDir()); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(w.RunsDir(), []byte("not a directory\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _, _, restore := cleanupFixture(t)
			defer restore()
			tc.corrupt(t, w)
			plan, err := PlanRepositoryCleanup(w, "core", time.Unix(4, 0))
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Items) != 1 || !plan.Items[0].Unknown || plan.Items[0].Eligible || !strings.Contains(strings.Join(plan.Items[0].Reasons, " "), "run/claim evidence") {
				t.Fatalf("unreadable run evidence did not fail closed: %+v", plan.Items)
			}
			for _, artifact := range plan.Artifacts {
				if artifact.Pruneable {
					t.Fatalf("unreadable run evidence left artifact eligible: %+v", artifact)
				}
			}
		})
	}
}

func TestCleanupApplyRefusesStaleDirtyPlan(t *testing.T) {
	w, _, checkout, restore := cleanupFixture(t)
	defer restore()
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "untracked"), []byte("do not delete"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0)); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("changed plan apply = %v, want stale refusal", err)
	}
	if _, err := os.Stat(filepath.Join(checkout, "untracked")); err != nil {
		t.Fatalf("stale apply changed worktree: %v", err)
	}
}

func TestCleanupPreservesEveryUnsafeClassification(t *testing.T) {
	w, task, checkout, restore := cleanupFixture(t)
	defer restore()
	// Marking the caller's worktree protected must dominate every other proof.
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0), checkout)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 || !plan.Items[0].Protected || plan.Items[0].Eligible {
		t.Fatalf("protected fixture became eligible: %+v", plan.Items)
	}

	// GitHub outage, open PR, closed-unmerged PR, missing PR, ambiguous PR,
	// dirty/untracked, unpushed, and non-terminal task each fail closed.
	cases := []struct {
		name string
		prs  []DeliveryPR
		err  error
		want string
	}{
		{"open", []DeliveryPR{{DeliveryConfidence: "OPEN", HeadRefName: TaskBranch(task)}}, nil, "open"},
		{"closed-unmerged", []DeliveryPR{{DeliveryConfidence: "CLOSED", HeadRefName: TaskBranch(task)}}, nil, "closed"},
		{"missing", nil, nil, "missing"},
		{"ambiguous", []DeliveryPR{{DeliveryConfidence: "OPEN", HeadRefName: TaskBranch(task)}, {DeliveryConfidence: "CLOSED", HeadRefName: TaskBranch(task)}}, nil, "ambiguous"},
		{"unknown", nil, os.ErrPermission, "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ObserveDeliveryPRs = func(string) ([]DeliveryPR, error) { return tc.prs, tc.err }
			p, err := PlanRepositoryCleanup(w, "core", time.Unix(2, 0))
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Items) != 1 || p.Items[0].Eligible || p.Items[0].PRState != tc.want {
				t.Fatalf("unsafe %s classification: %+v", tc.name, p.Items)
			}
		})
	}
}

func TestCleanupClassifiesHistoricalPRAsSuperseded(t *testing.T) {
	w, task, _, restore := cleanupFixture(t)
	defer restore()
	base, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	head := base.Items[0].Commit
	merge := &struct {
		OID string `json:"oid"`
	}{OID: "merged"}
	ObserveDeliveryPRs = func(string) ([]DeliveryPR, error) {
		return []DeliveryPR{
			{Number: 4, DeliveryConfidence: "CLOSED", HeadRefName: TaskBranch(task), HeadRefOid: head},
			{Number: 9, DeliveryConfidence: "MERGED", HeadRefName: TaskBranch(task), HeadRefOid: head, MergeCommit: merge},
		}, nil
	}
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	item := plan.Items[0]
	if !item.Eligible || item.PRState != "merged" || len(item.PRHistory) != 2 || item.PRHistory[0].State != "superseded" || item.PRHistory[1].State != "merged" {
		t.Fatalf("superseded/current PR classification = %+v", item)
	}
}

func TestCleanupAuditDistinguishesPartialOperationFailure(t *testing.T) {
	w, _, _, restore := cleanupFixture(t)
	defer restore()
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	original := removeCleanupBranch
	removeCleanupBranch = func(string, string) error { return os.ErrPermission }
	t.Cleanup(func() { removeCleanupBranch = original })
	audit, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0))
	if err == nil {
		t.Fatal("branch failure reported cleanup success")
	}
	if len(audit.Removed) != 0 || len(audit.Operations) != 2 || audit.Operations[0].Operation != "artifact-quarantine" || audit.Operations[1].Operation != "worktree-remove" || audit.Operations[1].Target != plan.Items[0].Worktree {
		t.Fatalf("partial cleanup audit overstated completion: %+v", audit)
	}
	raw, readErr := os.ReadFile(filepath.Join(w.Root, workspace.Dir, "audit", "cleanup", plan.ID+".json"))
	if readErr != nil || !strings.Contains(string(raw), `"operation": "worktree-remove"`) || strings.Contains(string(raw), `"operation": "branch-delete"`) {
		t.Fatalf("persisted partial audit is untruthful: err=%v\n%s", readErr, raw)
	}
}

func TestCleanupArtifactPartialMoveHasTruthfulRecoveryAudit(t *testing.T) {
	w, _, checkout, restore := cleanupFixture(t)
	defer restore()
	second := filepath.Join(w.RunDir("01M14CLEANUP0000000000001"), "z-second.tmp")
	if err := os.WriteFile(second, []byte("second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanRepositoryCleanup(w, "core", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	original := moveCleanupArtifact
	moves := 0
	moveCleanupArtifact = func(source, target string) error {
		moves++
		if moves == 2 {
			return os.ErrPermission
		}
		return os.Rename(source, target)
	}
	t.Cleanup(func() { moveCleanupArtifact = original })
	audit, err := ApplyRepositoryCleanup(w, "core", plan.ID, time.Unix(2, 0))
	if err == nil {
		t.Fatal("partial artifact failure reported success")
	}
	if len(audit.Quarantined) != 1 || len(audit.Operations) != 1 || audit.Operations[0].Operation != "artifact-quarantine" || audit.Operations[0].Recovery == "" {
		t.Fatalf("partial artifact audit is not actionable: %+v", audit)
	}
	if _, err := os.Stat(audit.Quarantined[0].Quarantine); err != nil {
		t.Fatalf("completed quarantine target missing: %v", err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatalf("failed artifact move changed source: %v", err)
	}
	if _, err := os.Stat(checkout); err != nil {
		t.Fatalf("artifact failure continued into worktree cleanup: %v", err)
	}
	raw, readErr := os.ReadFile(filepath.Join(w.Root, workspace.Dir, "audit", "cleanup", plan.ID+".json"))
	if readErr != nil || strings.Count(string(raw), `"operation": "artifact-quarantine"`) != 1 || !strings.Contains(string(raw), `"recovery": "dacli cleanup`) {
		t.Fatalf("persisted partial audit is not truthful/actionable: err=%v\n%s", readErr, raw)
	}
}
