package planning

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func projectLandingEnv(t *testing.T) (*workspace.Workspace, *clikit.Ctx) {
	t.Helper()
	unsetAgentEnv(t)
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	return w, &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
}

func TestProjectAddLandingValidationPrecedesPersistence(t *testing.T) {
	w, ctx := projectLandingEnv(t)
	for _, args := range [][]string{{"Bad mode", "--slug", "bad-mode", "--landing-mode", "merge"}, {"Bad base", "--slug", "bad-base", "--landing-base", "  "}} {
		err := cmdProjectAdd(ctx, args)
		if clikit.ExitCode(err) != 2 {
			t.Fatalf("exit = %d, want 2: %v", clikit.ExitCode(err), err)
		}
		slug := args[2]
		if _, err := os.Stat(w.ProjectPath(slug)); !os.IsNotExist(err) {
			t.Fatalf("invalid project %s was persisted", slug)
		}
	}
}

func TestProjectShowStructuredLandingOnly(t *testing.T) {
	w, ctx := projectLandingEnv(t)
	p, err := store.CreateProject(w, "a-root", "P", "p", "secret goal", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConfigureProjectLanding(p, model.LandingPolicy{Mode: model.LandingLocal, Base: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	ctx.JSON = true
	if err := cmdProjectShow(ctx, []string{"p", "--landing-mode", "pr"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["landing_override"] != true {
		t.Fatalf("landing_override = %#v", got["landing_override"])
	}
	if _, leaked := got["goal"]; leaked {
		t.Fatal("structured inspection leaked unrelated project goal")
	}
	effective := got["landing_effective"].(map[string]any)
	if effective["mode"] != "pr" || effective["base"] != "main" {
		t.Fatalf("effective = %#v", effective)
	}
}
