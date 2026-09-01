package execution

// Issue #874: launch compatibility is not mutation compatibility. A coding
// harness can edit source while its sandbox refuses .git or the shared event
// journal. Probe each concrete lifecycle capability before minting a worker,
// distinguish delegated publication from required worker authority, and never
// widen the selected grant or harness to make a probe pass.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

type mutationCapabilityResult struct {
	Capability  string `json:"capability"`
	Disposition string `json:"disposition"` // pass|required_refusal|planned_handoff
	Class       string `json:"class"`
	Detail      string `json:"detail"`
}

var (
	probeMutationDirectory = probeDirectoryMutation
	probeMutationGitLock   = probeGitMutationLock
	probeMutationCommand   = execMutationProbe
)

func probeDirectoryMutation(dir, prefix string) error {
	f, err := os.CreateTemp(dir, prefix)
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := f.WriteString("dacli mutation capability probe\n"); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}

func probeGitMutationLock(indexPath string) error {
	lock := indexPath + ".lock"
	f, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(lock) }()
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Remove(lock)
}

func execMutationProbe(dir, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := commandresult.Run(cmd, commandresult.RunOptions{
		Operation: "mutation capability probe", WorkspaceRoot: dir,
		TimedOut: func() bool { return ctx.Err() == context.DeadlineExceeded },
	})
	if err != nil {
		return fmt.Errorf("mutation capability probe: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func mutationFailureClass(err error) string {
	if err == nil {
		return "ok"
	}
	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return "missing_tool"
	}
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS) {
		return "filesystem_sandbox_refusal"
	}
	if errors.Is(err, fs.ErrExist) || errors.Is(err, syscall.EBUSY) || errors.Is(err, syscall.EAGAIN) {
		return "transient_contention"
	}
	lower := strings.ToLower(err.Error())
	for _, marker := range []string{"authentication", "not logged", "login", "network", "connection", "dns", "could not resolve", "timed out"} {
		if strings.Contains(lower, marker) {
			return "authentication_network_failure"
		}
	}
	for _, marker := range []string{"policy", "not allowed", "operation not permitted", "denied by"} {
		if strings.Contains(lower, marker) {
			return "policy_refusal"
		}
	}
	return "filesystem_sandbox_refusal"
}

func nearestExistingDir(root, claim string) (string, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve assignment worktree %s: %w", root, err)
	}
	resolvedRoot = filepath.Clean(resolvedRoot)
	contained := func(dir string) (string, error) {
		resolved, resolveErr := filepath.EvalSymlinks(dir)
		if resolveErr != nil {
			return "", resolveErr
		}
		resolved = filepath.Clean(resolved)
		rel, relErr := filepath.Rel(resolvedRoot, resolved)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("resolved probe directory %s escapes assignment worktree %s", resolved, resolvedRoot)
		}
		return resolved, nil
	}
	clean := filepath.Clean(claim)
	if clean == "." {
		return contained(root)
	}
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("claim %q escapes the assignment worktree", claim)
	}
	target := filepath.Join(root, clean)
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return contained(target)
	}
	dir := filepath.Dir(target)
	for {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			resolved, containErr := contained(dir)
			if containErr != nil {
				return "", fmt.Errorf("claim %q has no writable parent inside the worktree: %w", claim, containErr)
			}
			return resolved, nil
		}
		if dir == root || filepath.Dir(dir) == dir {
			return "", fmt.Errorf("claim %q has no existing parent inside the worktree", claim)
		}
		dir = filepath.Dir(dir)
	}
}

var backtickCommand = regexp.MustCompile("`([^`\\n]+)`")

var declaredBuildExecutables = map[string]bool{
	"bun": true, "cargo": true, "cmake": true, "dotnet": true, "go": true,
	"gradle": true, "gradlew": true, "java": true, "make": true, "mvn": true,
	"npm": true, "npx": true, "pnpm": true, "python": true, "python3": true,
	"pytest": true, "swift": true, "xcodebuild": true, "yarn": true,
}

func declaredVerificationTools(t *store.Task) []string {
	seen := map[string]bool{}
	var out []string
	for _, criterion := range t.Acceptance() {
		for _, match := range backtickCommand.FindAllStringSubmatch(criterion.Text, -1) {
			fields := strings.Fields(match[1])
			if len(fields) < 2 { // a single code token is normally a path or symbol
				continue
			}
			tool := strings.TrimPrefix(fields[0], "./")
			if tool == "" || !declaredBuildExecutables[tool] || seen[tool] {
				continue
			}
			seen[tool] = true
			out = append(out, tool)
		}
	}
	sort.Strings(out)
	return out
}

func mutationPreflight(p *launchPlan) ([]mutationCapabilityResult, error) {
	if p.Grant == model.GrantRO {
		results := []mutationCapabilityResult{{Capability: "source-write", Disposition: "pass", Class: "policy_refusal", Detail: "not required by read-only assignment"}}
		if p.f != nil && p.f.Bool("structured-review-result") {
			if err := probeMutationDirectory(p.w.EventsDir(), ".dacli-review-result-probe-*"); err != nil {
				class := mutationFailureClass(err)
				results = append(results, mutationCapabilityResult{Capability: "review-result-publication", Disposition: "required_refusal", Class: class, Detail: err.Error()})
				return results, clikit.Refusedf("mutation preflight review-result-publication failed (%s): %v", class, err)
			}
			results = append(results, mutationCapabilityResult{Capability: "review-result-publication", Disposition: "pass", Class: "ok", Detail: "parent can durably append the structured reviewer result"})
		}
		return results, nil
	}
	results := make([]mutationCapabilityResult, 0, 6)
	requiredFailure := func(capability string, err error) error {
		class := mutationFailureClass(err)
		results = append(results, mutationCapabilityResult{Capability: capability, Disposition: "required_refusal", Class: class, Detail: err.Error()})
		return clikit.Refusedf("mutation preflight %s failed (%s): %v", capability, class, err)
	}

	claims := p.Claims
	if len(claims) == 0 {
		claims = []string{"."}
	}
	seenDirs := map[string]bool{}
	probeRoot := p.ProbeWorkDir
	if probeRoot == "" {
		probeRoot = p.w.Root
	}
	for _, claim := range claims {
		dir, err := nearestExistingDir(probeRoot, claim)
		if err != nil {
			return results, requiredFailure("source-write", err)
		}
		if seenDirs[dir] {
			continue
		}
		seenDirs[dir] = true
		if err := probeMutationDirectory(dir, ".dacli-source-probe-*"); err != nil {
			return results, requiredFailure("source-write", err)
		}
	}
	results = append(results, mutationCapabilityResult{Capability: "source-write", Disposition: "pass", Class: "ok", Detail: "claimed source parents are writable"})

	gitIndex, err := gitx.Run(probeRoot, "rev-parse", "--git-path", "index")
	if err == nil {
		if !filepath.IsAbs(gitIndex) {
			gitIndex = filepath.Join(probeRoot, gitIndex)
		}
		err = probeMutationGitLock(gitIndex)
	}
	if err != nil {
		results = append(results, mutationCapabilityResult{Capability: "git-metadata-write", Disposition: "planned_handoff", Class: mutationFailureClass(err), Detail: err.Error()})
	} else {
		results = append(results, mutationCapabilityResult{Capability: "git-metadata-write", Disposition: "pass", Class: "ok", Detail: "Git administrative directory accepts an atomic lock peer"})
	}

	eventErr := probeMutationDirectory(p.w.EventsDir(), ".dacli-event-probe-*")
	if eventErr != nil {
		results = append(results, mutationCapabilityResult{Capability: "event-write", Disposition: "planned_handoff", Class: mutationFailureClass(eventErr), Detail: eventErr.Error()})
	} else {
		results = append(results, mutationCapabilityResult{Capability: "event-write", Disposition: "pass", Class: "ok", Detail: "shared event directory is writable"})
	}

	for _, tool := range declaredVerificationTools(p.Task) {
		if _, err := exec.LookPath(tool); err != nil {
			return results, requiredFailure("tool:"+tool, err)
		}
		results = append(results, mutationCapabilityResult{Capability: "tool:" + tool, Disposition: "pass", Class: "ok", Detail: "declared verification executable found"})
	}

	if p.f.Bool("pr") || p.f.Bool("review") {
		networkErr := probeMutationCommand(p.w.Root, "gh", "api", "rate_limit", "--silent")
		if networkErr != nil {
			results = append(results, mutationCapabilityResult{Capability: "github-network", Disposition: "planned_handoff", Class: mutationFailureClass(networkErr), Detail: networkErr.Error()})
		} else {
			results = append(results, mutationCapabilityResult{Capability: "github-network", Disposition: "pass", Class: "ok", Detail: "authenticated GitHub read probe succeeded"})
		}
	}
	return results, nil
}

func plannedHandoffCapabilities(results []mutationCapabilityResult) []string {
	var out []string
	for _, result := range results {
		if result.Disposition == "planned_handoff" {
			out = append(out, result.Capability+":"+result.Class)
		}
	}
	return out
}

func mutationPreflightInvocation(results []mutationCapabilityResult) string {
	var b strings.Builder
	for _, result := range results {
		fmt.Fprintf(&b, "mutation_capability: %s=%s/%s\n", result.Capability, result.Disposition, result.Class)
	}
	return b.String()
}

func handoffChannelPreamble(runDir string, planned []string) string {
	label := strings.Join(planned, ",")
	if label == "" {
		label = "none observed; use this channel for an unexpected publication failure"
	}
	path := filepath.Join(runDir, store.RootHandoffRequestFile)
	return fmt.Sprintf("\nROOT HANDOFF CONTRACT (planned capabilities: %s): If commit, event, or GitHub publication is unavailable after useful edits or verification, do not claim success and do not switch harness or seek broader authority. Write structured JSON to %s with schema %q and fields verification:[{command,exit_code,result}], unresolved_findings:[], failed_operation, failure_class, stderr, safe_owner_next_action. Continue preserving the worktree for root.\n", label, path, store.RootHandoffSchema)
}
