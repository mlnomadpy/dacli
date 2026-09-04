package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const (
	ParentCommitRequestSchema = "parent-commit-request/v1"
	ParentCommitReceiptSchema = "parent-commit-receipt/v1"
	ParentCommitRequestFile   = "parent-commit-request-v1.json"
	ParentCommitReceiptFile   = "parent-commit-receipt-v1.json"
)

type ParentCommitRequest struct {
	Schema       string                    `json:"schema"`
	RequestID    string                    `json:"request_id"`
	TaskID       string                    `json:"task_id"`
	RunID        string                    `json:"run_id"`
	ChildID      string                    `json:"child_id"`
	Role         string                    `json:"role"`
	Worktree     string                    `json:"worktree"`
	Branch       string                    `json:"branch"`
	ParentCommit string                    `json:"parent_commit"`
	TreeOID      string                    `json:"tree_oid"`
	AllowedPaths []string                  `json:"allowed_paths"`
	ChangedPaths []RootHandoffPath         `json:"changed_paths"`
	DiffSHA256   string                    `json:"diff_sha256"`
	TreeSHA256   string                    `json:"tree_sha256"`
	AuthorName   string                    `json:"author_name"`
	AuthorEmail  string                    `json:"author_email"`
	Message      string                    `json:"message"`
	Trailers     []string                  `json:"trailers"`
	Verification []RootHandoffVerification `json:"verification"`
	RequestedAt  time.Time                 `json:"requested_at"`
}

type ParentCommitReceipt struct {
	Schema    string    `json:"schema"`
	RequestID string    `json:"request_id"`
	RunID     string    `json:"run_id"`
	TaskID    string    `json:"task_id"`
	Branch    string    `json:"branch"`
	Commit    string    `json:"commit"`
	TreeOID   string    `json:"tree_oid"`
	AppliedAt time.Time `json:"applied_at"`
}

func ParentCommitRequestPath(w *workspace.Workspace, runID string) string {
	return filepath.Join(w.RunDir(runID), ParentCommitRequestFile)
}

func ParentCommitReceiptPath(w *workspace.Workspace, runID string) string {
	return filepath.Join(w.RunDir(runID), ParentCommitReceiptFile)
}

func parentCommitID(req ParentCommitRequest) (string, error) {
	req.RequestID = ""
	raw, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateParentCommitRequest(req ParentCommitRequest) error {
	if req.Schema != ParentCommitRequestSchema || req.RequestID == "" || req.TaskID == "" || req.RunID == "" || req.ChildID == "" || req.Worktree == "" || req.Branch == "" || req.ParentCommit == "" || req.TreeOID == "" || len(req.AllowedPaths) == 0 || len(req.ChangedPaths) == 0 || req.AuthorName == "" || req.AuthorEmail == "" || req.Message == "" || len(req.Verification) == 0 || req.RequestedAt.IsZero() {
		return fmt.Errorf("invalid %s", ParentCommitRequestSchema)
	}
	want, err := parentCommitID(req)
	if err != nil || want != req.RequestID {
		return fmt.Errorf("invalid %s content identity", ParentCommitRequestSchema)
	}
	return nil
}

func inParentCommitClaim(path string, claims []string) bool {
	clean := strings.Trim(strings.TrimSpace(filepath.ToSlash(path)), "/")
	if clean == "" || clean == "." || clean == workspace.Dir || strings.HasPrefix(clean, workspace.Dir+"/") {
		return false
	}
	_, _, overlap := procmon.PathsOverlap([]string{clean}, claims)
	return overlap
}

func parentCommitTree(runDir, worktree, parent string, claims []string) (string, []string, error) {
	index, err := os.CreateTemp(runDir, ".parent-commit-index-*")
	if err != nil {
		return "", nil, err
	}
	indexPath := index.Name()
	if err := index.Close(); err != nil {
		return "", nil, err
	}
	if err := os.Remove(indexPath); err != nil {
		return "", nil, err
	}
	defer func() { _ = os.Remove(indexPath) }()
	env := []string{"GIT_INDEX_FILE=" + indexPath}
	if _, err := gitx.RunEnv(worktree, env, "read-tree", parent); err != nil {
		return "", nil, fmt.Errorf("initialize parent commit index: %w", err)
	}
	if _, err := gitx.RunEnv(worktree, env, "add", "-A", "--", "."); err != nil {
		return "", nil, fmt.Errorf("stage parent commit index: %w", err)
	}
	raw, err := gitx.RunEnv(worktree, env, "diff", "--cached", "--name-only", "--no-renames", "-z", parent, "--")
	if err != nil {
		return "", nil, fmt.Errorf("inspect parent commit index: %w", err)
	}
	var staged []string
	for _, path := range strings.Split(raw, "\x00") {
		if path = strings.TrimSpace(path); path == "" {
			continue
		}
		path = filepath.ToSlash(path)
		if !inParentCommitClaim(path, claims) {
			return "", nil, fmt.Errorf("parent commit refuses path %s outside claims [%s]", path, strings.Join(claims, ", "))
		}
		staged = append(staged, path)
	}
	if len(staged) == 0 {
		return "", nil, fmt.Errorf("parent commit request contains no claimed changes")
	}
	slices.Sort(staged)
	tree, err := gitx.RunEnv(worktree, env, "write-tree")
	if err != nil {
		return "", nil, fmt.Errorf("write parent commit tree: %w", err)
	}
	return strings.TrimSpace(tree), staged, nil
}

func BuildParentCommitRequest(w *workspace.Workspace, h RootHandoff, now time.Time) (ParentCommitRequest, error) {
	var req ParentCommitRequest
	if err := ReobserveRootHandoff(h); err != nil {
		return req, err
	}
	rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(h.RunID), "proc.txt"))
	if err != nil || rec.RunID != h.RunID || rec.Task != h.TaskID || rec.Child != h.ChildID {
		return req, fmt.Errorf("parent commit run identity does not match handoff")
	}
	task, err := FindTask(w, h.TaskID)
	if err != nil {
		return req, err
	}
	canonical := w.WorktreePath(task.Project, task.Seq, task.Slug)
	canonicalInfo, canonicalErr := os.Stat(canonical)
	worktreeInfo, worktreeErr := os.Stat(h.Worktree)
	if canonicalErr != nil || worktreeErr != nil || !os.SameFile(canonicalInfo, worktreeInfo) {
		return req, fmt.Errorf("parent commit worktree %s does not match canonical task worktree %s", h.Worktree, canonical)
	}
	branch := strings.TrimSpace(gitx.CurrentBranch(h.Worktree))
	wantBranch := fmt.Sprintf("dacli/%03d-%s", task.Seq, task.Slug)
	if branch != wantBranch {
		return req, fmt.Errorf("parent commit branch %s does not match task branch %s", branch, wantBranch)
	}
	parent, err := gitx.Run(h.Worktree, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return req, err
	}
	claims := append([]string(nil), rec.Claims...)
	if len(claims) == 0 {
		claims = ClaimHints(w.Root, task)
	}
	slices.Sort(claims)
	claims = slices.Compact(claims)
	if len(claims) == 0 {
		return req, fmt.Errorf("parent commit has no explicit or safely inferred path claim")
	}
	for _, path := range h.ChangedPaths {
		if !inParentCommitClaim(path.Path, claims) {
			return req, fmt.Errorf("parent commit refuses changed path %s outside claims [%s]", path.Path, strings.Join(claims, ", "))
		}
	}
	tree, _, err := parentCommitTree(w.RunDir(h.RunID), h.Worktree, parent, claims)
	if err != nil {
		return req, err
	}
	domain, prefix := w.Attribution()
	name := rec.Child
	if rec.Role != "" && rec.Role != "root" {
		name += " (" + rec.Role + ")"
	}
	message := strings.TrimSpace(h.CommitMessage)
	if message == "" {
		message = task.Title
	}
	trailers := []string{prefix + "-Agent: " + rec.Child, prefix + "-Task: " + fmt.Sprintf("%03d-%s", task.Seq, task.Slug)}
	if rec.Role != "" {
		trailers = append(trailers, prefix+"-Role: "+rec.Role)
	}
	verification := append([]RootHandoffVerification(nil), h.Verification...)
	if len(verification) == 0 {
		verification = []RootHandoffVerification{{Command: "configured parent verification after commit", ExitCode: -1, Result: "pending"}}
	}
	req = ParentCommitRequest{
		Schema: ParentCommitRequestSchema, TaskID: task.ID, RunID: rec.RunID, ChildID: rec.Child, Role: rec.Role,
		Worktree: h.Worktree, Branch: branch, ParentCommit: strings.TrimSpace(parent), TreeOID: tree,
		AllowedPaths: claims, ChangedPaths: append([]RootHandoffPath(nil), h.ChangedPaths...), DiffSHA256: h.DiffSHA256, TreeSHA256: h.TreeSHA256,
		AuthorName: name, AuthorEmail: rec.Child + domain, Message: message, Trailers: trailers,
		Verification: verification, RequestedAt: now.UTC(),
	}
	req.RequestID, err = parentCommitID(req)
	return req, err
}

func persistParentCommitRequest(w *workspace.Workspace, req ParentCommitRequest) error {
	path := ParentCommitRequestPath(w, req.RunID)
	if raw, err := os.ReadFile(path); err == nil {
		var existing ParentCommitRequest
		if json.Unmarshal(raw, &existing) != nil || validateParentCommitRequest(existing) != nil || existing.RequestID != req.RequestID {
			return fmt.Errorf("existing parent commit request differs; refuse overwrite")
		}
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	raw, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	return writeRootHandoffAtomic(path, append(raw, '\n'), 0o600)
}

func LoadParentCommitRequest(w *workspace.Workspace, runID string) (ParentCommitRequest, error) {
	var req ParentCommitRequest
	raw, err := os.ReadFile(ParentCommitRequestPath(w, runID))
	if err != nil {
		return req, err
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return req, err
	}
	return req, validateParentCommitRequest(req)
}

func parentCommitRequestMatchesHandoff(req ParentCommitRequest, h RootHandoff) bool {
	return req.RunID == h.RunID && req.TaskID == h.TaskID && req.ChildID == h.ChildID &&
		req.Worktree == h.Worktree && req.DiffSHA256 == h.DiffSHA256 && req.TreeSHA256 == h.TreeSHA256 &&
		slices.Equal(req.ChangedPaths, h.ChangedPaths)
}

func commitMessage(req ParentCommitRequest) string {
	return req.Message + "\n\n" + strings.Join(req.Trailers, "\n")
}

func expectedParentCommit(req ParentCommitRequest) (string, error) {
	env := []string{
		"GIT_AUTHOR_NAME=" + req.AuthorName, "GIT_AUTHOR_EMAIL=" + req.AuthorEmail,
		"GIT_COMMITTER_NAME=" + req.AuthorName, "GIT_COMMITTER_EMAIL=" + req.AuthorEmail,
		"GIT_AUTHOR_DATE=" + req.RequestedAt.Format(time.RFC3339Nano),
		"GIT_COMMITTER_DATE=" + req.RequestedAt.Format(time.RFC3339Nano),
	}
	return gitx.RunEnv(req.Worktree, env, "commit-tree", req.TreeOID, "-p", req.ParentCommit, "-m", commitMessage(req))
}

func loadParentCommitReceipt(w *workspace.Workspace, runID string) (ParentCommitReceipt, error) {
	var receipt ParentCommitReceipt
	raw, err := os.ReadFile(ParentCommitReceiptPath(w, runID))
	if err != nil {
		return receipt, err
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return receipt, err
	}
	if receipt.Schema != ParentCommitReceiptSchema || receipt.RunID != runID || receipt.RequestID == "" || receipt.Commit == "" {
		return receipt, fmt.Errorf("invalid %s", ParentCommitReceiptSchema)
	}
	return receipt, nil
}

// ApplyParentCommit turns an exact handoff into one deterministic commit. The
// request is durable before the ref mutation; a retry after any instruction
// boundary derives the same commit OID and either completes or refuses drift.
func ApplyParentCommit(w *workspace.Workspace, h RootHandoff, now time.Time) (ParentCommitReceipt, error) {
	var receipt ParentCommitReceipt
	lock := filepath.Join(w.RunDir(h.RunID), ".parent-commit.lock")
	err := WithFileLock(lock, func() error {
		if existing, err := loadParentCommitReceipt(w, h.RunID); err == nil {
			req, reqErr := LoadParentCommitRequest(w, h.RunID)
			if reqErr != nil || !parentCommitRequestMatchesHandoff(req, h) || existing.RequestID != req.RequestID || existing.TaskID != req.TaskID || existing.Branch != req.Branch || existing.TreeOID != req.TreeOID {
				return fmt.Errorf("parent commit receipt does not match its immutable request")
			}
			head, headErr := gitx.Run(h.Worktree, "rev-parse", "--verify", "refs/heads/"+existing.Branch)
			if headErr != nil || strings.TrimSpace(head) != existing.Commit {
				return fmt.Errorf("parent commit receipt no longer matches branch head")
			}
			tree, treeErr := gitx.Run(h.Worktree, "show", "-s", "--format=%T", existing.Commit)
			if treeErr != nil || strings.TrimSpace(tree) != existing.TreeOID {
				return fmt.Errorf("parent commit receipt no longer matches committed tree")
			}
			receipt = existing
			return writeParentCommitConsumption(w, h, existing, now)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		req, err := LoadParentCommitRequest(w, h.RunID)
		if errors.Is(err, fs.ErrNotExist) {
			req, err = BuildParentCommitRequest(w, h, now)
			if err == nil {
				err = persistParentCommitRequest(w, req)
			}
		}
		if err != nil {
			return err
		}
		if !parentCommitRequestMatchesHandoff(req, h) {
			return fmt.Errorf("parent commit request does not match the exact root handoff")
		}
		commit, err := expectedParentCommit(req)
		if err != nil {
			return fmt.Errorf("create parent-mediated commit object: %w", err)
		}
		commit = strings.TrimSpace(commit)
		head, err := gitx.Run(req.Worktree, "rev-parse", "--verify", "refs/heads/"+req.Branch)
		if err != nil {
			return err
		}
		head = strings.TrimSpace(head)
		switch head {
		case req.ParentCommit:
			if err := ReobserveRootHandoff(h); err != nil {
				return err
			}
			if _, err := gitx.Run(req.Worktree, "update-ref", "refs/heads/"+req.Branch, commit, req.ParentCommit); err != nil {
				return fmt.Errorf("atomically advance parent-mediated commit: %w", err)
			}
		case commit:
			// Crash recovery: the ref advanced after the durable request, but the
			// index/receipt boundary did not finish. Continue with the same OID.
		default:
			return fmt.Errorf("parent commit is stale: branch %s moved from %s to %s", req.Branch, req.ParentCommit, head)
		}
		if _, err := gitx.Run(req.Worktree, "reset", "--mixed", commit); err != nil {
			return fmt.Errorf("synchronize parent-mediated worktree index: %w", err)
		}
		dirty, err := gitx.DirtyPaths(req.Worktree, workspace.Dir)
		if err != nil {
			return fmt.Errorf("inspect parent-mediated worktree after commit: %w", err)
		}
		if len(dirty) > 0 {
			return fmt.Errorf("parent commit left unexpected worktree changes: %v", dirty)
		}
		receipt = ParentCommitReceipt{Schema: ParentCommitReceiptSchema, RequestID: req.RequestID, RunID: req.RunID, TaskID: req.TaskID, Branch: req.Branch, Commit: commit, TreeOID: req.TreeOID, AppliedAt: now.UTC()}
		raw, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			return err
		}
		if err := writeRootHandoffAtomic(ParentCommitReceiptPath(w, req.RunID), append(raw, '\n'), 0o644); err != nil {
			return err
		}
		return writeParentCommitConsumption(w, h, receipt, now)
	})
	return receipt, err
}

type parentCommitConsumption struct {
	Schema     string    `json:"schema"`
	RunID      string    `json:"run_id"`
	Actor      string    `json:"actor"`
	TreeSHA256 string    `json:"tree_sha256"`
	RequestID  string    `json:"parent_commit_request_id"`
	Commit     string    `json:"commit"`
	ConsumedAt time.Time `json:"consumed_at"`
}

func writeParentCommitConsumption(w *workspace.Workspace, h RootHandoff, receipt ParentCommitReceipt, now time.Time) error {
	path := filepath.Join(w.RunDir(h.RunID), RootHandoffConsumedFile)
	if existingRaw, err := os.ReadFile(path); err == nil {
		var existing parentCommitConsumption
		if json.Unmarshal(existingRaw, &existing) != nil || existing.Schema != "root-handoff-consumption/v1" || existing.RunID != h.RunID || existing.TreeSHA256 != h.TreeSHA256 || existing.RequestID != receipt.RequestID || existing.Commit != receipt.Commit {
			return fmt.Errorf("existing root handoff consumption does not match parent commit receipt")
		}
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	record := parentCommitConsumption{"root-handoff-consumption/v1", h.RunID, "a-root", h.TreeSHA256, receipt.RequestID, receipt.Commit, now.UTC()}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return writeRootHandoffAtomic(path, append(raw, '\n'), 0o644)
}
