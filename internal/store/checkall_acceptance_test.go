package store

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// TestCheckAllAcceptancePreservesProseAndNesting verifies that CheckAllAcceptance
// marks checkboxes done while preserving prose lines, blank lines, and nested
// checkbox indentation (dacli 335).
func TestCheckAllAcceptancePreservesProseAndNesting(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := CreateProject(w, "a-root", "P", "p", "g", ""); err != nil {
		t.Fatal(err)
	}

	// Create a task with an Acceptance section containing:
	// - Leading prose line
	// - Regular unchecked checkbox
	// - A blank line
	// - A nested unchecked checkbox
	// - A trailing prose line
	tk, err := CreateTask(w, "a-root", "p", "test task", TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}

	acceptance := `This is the acceptance criteria.
- [ ] First requirement
- [ ] Second requirement

  - [ ] Nested sub-requirement

This is a trailing note about implementation.
`

	tk.Doc.SetSection("Acceptance", acceptance)
	if err := SaveTask(tk); err != nil {
		t.Fatal(err)
	}

	// Call CheckAllAcceptance to mark all boxes done
	newly := CheckAllAcceptance(tk)
	if newly != 3 {
		t.Errorf("CheckAllAcceptance returned %d newly-checked boxes, want 3", newly)
	}

	if err := SaveTask(tk); err != nil {
		t.Fatal(err)
	}

	// Re-read the task from disk to verify the written state
	got, err := FindTask(w, tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	sec, ok := got.Doc.Section("Acceptance")
	if !ok {
		t.Fatal("Acceptance section not found after CheckAllAcceptance")
	}

	content := sec.Content
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	// Verify that all non-checkbox lines are preserved exactly
	checks := []struct {
		line   int
		expect string
	}{
		{0, "This is the acceptance criteria."},
		{1, "- [x] First requirement"},
		{2, "- [x] Second requirement"},
		{3, ""},
		{4, "  - [x] Nested sub-requirement"},
		{5, ""},
		{6, "This is a trailing note about implementation."},
	}

	if len(lines) != 7 {
		t.Errorf("Acceptance section has %d lines, want 7", len(lines))
		t.Logf("Content:\n%q", content)
	}

	for _, c := range checks {
		if c.line >= len(lines) {
			t.Errorf("Line %d missing; content has %d lines", c.line, len(lines))
			continue
		}
		if lines[c.line] != c.expect {
			t.Errorf("Line %d: got %q, want %q", c.line, lines[c.line], c.expect)
		}
	}

	// Verify that all checkboxes are marked [x]
	boxes := mdstore.Checkboxes(content)
	if len(boxes) != 3 {
		t.Errorf("Found %d checkboxes, want 3", len(boxes))
	}
	for i, box := range boxes {
		if !box.Done {
			t.Errorf("Checkbox %d is not marked done", i)
		}
	}
}
