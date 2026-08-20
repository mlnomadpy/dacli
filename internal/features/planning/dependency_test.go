package planning

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestTaskDependRecordsAppliedAuditEvent(t *testing.T) {
	w, ctx := taskAddEnv(t)
	dep, _ := store.CreateTask(w, agentid.RootID, "p", "dependency", store.TaskOpts{})
	target, _ := store.CreateTask(w, agentid.RootID, "p", "target", store.TaskOpts{})
	if err := cmdTaskDepend(ctx, []string{target.ID, "--add", dep.ID + ":SS"}); err != nil {
		t.Fatal(err)
	}
	events, err := eventlog.List(w, eventlog.Query{About: target.ID, Kinds: []model.EventKind{model.EventDependency}})
	if err != nil || len(events) != 1 || !events[0].Applied {
		t.Fatalf("audit event = %+v, err=%v; want one applied dependency event", events, err)
	}
}

func TestTaskDependReadOnlyAgentProposesForOwnerSync(t *testing.T) {
	w, _ := taskAddEnv(t)
	dep, _ := store.CreateTask(w, agentid.RootID, "p", "dependency", store.TaskOpts{})
	target, _ := store.CreateTask(w, agentid.RootID, "p", "target", store.TaskOpts{})
	childID, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}, "reviewer", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdTaskDepend(ctx, []string{target.ID, "--add", dep.ID}); err != nil {
		t.Fatalf("read-only proposal: %v", err)
	}
	if !strings.Contains(out.String(), "proposed") {
		t.Fatalf("proposal output = %q", out.String())
	}
	unchanged, _ := store.FindTask(w, target.ID)
	if len(unchanged.Deps()) != 0 {
		t.Fatal("read-only proposal wrote the task directly")
	}

	res, err := eventlog.Sync(w, agentid.RootID, func(owner string) bool { return owner == agentid.RootID })
	if err != nil || res.Applied != 1 {
		t.Fatalf("owner sync = %+v, err=%v", res, err)
	}
	got, _ := store.FindTask(w, target.ID)
	if deps := got.Deps(); len(deps) != 1 || deps[0].Ref != dep.ID {
		t.Fatalf("synced dependencies = %#v", deps)
	}
	events, _ := eventlog.List(w, eventlog.Query{About: target.ID, Kinds: []model.EventKind{model.EventDependency}})
	if len(events) != 1 || events[0].Actor != childID || !events[0].Applied {
		t.Fatalf("proposal audit event = %+v", events)
	}
}
