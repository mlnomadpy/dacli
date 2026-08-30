package docs_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstalledSkillAndReleaseGuidanceAreCurrent(t *testing.T) {
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

	skill := read("skills/dacli/SKILL.md")
	for _, want := range []string{
		"dacli version --compatibility --json",
		"dacli capabilities --json",
		"dacli explain --project <project> --json",
		"https://github.com/mlnomadpy/dacli/blob/main/docs/OPERATOR_PLAYBOOK.md",
	} {
		if !strings.Contains(skill, want) {
			t.Errorf("installed skill guidance missing %q", want)
		}
	}
	if strings.Contains(skill, "../../docs/OPERATOR_PLAYBOOK.md") {
		t.Error("installed skill retains a source-tree-relative playbook link")
	}

	var requirements struct {
		Required []struct {
			ID string `json:"id"`
		} `json:"required"`
	}
	if err := json.Unmarshal([]byte(read("skills/dacli/capabilities.json")), &requirements); err != nil {
		t.Fatalf("parse skill capabilities: %v", err)
	}
	got := map[string]bool{}
	for _, requirement := range requirements.Required {
		got[requirement.ID] = true
	}
	for _, want := range []string{
		"cli.command.capabilities",
		"cli.command.version.flag.compatibility",
		"cli.command.whoami",
		"cli.command.status",
		"cli.command.next",
		"cli.command.explain",
		"cli.command.agents",
		"cli.command.loop.status",
	} {
		if !got[want] {
			t.Errorf("skill capability requirements missing %q", want)
		}
	}

	for name, bad := range map[string][]string{
		"README.md":             {"The direct-download path starts", "Until that release exists"},
		"docs/index.md":         {"first tagged release", "brew install mlnomadpy/tap/dacli"},
		"docs/COMPATIBILITY.md": {"Today that is three document emitters"},
	} {
		body := read(name)
		for _, phrase := range bad {
			if strings.Contains(body, phrase) {
				t.Errorf("%s retains stale guidance %q", name, phrase)
			}
		}
	}
}

// TestPublicDocsMatchCurrentAgentContracts protects the public entry points
// against documentation drift that sends an orchestrator down an invalid or
// over-disclosing path (issue #918). The command registry is authoritative;
// these assertions cover only the few policy boundaries the narrative must
// teach before directing an agent to capabilities --json.
func TestPublicDocsMatchCurrentAgentContracts(t *testing.T) {
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

	readme := read("README.md")
	for _, want := range []string{
		"public-safe task fields",
		"--include-internal",
		"primary shipped surface",
		"dacli start",
		"dacli task aggregate",
		"dacli review projection",
		"dacli pr diagnose",
		"dacli release train",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README.md missing current agent contract %q", want)
		}
	}
	for _, stale := range []string{
		"tasks→issues, decisions→issues, findings→issues",
		"The full shipped surface, grouped.",
	} {
		if strings.Contains(readme, stale) {
			t.Errorf("README.md retains stale claim %q", stale)
		}
	}

	walkthrough := read("docs/WALKTHROUGH.md")
	for _, want := range []string{
		"dacli github sync ledger --dry-run",
		"dacli review projection --task",
		"independent-review-result/v1",
	} {
		if !strings.Contains(walkthrough, want) {
			t.Errorf("docs/WALKTHROUGH.md missing executable lifecycle guidance %q", want)
		}
	}
	if strings.Contains(walkthrough, "dacli github sync --dry-run") {
		t.Error("docs/WALKTHROUGH.md retains project-less github sync")
	}
	if strings.Contains(walkthrough, "finding lands as\n                           # an issue comment") {
		t.Error("docs/WALKTHROUGH.md claims default public sync publishes an internal finding")
	}

	selfHosting := read("docs/SELFHOSTING.md")
	for _, want := range []string{"structured delivery", "independent-review-result/v1", "selected harness"} {
		if !strings.Contains(selfHosting, want) {
			t.Errorf("docs/SELFHOSTING.md missing current review boundary %q", want)
		}
	}
	if strings.Contains(selfHosting, "nobody, and nothing, ever having read the diff") {
		t.Error("docs/SELFHOSTING.md retains the pre-structured-review loop claim")
	}

	architecture := read("docs/ARCHITECTURE.md")
	for _, stale := range []string{
		"Every command accepts `--json`",
		"fourteen core tools",
		"`--explain` (proposed",
	} {
		if strings.Contains(architecture, stale) {
			t.Errorf("docs/ARCHITECTURE.md retains stale interface claim %q", stale)
		}
	}
	for _, want := range []string{
		"`clikit.Command.JSON`",
		"`dacli capabilities --json`",
		"`dacli explain`",
	} {
		if !strings.Contains(architecture, want) {
			t.Errorf("docs/ARCHITECTURE.md missing current interface contract %q", want)
		}
	}

	contributing := read("CONTRIBUTING.md")
	for _, want := range []string{"`dacli release train`", "`dacli github release`"} {
		if !strings.Contains(contributing, want) {
			t.Errorf("CONTRIBUTING.md missing release authority boundary %q", want)
		}
	}
}

// TestPagesLandingMatchesCurrentContracts covers the real Pages homepage.
// docs/index.md is only a fallback because overrides/home.html empties the
// normal content block; testing the markdown alone allowed stale public copy
// to deploy successfully in issue #920.
func TestPagesLandingMatchesCurrentContracts(t *testing.T) {
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

	home := read("overrides/home.html")
	for _, want := range []string{
		"v0.3.0",
		"--harness codex",
		"--hybrid",
		"public-safe",
		"--include-internal",
		"version --compatibility --json",
		"capabilities --json",
	} {
		if !strings.Contains(home, want) {
			t.Errorf("overrides/home.html missing current Pages contract %q", want)
		}
	}
	if strings.Contains(home, "Homebrew and prebuilt binaries come with the first tagged release") {
		t.Error("overrides/home.html still describes shipped release assets as future")
	}

	mkdocs := read("mkdocs.yml")
	for _, want := range []string{
		"site_url: https://www.tahabouhsine.com/dacli/",
		"Operator playbook: OPERATOR_PLAYBOOK.md",
		"Trust & mutation boundaries: TRUST.md",
		"GitHub App boundary: GITHUB_APP.md",
	} {
		if !strings.Contains(mkdocs, want) {
			t.Errorf("mkdocs.yml missing deployed Pages contract %q", want)
		}
	}
}

// TestPagesLandingDesignContract keeps the custom homepage focused on the
// product's distinctive governed lifecycle rather than drifting back to a
// generic card grid (issue #922).
func TestPagesLandingDesignContract(t *testing.T) {
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

	home := read("overrides/home.html")
	for _, want := range []string{
		"Keep coding-agent swarms moving",
		"Read the operator playbook",
		"dacli-cycle-console",
		"Exact-tree review",
		"What dacli controls",
		"What stays human",
		"Does dacli switch coding CLIs",
		"Can it run forever",
	} {
		if !strings.Contains(home, want) {
			t.Errorf("overrides/home.html missing landing design contract %q", want)
		}
	}
	for _, stale := range []string{"Get started →", "dacli-bento", "02 · what you get"} {
		if strings.Contains(home, stale) {
			t.Errorf("overrides/home.html retains generic landing pattern %q", stale)
		}
	}

	css := read("docs/stylesheets/landing.css")
	for _, want := range []string{
		".dacli-cycle-console",
		":focus-visible",
		"prefers-reduced-motion: reduce",
		"@media (max-width: 720px)",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("landing.css missing visual/accessibility contract %q", want)
		}
	}
}

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
		{"docs/index.md", []string{"first tagged release", "once the first release is tagged", "brew install mlnomadpy/tap/dacli"}},
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
