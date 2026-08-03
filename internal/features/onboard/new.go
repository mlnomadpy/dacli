package onboard

// `dacli new` is the greenfield half of onboarding (dacli 191). `adopt` reads
// a repo that already exists; on an empty directory it produced a codebase map
// of zero files and a placeholder goal, so the only path from "a product idea"
// to a workable backlog was a human hand-writing every task. `new` closes that:
// one command turns a product name plus a goal into an initialized workspace, a
// project whose Goal / Out of scope / Success criteria are FILLED from the
// flags, a Spec and an Architecture section a planner agent refines, a recorded
// stack (so later phases know the build and test commands), and a five-task
// starter backlog wired into a real DAG.
//
// Everything this command writes must clear gates.unfilled — the "present but
// not filled" rule that rejects empty content, the placeholder tokens, anything
// under 20 characters, and major-severity ambiguity (spm's vague, comparative,
// and completion categories). That is why the seeded prose below names concrete
// artifacts and commands instead of the "handle the requirements appropriately"
// register a template would otherwise drift into: text that trips the gate is
// worse than no text, because it blocks the stage it was meant to satisfy.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// newFlags is the exact flag set cmdNew reads. It is named once and reused for
// both ParseFlags' value-flag list and Reject's allowlist, so a flag can never
// be read but unlisted (silently rejected) or listed but unread (silently
// dropped) — the two halves of the dacli 143/175 failure.
var newFlags = []string{"name", "goal", "slug", "stack", "template", "out-of-scope", "success"}

// stackProfile is what a stack choice buys: the commands every later phase
// needs to know, and the layout the scaffold task is measured against. Recorded
// on the project so an agent briefed months later does not guess at `npm test`
// vs `pytest`.
type stackProfile struct {
	// label is the human name written into the project doc.
	label string
	// scaffold is the one command that creates the skeleton from nothing.
	scaffold string
	// build and test are the two commands the Success criteria bind to.
	build, test string
	// layout is the starting directory shape, stated concretely enough to be
	// an acceptance criterion rather than an aspiration.
	layout string
	// markers are the files whose presence identifies this stack on disk.
	markers []string
}

// stackProfiles is keyed by the value --stack accepts. `auto` is not a key —
// it is the instruction to detect one of these.
var stackProfiles = map[string]stackProfile{
	"go": {
		label:    "Go",
		scaffold: "go mod init <module-path>",
		build:    "go build ./...",
		test:     "go test ./...",
		layout:   "cmd/ for entry points, internal/ for packages the module owns, one package per bounded concern",
		markers:  []string{"go.mod"},
	},
	"python": {
		label:    "Python",
		scaffold: "python -m venv .venv && python -m pip install -e .",
		build:    "python -m build",
		test:     "pytest",
		layout:   "src/<package>/ for library code, tests/ mirroring that tree, pyproject.toml as the single build declaration",
		markers:  []string{"pyproject.toml", "setup.py", "requirements.txt", "Pipfile"},
	},
	"typescript": {
		label:    "TypeScript",
		scaffold: "npm init -y && npm install --save-dev typescript vitest",
		build:    "npm run build",
		test:     "npm test",
		layout:   "src/ for TypeScript sources, dist/ as compiled output, tsconfig.json as the single compiler declaration",
		markers:  []string{"tsconfig.json", "package.json", "deno.json"},
	},
	"rust": {
		label:    "Rust",
		scaffold: "cargo init",
		build:    "cargo build",
		test:     "cargo test",
		layout:   "src/main.rs or src/lib.rs as the crate root, tests/ for integration tests, Cargo.toml as the single manifest",
		markers:  []string{"Cargo.toml"},
	},
}

// stackNames lists the accepted --stack values in a fixed order, for error
// messages that name the real choices instead of shrugging.
var stackNames = []string{"go", "python", "typescript", "rust"}

func cmdNew(ctx *clikit.Ctx, args []string) error {
	// Every one of these flags always carries a value, so declaring them as
	// value-flags makes `--goal "..."` work without the = or -- escape, and
	// turns a value-less `--goal` into a usage error instead of the string
	// "true" silently becoming the project's goal.
	f, err := clikit.ParseFlags(args, newFlags...)
	if err != nil {
		return err
	}
	if err := f.Reject(newFlags...); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli new \"<product name>\" --goal \"<what it is>\" [--slug s] [--stack %s|auto] [--template solo|standard|product] [--out-of-scope x]... [--success \"criterion\"]...",
			strings.Join(stackNames, "|"))
	}
	name := strings.Join(f.Pos, " ")

	// A greenfield project with no goal is the exact blank the command exists
	// to prevent, and CreateProject would happily write an empty Goal section
	// that fails every template's first gate. Refuse at the door.
	goal := strings.TrimSpace(f.Get("goal"))
	if len(goal) < 20 {
		return clikit.Usagef("dacli new needs --goal with at least 20 characters describing what %s is — a shorter goal fails the first stage gate the moment a template is attached", name)
	}

	slug := f.Get("slug")
	if slug == "" {
		slug = store.Slugify(name)
	}
	// An explicit --slug becomes a path segment under projects/, so a "../.."
	// would write (and later `project rm` would delete) outside the workspace.
	// CreateProject checks this too; catching it here keeps the error a usage
	// error and, more importantly, aborts BEFORE workspace.Init has any effect.
	if !workspace.SafeSegment(slug) {
		return clikit.Usagef("invalid --slug %q: must be a single path segment without '/' or '..'", slug)
	}

	// Validate --template and --stack before anything is created — wscore's
	// lesson: a typo'd flag that is checked after init leaves a half-seeded
	// workspace behind and exits 0.
	tmpl := f.Get("template")
	if tmpl != "" {
		if _, err := gates.Get(nil, tmpl); err != nil {
			return clikit.Usagef("unknown template %q — run `dacli template list` for the available processes", tmpl)
		}
	}
	stackKey, prof, err := resolveStack(f.Get("stack"), ctx.Cwd)
	if err != nil {
		return err
	}

	// Init the workspace if this directory has none. `new` is by definition
	// first contact, so requiring a separate `dacli init` would be a step that
	// exists only to be typed — the same reasoning adopt applies.
	w, ferr := workspace.Find(ctx.Cwd)
	if ferr != nil {
		wsName := f.Get("name")
		if wsName == "" {
			wsName = name
		}
		w, err = workspace.Init(ctx.Cwd, wsName)
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stderr, "initialized workspace %q\n", w.Name)
	}
	id, err := agentid.Resolve(w)
	if err != nil {
		return err
	}
	// Creating a project, a stack record, and a backlog is a privileged write:
	// a read-only agent may report that a project should exist, never mint one.
	if err := clikit.RequireRW(id, "creating a greenfield project"); err != nil {
		return err
	}

	if _, err := store.LoadProject(w, slug); err == nil {
		return clikit.Refusedf("project %s already exists — `dacli new` is greenfield only; use `dacli adopt --project %s` to refresh an existing project, or pass a different --slug", slug, slug)
	}

	p, err := store.CreateProject(w, id.ID, name, slug, goal, "")
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "project %s created (stack: %s, stage: %s)\n", p.Slug, stackKey, p.Stage)

	// The four sections CreateProject seeds empty, filled from the flags, plus
	// the two the project object had nowhere to put. Spec and Architecture are
	// the known gap this task names: a brief could carry a goal and a codebase
	// map but there was no place for what the product IS and how it is put
	// together, so every planner started from the goal line alone.
	p.Doc.SetSection("Out of scope", outOfScopeBody(f.All("out-of-scope"), name))
	p.Doc.SetSection("Success criteria", successBody(f.All("success"), prof))
	p.Doc.SetSection("Constraints", constraintsBody(prof))
	p.Doc.SetSection("Spec", specBody(name, goal))
	p.Doc.SetSection("Architecture", architectureBody(prof))
	if err := store.SaveProject(p); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "spec + architecture sections written — a planner agent replaces the open questions with decisions\n")

	// Template attach comes after the save: gates.Attach reloads the project
	// from disk, so writing the sections first is what makes its first-stage
	// project_sections check see the filled Goal and Out of scope.
	if tmpl != "" && tmpl != "solo" {
		first, err := gates.Attach(w, p.Slug, tmpl)
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "template %s attached (stage: %s)\n", tmpl, first.Name)
	}

	seeded, err := seedBacklog(w, id.ID, p.Slug, name, prof)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "seeded %d task(s): the minimum arc from empty directory to first release\n", len(seeded))

	printNextSteps(ctx, p.Slug, seeded)
	return nil
}

// resolveStack turns the --stack flag into a profile. An empty flag means
// `auto`: an existing directory is inspected for manifest files. A genuinely
// empty directory has nothing to inspect, so rather than guessing a stack the
// operator did not choose — a guess that would be baked into five acceptance
// criteria — it refuses and names the choices.
func resolveStack(flag, cwd string) (string, stackProfile, error) {
	key := strings.ToLower(strings.TrimSpace(flag))
	if key != "" && key != "auto" {
		prof, ok := stackProfiles[key]
		if !ok {
			return "", stackProfile{}, clikit.Usagef("unknown --stack %q — one of: %s, auto", flag, strings.Join(stackNames, ", "))
		}
		return key, prof, nil
	}
	if found, prof, ok := detectStack(cwd); ok {
		return found, prof, nil
	}
	return "", stackProfile{}, clikit.Usagef("cannot detect a stack in %s (no manifest file found) — pass --stack %s", cwd, strings.Join(stackNames, "|"))
}

// detectStack reads the directory's top level for the manifest that identifies
// a stack. Top level only: a go.mod buried under a vendored example directory
// says nothing about what the operator is building here.
func detectStack(dir string) (string, stackProfile, bool) {
	for _, key := range stackNames {
		prof := stackProfiles[key]
		for _, marker := range prof.markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return key, prof, true
			}
		}
	}
	return "", stackProfile{}, false
}

// --- Section bodies.
//
// All of these are the CONTENT of a "## <name>" section, so labels inside them
// are BOLD, never ATX headings — an inner "##" would be re-parsed as a sibling
// section and leave the section it belonged to empty (the trap renderMap
// already documents, found once in a test).

// outOfScopeBody renders the --out-of-scope flags, or a real default boundary.
// The default is a stated scope line, not a placeholder: an empty Out of scope
// blocks the discovery gate of both the standard and product templates.
func outOfScopeBody(outs []string, name string) string {
	if len(outs) > 0 {
		var b strings.Builder
		for _, o := range outs {
			if o = strings.TrimSpace(o); o != "" {
				fmt.Fprintf(&b, "- %s\n", o)
			}
		}
		if b.Len() > 0 {
			return b.String()
		}
	}
	return fmt.Sprintf("Everything past the first working release of %s: additional platforms, paid tiers, and localisation stay outside this project until that release ships.\nA planner agent narrows this line to the real boundary once discovery has run.\n", name)
}

// successBody renders the --success flags, or binds success to the two commands
// the stack actually runs. Criteria a machine can check are the only ones an
// agent can be held to.
func successBody(criteria []string, prof stackProfile) string {
	var b strings.Builder
	for _, c := range criteria {
		if c = strings.TrimSpace(c); c != "" {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}
	if b.Len() > 0 {
		return b.String()
	}
	fmt.Fprintf(&b, "- `%s` and `%s` both exit 0 from a clean checkout.\n", prof.build, prof.test)
	b.WriteString("- The first feature slice runs from its user-visible entry point through to a returned result.\n")
	b.WriteString("- A reader who has never seen this repository reaches a green test run from the README alone.\n")
	return b.String()
}

// constraintsBody records the stack. It lands in Constraints specifically
// because that is one of the project sections the brief assembler already
// carries into every brief — so the build and test commands reach an agent
// without anyone editing the assembler.
func constraintsBody(prof stackProfile) string {
	return fmt.Sprintf("Stack: %s. Build with `%s`; test with `%s`. A task in this project is done only when both exit 0.\n",
		prof.label, prof.build, prof.test)
}

// specBody is the product definition the project object previously had nowhere
// to hold. It is seeded from the goal and structured as the four questions a
// planner agent must answer before implementation, each one phrased as the
// artifact that answers it — a question with a named output is work, a question
// without one is a placeholder wearing a question mark.
func specBody(name, goal string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**What %s is:** %s\n\n", name, strings.TrimSpace(goal))
	b.WriteString("**Primary user:** one sentence naming who runs it and the outcome they came for. A planner agent writes this line from discovery, before the first implementation task is claimed.\n\n")
	b.WriteString("**User-visible surface:** every entry point a person touches — commands, screens, or endpoints. Name each one and the result it returns.\n\n")
	b.WriteString("**Stated behaviour on failure and on empty input:** one written rule per case. A rule nobody wrote down arrives later as a defect report.\n\n")
	b.WriteString("**Scope boundary:** the Out of scope section of this document. Every line there is a boundary this spec inherits.\n\n")
	b.WriteString("**Open questions a planner answers before build:** data model, persistence, authentication, deployment target. Each answer belongs in `dacli note add decision` with the rejected alternative recorded alongside it.\n")
	return b.String()
}

// architectureBody records how the product is put together, seeded with the
// stack's real layout so the scaffold task has something to be measured
// against, and with the two rules that keep a young codebase from calcifying:
// no cycles between concerns, and no silent dependency.
func architectureBody(prof stackProfile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Stack:** %s — scaffold with `%s`, build with `%s`, test with `%s`.\n\n", prof.label, prof.scaffold, prof.build, prof.test)
	fmt.Fprintf(&b, "**Starting layout:** %s.\n\n", prof.layout)
	b.WriteString("**Module boundaries:** one directory per bounded concern, with no import cycles between concerns. A planner agent names the concerns from the spec's user-visible surface and records each boundary as a decision note.\n\n")
	b.WriteString("**Data flow:** the single path a request takes from the entry point to storage and back. Record it as one diagram in the README.\n\n")
	b.WriteString("**Dependency rule:** a new third-party dependency needs a decision note naming what it replaces and what was rejected.\n")
	return b.String()
}

// --- The starter backlog.

// starterTask is one rung of the minimum arc. deps indexes earlier rungs in
// this same slice, which is what makes the seeded backlog a DAG rather than a
// flat list: rung 2 fans out into CI and the first feature slice, which are
// genuinely parallel, and the README waits on the feature slice it documents.
type starterTask struct {
	title    string
	soThat   string
	priority string
	estimate string
	context  string
	accept   []string
	deps     []int
}

// starterArc is the five-task minimum for any new product. Estimates are
// present on every rung on purpose: both the standard and product templates
// gate their build stage on `tasks: all_have_estimate`, so a backlog seeded
// without them would block the first stage advance it was seeded to enable.
func starterArc(name string, prof stackProfile) []starterTask {
	return []starterTask{
		{
			title:    "Scaffold the " + prof.label + " project skeleton",
			soThat:   "every later task starts from a repository that already builds",
			priority: "must",
			estimate: "1,2,4",
			context:  fmt.Sprintf("Run `%s`. Target layout: %s.", prof.scaffold, prof.layout),
			accept: []string{
				"The starting layout exists: " + prof.layout,
				"`" + prof.build + "` exits 0 from a clean checkout",
				"A .gitignore excludes build output and local environment files",
			},
		},
		{
			title:    "Set up the test harness",
			soThat:   "a green run means something before there is code to trust",
			priority: "must",
			estimate: "1,2,4",
			context:  fmt.Sprintf("Wire `%s` so it runs from the repository root with no arguments.", prof.test),
			accept: []string{
				"`" + prof.test + "` exits 0 and runs at least one real assertion",
				"One deliberately failing test was observed to fail, then removed or fixed",
			},
			deps: []int{0},
		},
		{
			title:    "Set up continuous integration",
			soThat:   "a broken main branch is caught by the repository, not by the next agent",
			priority: "should",
			estimate: "1,2,3",
			context:  fmt.Sprintf("The workflow runs exactly the two commands recorded in this project's Constraints: `%s` and `%s`.", prof.build, prof.test),
			accept: []string{
				"A CI workflow runs `" + prof.build + "` and `" + prof.test + "` on every push",
				"The workflow has completed green once on the default branch",
			},
			deps: []int{1},
		},
		{
			title:    "Implement the first feature slice end to end",
			soThat:   "the spec is proven by running code rather than by agreement",
			priority: "must",
			estimate: "2,5,10",
			context:  fmt.Sprintf("Pick the single entry point named first in %s's Spec section and carry one request from that entry point to a returned result.", name),
			accept: []string{
				"One entry point named in the project Spec runs from input to returned result",
				"`" + prof.test + "` covers that path with at least one test that fails when the path breaks",
			},
			deps: []int{1},
		},
		{
			title:    "Write the README",
			soThat:   "the next reader starts from the repository instead of from a briefing",
			priority: "should",
			estimate: "1,2,3",
			context:  fmt.Sprintf("State what %s is in one sentence matching the project Goal, then the exact commands to build and test it.", name),
			accept: []string{
				"The README states what " + name + " is in one sentence matching the project Goal",
				"A reader reaches a green `" + prof.test + "` run from the README alone",
			},
			deps: []int{3},
		},
	}
}

// seedBacklog creates the arc in order, rewriting each rung's dependency
// indexes into the ULID of the task already created. The ULID is deliberate:
// an "001" style ref resolves across the WHOLE workspace, so in a second
// project's presence it would become ambiguous and break the DAG that this
// function exists to make real.
func seedBacklog(w *workspace.Workspace, actor, slug, name string, prof stackProfile) ([]*store.Task, error) {
	arc := starterArc(name, prof)
	created := make([]*store.Task, 0, len(arc))
	for _, st := range arc {
		var deps []string
		for _, i := range st.deps {
			if i < len(created) {
				deps = append(deps, created[i].ID)
			}
		}
		t, err := store.CreateTask(w, actor, slug, st.title, store.TaskOpts{
			Priority:  st.priority,
			Estimate:  st.estimate,
			Accept:    st.accept,
			SoThat:    st.soThat,
			Context:   st.context,
			DependsOn: deps,
		})
		if err != nil {
			return created, err
		}
		created = append(created, t)
	}
	return created, nil
}

// printNextSteps mirrors init's getting-started block: the shortest real path
// on from here, with the seeded task refs filled in so the operator can copy a
// line rather than look one up. Suppressed under --json, where a machine caller
// wants the facts above and not a reading list.
func printNextSteps(ctx *clikit.Ctx, slug string, seeded []*store.Task) {
	if ctx.JSON {
		return
	}
	first := "<task>"
	if len(seeded) > 0 {
		first = fmt.Sprintf("%03d", seeded[0].Seq)
	}
	pal := clikit.NewPalette(ctx)
	steps := [][2]string{
		{"dacli project show " + slug, "read the Spec and Architecture a planner must fill in"},
		{"dacli next", "see which rung of the arc is ready"},
		{"dacli context " + first, "brief an agent on the scaffold task"},
		{"dacli spawn --task " + first + " --runtime <runtime> --grant rw", "start building"},
	}
	width := 0
	for _, s := range steps {
		if len(s[0]) > width {
			width = len(s[0])
		}
	}
	fmt.Fprintf(ctx.Stdout, "\n%s\n", pal.Bold("Next steps"))
	for _, s := range steps {
		fmt.Fprintf(ctx.Stdout, "  %s  %s\n", pal.Cyan(fmt.Sprintf("%-*s", width, s[0])), s[1])
	}
}
