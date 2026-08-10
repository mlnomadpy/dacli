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
	if v, ok := os.LookupEnv("DACLI_AGENT"); ok {
		t.Setenv("DACLI_AGENT", v)
		_ = os.Unsetenv("DACLI_AGENT")
	}
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

// --- CI (dacli 195).
//
// The generated workflow is the repository's ONLY source of check history, so
// these tests hold two lines at once: the YAML has to be well-formed (a file
// GitHub refuses to parse reports no checks, which is the exact hole this work
// closes), and it has to run the stack's real build and test commands (a
// workflow that runs something else is a green check that means nothing).

// lintYAML parses the emitted workflow structurally without a YAML library —
// go.mod has zero requires and stays that way. It is not a general parser: it
// is the subset a workflow file uses (block mappings, block sequences, flow
// sequences as scalars) with every rule that would make GitHub reject the file.
// It returns the set of "indent+key" lines so a caller can assert nesting.
func lintYAML(t *testing.T, src string) {
	t.Helper()
	levels := []int{0}
	seenTop := map[string]bool{}
	for n, line := range strings.Split(src, "\n") {
		ln := n + 1
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.ContainsRune(line, '\t') {
			t.Errorf("line %d has a tab — YAML forbids tabs for indentation: %q", ln, line)
			continue
		}
		if strings.TrimRight(line, " ") != line {
			t.Errorf("line %d has trailing whitespace: %q", ln, line)
		}
		ind := len(line) - len(strings.TrimLeft(line, " "))
		if ind%2 != 0 {
			t.Errorf("line %d indents %d spaces, not a multiple of 2: %q", ln, ind, line)
			continue
		}
		for len(levels) > 1 && ind < levels[len(levels)-1] {
			levels = levels[:len(levels)-1]
		}
		if ind != levels[len(levels)-1] {
			t.Errorf("line %d indents to column %d, which is not an open block level %v: %q", ln, ind, levels, line)
			continue
		}
		body := strings.TrimSpace(line)
		item := strings.HasPrefix(body, "- ")
		if item {
			body = strings.TrimPrefix(body, "- ")
		}
		key, val, isPair := strings.Cut(body, ":")
		switch {
		case !isPair:
			if !item {
				t.Errorf("line %d is neither a mapping key nor a sequence item: %q", ln, line)
			}
		case key == "" || strings.ContainsAny(key, " \t"):
			t.Errorf("line %d has an unusable mapping key %q: %q", ln, key, line)
		case val != "" && !strings.HasPrefix(val, " "):
			t.Errorf("line %d needs a space after the colon (%q parses as part of the key): %q", ln, val, line)
		}
		if ind == 0 && isPair {
			if seenTop[key] {
				t.Errorf("line %d repeats top-level key %q — the second one silently wins", ln, key)
			}
			seenTop[key] = true
		}
		// A key with no value opens a nested block; so does a sequence item
		// carrying a key, whose siblings sit one level in from the dash.
		if isPair && strings.TrimSpace(val) == "" {
			levels = append(levels, ind+2)
		} else if item {
			levels = append(levels, ind+2)
		}
	}
	for _, want := range []string{"name", "on", "permissions", "jobs"} {
		if !seenTop[want] {
			t.Errorf("workflow has no top-level %q key", want)
		}
	}
}

// acceptText flattens a task's acceptance checkboxes into one blob to assert
// against.
func acceptText(t *store.Task) string {
	var lines []string
	for _, c := range t.Acceptance() {
		lines = append(lines, c.Text)
	}
	return strings.Join(lines, "\n")
}

func hasLine(src, want string) bool {
	for _, line := range strings.Split(src, "\n") {
		if line == want {
			return true
		}
	}
	return false
}

// Every supported stack gets a workflow that runs THAT stack's build and test
// commands — the same two the project's Constraints section records, so a green
// check and a green local run are the same claim.
func TestNewWritesRunnableCIWorkflowPerStack(t *testing.T) {
	for _, stack := range stackNames {
		t.Run(stack, func(t *testing.T) {
			dir, ctx, out := newEnv(t)
			if err := cmdNew(ctx, []string{
				"Sample " + stack,
				"--goal", "A product used to prove the generated CI workflow matches its stack.",
				"--stack", stack,
			}); err != nil {
				t.Fatalf("dacli new: %v", err)
			}

			path := filepath.Join(dir, ".github", "workflows", "ci.yml")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("no CI workflow at .github/workflows/ci.yml: %v", err)
			}
			src := string(raw)
			lintYAML(t, src)

			prof := stackProfiles[stack]
			for _, want := range []string{
				"name: CI",
				"permissions:",
				"  contents: read",
				// The manual re-trigger fallback for a pull_request event that
				// silently didn't fire (dacli 263).
				"  workflow_dispatch:",
				"      - name: Check out the repository",
				"        uses: actions/checkout@v4",
				"        run: " + prof.build,
				"        run: " + prof.test,
			} {
				if !hasLine(src, want) {
					t.Errorf("workflow is missing the line %q:\n%s", want, src)
				}
			}
			// The toolchain setup the stack declared has to be in the file, or
			// the build command runs against whatever the runner happened to
			// ship with.
			for _, step := range prof.ci {
				if step.uses != "" {
					if !hasLine(src, "        uses: "+step.uses) {
						t.Errorf("workflow does not use %q for %s:\n%s", step.uses, stack, src)
					}
					continue
				}
				if !hasLine(src, "        run: "+step.run) {
					t.Errorf("workflow does not run the setup command %q:\n%s", step.run, src)
				}
			}
			// Every action is pinned to a major version tag: an unpinned action
			// re-resolves on each run, so a workflow can go red with no commit.
			for _, line := range strings.Split(src, "\n") {
				ref, ok := strings.CutPrefix(strings.TrimSpace(line), "uses: ")
				if !ok {
					continue
				}
				owner, ver, cut := strings.Cut(ref, "@")
				if !cut || !strings.HasPrefix(ver, "v") || !strings.Contains(owner, "/") {
					t.Errorf("action %q is not pinned to a major version tag", ref)
				}
			}
			if o := out.String(); !strings.Contains(o, ".github/workflows/ci.yml") {
				t.Errorf("new did not report the workflow it wrote:\n%s", o)
			}

			// The CI rung of the backlog now points at the file that exists
			// instead of asking an agent to invent a second pipeline.
			w, err := workspace.Find(dir)
			if err != nil {
				t.Fatal(err)
			}
			tasks, err := store.ListTasks(w, store.Slugify("Sample "+stack), "")
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, task := range tasks {
				if !strings.Contains(strings.ToLower(task.Title), "ci workflow") {
					continue
				}
				found = true
				accept := acceptText(task)
				if !strings.Contains(accept, ".github/workflows/ci.yml") {
					t.Errorf("CI task acceptance does not name the seeded workflow: %q", accept)
				}
				if !strings.Contains(accept, "green") {
					t.Errorf("CI task acceptance does not require a green run: %q", accept)
				}
			}
			if !found {
				t.Error("no seeded task refers to the CI workflow that was written")
			}
		})
	}
}

// --no-ci is for a repository whose pipeline lives elsewhere. It must write
// nothing at all — not an empty directory, not a disabled workflow — and the
// seeded rung must fall back to asking for CI rather than asserting a file that
// is not there.
func TestNewNoCIWritesNothing(t *testing.T) {
	dir, ctx, out := newEnv(t)

	if err := cmdNew(ctx, []string{
		"Externally Built",
		"--goal", "A product whose continuous integration is configured outside this repository.",
		"--stack", "go",
		"--no-ci",
	}); err != nil {
		t.Fatalf("dacli new: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github")); !os.IsNotExist(err) {
		t.Errorf("--no-ci created .github anyway (stat err = %v)", err)
	}
	if !strings.Contains(out.String(), "--no-ci") {
		t.Errorf("new did not report that CI was skipped:\n%s", out.String())
	}

	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := store.ListTasks(w, "externally-built", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		accept := acceptText(task)
		if strings.Contains(accept, ".github/workflows/ci.yml") {
			t.Errorf("task %03d claims a workflow --no-ci never wrote: %q", task.Seq, accept)
		}
	}
}

// A workflow already in the repository is CI dacli did not write. Overwriting a
// hand-tuned pipeline is far worse than declining to add one, and the extension
// checked has to cover .yaml too — GitHub reads both, so matching only our own
// ci.yml would leave a second, competing pipeline behind.
func TestNewNeverOverwritesExistingWorkflow(t *testing.T) {
	dir, ctx, out := newEnv(t)
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "name: hand written pipeline\non: push\n"
	if err := os.WriteFile(filepath.Join(wfDir, "build.yaml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cmdNew(ctx, []string{
		"Adopted",
		"--goal", "A repository that already carries a hand written continuous integration pipeline.",
		"--stack", "go",
	}); err != nil {
		t.Fatalf("dacli new: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(wfDir, "build.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("existing workflow was rewritten:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(wfDir, "ci.yml")); !os.IsNotExist(err) {
		t.Errorf("a second, competing workflow was added next to the existing one (stat err = %v)", err)
	}
	if !strings.Contains(out.String(), "build.yaml") {
		t.Errorf("new did not report which workflow it left alone:\n%s", out.String())
	}
}

// --gitignore-workspace keeps the workspace out of the generated product repo's
// trunk (dacli 222): it adds ".dacli/" to the root .gitignore, extending an
// existing file rather than clobbering it, and never touches trunk when the
// flag is absent.
func TestNewGitignoresWorkspaceWhenAsked(t *testing.T) {
	dir, ctx, out := newEnv(t)
	// A .gitignore already in the repo must be extended, not overwritten.
	gi := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gi, []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cmdNew(ctx, []string{
		"Shipped",
		"--goal", "A product whose agent workspace is kept out of the generated repository trunk.",
		"--stack", "go",
		"--gitignore-workspace",
	}); err != nil {
		t.Fatalf("dacli new: %v", err)
	}

	raw, err := os.ReadFile(gi)
	if err != nil {
		t.Fatalf("no root .gitignore after --gitignore-workspace: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "node_modules/") {
		t.Errorf("--gitignore-workspace clobbered the existing .gitignore:\n%s", body)
	}
	if !hasLine(body, ".dacli/") {
		t.Errorf(".dacli/ was not gitignored:\n%s", body)
	}
	if !strings.Contains(out.String(), "record-branch") {
		t.Errorf("new did not point the operator at the record branch that preserves the workspace:\n%s", out.String())
	}

	// A second run must not append a duplicate entry.
	w2, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := gitignoreWorkspace(ctx, w2, mustParse(t, "--gitignore-workspace")); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(mustRead(t, gi), ".dacli/\n"); n != 1 {
		t.Errorf(".dacli/ appears %d times, want exactly 1 (idempotent)", n)
	}
}

// The ignore is now the DEFAULT, and it never travels alone: a workspace that
// trunk does not track must have a record branch, or its history is gone
// rather than merely tidied. These two facts are one decision, so they are
// asserted together.
func TestNewGitignoresTheWorkspaceAndRecordsWhereItsHistoryLives(t *testing.T) {
	dir, ctx, _ := newEnv(t)

	if err := cmdNew(ctx, []string{
		"Defaulted",
		"--goal", "A product created without naming the workspace-ignore flag at all.",
		"--stack", "go",
	}); err != nil {
		t.Fatalf("dacli new: %v", err)
	}
	body := mustRead(t, filepath.Join(dir, ".gitignore"))
	if !hasLine(body, ".dacli/") {
		t.Errorf(".dacli/ must be ignored by default:\n%s", body)
	}
	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w.RecordBranch == "" {
		t.Error("the workspace is ignored but no record branch was configured — its history would be lost, which is exactly why this was opt-in before")
	}
}

// The opt-out still works, and leaves trunk exactly as it was.
func TestNewLeavesTrunkAloneWhenOptedOut(t *testing.T) {
	dir, ctx, _ := newEnv(t)

	if err := cmdNew(ctx, []string{
		"Untouched",
		"--goal", "A product created with the workspace ignore explicitly declined.",
		"--stack", "go",
		"--gitignore-workspace=false",
	}); err != nil {
		t.Fatalf("dacli new: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Errorf("new wrote a root .gitignore despite --gitignore-workspace=false (stat err = %v)", err)
	}
	w, err := workspace.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w.RecordBranch != "" {
		t.Errorf("record branch = %q; a tracked workspace records on trunk as before", w.RecordBranch)
	}
}

func mustParse(t *testing.T, args ...string) *clikit.Flags {
	t.Helper()
	f, err := clikit.ParseFlags(args, newFlags...)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// An unknown flag must be a usage error naming the offender, not silently
// dropped intent (dacli 143/175).
//
// The flag below is MISSPELLED ON PURPOSE: --success is a real flag of `new`, so a
// correctly-spelled one would be accepted and this test would prove nothing.
// The misspelling is the fixture. nolint:misspell keeps an auto-fixer from
// "correcting" it — which is exactly what happened when the linter was first
// introduced, silently turning the unknown flag into a known one and leaving a
// test that passed while measuring nothing.
func TestNewRejectsUnknownFlags(t *testing.T) {
	_, ctx, _ := newEnv(t)

	//nolint:misspell // the typo IS the test fixture; see the comment above
	const typoFlag = "--sucess"

	err := cmdNew(ctx, []string{"X", "--goal", "A goal long enough to clear the gate.", "--stack", "go", typoFlag, "typo"})
	if err == nil || clikit.ExitCode(err) != 2 {
		t.Fatalf("unknown flag: err = %v (exit %d), want a usage error", err, clikit.ExitCode(err))
	}
	if !strings.Contains(err.Error(), strings.TrimPrefix(typoFlag, "--")) {
		t.Errorf("usage error should name %s, got: %v", typoFlag, err)
	}
}
