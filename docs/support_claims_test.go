package docs_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestLifecycleBoundaryLanguage distinguishes the current product contract
// from the wording retained in dated research evidence (issue #825). An
// unqualified "runs agents, not work" can be read as denying the lifecycle
// orchestration the shipped loop performs; research quotes may keep it only
// when the artifact explicitly identifies it as historical terminology.
func TestLifecycleBoundaryLanguage(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(here), ".."))
	read := func(name string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(body)
	}

	runtimes := read("docs/RUNTIMES.md")
	for _, want := range []string{
		"governs coding-agent lifecycles",
		"agents do the engineering work",
		"arbitrary DAG of jobs",
		"cron trigger",
	} {
		if !strings.Contains(runtimes, want) {
			t.Errorf("docs/RUNTIMES.md missing lifecycle boundary %q", want)
		}
	}

	const stale = "runs agents, not work"
	const historicalMarker = "Historical terminology note:"
	historicalResearch := map[string]bool{
		"docs/research/DASHBOARD_UX_RESEARCH.md":     true,
		"docs/research/INTERVIEW_GUIDE.md":           true,
		"docs/research/interviews/adopter.md":        true,
		"docs/research/interviews/operator.md":       true,
		"docs/research/interviews/reviewer-agent.md": true,
	}
	for name := range historicalResearch {
		body := read(name)
		if !strings.Contains(strings.ToLower(body), stale) {
			t.Errorf("%s no longer preserves the historical wording it analyzes", name)
		}
		if !strings.Contains(body, historicalMarker) {
			t.Errorf("%s retains the old boundary without labeling it historical", name)
		}
		for _, want := range []string{"governs coding-agent lifecycles", "agents do the", "engineering work"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s does not point readers from historical wording to the current boundary %q", name, want)
			}
		}
	}

	err := fs.WalkDir(os.DirFS(root), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch name {
			case ".git", ".dacli", "node_modules":
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(name) != ".md" {
			return nil
		}
		body, readErr := fs.ReadFile(os.DirFS(root), name)
		if readErr != nil {
			return readErr
		}
		if !strings.Contains(strings.ToLower(string(body)), stale) {
			return nil
		}
		clean := filepath.ToSlash(name)
		if historicalResearch[clean] || clean == "docs/REVIEW.md" {
			return nil
		}
		t.Errorf("%s contains unqualified stale lifecycle boundary %q", clean, stale)
		return nil
	})
	if err != nil {
		t.Fatalf("scan markdown lifecycle claims: %v", err)
	}
}

func TestOperatorGuidanceDoesNotPresentClosedDefectsAsCurrent(t *testing.T) {
	for _, tc := range []struct {
		path string
		bad  []string
	}{
		{"../skills/dacli/references/swarms-loops.md", []string{"open defects", "#629", "#641"}},
		{"../skills/dacli/references/github-landing.md", []string{"issue #651 tracks", "Until it is fixed"}},
		{"OPERATOR_PLAYBOOK.md", []string{"no task-edit command that can add missing criteria"}},
		{"../skills/dacli/references/critical-path-github.md", []string{"no task-edit command that can add missing criteria"}},
	} {
		raw, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range tc.bad {
			if strings.Contains(string(raw), bad) {
				t.Errorf("%s still presents retired guidance %q as current", tc.path, bad)
			}
		}
	}
}

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
	for _, file := range []string{
		"docs/OPERATOR_PLAYBOOK.md",
		"skills/dacli/references/critical-path-github.md",
		"skills/dacli/references/github-landing.md",
	} {
		if body := read(file); !strings.Contains(body, "github pull <project> --dry-run") {
			t.Errorf("%s must document the shipped inbound-only GitHub preview", file)
		}
	}
	ghMirrorSource := read("internal/features/ghmirror/ghmirror.go") + read("internal/features/ghmirror/adoption.go")
	for _, want := range []string{`Usage: "dacli github pull <project> [--dry-run]"`, `f.Reject("dry-run")`, `func pullParsed`} {
		if !strings.Contains(ghMirrorSource, want) {
			t.Errorf("canonical pull preview requires the exact shipped command contract %q", want)
		}
	}
	syncStart := strings.Index(ghMirrorSource, "func cmdSync")
	if syncStart < 0 {
		t.Fatal("cannot locate github sync implementation")
	}
	syncEnd := strings.Index(ghMirrorSource[syncStart:], "// --- findings")
	if syncEnd < 0 {
		t.Fatal("cannot locate end of github sync implementation")
	}
	syncSource := ghMirrorSource[syncStart : syncStart+syncEnd]
	validation := `f.Reject("findings-as-issues", "with-tasks", "since", "include-internal", "dry-run")`
	if rejectAt, pullAt := strings.Index(syncSource, validation), strings.Index(syncSource, "pullParsed(ctx, f)"); rejectAt < 0 || pullAt < 0 || rejectAt > pullAt {
		t.Error("github sync must validate its complete flag union before entering the mutating pull half")
	}
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
		"Before owner acceptance or GitHub issue closure",
		"observe both the merged PR and its commit on trunk",
		"separate governed wave transaction that accepts and integrates",
		"No dedicated runtime-cooldown clear or expiry command is shipped",
		"`github sync <project> --dry-run` preview projection changes first",
		"do not use `critical-path` or `next` to claim that incomplete graph is authoritative",
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
	for _, want := range []string{`Path: "next"`, `f.Reject("parallel", "project", "critical-path")`} {
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
	recovery := read("skills/dacli/references/recovery.md")
	if strings.Contains(recovery, "dacli project rm <project> --force   # inspect exact project first") {
		t.Error("recovery reference must not present irreversible project deletion as a preview")
	}
	if !strings.Contains(recovery, "has no preview mode and irreversibly") {
		t.Error("recovery reference must state the project deletion boundary explicitly")
	}
	swarms := read("skills/dacli/references/swarms-loops.md")
	for _, want := range []string{"--impl-role <implementer>", "--review-role <reviewer>", "dacli logs <run-id-prefix|child-id> -f"} {
		if !strings.Contains(swarms, want) {
			t.Errorf("swarms-loops reference missing exact executable guidance %q", want)
		}
	}

	mcpDoc := read("docs/MCP.md")
	compatDoc := read("docs/COMPATIBILITY.md")
	for _, want := range []string{"Twenty-two schemas", "`check_task`", "`release_train`", "`github_projection`", "`reconcile_delivery`", "manually maintained"} {
		if !strings.Contains(mcpDoc, want) {
			t.Errorf("docs/MCP.md missing shipped-surface claim %q", want)
		}
	}
	if !strings.Contains(compatDoc, "twenty-two Tier-1 tools") || !strings.Contains(compatDoc, "manually maintained") {
		t.Error("docs/COMPATIBILITY.md must describe the twenty-two manually maintained Tier-1 tools")
	}
	toolSource := read("internal/mcp/tools.go")
	toolTable := toolSource[strings.Index(toolSource, "var tools = []tool{"):strings.Index(toolSource, "// refCmd builds")]
	if got := strings.Count(toolTable, "\n\t\tname:"); got != 22 { // twenty-one core schemas plus cli
		t.Errorf("MCP tool table has %d entries; update the documented support surface with it", got)
	}

	runtimes := read("docs/RUNTIMES.md")
	for _, want := range []string{"nine shipped presets", "require user configuration", "`copilot-rw`"} {
		if !strings.Contains(runtimes, want) {
			t.Errorf("docs/RUNTIMES.md missing preset boundary %q", want)
		}
	}
	presetSource := read("internal/features/execution/execution.go") + read("internal/features/execution/runtime_adapters.go")
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
