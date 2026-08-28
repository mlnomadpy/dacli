package store

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func fixture(t *testing.T) (*workspace.Workspace, *Task) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "reconcile")
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		if out, e := exec.Command("git", append([]string{"-C", w.Root}, args...)...).CombinedOutput(); e != nil {
			t.Fatalf("git %v: %v\n%s", args, e, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "test@example.test")
	git("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(w.Root, "seed"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "seed")
	git("commit", "-qm", "seed")
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	orphan, err := CreateTask(w, "child-dead", "core", "orphan", TaskOpts{Accept: []string{"classified"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := MoveTask(w, orphan, model.StatusActive); err != nil {
		t.Fatal(err)
	}
	done, err := CreateTask(w, "a-root", "core", "terminal", TaskOpts{Accept: []string{"classified"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := MoveTask(w, done, model.StatusDone); err != nil {
		t.Fatal(err)
	}
	eventDoc := &mdstore.Doc{}
	eventDoc.Front.Set("id", "event-terminal")
	eventDoc.Front.Set("about", "[["+done.ID+"]]")
	eventDoc.Front.Set("applied", "false")
	if err := mdstore.WriteFile(w.EventPath("2026/08/28", "event-terminal", "child-dead", model.EventKind("finding")), eventDoc); err != nil {
		t.Fatal(err)
	}
	runID := "01M146BA62817V08T9P6D6RUN"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(runID), "proc.txt"), procmon.Record{RunID: runID, Child: "child-dead", Task: orphan.ID, PID: 99999999, Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.RunDir(runID), "outcome.md"), []byte("outcome: running (detached)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(w.Root, workspace.Dir, "loop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, workspace.Dir, "loop", "core.txt"), []byte("trunk_marker: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return w, orphan
}

func TestReconcileDistinctFixtureClassificationsAndReadOnlyDigest(t *testing.T) {
	w, orphan := fixture(t)
	old := ObserveDeliveryPRs
	ObserveDeliveryPRs = func(string) ([]DeliveryPR, error) {
		return []DeliveryPR{{DeliveryConfidence: "CLOSED", URL: "https://example.test/pr/1", HeadRefName: TaskBranch(orphan), HeadRefOid: "head", BaseRefOid: "base"}}, nil
	}
	t.Cleanup(func() { ObserveDeliveryPRs = old })
	before := treeDigest(t, w.Root)
	p, err := ReconcileDelivery(w, "core", time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	after := treeDigest(t, w.Root)
	if before != after {
		t.Fatal("read-only reconciliation mutated the fixture")
	}
	got := map[string]bool{}
	for _, f := range p.Findings {
		got[f.Classification] = true
		if f.ID == "" || f.ObjectID == "" || f.Source == "" || f.ObservedAt.IsZero() || f.Severity == "" || f.Confidence == "" || f.NextAction == "" {
			t.Fatalf("incomplete finding: %+v", f)
		}
	}
	for _, want := range []string{"orphaned-active-task", "finished-unfinalized-run", "stale-loop-trunk-marker", "terminal-task-event", "closed-unmerged-pr"} {
		if !got[want] {
			t.Errorf("missing %s in %#v", want, got)
		}
	}
	if p.Schema != DeliverySchemaVersion || p.Version != 1 || !p.Reconciled {
		t.Fatalf("projection envelope = %+v", p)
	}
}

func TestGitHubFailureIsUnknownAndNotReconciled(t *testing.T) {
	w, _ := fixture(t)
	old := ObserveDeliveryPRs
	ObserveDeliveryPRs = func(string) ([]DeliveryPR, error) { return nil, fmt.Errorf("authentication required") }
	t.Cleanup(func() { ObserveDeliveryPRs = old })
	p, err := ReconcileDelivery(w, "core", time.Unix(1, 0))
	if err == nil || p.Reconciled {
		t.Fatalf("err=%v reconciled=%t, want closed failure", err, p.Reconciled)
	}
	found := false
	for _, f := range p.Findings {
		if f.Classification == "github-state-unknown" && f.Confidence == DeliveryUnknown {
			found = true
		}
	}
	if !found {
		t.Fatalf("unknown GitHub classification absent: %+v", p.Findings)
	}
}

func treeDigest(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
			return nil
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		fmt.Fprintf(h, "%s\x00", strings.TrimPrefix(path, root))
		_, _ = h.Write(raw)
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil))
}
