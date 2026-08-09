package briefing

import (
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// The failure this exists for: a wave's fourth agent reads a brief assembled
// before the first three found anything, so it re-derives what a sibling filed
// ten minutes ago. catchup must show the sibling's filing WITHOUT the owner
// having synced it — the append-only log is the only place it lands
// immediately.
func TestCatchupShowsALiveSiblingsFinding(t *testing.T) {
	w, task := catchupWS(t)

	if _, err := eventlog.Append(w, "a-sibling", model.EventFinding, task.Slug, "",
		"the batch job writes balances directly\ncron/settle.go:112"); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdCatchup(ctx, nil); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "a-sibling") || !strings.Contains(got, "writes balances directly") {
		t.Errorf("a sibling's unsynced finding did not reach catchup:\n%s", got)
	}
	// One line per item, not a re-assembled brief: the whole point is that it
	// is cheap enough to run between steps.
	if strings.Contains(got, "## Task:") {
		t.Errorf("catchup emitted a full brief; it must be a short digest:\n%s", got)
	}
}

// An agent's own filings are not news to it, and including them is what makes
// the output too long to be worth reading mid-run.
func TestCatchupExcludesYourOwnFilings(t *testing.T) {
	w, task := catchupWS(t)
	if _, err := eventlog.Append(w, "a-me", model.EventFinding, task.Slug, "", "something I found"); err != nil {
		t.Fatal(err)
	}
	if _, err := eventlog.Append(w, "a-other", model.EventFinding, task.Slug, "", "something they found"); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdCatchup(ctx, []string{"--mine", "a-me"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "something I found") {
		t.Errorf("catchup echoed the caller's own filing back at it:\n%s", got)
	}
	if !strings.Contains(got, "something they found") {
		t.Errorf("catchup dropped a sibling's filing:\n%s", got)
	}
}

// --since windows to what is actually new. A malformed duration is a usage
// error, never a silent "show everything".
func TestCatchupSinceWindow(t *testing.T) {
	w, task := catchupWS(t)
	if _, err := eventlog.Append(w, "a-sibling", model.EventFinding, task.Slug, "", "recent thing"); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdCatchup(ctx, []string{"--since", "1h"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "recent thing") {
		t.Errorf("an event inside the window was filtered out:\n%s", out.String())
	}

	// A window that ends before the event was written shows nothing, and says
	// so rather than printing an empty list.
	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdCatchup(ctx2, []string{"--since", "1ns"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "nothing new") {
		t.Errorf("an empty result must say so explicitly:\n%s", out2.String())
	}

	if _, err := sinceCutoff("20 minutes"); err == nil {
		t.Error("a malformed --since must be a usage error, not a silent whole-log read")
	}
}

func catchupWS(t *testing.T) (*workspace.Workspace, *store.Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "c")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "core", "the work", store.TaskOpts{Accept: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	return w, task
}

var _ = time.Now
