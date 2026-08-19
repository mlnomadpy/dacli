package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDeclaration = "<!-- dacli-prompt schema: dacli-prompt/v1 base: autonomous-delivery/v1 -->\n"

func TestRenderPreamble(t *testing.T) {
	taskID := "t-01KZYQ5EF1YQ05NRA8GW3N9PQM"
	out, err := Render("", "protocol_preamble", map[string]any{
		"ChildID": "a-x", "Grant": "ro", "Ref": taskID, "Slug": "audit",
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
		"ChildID": "a-x", "Grant": "rw", "Ref": taskID, "Slug": "s", "Project": "p", "Exe": "d", "RW": true,
	})
	if !strings.Contains(rw, "task check") || !strings.Contains(rw, "task done") {
		t.Error("rw preamble must include the completion verbs")
	}
	for _, command := range []string{"task check " + taskID, "task done " + taskID} {
		if !strings.Contains(rw, command) {
			t.Errorf("rw preamble must use stable task ID in %q", command)
		}
	}
}

// The RW and RO preambles describe different lifecycles: RW agents follow
// claim -> work -> commit -> pr -> accept/ship, while RO agents follow
// claim -> work -> report -> sync (via the propose-and-sync path).
func TestROAndRWPreamblesDescribeDifferentLifecycles(t *testing.T) {
	ro, _ := Render("", "protocol_preamble", map[string]any{
		"ChildID": "a-ro", "Grant": "ro", "Ref": "287", "Slug": "test",
		"Project": "core", "Exe": "/usr/bin/dacli", "RW": false,
	})
	rw, _ := Render("", "protocol_preamble", map[string]any{
		"ChildID": "a-rw", "Grant": "rw", "Ref": "287", "Slug": "test",
		"Project": "core", "Exe": "/usr/bin/dacli", "RW": true,
	})

	// RW agents should see the commit -> pr lifecycle
	if !strings.Contains(rw, "commit") {
		t.Error("rw preamble must mention committing")
	}
	if !strings.Contains(rw, "pr") || !strings.Contains(rw, "PR") {
		t.Error("rw preamble must mention opening a PR")
	}

	// RO agents must NOT see the commit -> pr lifecycle
	if strings.Contains(ro, "commit") {
		t.Errorf("ro preamble must not mention committing, but got:\n%s", ro)
	}
	// RO preamble should describe propose-and-sync instead
	// (either directly mentioning "propose" and "sync", or describing the report/findings path)
	hasProposePath := strings.Contains(ro, "propose") || strings.Contains(ro, "sync")
	if !hasProposePath && !strings.Contains(ro, "report") {
		t.Errorf("ro preamble must describe propose-and-sync path or report mechanism, got:\n%s", ro)
	}
}

// The override rule: a workspace file of the same name wins over the
// embedded default — prompt tuning becomes a workspace commit, not a rebuild.
func TestWorkspaceOverrideWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "refusal_next.md"), []byte(testDeclaration+"custom: {{.X}}"), 0o644); err != nil {
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

func TestOverrideRequiresCompatibleBaseVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "refusal_next.md")
	if err := os.WriteFile(path, []byte("custom without a base"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Render(dir, "refusal_next", nil); err == nil || !strings.Contains(err.Error(), "missing declaration") {
		t.Fatalf("undeclared override error = %v", err)
	}
	if err := os.WriteFile(path, []byte("<!-- dacli-prompt schema: dacli-prompt/v1 base: autonomous-delivery/v0 -->\nold"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Render(dir, "refusal_next", nil); err == nil || !strings.Contains(err.Error(), "incompatible base version") {
		t.Fatalf("incompatible override error = %v", err)
	}
}

func TestAutonomousContractGoldenAndRequiredBehavior(t *testing.T) {
	contract, err := AutonomousContract("")
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "autonomous_delivery.golden"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(golden))
	got := contract.Schema + " " + contract.Version + " sha256:" + contract.Hash
	if got != want {
		t.Fatalf("autonomous delivery contract changed\n got: %s\nwant: %s", got, want)
	}
	if len(contract.Text) > 7000 {
		t.Fatalf("contract grew beyond its 7000-byte token guardrail: %d", len(contract.Text))
	}

	// Behavioral fixtures name the observable safe action each scenario must
	// make available. Removing the governing clause makes the fixture fail.
	scenarios := []struct {
		name, evidence, action string
	}{
		{"duplicate work", "never create a\nsecond task", "do not file"},
		{"critical path", "select ready\nzero-slack work first", "select critical-path work"},
		{"cheap capable model", "cheapest available model profile", "prefer cheap capable profile"},
		{"high consequence uplift", "Uplift for high-consequence", "uplift and independently review"},
		{"exit three", "Exit 3 is a policy answer and is\nnever retried unchanged", "stop without retry"},
		{"empty audit", "reports an honest empty cycle; it never invents backlog", "report empty"},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name+" chooses "+scenario.action, func(t *testing.T) {
			if !strings.Contains(contract.Text, scenario.evidence) {
				t.Errorf("contract no longer chooses %q; missing %q", scenario.action, scenario.evidence)
			}
		})
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
