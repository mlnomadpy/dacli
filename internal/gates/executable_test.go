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
