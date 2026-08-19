package docs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestPublicSupportClaimsMatchShippedSurface keeps the small support matrix
// honest. These documents used to claim generated MCP schemas, unshipped
// runtime presets, unsupported lint flags, and an unreleased state after
// v0.1.0 was tagged (task 367).
func TestPublicSupportClaimsMatchShippedSurface(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(here), ".."))
	read := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(b)
	}

	playbook := read("docs/OPERATOR_PLAYBOOK.md")
	for _, want := range []string{
		"Choose the smallest operating profile",
		"GitHub-first critical-path cycle",
		"Continuous means repeated bounded transactions with durable checkpoints",
		"`dacli push <ref>`",
		"`dacli logs <run-id-prefix|child-id> -f`",
		"`dacli project show <slug> --landing-mode pr --landing-base main`",
		"`dacli retro <task-or-project-ref> --well \"...\" --bad \"...\" --improve \"...\"`",
		"Direct task references are workspace-wide",
		"## Shipped, experimental, and future",
	} {
		if !strings.Contains(playbook, want) {
			t.Errorf("docs/OPERATOR_PLAYBOOK.md missing canonical operator guidance %q", want)
		}
	}
	pushSource := read("internal/features/vcs/lifecycle.go")
	if !strings.Contains(pushSource, `Path: "push"`) {
		t.Error("docs/OPERATOR_PLAYBOOK.md documents dacli push, but the branch-push command is absent")
	}
	nextSource := read("internal/features/insight/insight.go")
	for _, want := range []string{`Path: "next"`, `f.Reject("parallel", "project")`} {
		if !strings.Contains(nextSource, want) {
			t.Errorf("docs/OPERATOR_PLAYBOOK.md documents critical-path scheduling, but next surface is missing %q", want)
		}
	}
	skill := read("skills/dacli/SKILL.md")
	for _, want := range []string{
		"references/operating-profiles.md",
		"references/model-economics.md",
		"references/critical-path-github.md",
		"references/continuous-operations.md",
		"references/roster-design.md",
		"references/swarms-loops.md",
		"references/recovery.md",
		"references/github-landing.md",
	} {
		if !strings.Contains(skill, want) {
			t.Errorf("skills/dacli/SKILL.md must route to focused reference %q", want)
		}
	}
	continuous := read("skills/dacli/references/continuous-operations.md")
	for _, bad := range []string{"Circuit breakers/cooldowns stop retry storms", "terminal failures become blocked or dead-lettered work"} {
		if strings.Contains(continuous, bad) {
			t.Errorf("continuous-operations reference presents future service behavior as shipped: %q", bad)
		}
	}
	swarms := read("skills/dacli/references/swarms-loops.md")
	for _, want := range []string{"--impl-role <implementer>", "--review-role <reviewer>", "dacli logs <run-id-prefix|child-id> -f"} {
		if !strings.Contains(swarms, want) {
			t.Errorf("swarms-loops reference missing exact executable guidance %q", want)
		}
	}

	mcpDoc := read("docs/MCP.md")
	compatDoc := read("docs/COMPATIBILITY.md")
	for _, want := range []string{"Fifteen schemas", "`check_task`", "manually maintained"} {
		if !strings.Contains(mcpDoc, want) {
			t.Errorf("docs/MCP.md missing shipped-surface claim %q", want)
		}
	}
	if !strings.Contains(compatDoc, "fifteen Tier-1 tools") || !strings.Contains(compatDoc, "manually maintained") {
		t.Error("docs/COMPATIBILITY.md must describe the fifteen manually maintained Tier-1 tools")
	}
	toolSource := read("internal/mcp/tools.go")
	toolTable := toolSource[strings.Index(toolSource, "var tools = []tool{"):strings.Index(toolSource, "// refCmd builds")]
	if got := strings.Count(toolTable, "\n\t\tname:"); got != 16 { // fifteen core schemas plus cli
		t.Errorf("MCP tool table has %d entries; update the documented support surface with it", got)
	}

	runtimes := read("docs/RUNTIMES.md")
	for _, want := range []string{"nine shipped presets", "require user configuration", "`copilot-rw`"} {
		if !strings.Contains(runtimes, want) {
			t.Errorf("docs/RUNTIMES.md missing preset boundary %q", want)
		}
	}
	presetSource := read("internal/features/execution/execution.go")
	presetTable := presetSource[strings.Index(presetSource, "var presets = map[string]store.Runtime{"):strings.Index(presetSource, "func cmdRuntimeAdd")]
	for _, want := range []string{`"claude-code":`, `"claude-code-rw":`, `"codex":`, `"codex-rw":`, `"gemini":`, `"gemini-rw":`, `"copilot":`, `"copilot-rw":`, `"generic-exec":`} {
		if !strings.Contains(presetTable, want) {
			t.Errorf("shipped runtime table missing documented preset %s", want)
		}
	}
	if got := strings.Count(presetTable, `": {`); got != 9 {
		t.Errorf("runtime table has %d presets; update docs/RUNTIMES.md with it", got)
	}

	for _, tc := range []struct {
		file string
		bad  []string
	}{
		{"DESIGN.md", []string{"spec only — runtime"}},
		{"README.md", []string{"first tagged release", "once the first release is tagged", "lint --ambiguity"}},
		{"docs/SPM.md", []string{"lint --ambiguity", "`--strict`"}},
		{"SECURITY.md", []string{"pre-1.0 and unreleased"}},
	} {
		body := read(tc.file)
		for _, bad := range tc.bad {
			if strings.Contains(body, bad) {
				t.Errorf("%s retains stale public claim %q", tc.file, bad)
			}
		}
	}
}
