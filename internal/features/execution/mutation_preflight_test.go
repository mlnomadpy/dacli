package execution

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func mutationPlanFixture(t *testing.T) *launchPlan {
	t.Helper()
	w := newExecWS(t)
	for _, args := range [][]string{{"init"}, {"config", "user.name", "Test"}, {"config", "user.email", "test@example.invalid"}} {
		if _, err := gitx.Run(w.Root, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(w.Root, "source.go"), []byte("package fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "add", "source.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(w.Root, "commit", "-m", "fixture"); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, agentid.RootID, testProject, "mutation probe", store.TaskOpts{Accept: []string{"source is updated"}})
	if err != nil {
		t.Fatal(err)
	}
	flags, err := clikit.ParseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &launchPlan{Task: task, Grant: model.GrantRW, Claims: []string{"source.go"}, w: w, f: flags}
}

func withMutationDirectoryProbe(t *testing.T, fn func(string, string) error) {
	t.Helper()
	old := probeMutationDirectory
	probeMutationDirectory = fn
	t.Cleanup(func() { probeMutationDirectory = old })
}

func TestMutationPreflightRefusesSourceWriteBeforeWorkerCreation(t *testing.T) {
	p := mutationPlanFixture(t)
	withMutationDirectoryProbe(t, func(_ string, prefix string) error {
		if strings.Contains(prefix, "source") {
			return fs.ErrPermission
		}
		return nil
	})
	_, err := mutationPreflight(p)
	if err == nil || !strings.Contains(err.Error(), "source-write") || !strings.Contains(err.Error(), "filesystem_sandbox_refusal") {
		t.Fatalf("source refusal = %v", err)
	}
	entries, readErr := os.ReadDir(p.w.RunsDir())
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("preflight refusal created %d run records", len(entries))
	}
}

func TestMutationPreflightRefusesGovernedReviewWithoutResultPublication(t *testing.T) {
	p := mutationPlanFixture(t)
	p.Grant = model.GrantRO
	flags, err := clikit.ParseFlags([]string{"--structured-review-result"})
	if err != nil {
		t.Fatal(err)
	}
	p.f = flags
	withMutationDirectoryProbe(t, func(dir, prefix string) error {
		if dir == p.w.EventsDir() && strings.Contains(prefix, "review-result") {
			return fs.ErrPermission
		}
		return nil
	})

	results, err := mutationPreflight(p)
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "review-result-publication") || !strings.Contains(err.Error(), "filesystem_sandbox_refusal") {
		t.Fatalf("review result publication refusal = %v", err)
	}
	if len(results) != 2 || results[1].Capability != "review-result-publication" || results[1].Disposition != "required_refusal" {
		t.Fatalf("review result capability evidence = %+v", results)
	}
	entries, readErr := os.ReadDir(p.w.RunsDir())
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("review preflight refusal created %d run records", len(entries))
	}
}

func TestMutationPreflightRefusesSymlinkedClaimOutsideWorktreeWithoutWriting(t *testing.T) {
	p := mutationPlanFixture(t)
	outside := t.TempDir()
	outsideResolved, resolveErr := filepath.EvalSymlinks(outside)
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	link := filepath.Join(p.w.Root, "escaped-claim")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink fixture unavailable: %v", err)
	}
	calledOutside := false
	withMutationDirectoryProbe(t, func(dir, prefix string) error {
		if strings.Contains(prefix, "source") && filepath.Clean(dir) == filepath.Clean(outsideResolved) {
			calledOutside = true
			return os.WriteFile(filepath.Join(outside, "probe-escaped"), []byte("outside write\n"), 0o600)
		}
		return nil
	})
	p.Claims = []string{"escaped-claim/new-source.go"}
	_, err := mutationPreflight(p)
	entries, readErr := os.ReadDir(outside)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("source preflight wrote outside assignment through symlink: %v", entries)
	}
	if calledOutside {
		t.Fatal("source mutation probe was invoked outside the assignment worktree")
	}
	if err == nil || !strings.Contains(err.Error(), "escapes assignment worktree") {
		t.Fatalf("symlinked source claim = %v, want containment refusal", err)
	}
}

func TestMutationPreflightPlansGitIndexHandoffAfterSourcePasses(t *testing.T) {
	p := mutationPlanFixture(t)
	old := probeMutationGitLock
	probeMutationGitLock = func(string) error { return fs.ErrPermission }
	t.Cleanup(func() { probeMutationGitLock = old })
	results, err := mutationPreflight(p)
	if err != nil {
		t.Fatal(err)
	}
	planned := plannedHandoffCapabilities(results)
	if len(planned) != 1 || planned[0] != "git-metadata-write:filesystem_sandbox_refusal" {
		t.Fatalf("planned handoff = %v", planned)
	}
}

func TestMutationPreflightPlansEventWriteHandoff(t *testing.T) {
	p := mutationPlanFixture(t)
	withMutationDirectoryProbe(t, func(_ string, prefix string) error {
		if strings.Contains(prefix, "event") {
			return fs.ErrPermission
		}
		return nil
	})
	results, err := mutationPreflight(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(plannedHandoffCapabilities(results), ","); got != "event-write:filesystem_sandbox_refusal" {
		t.Fatalf("planned event handoff = %q", got)
	}
}

func TestMutationFailureClassDistinguishesContractStates(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want string
	}{
		"sandbox": {fs.ErrPermission, "filesystem_sandbox_refusal"},
		"missing": {&exec.Error{Name: "missing", Err: exec.ErrNotFound}, "missing_tool"},
		"auth":    {errors.New("authentication failed while connecting"), "authentication_network_failure"},
		"policy":  {errors.New("operation denied by policy"), "policy_refusal"},
		"busy":    {syscall.EBUSY, "transient_contention"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := mutationFailureClass(tc.err); got != tc.want {
				t.Fatalf("class = %q, want %q", got, tc.want)
			}
		})
	}
}
