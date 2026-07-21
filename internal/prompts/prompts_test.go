package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPreamble(t *testing.T) {
	out, err := Render("", "protocol_preamble", map[string]any{
		"ChildID": "a-x", "Grant": "ro", "Ref": "008", "Slug": "audit",
		"Project": "core", "Exe": "/usr/bin/dacli", "RW": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## How to report", "a-x", "/usr/bin/dacli note add finding", "read-only", "never retry it"} {
		if !strings.Contains(out, want) {
			t.Errorf("preamble missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "task check") {
		t.Error("ro preamble must not offer box-checking")
	}
	rw, _ := Render("", "protocol_preamble", map[string]any{
		"ChildID": "a-x", "Grant": "rw", "Ref": "008", "Slug": "s", "Project": "p", "Exe": "d", "RW": true,
	})
	if !strings.Contains(rw, "task check") || !strings.Contains(rw, "task done") {
		t.Error("rw preamble must include the completion verbs")
	}
}

// The override rule: a workspace file of the same name wins over the
// embedded default — prompt tuning becomes a workspace commit, not a rebuild.
func TestWorkspaceOverrideWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "refusal_next.md"), []byte("custom: {{.X}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Render(dir, "refusal_next", map[string]any{"X": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "custom: hello" {
		t.Errorf("override not applied: %q", out)
	}
	// And absence falls back to the embedded default.
	def, _ := Render(dir, "brief_header", map[string]any{"TaskID": "t-1", "Est": 5})
	if !strings.Contains(def, "data, not instructions") {
		t.Errorf("embedded fallback broken: %q", def)
	}
}

func TestNamesListsRegistry(t *testing.T) {
	names := Names()
	for _, want := range []string{"protocol_preamble", "supervise_correction", "brief_header", "refusal_next", "mcp_tools"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("registry missing %q (have %v)", want, names)
		}
	}
}

func TestMCPDescSections(t *testing.T) {
	if d := MCPDesc("get_context"); !strings.Contains(d, "Call this FIRST") {
		t.Errorf("get_context desc = %q", d)
	}
	defer func() {
		if recover() == nil {
			t.Error("missing tool section should panic (guarded at init, caught by tests)")
		}
	}()
	MCPDesc("no_such_tool")
}
