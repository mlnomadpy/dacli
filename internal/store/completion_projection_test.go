package store

import (
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestImplementedUnlandedRemainsNonterminalAcrossDerivedStoreProjections(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject(w, "a-root", "P", "p", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := CreateTask(w, "a-root", "p", "Await landing", TaskOpts{Accept: []string{"verified"}, Estimate: "1,2,3"})
	if err != nil {
		t.Fatal(err)
	}
	task.Doc.Front.Set("completion_state", "implemented-unlanded")
	if err := SaveTask(task); err != nil {
		t.Fatal(err)
	}

	projection, err := LocalDeliveryProjection(w, "p", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, finding := range projection.Findings {
		if finding.Classification == "implemented-unlanded" && finding.ObjectID == task.ID && strings.Contains(finding.NextAction, "dacli ship") {
			found = true
		}
	}
	if !found {
		t.Fatalf("reconciliation omitted intermediate lifecycle: %#v", projection.Findings)
	}
	if done, err := ListTasks(w, "p", model.StatusDone); err != nil || len(done) != 0 {
		t.Fatalf("implemented task entered done set: %d err=%v", len(done), err)
	}
	if samples := CalibrationSamples(w); len(samples) != 0 {
		t.Fatalf("implemented-unlanded task polluted calibration: %#v", samples)
	}
}
