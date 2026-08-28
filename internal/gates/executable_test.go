package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func gateEnv(t *testing.T) (*workspace.Workspace, *store.Project) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.CreateProject(w, agentid.RootID, "Core", "core", "", "")
	if err != nil {
		t.Fatal(err)
	}
	return w, p
}

// Before this, every predicate inspected markdown, so a stage could pass with a
// broken build: the gates certified the paperwork, not the software (dacli 186).
func TestCommandPredicateGatesOnExitStatus(t *testing.T) {
	w, p := gateEnv(t)

	if c := evaluate(w, p, Predicate{Kind: "command", Arg: "true"}); !c.OK {
		t.Errorf("a passing command must satisfy the gate; got Why=%q", c.Why)
	}
	c := evaluate(w, p, Predicate{Kind: "command", Arg: "echo boom >&2; exit 1"})
	if c.OK {
		t.Fatal("a failing command must shut the gate")
	}
	if !strings.Contains(c.Why, "boom") {
		t.Errorf("the failure output must explain WHY the gate is shut; got Why=%q", c.Why)
	}
	if c := evaluate(w, p, Predicate{Kind: "command", Arg: "  "}); c.OK {
		t.Error("an empty command must not pass")
	}
}

func TestCommandPredicateUsesGovernedBoundedDiagnostic(t *testing.T) {
	w, p := gateEnv(t)
	t.Setenv("GITHUB_TOKEN", "ghp_gate_secret_123456")
	c := evaluate(w, p, Predicate{Kind: "command", Arg: "printf 'auth ghp_gate_secret_123456 /private/outside/path' >&2; exit 23"})
	if c.OK {
		t.Fatal("failing command passed")
	}
	for _, want := range []string{"stage gate command", "exit 23", "<redacted>", "<outside-workspace>"} {
		if !strings.Contains(c.Why, want) {
			t.Fatalf("diagnostic %q missing %q", c.Why, want)
		}
	}
	if strings.Contains(c.Why, "ghp_gate_secret_123456") || strings.Contains(c.Why, "/private/outside/path") {
		t.Fatalf("diagnostic leaked governed content: %q", c.Why)
	}
}

// The command runs at the workspace root, not the process cwd — a gate must
// judge the project it belongs to.
func TestCommandPredicateRunsAtWorkspaceRoot(t *testing.T) {
	w, p := gateEnv(t)
	if err := os.WriteFile(filepath.Join(w.Root, "marker.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if c := evaluate(w, p, Predicate{Kind: "command", Arg: "test -f marker.txt"}); !c.OK {
		t.Errorf("command should run at the workspace root; got Why=%q", c.Why)
	}
}

// Coverage is the one quality signal a reader can check independently, so it
// gets a gate rather than living only in a CI log nobody reads (dacli 187).
func TestCoveragePredicate(t *testing.T) {
	w, p := gateEnv(t)

	// Above the floor: passes, and reports what was measured.
	c := evaluate(w, p, Predicate{Kind: "coverage", Arg: "80 echo 'coverage: 87.3% of statements'"})
	if !c.OK {
		t.Errorf("87.3%% must clear an 80%% floor; Why=%q", c.Why)
	}
	if !strings.Contains(c.Why, "87.3") {
		t.Errorf("the measured value must be reported; Why=%q", c.Why)
	}
	// Below the floor: fails.
	if c := evaluate(w, p, Predicate{Kind: "coverage", Arg: "80 echo 'TOTAL 120 40 66%'"}); c.OK {
		t.Error("66% must not clear an 80% floor")
	}
	// THE BOUNDARY. A floor is inclusive: exactly 80% clears an 80% floor.
	// Mutation testing caught this gap — flipping `>=` to `>` here survived the
	// suite, meaning nothing asserted the boundary and a project sitting
	// precisely on its floor could have been failed by a one-character edit.
	if c := evaluate(w, p, Predicate{Kind: "coverage", Arg: "80 echo 'coverage: 80.0% of statements'"}); !c.OK {
		t.Errorf("exactly 80%% must clear an 80%% floor (inclusive); Why=%q", c.Why)
	}
	if c := evaluate(w, p, Predicate{Kind: "coverage", Arg: "80 echo 'coverage: 79.9% of statements'"}); c.OK {
		t.Error("79.9% must not clear an 80% floor")
	}
	// The LAST percentage wins — a per-package list ending in the total.
	c = evaluate(w, p, Predicate{Kind: "coverage", Arg: "50 printf 'pkg a 10%%\\npkg b 20%%\\ntotal: 91.4%%\\n'"})
	if !c.OK {
		t.Errorf("the summary (last) percentage must be used; Why=%q", c.Why)
	}
	// Malformed / missing pieces fail closed.
	for _, arg := range []string{"", "80", "notanumber echo 50%"} {
		if c := evaluate(w, p, Predicate{Kind: "coverage", Arg: arg}); c.OK {
			t.Errorf("malformed coverage predicate %q must not pass", arg)
		}
	}
	// A command that fails, or emits no percentage, must not pass.
	if c := evaluate(w, p, Predicate{Kind: "coverage", Arg: "80 exit 1"}); c.OK {
		t.Error("a failing coverage command must not pass")
	}
	if c := evaluate(w, p, Predicate{Kind: "coverage", Arg: "80 echo no-number-here"}); c.OK {
		t.Error("output with no percentage must not pass")
	}
}

func TestArtifactPredicate(t *testing.T) {
	w, p := gateEnv(t)
	if err := os.MkdirAll(filepath.Join(w.Root, "internal", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, "internal", "api", "server.go"), []byte("package api\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if c := evaluate(w, p, Predicate{Kind: "artifact", Arg: "internal/api/server.go"}); !c.OK {
		t.Errorf("an existing artifact must satisfy the gate; got Why=%q", c.Why)
	}
	// Multiple paths, one missing.
	c := evaluate(w, p, Predicate{Kind: "artifact", Arg: "internal/api/server.go | README.md"})
	if c.OK {
		t.Fatal("a missing artifact must shut the gate")
	}
	if !strings.Contains(c.Why, "README.md") {
		t.Errorf("the missing path must be named; got Why=%q", c.Why)
	}
	// A traversing path must never be allowed to probe outside the workspace.
	if c := evaluate(w, p, Predicate{Kind: "artifact", Arg: "../../../etc/passwd"}); c.OK {
		t.Error("a traversing artifact path must be rejected, not resolved")
	}
}
