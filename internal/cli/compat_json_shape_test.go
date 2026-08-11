package cli

import (
	"encoding/json"
	"testing"
)

// TestDocumentedJSONShapesStillParse is the enforcement half of
// docs/COMPATIBILITY.md: it decodes the two document-emitting --json surfaces
// (`context`, `task list`) and checks every field that document commits to is
// present with the documented type. A rename or removal fails this test, not
// just a sentence in a doc nobody re-reads before shipping.
//
// It deliberately does NOT fail on an EXTRA field: docs/COMPATIBILITY.md and
// FORMAT.md both promise additive-only changes, so a new field is compatible
// and this test must not punish adding one.
func TestDocumentedJSONShapesStillParse(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a real goal for this project")
	run(t, dir, 0, "task", "add", "a task", "--project", "p", "--accept", "it works")

	t.Run("context --json", func(t *testing.T) {
		out, msg, code := executor(dir)([]string{"context", "001"}, true)
		if code != 0 {
			t.Fatalf("context --json: exit %d: %s", code, msg)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("context --json did not parse as an object: %v\n%s", err, out)
		}

		var taskID string
		requireField(t, doc, "task_id", &taskID)
		if taskID == "" {
			t.Errorf("task_id must be non-empty, got %q", taskID)
		}

		var omitted []string
		requireField(t, doc, "omitted", &omitted) // may be empty, must be an array

		var sections []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		requireField(t, doc, "sections", &sections)
		if len(sections) == 0 {
			t.Errorf("sections must be a non-empty array for a real task")
		}
	})

	t.Run("task list --json", func(t *testing.T) {
		out, msg, code := executor(dir)([]string{"task", "list", "--project", "p"}, true)
		if code != 0 {
			t.Fatalf("task list --json: exit %d: %s", code, msg)
		}
		var tasks []map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &tasks); err != nil {
			t.Fatalf("task list --json did not parse as an array: %v\n%s", err, out)
		}
		if len(tasks) == 0 {
			t.Fatalf("expected at least one task in the fixture, got none")
		}
		task := tasks[0]

		var s string
		requireField(t, task, "id", &s)
		if s == "" {
			t.Errorf("id must be non-empty")
		}
		var n int
		requireField(t, task, "seq", &n)
		requireField(t, task, "slug", &s)
		requireField(t, task, "project", &s)
		requireField(t, task, "status", &s)
		requireField(t, task, "title", &s)
		requireField(t, task, "acceptance_done", &n)
		requireField(t, task, "acceptance_total", &n)
		// priority is documented as omitempty — it must decode as a string when
		// present, but is not required to be present on every task.
		if raw, ok := task["priority"]; ok {
			var p string
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Errorf("priority must be a string when present: %v", err)
			}
		}
	})
}

// requireField fails the test if key is missing from doc, or if its value
// does not unmarshal into the documented Go type of dst.
func requireField(t *testing.T, doc map[string]json.RawMessage, key string, dst any) {
	t.Helper()
	raw, ok := doc[key]
	if !ok {
		t.Errorf("documented field %q is missing from the response", key)
		return
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Errorf("documented field %q has the wrong shape: %v (raw: %s)", key, err, raw)
	}
}
