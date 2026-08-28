package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const CleanupPlanSchema = "repository-cleanup/v1"

// CleanupItem is one managed worktree/branch pair. Eligible is deliberately
// derived from every proof below; absence or failure of any proof preserves it.
type CleanupItem struct {
	Worktree   string       `json:"worktree"`
	Branch     string       `json:"branch"`
	Task       string       `json:"task,omitempty"`
	TaskStatus string       `json:"task_status,omitempty"`
	Owner      string       `json:"owner,omitempty"`
	Runs       []CleanupRun `json:"runs,omitempty"`
	PRHistory  []CleanupPR  `json:"pr_history,omitempty"`
	PRState    string       `json:"pr_state"`
	Commit     string       `json:"commit,omitempty"`
	Protected  bool         `json:"protected"`
	Dirty      bool         `json:"dirty"`
	Unpushed   bool         `json:"unpushed"`
	Unknown    bool         `json:"unknown"`
	Eligible   bool         `json:"eligible"`
	Reasons    []string     `json:"reasons"`
	Operations []string     `json:"operations,omitempty"`
	Recovery   []string     `json:"recovery,omitempty"`
}

type CleanupRun struct {
	ID     string   `json:"id"`
	Agent  string   `json:"agent,omitempty"`
	State  string   `json:"state"`
	Claims []string `json:"claims,omitempty"`
}

type CleanupPR struct {
	Number int    `json:"number"`
	State  string `json:"state"`
}

type CleanupArtifact struct {
	Path           string `json:"path"`
	RunID          string `json:"run_id"`
	Task           string `json:"task,omitempty"`
	Classification string `json:"classification"`
	Pruneable      bool   `json:"pruneable"`
	Reason         string `json:"reason"`
}

// CleanupPlan is a content-addressed, versioned observation. ObservedAt is
// informational and intentionally excluded from ID so an unchanged rerun can
// apply exactly the plan the operator reviewed.
type CleanupPlan struct {
	Schema        string            `json:"schema"`
	Version       int               `json:"version"`
	ID            string            `json:"id"`
	Project       string            `json:"project"`
	Base          string            `json:"base"`
	BaseCommit    string            `json:"base_commit"`
	CurrentBranch string            `json:"current_branch"`
	ObservedAt    time.Time         `json:"observed_at"`
	Items         []CleanupItem     `json:"items"`
	Artifacts     []CleanupArtifact `json:"artifacts"`
}

// PlanRepositoryCleanup reuses ReclaimableWorktrees as the canonical local
// worktree classifier, then narrows it with remote and terminal evidence.
// Cleanup is never allowed to be more permissive than worktree prune.
func PlanRepositoryCleanup(w *workspace.Workspace, project string, now time.Time, protect ...string) (CleanupPlan, error) {
	p := CleanupPlan{Schema: CleanupPlanSchema, Version: 1, Project: project, ObservedAt: now.UTC(), Items: []CleanupItem{}, Artifacts: []CleanupArtifact{}}
	prj, err := LoadProject(w, project)
	if err != nil {
		return p, err
	}
	p.Base = prj.Landing.Base
	if p.Base == "" {
		p.Base = "main"
	}
	baseCommit, err := gitx.Run(w.Root, "rev-parse", p.Base)
	if err != nil {
		return p, fmt.Errorf("observe configured base %s: %w", p.Base, err)
	}
	p.BaseCommit = strings.TrimSpace(baseCommit)
	prs, ghErr := ObserveDeliveryPRs(w.Root)
	wts, wtErr := gitx.ListWorktrees(w.Root)
	if wtErr != nil {
		return p, fmt.Errorf("observe worktrees: %w", wtErr)
	}
	reclaimable, reclaimErr := ReclaimableWorktrees(w, p.Base)
	reclaim := map[string]ReclaimableWorktree{}
	if reclaimErr == nil {
		for _, c := range reclaimable {
			reclaim[cleanPath(c.Path)] = c
		}
	}
	tasks, taskErr := ListTasks(w, project, "")
	byBranch := map[string]*Task{}
	taskByID := map[string]*Task{}
	if taskErr == nil {
		for _, t := range tasks {
			byBranch[TaskBranch(t)] = t
			taskByID[t.ID] = t
		}
	}
	runsByTask, artifacts, runsErr := observeCleanupRuns(w, taskByID)
	p.Artifacts = artifacts
	current := gitx.CurrentBranch(w.Root)
	p.CurrentBranch = current
	protectedPaths := map[string]bool{cleanPath(w.Root): true}
	for _, path := range protect {
		if path != "" {
			protectedPaths[cleanPath(path)] = true
		}
	}
	managedRoot := cleanPath(w.WorktreesDir())
	for _, wt := range wts {
		path := cleanPath(wt.Path)
		if path != managedRoot && !strings.HasPrefix(path, managedRoot+string(filepath.Separator)) {
			continue
		}
		item := CleanupItem{Worktree: wt.Path, Branch: wt.Branch, PRState: "missing", Reasons: []string{}}
		if t := byBranch[wt.Branch]; t != nil {
			item.Task, item.TaskStatus = t.ID, string(t.Status)
			item.Owner, item.Runs = t.Owner(), runsByTask[t.ID]
		}
		if runsErr != nil {
			item.Unknown = true
			item.Reasons = append(item.Reasons, "run/claim evidence is unobservable: "+runsErr.Error())
		}
		item.Protected = protectedPaths[path] || wt.Branch == current || wt.Branch == p.Base
		if item.Protected {
			item.Reasons = append(item.Reasons, "protected current/base worktree")
		}
		dirty, dirtyErr := gitx.DirtyPaths(wt.Path)
		item.Dirty = dirtyErr != nil || len(dirty) > 0
		if item.Dirty {
			item.Reasons = append(item.Reasons, "dirty or untracked worktree")
		}
		commit, commitErr := gitx.Run(w.Root, "rev-parse", wt.Branch)
		item.Commit = strings.TrimSpace(commit)
		remoteHead := ""
		if ghErr != nil {
			item.Unknown = true
			item.PRState = "unknown"
			item.Reasons = append(item.Reasons, "GitHub PR state is unobservable")
		} else {
			var matches []DeliveryPR
			for _, pr := range prs {
				if pr.HeadRefName == wt.Branch {
					matches = append(matches, pr)
				}
			}
			if len(matches) > 0 {
				sort.Slice(matches, func(i, j int) bool { return matches[i].Number < matches[j].Number })
				canonicalPR := matches[len(matches)-1]
				for i, pr := range matches {
					state := cleanupPRState(pr)
					if i < len(matches)-1 {
						state = "superseded"
					}
					item.PRHistory = append(item.PRHistory, CleanupPR{Number: pr.Number, State: state})
				}
				if len(matches) > 1 && matches[len(matches)-2].Number == canonicalPR.Number {
					item.Unknown, item.PRState = true, "ambiguous"
					item.Reasons = append(item.Reasons, "multiple canonical PR observations share the latest number")
				} else {
					item.PRState = cleanupPRState(canonicalPR)
					remoteHead = canonicalPR.HeadRefOid
				}
			}
		}
		// The PR head OID is the canonical pushed snapshot and remains useful
		// after GitHub deletes the merged remote branch. Fall back to the local
		// remote-tracking ref only when GitHub omitted it; any failure preserves.
		if remoteHead == "" {
			remoteCommit, remoteErr := gitx.Run(w.Root, "rev-parse", "refs/remotes/origin/"+wt.Branch)
			if remoteErr != nil {
				item.Unknown = true
				item.Reasons = append(item.Reasons, "remote branch is unobservable")
			} else {
				remoteHead = strings.TrimSpace(remoteCommit)
			}
		}
		item.Unpushed = commitErr == nil && remoteHead != "" && remoteHead != item.Commit
		if item.Unpushed {
			item.Reasons = append(item.Reasons, "branch contains unpushed commits")
		}
		if item.PRState != "merged" {
			item.Reasons = append(item.Reasons, "PR state is "+item.PRState)
		}
		if item.Task == "" || item.TaskStatus != string(model.StatusDone) {
			item.Reasons = append(item.Reasons, "task is missing or non-terminal")
		}
		runsTerminal := true
		for _, run := range item.Runs {
			if run.State != "terminal" || len(run.Claims) > 0 {
				runsTerminal = false
				item.Unknown = item.Unknown || run.State == "unknown"
				item.Reasons = append(item.Reasons, "run "+run.ID+" is "+run.State+" or still owns claims")
			}
		}
		canonical := reclaim[path]
		if !canonical.Merged {
			item.Reasons = append(item.Reasons, "canonical worktree classifier has not proven a merge")
		}
		if commitErr != nil || reclaimErr != nil || taskErr != nil {
			item.Unknown = true
			item.Reasons = append(item.Reasons, "local cleanup evidence is incomplete")
		}
		item.Reasons = uniqueStrings(item.Reasons)
		item.Eligible = !item.Protected && !item.Dirty && !item.Unpushed && !item.Unknown && runsTerminal && item.PRState == "merged" && item.TaskStatus == string(model.StatusDone) && canonical.Merged
		if item.Eligible {
			item.Reasons = []string{"merged, clean, pushed, non-protected, terminal task"}
			item.Operations = []string{"git worktree remove -- " + wt.Path, "git branch -d -- " + wt.Branch}
			item.Recovery = []string{"git branch recovered/" + safeRecoveryRef(wt.Branch) + " " + item.Commit, "git worktree add <path> " + item.Commit}
		}
		p.Items = append(p.Items, item)
	}
	sort.Slice(p.Items, func(i, j int) bool { return p.Items[i].Worktree < p.Items[j].Worktree })
	sort.Slice(p.Artifacts, func(i, j int) bool { return p.Artifacts[i].Path < p.Artifacts[j].Path })
	p.ID = cleanupPlanID(p)
	return p, nil
}

func cleanupPRState(pr DeliveryPR) string {
	state := strings.ToLower(pr.DeliveryConfidence)
	if state == "closed" && pr.MergeCommit != nil {
		return "merged"
	}
	return state
}

func observeCleanupRuns(w *workspace.Workspace, tasks map[string]*Task) (map[string][]CleanupRun, []CleanupArtifact, error) {
	byTask := map[string][]CleanupRun{}
	var artifacts []CleanupArtifact
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return byTask, artifacts, nil
		}
		return byTask, artifacts, fmt.Errorf("read runs directory: %w", err)
	}
	var observeErr error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runID, runDir := entry.Name(), w.RunDir(entry.Name())
		rec, recErr := procmon.ReadRecord(filepath.Join(runDir, "proc.txt"))
		if recErr == nil && (rec.RunID == "" || rec.Task == "") {
			recErr = fmt.Errorf("missing run/task identity")
		}
		run := CleanupRun{ID: runID, State: "unknown"}
		if recErr == nil {
			run.Agent, run.Claims = rec.Child, append([]string(nil), rec.Claims...)
			switch {
			case rec.Outcome != "" && len(rec.Claims) == 0:
				run.State = "terminal"
			case procmon.AliveIdentity(rec.PID, rec.PIDStart) || len(rec.Claims) > 0:
				run.State = "live"
			}
			byTask[rec.Task] = append(byTask[rec.Task], run)
		} else {
			observeErr = fmt.Errorf("read run %s process evidence: %w", runID, recErr)
		}
		files, fileErr := os.ReadDir(runDir)
		if fileErr != nil {
			observeErr = fmt.Errorf("read run %s artifacts: %w", runID, fileErr)
			continue
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			a := CleanupArtifact{Path: filepath.Join(runDir, file.Name()), RunID: runID, Task: rec.Task, Classification: "durable-evidence", Reason: "run records and transcripts are retained as durable evidence"}
			if strings.HasSuffix(file.Name(), ".tmp") && recErr == nil && run.State == "terminal" {
				if t := tasks[rec.Task]; t != nil && t.Status == model.StatusDone {
					a.Classification, a.Pruneable = "generated-run-artifact", true
					a.Reason = "terminal task and run; generated temporary artifact is classified safe but preserved until a recoverable quarantine operation is available"
				}
			}
			artifacts = append(artifacts, a)
		}
	}
	return byTask, artifacts, observeErr
}

func cleanupPlanID(p CleanupPlan) string {
	p.ID = ""
	p.ObservedAt = time.Time{}
	raw, _ := json.Marshal(p)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func safeRecoveryRef(branch string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(branch)
	return strings.Trim(r, "-")
}

type CleanupAudit struct {
	Schema     string                  `json:"schema"`
	PlanID     string                  `json:"plan_id"`
	Project    string                  `json:"project"`
	AppliedAt  time.Time               `json:"applied_at"`
	Planned    []CleanupItem           `json:"planned"`
	Removed    []CleanupItem           `json:"removed"`
	Operations []CleanupOperationAudit `json:"completed_operations"`
}

type CleanupOperationAudit struct {
	Operation   string    `json:"operation"`
	Target      string    `json:"target"`
	CompletedAt time.Time `json:"completed_at"`
}

var (
	removeCleanupWorktree = gitx.RemoveCleanWorktree
	removeCleanupBranch   = func(root, branch string) error { _, err := gitx.Run(root, "branch", "-d", "--", branch); return err }
)

// ApplyRepositoryCleanup executes only the operations enumerated by a fresh,
// matching plan. Worktrees are removed through git, never recursive deletion;
// branches use non-forcing -d and retain their exact recovery commit in audit.
func ApplyRepositoryCleanup(w *workspace.Workspace, project, requestedID string, now time.Time, protect ...string) (CleanupAudit, error) {
	var audit CleanupAudit
	err := WithFileLock(filepath.Join(w.Root, workspace.Dir, ".cleanup.lock"), func() error {
		p, err := PlanRepositoryCleanup(w, project, now, protect...)
		if err != nil {
			return err
		}
		if requestedID == "" || requestedID != p.ID || cleanupPlanID(p) != p.ID {
			return fmt.Errorf("cleanup plan is stale or unknown; review a new dry-run")
		}
		for _, item := range p.Items {
			if item.Eligible {
				audit.Planned = append(audit.Planned, item)
			}
		}
		audit.Schema, audit.PlanID, audit.Project, audit.AppliedAt = "repository-cleanup-audit/v1", p.ID, p.Project, now.UTC()
		audit.Removed = []CleanupItem{}
		auditPath, err := writeCleanupAudit(w, audit)
		if err != nil {
			return err // prove the recovery ledger is writable before mutation
		}
		for _, item := range audit.Planned {
			if err := removeCleanupWorktree(w.Root, item.Worktree); err != nil {
				return fmt.Errorf("remove worktree %s (audit %s): %w", item.Worktree, auditPath, err)
			}
			audit.Operations = append(audit.Operations, CleanupOperationAudit{Operation: "worktree-remove", Target: item.Worktree, CompletedAt: time.Now().UTC()})
			if _, err := writeCleanupAudit(w, audit); err != nil {
				return fmt.Errorf("record removed worktree in %s: %w", auditPath, err)
			}
			if item.Branch != "" {
				if err := removeCleanupBranch(w.Root, item.Branch); err != nil {
					return fmt.Errorf("remove merged branch %s (worktree recovery recorded in %s): %w", item.Branch, auditPath, err)
				}
				audit.Operations = append(audit.Operations, CleanupOperationAudit{Operation: "branch-delete", Target: item.Branch, CompletedAt: time.Now().UTC()})
			}
			audit.Removed = append(audit.Removed, item)
			if _, err := writeCleanupAudit(w, audit); err != nil {
				return fmt.Errorf("record completed cleanup in %s: %w", auditPath, err)
			}
		}
		_, err = writeCleanupAudit(w, audit)
		return err
	})
	return audit, err
}

func writeCleanupAudit(w *workspace.Workspace, audit CleanupAudit) (string, error) {
	raw, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return "", err
	}
	dir := filepath.Join(w.Root, workspace.Dir, "audit", "cleanup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, audit.PlanID+".json")
	return path, mdstore.WriteBytes(path, append(raw, '\n'), 0o644)
}
