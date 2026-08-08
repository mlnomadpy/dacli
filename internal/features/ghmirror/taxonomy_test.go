package ghmirror

import (
	"encoding/json"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// snapshotIndex builds a marker index pre-loaded with one issue, standing in
// for the `gh issue list` the push already makes.
func snapshotIndex(t *testing.T, w *workspace.Workspace, repo string, iss ghIssue) *markerIndex {
	t.Helper()
	m := newMarkerIndex(w, repo)
	m.issues = []ghIssue{iss}
	m.loaded = true
	return m
}

// An issue whose labels and milestone already match must cost ZERO gh calls.
// Before this, every mapped issue took five unconditional invocations per push
// (add, three removes, taxonomy), so an idempotent re-push of ~300 mirrored
// tasks spent ~2,100 network round-trips changing nothing.
func TestTaxonomySyncSkipsAnAlreadyCurrentIssue(t *testing.T) {
	const repo = "octo/linked"
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := captureArgs(t, "[]")

	idx := snapshotIndex(t, w, repo, ghIssue{
		Number:    7,
		Labels:    []ghLabel{{Name: statusLabel(model.StatusDone)}, {Name: "type:task"}, {Name: "area:core"}},
		Milestone: ghMilestone{Title: "core"},
	})
	syncIssueTaxonomy(w, repo, idx, 7, model.StatusDone, "area:core", "core", true)

	if len(*calls) != 0 {
		t.Errorf("a current issue must cost no gh calls, got %d: %v", len(*calls), *calls)
	}
}

// A stale issue is corrected in ONE edit that carries every add and remove,
// rather than one invocation per label.
func TestTaxonomySyncCorrectsInASingleEdit(t *testing.T) {
	const repo = "octo/linked"
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := captureArgs(t, "[]")

	// Currently labelled open; the task is now done and needs the area label.
	idx := snapshotIndex(t, w, repo, ghIssue{
		Number: 7,
		Labels: []ghLabel{{Name: statusLabel(model.StatusOpen)}, {Name: "type:task"}},
	})
	syncIssueTaxonomy(w, repo, idx, 7, model.StatusDone, "area:core", "core", true)

	var edits [][]string
	for _, c := range *calls {
		if len(c) > 1 && c[0] == "issue" && c[1] == "edit" {
			edits = append(edits, c)
		}
	}
	if len(edits) != 1 {
		t.Fatalf("want exactly one issue edit, got %d: %v", len(edits), edits)
	}
	e := edits[0]
	if !argPairPresent(e, "--add-label", statusLabel(model.StatusDone)) {
		t.Errorf("the new status label must be added: %v", e)
	}
	if !argPairPresent(e, "--remove-label", statusLabel(model.StatusOpen)) {
		t.Errorf("the stale status label must be removed: %v", e)
	}
	if !argPairPresent(e, "--add-label", "area:core") {
		t.Errorf("the missing area label must be added: %v", e)
	}
	if !argPairPresent(e, "--milestone", "core") {
		t.Errorf("the missing milestone must be set: %v", e)
	}
	// type:task is already present and must NOT be re-added.
	if argPairPresent(e, "--add-label", "type:task") {
		t.Errorf("a label the issue already carries must not be re-added: %v", e)
	}
}

// An index that never loaded (transient gh failure) must still write: an
// unknown issue is treated as needing the edit, never silently skipped.
func TestTaxonomySyncWritesWhenTheSnapshotIsUnavailable(t *testing.T) {
	const repo = "octo/linked"
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := captureArgs(t, "[]")

	idx := newMarkerIndex(w, repo)
	idx.loaded = true // loaded, but empty: the fetch failed
	syncIssueTaxonomy(w, repo, idx, 7, model.StatusDone, "area:core", "", false)

	if len(*calls) == 0 {
		t.Error("an issue missing from the snapshot must still be written, not skipped")
	}
}

// The snapshot must actually request labels and milestone, or every diff above
// silently degrades to "no labels" and the savings disappear.
func TestIndexSnapshotRequestsLabelsAndMilestone(t *testing.T) {
	const repo = "octo/linked"
	w := &workspace.Workspace{Root: t.TempDir()}
	body, _ := json.Marshal([]ghIssue{})
	calls := captureArgs(t, string(body))

	newMarkerIndex(w, repo).load()

	for _, c := range *calls {
		for i := 0; i+1 < len(c); i++ {
			if c[i] == "--json" {
				for _, want := range []string{"labels", "milestone"} {
					if !containsField(c[i+1], want) {
						t.Errorf("issue list --json %q must include %q so the taxonomy diff has data", c[i+1], want)
					}
				}
				return
			}
		}
	}
	t.Fatal("no gh issue list --json call captured")
}

func argPairPresent(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}

func containsField(csv, field string) bool {
	for _, f := range splitCSV(csv) {
		if f == field {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}
