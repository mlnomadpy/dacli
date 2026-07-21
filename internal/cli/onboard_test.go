package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adopt onboards an existing repo: inits, creates a project, writes a
// codebase map that reaches briefs, and (--todos) seeds tasks from markers.
func TestAdoptExistingRepo(t *testing.T) {
	dir := t.TempDir()
	// A pre-existing project with no dacli workspace.
	writeAt(t, dir, "README.md", "# Ledger Service\n\nHandles money.\n")
	writeAt(t, dir, "internal/pay/pay.go", "package pay\n\n// TODO: handle the batch path\nfunc Pay() {}\n")
	writeAt(t, dir, "internal/pay/pay_test.go", "package pay\n\n// FIXME flaky under -race\n")
	writeAt(t, dir, "web/app.ts", "// XXX no error handling\nexport const app = 1\n")
	writeAt(t, dir, "docs/design.md", "# Design\n")

	// adopt with no workspace: inits + project + map, infers goal from README.
	out := run(t, dir, 0, "adopt")
	if !strings.Contains(out, "project ") || !strings.Contains(out, "codebase map written") {
		t.Fatalf("adopt did not onboard:\n%s", out)
	}
	if !strings.Contains(out, "markers found") || !strings.Contains(out, "--todos") {
		t.Errorf("adopt should report unadopted markers:\n%s", out)
	}

	slug := filepath.Base(dir)
	// The goal came from the README heading, not a placeholder.
	show := run(t, dir, 0, "project", "show", slug)
	if !strings.Contains(show, "Continue Ledger Service") {
		t.Errorf("goal not inferred from README:\n%s", show)
	}
	// The codebase map reaches a brief. Add a task, check its brief.
	run(t, dir, 0, "task", "add", "First real task", "--project", slug, "--accept", "a")
	brief := run(t, dir, 0, "context", "first-real-task")
	for _, want := range []string{"Codebase map", "Go (", "TypeScript", "internal/", "handle the batch path"} {
		if !strings.Contains(brief, want) {
			t.Errorf("codebase map missing %q from brief:\n%s", want, brief)
		}
	}

	// Re-run with --todos: markers become tasks with file:line context.
	todoOut := run(t, dir, 0, "adopt", "--todos")
	if !strings.Contains(todoOut, "seeded 3 task(s)") {
		t.Errorf("todos not seeded:\n%s", todoOut)
	}
	tasks := run(t, dir, 0, "task", "list", "--project", slug)
	if !strings.Contains(tasks, "handle-the-batch-path") {
		t.Errorf("TODO not turned into a task:\n%s", tasks)
	}
	show = run(t, dir, 0, "task", "show", "handle-the-batch-path")
	if !strings.Contains(show, "pay.go:3") || !strings.Contains(show, "TODO marker") {
		t.Errorf("task context missing file:line:\n%s", show)
	}
}

func writeAt(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
