package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestAcceptJSONSeparatesDeltaFromTotalSatisfaction(t *testing.T) {
	w, _, ctx := acceptEnv(t)
	task, err := store.CreateTask(w, agentid.RootID, "p", "json result", store.TaskOpts{Accept: []string{"one", "two"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx.JSON = true
	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
	if err := acceptOne(ctx, w, root, task, "", false, false, false, false, true, ""); err != nil {
		t.Fatal(err)
	}
	var got acceptanceResult
	if err := json.Unmarshal(ctx.Stdout.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatalf("JSON result: %v\n%s", err, ctx.Stdout.(*bytes.Buffer).String())
	}
	if got.NewlyChecked != 2 || got.Satisfied != 2 || got.Total != 2 || len(got.Tasks) != 1 || got.Tasks[0].Unverified {
		t.Fatalf("JSON counts = %#v", got)
	}
}

func TestAcceptReportsDeltaAndTotalSatisfaction(t *testing.T) {
	tests := []struct {
		name            string
		criteria        []string
		prechecked      int
		allowUnverified bool
		newly           int
		satisfied       int
		total           int
		unverified      bool
		human           string
	}{
		{name: "all prechecked", criteria: []string{"one", "two", "three"}, prechecked: 3, newly: 0, satisfied: 3, total: 3, human: "0 newly checked; 3/3 acceptance criteria satisfied"},
		{name: "none prechecked", criteria: []string{"one", "two", "three"}, newly: 3, satisfied: 3, total: 3, human: "3 newly checked; 3/3 acceptance criteria satisfied"},
		{name: "partially checked", criteria: []string{"one", "two", "three"}, prechecked: 1, newly: 2, satisfied: 3, total: 3, human: "2 newly checked; 3/3 acceptance criteria satisfied"},
		{name: "no criteria override", allowUnverified: true, unverified: true, human: "NO acceptance criteria; explicitly UNVERIFIED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, ctx := acceptEnv(t)
			task, err := store.CreateTask(w, agentid.RootID, "p", tt.name, store.TaskOpts{Accept: tt.criteria})
			if err != nil {
				t.Fatal(err)
			}
			if tt.prechecked > 0 {
				lines := make([]string, 0, len(tt.criteria))
				for i, criterion := range tt.criteria {
					mark := " "
					if i < tt.prechecked {
						mark = "x"
					}
					lines = append(lines, fmt.Sprintf("- [%s] %s", mark, criterion))
				}
				task.Doc.SetSection("Acceptance", strings.Join(lines, "\n")+"\n")
				if err := store.SaveTask(task); err != nil {
					t.Fatal(err)
				}
			}

			root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
			if err := acceptOne(ctx, w, root, task, "", false, false, tt.allowUnverified, false, true, ""); err != nil {
				t.Fatal(err)
			}
			result, ok := ctx.Result.(acceptanceResult)
			if !ok || len(result.Tasks) != 1 {
				t.Fatalf("result = %#v", ctx.Result)
			}
			got := result.Tasks[0]
			if got.NewlyChecked != tt.newly || got.Satisfied != tt.satisfied || got.Total != tt.total || got.Unverified != tt.unverified {
				t.Fatalf("task result = %#v", got)
			}
			if result.NewlyChecked != tt.newly || result.Satisfied != tt.satisfied || result.Total != tt.total {
				t.Fatalf("aggregate result = %#v", result)
			}
			if out := ctx.Stdout.(*bytes.Buffer).String(); !strings.Contains(out, tt.human) {
				t.Fatalf("human summary = %q, want %q", out, tt.human)
			}
		})
	}
}

func TestAcceptNoCriteriaRefusalHasNoSuccessSummary(t *testing.T) {
	w, _, ctx := acceptEnv(t)
	task, err := store.CreateTask(w, agentid.RootID, "p", "refused", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
	err = acceptOne(ctx, w, root, task, "", false, false, false, false, true, "")
	if err == nil || ctx.Result != nil || ctx.Stdout.(*bytes.Buffer).Len() != 0 {
		t.Fatalf("refusal err=%v result=%#v stdout=%q", err, ctx.Result, ctx.Stdout.(*bytes.Buffer).String())
	}
	if got, findErr := store.FindTask(w, task.ID); findErr != nil || got.Status == model.StatusDone {
		t.Fatalf("refusal mutated task: status=%v err=%v", got.Status, findErr)
	}
}

func TestAcceptAllUsesTheSameSatisfactionSemantics(t *testing.T) {
	w, checked, ctx := acceptEnv(t)
	checked.Doc.SetSection("Acceptance", "- [x] done\n")
	if err := store.SaveTask(checked); err != nil {
		t.Fatal(err)
	}
	empty, err := store.CreateTask(w, "a-deadchild", "p", "no criteria", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	child := &agentid.Identity{ID: "a-deadchild", Grant: model.GrantRW, Role: "worker"}
	if err := propose(ctx, w, child, checked); err != nil {
		t.Fatal(err)
	}
	if err := propose(ctx, w, child, empty); err != nil {
		t.Fatal(err)
	}
	ctx.Stdout.(*bytes.Buffer).Reset()

	root := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
	if err := acceptAll(ctx, w, root, "", true, false, false, true, false, true, ""); err != nil {
		t.Fatal(err)
	}
	result, ok := ctx.Result.(acceptanceResult)
	if !ok || result.Accepted != 2 || result.NewlyChecked != 0 || result.Satisfied != 1 || result.Total != 1 || result.UnverifiedTasks != 1 {
		t.Fatalf("batch result = %#v", ctx.Result)
	}
	out := ctx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{"0 newly checked; 1/1 acceptance criteria satisfied", "NO acceptance criteria; explicitly UNVERIFIED", "1 explicitly UNVERIFIED task(s)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("batch summary missing %q:\n%s", want, out)
		}
	}
}
