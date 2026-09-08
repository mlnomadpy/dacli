package insight

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestOverviewJSONIsVersionedAndBounded(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "overview-json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRole(w, "a-root", team.Role{Name: "builder", Kind: "implementer", Grant: "rw", WIP: 4}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if _, err := store.CreateTask(w, "a-root", "core", "Task "+string(rune('A'+i)), store.TaskOpts{Accept: []string{"done"}}); err != nil {
			t.Fatal(err)
		}
	}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdOverview(ctx, nil); err != nil {
		t.Fatal(err)
	}
	var got overviewJSON
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != "workspace-overview/v1" || got.Version != 1 || got.Projects != 1 || got.Tasks != 20 || got.Counts["open"] != 20 || got.WIPCapacity != 4 {
		t.Fatalf("overview facts = %+v", got)
	}
	if got.ReadyLimit != 3 || len(got.ReadyNow) != 3 {
		t.Fatalf("ready projection is not bounded: limit=%d len=%d", got.ReadyLimit, len(got.ReadyNow))
	}
}
