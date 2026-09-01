// Package store owns the durable root-handoff contract shared by execution,
// reconciliation, and loop recovery. A handoff is evidence, not worker prose:
// every material path is hashed and consumption re-observes the worktree before
// trusting the record (issue #874).
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
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const (
	RootHandoffSchema       = "root-handoff/v1"
	RootHandoffFile         = "root-handoff-v1.json"
	RootHandoffRequestFile  = "root-handoff-request-v1.json"
	RootHandoffConsumedFile = "root-handoff-consumed-v1.json"
)

type RootHandoffPath struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   string `json:"mode"`
}

type RootHandoffVerification struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Result   string `json:"result,omitempty"`
}

// RootHandoffRequest is the narrow structured channel a worker may write when
// it cannot publish its own lifecycle result. Identity and filesystem evidence
// are deliberately absent: CaptureRootHandoff derives those independently.
type RootHandoffRequest struct {
	Schema          string                    `json:"schema"`
	Verification    []RootHandoffVerification `json:"verification"`
	Unresolved      []string                  `json:"unresolved_findings"`
	FailedOperation string                    `json:"failed_operation"`
	FailureClass    string                    `json:"failure_class"`
	Stderr          string                    `json:"stderr"`
	NextAction      string                    `json:"safe_owner_next_action"`
}

type RootHandoff struct {
	Schema          string                    `json:"schema"`
	Version         int                       `json:"version"`
	TaskID          string                    `json:"task_id"`
	RunID           string                    `json:"run_id"`
	ChildID         string                    `json:"child_id"`
	Worktree        string                    `json:"worktree"`
	ChangedPaths    []RootHandoffPath         `json:"changed_paths"`
	DiffSHA256      string                    `json:"diff_sha256"`
	TreeSHA256      string                    `json:"tree_sha256"`
	Verification    []RootHandoffVerification `json:"verification"`
	Unresolved      []string                  `json:"unresolved_findings"`
	FailedOperation string                    `json:"failed_operation"`
	FailureClass    string                    `json:"failure_class"`
	Stderr          string                    `json:"stderr"`
	NextAction      string                    `json:"safe_owner_next_action"`
	CreatedAt       time.Time                 `json:"created_at"`
}

func RootHandoffPathForRun(w *workspace.Workspace, runID string) string {
	return filepath.Join(w.RunDir(runID), RootHandoffFile)
}

func LoadRootHandoff(w *workspace.Workspace, runID string) (RootHandoff, error) {
	var h RootHandoff
	if !workspace.SafeSegment(runID) {
		return h, fmt.Errorf("invalid root handoff run id %q", runID)
	}
	raw, err := os.ReadFile(RootHandoffPathForRun(w, runID))
	if err != nil {
		return h, err
	}
	if err := json.Unmarshal(raw, &h); err != nil {
		return h, fmt.Errorf("decode root handoff: %w", err)
	}
	if h.Schema != RootHandoffSchema || h.Version != 1 || h.RunID != runID || h.TaskID == "" || h.Worktree == "" {
		return h, fmt.Errorf("invalid or unsupported root handoff for run %s", runID)
	}
	resolved, err := workspace.Find(h.Worktree)
	if err != nil || resolved.Root != w.Root {
		return h, fmt.Errorf("root handoff worktree does not resolve to workspace %s", w.Root)
	}
	return h, nil
}

func readRootHandoffRequest(runDir string) (RootHandoffRequest, bool, error) {
	var req RootHandoffRequest
	raw, err := os.ReadFile(filepath.Join(runDir, RootHandoffRequestFile))
	if errors.Is(err, fs.ErrNotExist) {
		return req, false, nil
	}
	if err != nil {
		return req, false, err
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return req, false, fmt.Errorf("decode root handoff request: %w", err)
	}
	if req.Schema != RootHandoffSchema || strings.TrimSpace(req.FailedOperation) == "" || strings.TrimSpace(req.NextAction) == "" {
		return req, false, fmt.Errorf("invalid root handoff request: schema, failed_operation, and safe_owner_next_action are required")
	}
	return req, true, nil
}

func RootHandoffRequested(w *workspace.Workspace, runID string) bool {
	_, err := os.Stat(filepath.Join(w.RunDir(runID), RootHandoffRequestFile))
	return err == nil
}

// WriteRootHandoffRequest persists the narrow worker-to-owner fallback channel.
// The parent may use it after a provider completed useful analysis but the
// governed result publication failed; CaptureRootHandoff independently binds
// the request to run, task, worktree, and current filesystem evidence.
func WriteRootHandoffRequest(w *workspace.Workspace, runID string, req RootHandoffRequest) error {
	if !workspace.SafeSegment(runID) {
		return fmt.Errorf("invalid root handoff run id %q", runID)
	}
	if req.Schema != RootHandoffSchema || strings.TrimSpace(req.FailedOperation) == "" || strings.TrimSpace(req.NextAction) == "" {
		return fmt.Errorf("invalid root handoff request: schema, failed_operation, and safe_owner_next_action are required")
	}
	raw, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	return writeRootHandoffAtomic(filepath.Join(w.RunDir(runID), RootHandoffRequestFile), append(raw, '\n'), 0o600)
}

func digestFile(root, rel string) (RootHandoffPath, error) {
	clean := filepath.Clean(rel)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return RootHandoffPath{}, fmt.Errorf("unsafe handoff path %q", rel)
	}
	abs := filepath.Join(root, clean)
	info, err := os.Lstat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return RootHandoffPath{Path: filepath.ToSlash(clean), SHA256: "deleted", Mode: "deleted"}, nil
	}
	if err != nil {
		return RootHandoffPath{}, err
	}
	var payload []byte
	mode := info.Mode().String()
	if info.Mode()&os.ModeSymlink != 0 {
		target, readErr := os.Readlink(abs)
		if readErr != nil {
			return RootHandoffPath{}, readErr
		}
		payload = []byte("symlink\x00" + target)
	} else if info.Mode().IsRegular() {
		payload, err = os.ReadFile(abs)
		if err != nil {
			return RootHandoffPath{}, err
		}
	} else {
		return RootHandoffPath{}, fmt.Errorf("handoff path %s is not a regular file or symlink", clean)
	}
	sum := sha256.Sum256(payload)
	return RootHandoffPath{Path: filepath.ToSlash(clean), SHA256: hex.EncodeToString(sum[:]), Mode: mode}, nil
}

func observeRootHandoff(worktree string) ([]RootHandoffPath, string, string, error) {
	paths, err := gitx.DirtyPaths(worktree, workspace.Dir)
	if err != nil {
		return nil, "", "", fmt.Errorf("observe handoff paths: %w", err)
	}
	sort.Strings(paths)
	observed := make([]RootHandoffPath, 0, len(paths))
	for _, path := range paths {
		item, err := digestFile(worktree, path)
		if err != nil {
			return nil, "", "", fmt.Errorf("hash handoff path %s: %w", path, err)
		}
		observed = append(observed, item)
	}
	diff, err := gitx.Run(worktree, "diff", "--binary", "HEAD", "--")
	if err != nil {
		return nil, "", "", fmt.Errorf("observe handoff diff: %w", err)
	}
	diffSum := sha256.Sum256([]byte(diff))
	h := sha256.New()
	for _, item := range observed {
		_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00", item.Path, item.Mode, item.SHA256)
	}
	return observed, hex.EncodeToString(diffSum[:]), hex.EncodeToString(h.Sum(nil)), nil
}

// CaptureRootHandoff materializes exact worktree evidence. It returns false
// when neither changed work nor a structured request exists, so an ordinary
// empty run is not mislabeled as a lifecycle handoff.
func CaptureRootHandoff(w *workspace.Workspace, runID, taskID, childID, worktree string, fallback RootHandoffRequest, now time.Time) (RootHandoff, bool, error) {
	var h RootHandoff
	req, requested, err := readRootHandoffRequest(w.RunDir(runID))
	if err != nil {
		return h, false, err
	}
	if !requested {
		req = fallback
	}
	paths, diffDigest, treeDigest, err := observeRootHandoff(worktree)
	if err != nil {
		return h, false, err
	}
	if len(paths) == 0 && !requested {
		return h, false, nil
	}
	if strings.TrimSpace(req.FailedOperation) == "" {
		req.FailedOperation = "worker lifecycle publication"
	}
	if strings.TrimSpace(req.FailureClass) == "" {
		req.FailureClass = "filesystem_sandbox_refusal"
	}
	if strings.TrimSpace(req.NextAction) == "" {
		req.NextAction = "owner re-observes this handoff, reruns verification, then commits and publishes the recorded work without changing worker grant or harness"
	}
	h = RootHandoff{
		Schema: RootHandoffSchema, Version: 1, TaskID: taskID, RunID: runID, ChildID: childID,
		Worktree: worktree, ChangedPaths: paths, DiffSHA256: diffDigest, TreeSHA256: treeDigest,
		Verification: append([]RootHandoffVerification{}, req.Verification...), Unresolved: append([]string{}, req.Unresolved...), FailedOperation: req.FailedOperation,
		FailureClass: req.FailureClass, Stderr: req.Stderr, NextAction: req.NextAction, CreatedAt: now.UTC(),
	}
	raw, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return RootHandoff{}, false, err
	}
	raw = append(raw, '\n')
	if err := writeRootHandoffAtomic(RootHandoffPathForRun(w, runID), raw, 0o644); err != nil {
		return RootHandoff{}, false, fmt.Errorf("write root handoff: %w", err)
	}
	return h, true, nil
}

// ReobserveRootHandoff refuses stale material. No owner action is safe until
// both the path set and every content/diff digest still match.
func ReobserveRootHandoff(h RootHandoff) error {
	paths, diffDigest, treeDigest, err := observeRootHandoff(h.Worktree)
	if err != nil {
		return err
	}
	left, _ := json.Marshal(paths)
	right, _ := json.Marshal(h.ChangedPaths)
	if string(left) != string(right) || diffDigest != h.DiffSHA256 || treeDigest != h.TreeSHA256 {
		return fmt.Errorf("root handoff is stale: worktree paths or hashes changed; inspect fresh state and create a new handoff")
	}
	return nil
}

func MarkRootHandoffConsumed(w *workspace.Workspace, h RootHandoff, actor string, now time.Time) error {
	if err := ReobserveRootHandoff(h); err != nil {
		return err
	}
	record := struct {
		Schema     string    `json:"schema"`
		RunID      string    `json:"run_id"`
		Actor      string    `json:"actor"`
		TreeSHA256 string    `json:"tree_sha256"`
		ConsumedAt time.Time `json:"consumed_at"`
	}{"root-handoff-consumption/v1", h.RunID, actor, h.TreeSHA256, now.UTC()}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return writeRootHandoffAtomic(filepath.Join(w.RunDir(h.RunID), RootHandoffConsumedFile), append(raw, '\n'), 0o644)
}

func writeRootHandoffAtomic(path string, raw []byte, mode fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".root-handoff-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	ok = true
	return nil
}
