package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func handoffRepo(t *testing.T) *workspace.Workspace {
	t.Helper()
	root := t.TempDir()
	w, err := workspace.Init(root, "handoff")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init"}, {"config", "user.name", "Test"}, {"config", "user.email", "test@example.invalid"}} {
		if _, err := gitx.Run(root, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".dacli/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(root, "add", ".gitignore", "source.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(root, "commit", "-m", "fixture"); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestRootHandoffCapturesStructuredEvidenceAndRefusesStaleConsumption(t *testing.T) {
	w := handoffRepo(t)
	runID := "01ROOT-HANDOFF-TEST"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	req := RootHandoffRequest{
		Schema:       RootHandoffSchema,
		Verification: []RootHandoffVerification{{Command: "go test ./...", ExitCode: 0, Result: "pass"}},
		Unresolved:   []string{"publish event"}, FailedOperation: "git index lock", FailureClass: "filesystem_sandbox_refusal",
		Stderr: "permission denied", NextAction: "owner commits after re-observation",
	}
	raw, _ := json.Marshal(req)
	if err := os.WriteFile(filepath.Join(w.RunDir(runID), RootHandoffRequestFile), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, "source.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h, ok, err := CaptureRootHandoff(w, runID, "t-task", "a-worker", w.Root, RootHandoffRequest{}, time.Unix(1, 0))
	if err != nil || !ok {
		t.Fatalf("capture = ok %t, err %v", ok, err)
	}
	if h.Schema != RootHandoffSchema || h.TaskID != "t-task" || h.RunID != runID || h.ChildID != "a-worker" {
		t.Fatalf("identity/schema not derived by owner: %#v", h)
	}
	if len(h.ChangedPaths) != 1 || h.ChangedPaths[0].Path != "source.txt" || h.ChangedPaths[0].SHA256 == "" || h.DiffSHA256 == "" || h.TreeSHA256 == "" {
		t.Fatalf("exact changed evidence missing: %#v", h)
	}
	if len(h.Verification) != 1 || h.Verification[0].Command != "go test ./..." || h.FailedOperation != "git index lock" || h.Stderr != "permission denied" {
		t.Fatalf("structured lifecycle evidence missing: %#v", h)
	}
	if err := MarkRootHandoffConsumed(w, h, "a-root", time.Unix(2, 0)); err != nil {
		t.Fatalf("fresh consume: %v", err)
	}
	if err := os.WriteFile(filepath.Join(w.Root, "source.txt"), []byte("changed after handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MarkRootHandoffConsumed(w, h, "a-root", time.Unix(3, 0)); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("changed handoff consume = %v, want stale refusal", err)
	}
}

func TestRootHandoffRequiresUsefulWorkOrStructuredRequest(t *testing.T) {
	w := handoffRepo(t)
	runID := "01ROOT-HANDOFF-EMPTY"
	if err := os.MkdirAll(w.RunDir(runID), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := CaptureRootHandoff(w, runID, "t-task", "a-worker", w.Root, RootHandoffRequest{}, time.Now()); err != nil || ok {
		t.Fatalf("empty capture = ok %t, err %v; want no handoff", ok, err)
	}
}
