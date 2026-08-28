package reconciliation

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestCommandRendersSameVersionedProjectionAsTextAndJSON(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "reconcile")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = w.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatal(err)
	}
	old := store.ObserveDeliveryPRs
	store.ObserveDeliveryPRs = func(string) ([]store.DeliveryPR, error) { return nil, nil }
	t.Cleanup(func() { store.ObserveDeliveryPRs = old })
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdReconcile(ctx, []string{"--project", "core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	textOut := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(textOut, store.DeliverySchemaVersion) || !strings.Contains(textOut, "dry-run: read-only projection") {
		t.Fatalf("human rendering omitted contract:\n%s", textOut)
	}
	jsonCtx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdReconcile(jsonCtx, []string{"--project", "core"}); err != nil {
		t.Fatal(err)
	}
	jsonOut := jsonCtx.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{`"schema": "` + store.DeliverySchemaVersion + `"`, `"version": 1`, `"findings":`} {
		if !strings.Contains(jsonOut, want) {
			t.Errorf("JSON missing %q:\n%s", want, jsonOut)
		}
	}
}
