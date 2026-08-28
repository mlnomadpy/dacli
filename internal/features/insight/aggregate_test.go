package insight

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func aggregateInsightFixture(t *testing.T) (*workspace.Workspace, *store.Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "aggregate-insight")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	parent, err := store.CreateTask(w, "a-root", "core", "aggregate milestone", store.TaskOpts{Accept: []string{"children done"}, Estimate: "8,13,21"})
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"one", "two", "three", "four"} {
		if _, err := store.CreateTask(w, "a-root", "core", title, store.TaskOpts{Parent: parent.ID, Accept: []string{title}, Estimate: "1,2,3"}); err != nil {
			t.Fatal(err)
		}
	}
	plan, _ := store.BuildAggregateRepairPlan(w, parent)
	if _, err := store.ApplyAggregateRepairPlan(w, parent, plan.ID); err != nil {
		t.Fatal(err)
	}
	return w, parent
}

func TestNextCriticalPathAndWBSAgreeOnAggregateState(t *testing.T) {
	w, parent := aggregateInsightFixture(t)
	for name, run := range map[string]func(*clikit.Ctx) error{
		"next":          func(ctx *clikit.Ctx) error { return cmdNext(ctx, []string{"--project", "core", "--parallel", "10"}) },
		"critical-path": func(ctx *clikit.Ctx) error { return cmdCriticalPath(ctx, []string{"--project", "core"}) },
		"wbs":           func(ctx *clikit.Ctx) error { return cmdWBS(ctx, []string{"--project", "core"}) },
	} {
		t.Run(name, func(t *testing.T) {
			out := &bytes.Buffer{}
			ctx := &clikit.Ctx{Cwd: w.Root, Stdout: out, Stderr: &bytes.Buffer{}}
			if err := run(ctx); err != nil {
				t.Fatal(err)
			}
			text := out.String()
			if name == "wbs" {
				if !strings.Contains(text, "aggregate 0/4") {
					t.Fatalf("WBS omitted derived aggregate progress:\n%s", text)
				}
			} else if strings.Contains(text, parent.Slug) {
				t.Fatalf("%s scheduled aggregate parent:\n%s", name, text)
			}
		})
	}
}
