package cli

import (
	"bytes"
	"testing"

	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// runJSON runs a command with ctx.JSON set, returning stdout only.
func runJSON(t *testing.T, dir string, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	ctx := &Ctx{Stdout: &out, Stderr: &errb, Cwd: dir, JSON: true}
	cmd, rest := match(args)
	if cmd == nil {
		t.Fatalf("no such command: %v", args)
	}
	if err := cmd.Run(ctx, rest); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, errb.String())
	}
	return out.String()
}

// findTask resolves a ref to its ULID id.
func findTask(t *testing.T, dir, ref string) string {
	t.Helper()
	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := store.FindTask(w, ref)
	if err != nil {
		t.Fatal(err)
	}
	return tk.ID
}

// appendEvent writes an event as an arbitrary actor — the same file a real
// read-only child would produce, without needing a second process.
func appendEvent(t *testing.T, w *workspace.Workspace, actor, kind, about, body string) {
	t.Helper()
	if _, err := eventlog.Append(w, actor, model.EventKind(kind), about, "", body); err != nil {
		t.Fatal(err)
	}
}
