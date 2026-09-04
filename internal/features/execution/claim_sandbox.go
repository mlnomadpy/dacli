package execution

// Issue #1021: a commit-time claim check is too late. A provider may already
// have damaged an unrelated path before Git refuses its commit. Writable runs
// therefore execute in an independent disposable clone. Only an exact,
// re-observed set of claimed regular-file additions/modifications is projected
// into the canonical task checkout after the provider exits.

import (
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
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const (
	claimSandboxSchema = "claim-sandbox/v1"
	claimSandboxFile   = "claim-sandbox-v1.json"
)

type claimSandbox struct {
	Schema       string    `json:"schema"`
	RunID        string    `json:"run_id"`
	CanonicalDir string    `json:"canonical_dir"`
	SandboxDir   string    `json:"sandbox_dir"`
	BaseCommit   string    `json:"base_commit"`
	Branch       string    `json:"branch"`
	Claims       []string  `json:"claims"`
	CreatedAt    time.Time `json:"created_at"`
	ProjectedAt  time.Time `json:"projected_at,omitempty"`
	Paths        []string  `json:"projected_paths,omitempty"`
}

// taskWriteClaims keeps claim inference at the launch boundary: explicit
// durable claims win, while the existing path-token inference is only a
// conservative fallback. normalizeWriteClaims still validates the result
// against the actual assignment checkout before a provider starts.
func taskWriteClaims(w *workspace.Workspace, task *store.Task) []string {
	if claims := task.Claims(); len(claims) > 0 {
		return claims
	}
	return store.ClaimHints(w.Root, task)
}

func normalizeWriteClaims(root string, claims []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(claims))
	for _, raw := range claims {
		claim := strings.Trim(strings.TrimSpace(filepath.ToSlash(raw)), "/")
		claim = filepath.ToSlash(filepath.Clean(filepath.FromSlash(claim)))
		if claim == "" || claim == "." || claim == ".." || strings.HasPrefix(claim, "../") || filepath.IsAbs(raw) {
			return nil, fmt.Errorf("claim %q is not an exact repository-relative scope", raw)
		}
		if claim == workspace.Dir || strings.HasPrefix(claim, workspace.Dir+"/") {
			return nil, fmt.Errorf("claim %q targets dacli's shared control record", raw)
		}
		candidate := filepath.Join(root, filepath.FromSlash(claim))
		resolvedParent, err := filepath.EvalSymlinks(nearestExistingParent(candidate))
		if err != nil {
			return nil, fmt.Errorf("resolve claim %q: %w", raw, err)
		}
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			return nil, fmt.Errorf("resolve assignment root: %w", err)
		}
		rel, err := filepath.Rel(resolvedRoot, resolvedParent)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("claim %q resolves outside the assignment checkout", raw)
		}
		if !seen[claim] {
			seen[claim] = true
			out = append(out, claim)
		}
	}
	slices.Sort(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("writable assignment has no exact path claim")
	}
	return out, nil
}

func nearestExistingParent(path string) string {
	for candidate := filepath.Clean(path); ; candidate = filepath.Dir(candidate) {
		if _, err := os.Lstat(candidate); err == nil {
			return candidate
		}
		if filepath.Dir(candidate) == candidate {
			return candidate
		}
	}
}

func claimSandboxPath(w *workspace.Workspace, runID string) string {
	return filepath.Join(w.RunDir(runID), claimSandboxFile)
}

func prepareClaimSandbox(w *workspace.Workspace, runID, canonical string, claims []string, now time.Time) (claimSandbox, error) {
	var plan claimSandbox
	normalized, err := normalizeWriteClaims(canonical, claims)
	if err != nil {
		return plan, err
	}
	dirty, err := gitx.DirtyPaths(canonical, workspace.Dir)
	if err != nil {
		return plan, fmt.Errorf("inspect canonical assignment before claim sandbox: %w", err)
	}
	if len(dirty) != 0 {
		return plan, fmt.Errorf("canonical assignment must be clean before a claim sandbox launch: %s", strings.Join(dirty, ", "))
	}
	base, err := gitx.Run(canonical, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return plan, fmt.Errorf("resolve claim sandbox base: %w", err)
	}
	branch := strings.TrimSpace(gitx.CurrentBranch(canonical))
	if branch == "" || branch == "HEAD" {
		return plan, fmt.Errorf("claim sandbox requires an attached canonical task branch")
	}
	sandbox := filepath.Join(w.WorktreesDir(), ".claim-sandboxes", runID)
	if _, err := os.Stat(sandbox); err == nil {
		return plan, fmt.Errorf("claim sandbox already exists without a durable plan: %s", sandbox)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return plan, err
	}
	if err := os.MkdirAll(filepath.Dir(sandbox), 0o700); err != nil {
		return plan, err
	}
	if _, err := gitx.Run(w.Root, "clone", "--quiet", "--no-hardlinks", "--no-checkout", "--", canonical, sandbox); err != nil {
		return plan, fmt.Errorf("create independent claim sandbox: %w", err)
	}
	if _, err := gitx.Run(sandbox, "checkout", "--quiet", branch); err != nil {
		return plan, fmt.Errorf("checkout claim sandbox base: %w", err)
	}
	// A local clone records the canonical checkout as `origin`. Leaving that
	// remote would let an otherwise isolated worker mutate canonical refs with
	// `git push origin`, bypassing the projection transaction entirely.
	if _, err := gitx.Run(sandbox, "remote", "remove", "origin"); err != nil {
		return plan, fmt.Errorf("disconnect claim sandbox from canonical repository: %w", err)
	}
	checkedOut, err := gitx.Run(sandbox, "rev-parse", "--verify", "HEAD")
	if err != nil || strings.TrimSpace(checkedOut) != strings.TrimSpace(base) {
		return plan, fmt.Errorf("claim sandbox checkout does not match frozen base")
	}
	plan = claimSandbox{Schema: claimSandboxSchema, RunID: runID, CanonicalDir: filepath.Clean(canonical), SandboxDir: sandbox, BaseCommit: strings.TrimSpace(base), Branch: branch, Claims: normalized, CreatedAt: now.UTC()}
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return claimSandbox{}, err
	}
	if err := os.WriteFile(claimSandboxPath(w, runID), append(raw, '\n'), 0o600); err != nil {
		return claimSandbox{}, err
	}
	return plan, nil
}

func loadClaimSandbox(w *workspace.Workspace, runID string) (claimSandbox, error) {
	var plan claimSandbox
	raw, err := os.ReadFile(claimSandboxPath(w, runID))
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		return plan, err
	}
	if plan.Schema != claimSandboxSchema || plan.RunID != runID || plan.CanonicalDir == "" || plan.SandboxDir == "" || plan.BaseCommit == "" || len(plan.Claims) == 0 {
		return plan, fmt.Errorf("invalid %s", claimSandboxSchema)
	}
	return plan, nil
}

func sandboxChangedPaths(plan claimSandbox) ([]string, error) {
	raw, err := gitx.Run(plan.SandboxDir, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignored=matching")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, entry := range strings.Split(raw, "\x00") {
		if len(entry) < 4 {
			continue
		}
		status, path := entry[:2], filepath.ToSlash(strings.TrimSpace(entry[2:]))
		if strings.Contains(status, "R") || strings.Contains(status, "C") || strings.Contains(status, "D") {
			return nil, fmt.Errorf("claim sandbox refuses rename, copy, or delete at %s (%s)", path, status)
		}
		if path == ".git" || strings.HasPrefix(path, ".git/") || path == workspace.Dir || strings.HasPrefix(path, workspace.Dir+"/") {
			continue
		}
		if status == "!!" || status == "??" || strings.TrimSpace(status) != "" {
			if !seen[path] {
				seen[path] = true
				out = append(out, path)
			}
		}
	}
	// A worker may commit inside its independent repository, making status
	// clean. Compare the frozen base to its current worktree as well.
	diff, err := gitx.Run(plan.SandboxDir, "diff", "--name-only", "--no-renames", "-z", plan.BaseCommit, "--")
	if err != nil {
		return nil, err
	}
	for _, path := range strings.Split(diff, "\x00") {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path != "" && !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	slices.Sort(out)
	return out, nil
}

func pathInWriteClaims(path string, claims []string) bool {
	_, _, overlap := procmon.PathsOverlap([]string{path}, claims)
	return overlap
}

func projectClaimSandbox(w *workspace.Workspace, runID string, now time.Time) ([]string, error) {
	plan, err := loadClaimSandbox(w, runID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if !plan.ProjectedAt.IsZero() {
		return append([]string(nil), plan.Paths...), nil
	}
	head, err := gitx.Run(plan.CanonicalDir, "rev-parse", "--verify", "HEAD")
	if err != nil || strings.TrimSpace(head) != plan.BaseCommit || gitx.CurrentBranch(plan.CanonicalDir) != plan.Branch {
		return nil, fmt.Errorf("canonical assignment moved after claim sandbox launch")
	}
	dirty, err := gitx.DirtyPaths(plan.CanonicalDir, workspace.Dir)
	if err != nil {
		return nil, fmt.Errorf("inspect canonical assignment at claim projection: %w", err)
	}
	if len(dirty) != 0 {
		return nil, fmt.Errorf("canonical assignment is not clean at claim projection: paths=%v", dirty)
	}
	paths, err := sandboxChangedPaths(plan)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if !pathInWriteClaims(path, plan.Claims) {
			return nil, fmt.Errorf("worker wrote %s outside exact claims [%s]; canonical checkout was not mutated", path, strings.Join(plan.Claims, ", "))
		}
		source := filepath.Join(plan.SandboxDir, filepath.FromSlash(path))
		info, err := os.Lstat(source)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("claim sandbox refuses non-regular generated path %s", path)
		}
	}
	for _, path := range paths {
		source := filepath.Join(plan.SandboxDir, filepath.FromSlash(path))
		target := filepath.Join(plan.CanonicalDir, filepath.FromSlash(path))
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(source)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		tmp, err := os.CreateTemp(filepath.Dir(target), ".dacli-claim-project-*")
		if err != nil {
			return nil, err
		}
		name := tmp.Name()
		if _, err = tmp.Write(data); err == nil {
			err = tmp.Chmod(info.Mode().Perm())
		}
		if closeErr := tmp.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(name, target)
		}
		if err != nil {
			_ = os.Remove(name)
			return nil, err
		}
	}
	plan.ProjectedAt, plan.Paths = now.UTC(), append([]string(nil), paths...)
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(claimSandboxPath(w, runID), append(raw, '\n'), 0o600); err != nil {
		return nil, err
	}
	return paths, nil
}

func projectAndCommitClaimSandbox(w *workspace.Workspace, runID string, task *store.Task, childID, workDir string, now time.Time) ([]string, error) {
	paths, err := projectClaimSandbox(w, runID, now)
	if err != nil || len(paths) == 0 {
		return paths, err
	}
	handoff, required, err := store.CaptureRootHandoff(w, runID, task.ID, childID, workDir, store.RootHandoffRequest{
		Schema: store.RootHandoffSchema, FailedOperation: "claim sandbox parent commit", FailureClass: "filesystem_sandbox_refusal",
		NextAction: "parent creates the exact claim-bound commit before delivery continues",
	}, now)
	if err != nil {
		return nil, fmt.Errorf("capture claim sandbox parent commit: %w", err)
	}
	if !required {
		return paths, nil
	}
	_, resolved, err := applyParentCommitIfPlanned(w, handoff, []string{"git-metadata-write:claim-sandbox"}, now)
	if err != nil {
		return nil, fmt.Errorf("apply claim sandbox parent commit: %w", err)
	}
	if !resolved {
		return nil, fmt.Errorf("claim sandbox parent commit was not resolved")
	}
	return paths, nil
}
