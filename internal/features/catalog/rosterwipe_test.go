package catalog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// docs/ROSTER.md is a generated view committed alongside the source of truth.
// collectRoles discarded LoadRoles' error — which store/roles.go deliberately
// raises to distinguish "no roles" from "could not read" — so an unreadable
// roles directory produced an EMPTY roster, wrote it over the committed file,
// and printed success. A view that silently goes blank is worse than no view
// (dacli 208).
func TestCatalogRefusesRatherThanWritingAnEmptyRoster(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission bits do not constrain reads")
	}
	root := t.TempDir()
	w, err := workspace.Init(root, "x")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRole(w, agentid.RootID, team.Role{Name: "reviewer", Summary: "reviews things"}); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(root, "docs", "ROSTER.md")
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: root}

	// Healthy: the roster is written and names the role.
	if err := cmdCatalog(ctx, []string{"--out", out}); err != nil {
		t.Fatalf("catalog on a healthy workspace: %v", err)
	}
	good, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(good), "reviewer") {
		t.Fatalf("expected the role in the roster, got:\n%s", good)
	}

	// Now make the roles directory unreadable and run again.
	rolesDir := w.RolesDir()
	if err := os.Chmod(rolesDir, 0o000); err != nil {
		t.Skipf("cannot make the roles dir unreadable here: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(rolesDir, 0o755) })

	err = cmdCatalog(ctx, []string{"--out", out})
	if err == nil {
		t.Error("catalog must refuse when the roles cannot be read, not publish an empty roster")
	}
	// Whatever it decided, it must NOT have replaced the good roster with a
	// blank one.
	after, rerr := os.ReadFile(out)
	if rerr != nil {
		t.Fatalf("the existing roster was destroyed: %v", rerr)
	}
	if !strings.Contains(string(after), "reviewer") {
		t.Errorf("the roster was overwritten with an empty one:\n%s", after)
	}
}
