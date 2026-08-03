package onboard

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// newEnv is an EMPTY directory with no workspace — the exact starting point
// `dacli new` exists for, and the one `dacli adopt` mapped as zero files.
// DACLI_AGENT is cleared so the acting identity is root regardless of who runs
// the suite.
func newEnv(t *testing.T) (string, *clikit.Ctx, *bytes.Buffer) {
	t.Helper()
	t.Setenv("DACLI_AGENT", "")
	dir := t.TempDir()
	out := &bytes.Buffer{}
	return dir, &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: dir}, out
}

func section(t *testing.T, p *store.Project, name string) string {
	t.Helper()
	s, ok := p.Doc.Section(name)
	if !ok {
		t.Fatalf("project has no %q section", name)
	}
	return strings.TrimSpace(s.Content)
}

// The headline guarantee of dacli 191: one command turns an empty directory
// plus a product idea into a workspace, a project whose Goal / Out of scope /
// Success criteria are FILLED from the flags, a Spec and an Architecture
// section, a recorded stack, and an ORDERED backlog — no hand-written tasks.
func TestNewBootstrapsGreenfieldProject(t *testing.T) {
	dir, ctx, out := newEnv(t)

	err := cmdNew(ctx, []string{
		"Ledger Service",
		"--goal", "A service that records double-entry transactions and answers balance queries.",
		"--stack", "go",
		"--out-of-scope", "reporting dashboards",
		"--success", "a balance query returns within one round trip",
	})
	if err != nil {
		t.Fatalf("dacli new: %v", err)
	}

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatalf("new did not initialize a workspace: %v", err)
	}
	p, err := store.LoadProject(w, "ledger-service")
	if err != nil {
		t.Fatalf("load project: %v", err)
	}

	// 1. The goal is the operator's, verbatim — not a placeholder.
	if goal := section(t, p, "Goal"); !strings.Contains(goal, "double-entry transactions") {
		t.Errorf("Goal not filled from --goal: %q", goal)
	}
	// 2. The flags reached the scope and success sections.
	if got := section(t, p, "Out of scope"); !strings.Contains(got, "reporting dashboards") {
		t.Errorf("Out of scope not filled from --out-of-scope: %q", got)
	}
	if got := section(t, p, "Success criteria"); !strings.Contains(got, "one round trip") {
		t.Errorf("Success criteria not filled from --success: %q", got)
	}
	// 3. Spec and Architecture exist — the sections the project object had
	// nowhere to put before this command.
	if spec := section(t, p, "Spec"); len(spec) < 100 || !strings.Contains(spec, "double-entry transactions") {
		t.Errorf("Spec section missing or not seeded from the goal: %q", spec)
	}
	// 4. The stack is recorded with its real build and test commands, in both
	// Architecture and Constraints (the latter is what reaches every brief).
	arch := section(t, p, "Architecture")
	for _, want := range []string{"Go", "go build ./...", "go test ./..."} {
		if !strings.Contains(arch, want) {
			t.Errorf("Architecture missing %q:\n%s", want, arch)
		}
	}
	if cons := section(t, p, "Constraints"); !strings.Contains(cons, "go test ./...") {
		t.Errorf("stack commands do not reach Constraints (and so never reach a brief): %q", cons)
	}

	// 5. The backlog is real and ORDERED: the first rung has no dependency,
	// every later rung depends on an earlier one, and the README waits on the
	// feature slice rather than on nothing.
	tasks, err := store.ListTasks(w, "ledger-service", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) < 5 {
		t.Fatalf("starter backlog has %d task(s), want at least 5", len(tasks))
	}
	byID := map[string]*store.Task{}
	for _, task := range tasks {
		byID[task.ID] = task
		if len(task.Acceptance()) == 0 {
			t.Errorf("task %03d %q has no acceptance criteria", task.Seq, task.Title)
		}
	}
	if deps := tasks[0].Deps(); len(deps) != 0 {
		t.Errorf("first rung should be unblocked, got deps %v", deps)
	}
	edges := 0
	for _, task := range tasks[1:] {
		deps := task.Deps()
		if len(deps) == 0 {
			t.Errorf("task %03d %q has no dependency — the backlog is flat, not a DAG", task.Seq, task.Title)
			continue
		}
		for _, d := range deps {
			dep, ok := byID[d.Ref]
			if !ok {
				t.Errorf("task %03d depends on unresolvable ref %q", task.Seq, d.Ref)
				continue
			}
			if dep.Seq >= task.Seq {
				t.Errorf("task %03d depends on %03d — the arc must point backwards", task.Seq, dep.Seq)
			}
			edges++
		}
	}
	if edges < 4 {
		t.Errorf("backlog has %d dependency edge(s), want the full arc", edges)
	}

	if o := out.String(); !strings.Contains(o, "seeded") || !strings.Contains(o, "Next steps") {
		t.Errorf("new did not print next steps:\n%s", o)
	}
}

// The seeded prose is only worth writing if it clears gates.unfilled — the
// "present but not filled" rule that rejects placeholders, sub-20-character
// content, and major-severity ambiguity. A project born from `dacli new` must
// pass the discovery gate's project_sections check on its own, or the command
// would ship a backlog that is blocked the moment a template is attached.
func TestNewSeededSectionsClearTheStageGate(t *testing.T) {
	dir, ctx, _ := newEnv(t)

	if err := cmdNew(ctx, []string{
		"Pulse",
		"--goal", "A command-line reader that turns a feed subscription list into a single ranked digest.",
		"--stack", "rust",
		"--template", "product",
	}); err != nil {
		t.Fatalf("dacli new: %v", err)
	}

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := gates.Status(w, "pulse")
	if err != nil {
		t.Fatal(err)
	}
	if st.Template != "product" {
		t.Fatalf("template = %q, want product", st.Template)
	}
	found := false
	for _, c := range st.Checks {
		if !strings.Contains(c.Desc, "project sections") {
			continue
		}
		found = true
		if !c.OK {
			t.Errorf("seeded sections fail the discovery gate: %s", c.Why)
		}
	}
	if !found {
		t.Fatalf("discovery stage has no project_sections check: %+v", st.Checks)
	}
}

// A --slug is written as a path segment under projects/, so a traversing one
// would place a project file outside the workspace — and `project rm` would
// later RemoveAll it there. It must be refused as a usage error, and refused
// EARLY: nothing, not even the workspace, may exist afterwards.
func TestNewRejectsTraversingSlug(t *testing.T) {
	dir, ctx, _ := newEnv(t)

	err := cmdNew(ctx, []string{
		"Escape",
		"--goal", "A project whose slug tries to climb out of the workspace directory.",
		"--stack", "go",
		"--slug", "../escape",
	})
	if err == nil {
		t.Fatal("traversing --slug was accepted, want a usage error")
	}
	if code := clikit.ExitCode(err); code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if _, ferr := workspace.Find(dir); ferr == nil {
		t.Error("refused new still initialized a workspace — the slug must be checked first")
	}
	if _, serr := os.Stat(filepath.Join(dir, "..", "escape")); serr == nil {
		t.Error("a project escaped the workspace directory")
	}
}

// On an empty directory there is no manifest to detect, and guessing a stack
// would bake the wrong build and test commands into five acceptance criteria.
// `auto` must refuse and name the real choices instead.
func TestNewRefusesUndetectableStack(t *testing.T) {
	_, ctx, _ := newEnv(t)

	err := cmdNew(ctx, []string{
		"Blank",
		"--goal", "A product whose directory carries no manifest to detect a stack from.",
	})
	if err == nil {
		t.Fatal("undetectable stack was accepted, want a usage error")
	}
	if code := clikit.ExitCode(err); code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(err.Error(), "--stack") {
		t.Errorf("error should name --stack, got: %v", err)
	}
}

// --stack auto in a directory that already carries a manifest picks that stack
// up, so `dacli new` inside a checked-out but unstarted repo does not need the
// operator to restate what the repo already says.
func TestNewAutoDetectsStackFromManifest(t *testing.T) {
	dir, ctx, _ := newEnv(t)
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cmdNew(ctx, []string{
		"Crabwise",
		"--goal", "A crate that converts a stream of measurements into a rolling summary.",
		"--stack", "auto",
	}); err != nil {
		t.Fatalf("dacli new: %v", err)
	}

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.LoadProject(w, "crabwise")
	if err != nil {
		t.Fatal(err)
	}
	if got := section(t, p, "Constraints"); !strings.Contains(got, "cargo test") {
		t.Errorf("auto did not detect Rust from Cargo.toml: %q", got)
	}
}

// An unknown flag must be a usage error naming the offender, not silently
// dropped intent (dacli 143/175).
func TestNewRejectsUnknownFlags(t *testing.T) {
	_, ctx, _ := newEnv(t)

	err := cmdNew(ctx, []string{"X", "--goal", "A goal long enough to clear the gate.", "--stack", "go", "--sucess", "typo"})
	if err == nil || clikit.ExitCode(err) != 2 {
		t.Fatalf("unknown flag: err = %v (exit %d), want a usage error", err, clikit.ExitCode(err))
	}
	if !strings.Contains(err.Error(), "sucess") {
		t.Errorf("usage error should name --sucess, got: %v", err)
	}
}
