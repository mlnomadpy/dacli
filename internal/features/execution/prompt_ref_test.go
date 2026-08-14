package execution

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

// Generated worker commands cross the project boundary through the shared
// workspace resolver. They must carry the stable task identity even when the
// sequence shown to humans is ambiguous (issue #636).
func TestPromptSuffixUsesStableTaskIDForMutatingCommands(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "assigned task", store.TaskOpts{Accept: []string{"done"}})
	if _, err := store.CreateProject(w, agentid.RootID, "Other", "other", "", ""); err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateTask(w, agentid.RootID, "other", "colliding task", store.TaskOpts{Accept: []string{"done"}})
	if err != nil {
		t.Fatal(err)
	}
	if other.Seq != task.Seq {
		t.Fatalf("fixture must collide on sequence: got %d and %d", task.Seq, other.Seq)
	}

	f, err := clikit.ParseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := promptSuffix(w, f, task, "a-child", model.GrantRW)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		"task check " + task.ID,
		"task done " + task.ID,
		"accept " + task.ID,
		"commit \"" + task.ID + ": <what changed>\" --task " + task.ID,
	} {
		if !strings.Contains(out, command) {
			t.Errorf("generated instructions missing stable command %q", command)
		}
	}
	numeric := fmt.Sprintf("%03d", task.Seq)
	for _, command := range []string{"task check " + numeric, "task done " + numeric, "accept " + numeric} {
		if strings.Contains(out, command) {
			t.Errorf("generated instructions contain ambiguous command %q", command)
		}
	}
}
