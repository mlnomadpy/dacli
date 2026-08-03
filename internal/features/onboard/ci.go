package onboard

// Continuous integration is authored at project birth (dacli 195). Nothing in
// dacli used to write a workflow, so a generated repository had no
// `.github/workflows/` at all — and a repository with no workflow reports no
// checks, which means every pull request it ever opens carries an EMPTY check
// list. That is indistinguishable, to anything reading the PR, from a project
// whose tests were never written: the merge is green because nothing ran. The
// buyer of these repositories asks for pull requests and test coverage, and an
// empty check list is neither.
//
// So `dacli new` writes the workflow itself instead of asking the seeded
// "set up continuous integration" task to invent one. Two reasons the ordering
// matters. First, the workflow that lands is the one that runs the stack's real
// build and test commands — the same two commands recorded in the project's
// Constraints section — rather than whatever an agent improvised from a blank
// page. Second, it exists before the first pull request does, so the very first
// PR of the repository has a check attached to it.
//
// Two refusals are deliberate. `--no-ci` opts out entirely, for someone
// adopting dacli into a pipeline that already exists elsewhere. And a
// `.github/workflows/` that already holds a workflow is never touched: dacli
// does not own CI it did not write, and silently overwriting a hand-tuned
// pipeline is a far worse failure than declining to add one.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ciRelPath is the workflow's path from the repository root, in the slash form
// that goes into task prose and stdout on every platform.
const ciRelPath = ".github/workflows/ci.yml"

// ciStep is one step of the generated job. A step is either an action
// (uses + optional with) or a shell command (run) — never both, which is the
// one shape GitHub Actions rejects outright.
type ciStep struct {
	// name is the step label shown in the checks UI.
	name string
	// uses is an action reference, ALWAYS pinned to a major version tag: an
	// unpinned action re-resolves on every run, so a workflow that passed on
	// Monday can fail on Tuesday without a commit.
	uses string
	// with are the action's inputs, as an ordered slice rather than a map so
	// the emitted YAML is byte-stable across runs and diffs cleanly.
	with [][2]string
	// run is the shell command, when uses is empty.
	run string
}

// ciWorkflow renders the workflow for a stack. It is emitted by hand rather
// than by a YAML marshaller because dacli carries zero third-party
// dependencies, and a workflow file is a fixed shape: three top-level keys and
// one job. The emitter therefore has exactly one job of its own — never write a
// value that needs quoting it is not given. Every `with` value is single-quoted
// for that reason ('3.12' and '20' are otherwise a float and an int), and step
// names and run commands are free of the ": " and leading-indicator characters
// that would make a plain scalar ambiguous.
func ciWorkflow(prof stackProfile) string {
	var b strings.Builder
	b.WriteString("# Continuous integration for this repository, written by `dacli new`.\n")
	fmt.Fprintf(&b, "# It runs the two commands recorded in the project's Constraints section —\n# `%s` and `%s` — so a red check here means what a red local run means.\n", prof.build, prof.test)
	b.WriteString("name: CI\n\n")

	// Pull requests are the surface the check list attaches to, and pushes to
	// the default branch are what catch a merge that broke main. Both spellings
	// of the default branch are listed because a repository created by hand and
	// one created by `gh repo create` do not agree on it.
	b.WriteString("on:\n")
	b.WriteString("  push:\n")
	b.WriteString("    branches: [main, master]\n")
	b.WriteString("  pull_request:\n\n")

	// Least privilege. The default token grants write access to the whole
	// repository; this job reads a checkout and reports a status, so anything
	// beyond contents:read is standing authority nobody asked for.
	b.WriteString("permissions:\n")
	b.WriteString("  contents: read\n\n")

	fmt.Fprintf(&b, "jobs:\n  build:\n    name: %s build and test\n    runs-on: ubuntu-latest\n    steps:\n", prof.label)
	for _, s := range ciJobSteps(prof) {
		fmt.Fprintf(&b, "      - name: %s\n", s.name)
		if s.uses != "" {
			fmt.Fprintf(&b, "        uses: %s\n", s.uses)
			if len(s.with) > 0 {
				b.WriteString("        with:\n")
				for _, kv := range s.with {
					fmt.Fprintf(&b, "          %s: '%s'\n", kv[0], kv[1])
				}
			}
			continue
		}
		fmt.Fprintf(&b, "        run: %s\n", s.run)
	}
	return b.String()
}

// ciJobSteps is checkout, then the stack's toolchain and dependency steps, then
// the build and test commands the stackProfiles table already records. Build
// and test are appended here rather than restated per stack so the workflow and
// the project's Constraints section can never drift apart — they read the same
// two fields.
func ciJobSteps(prof stackProfile) []ciStep {
	steps := []ciStep{{name: "Check out the repository", uses: "actions/checkout@v4"}}
	steps = append(steps, prof.ci...)
	return append(steps,
		ciStep{name: "Build", run: prof.build},
		ciStep{name: "Test", run: prof.test},
	)
}

// writeCIWorkflow writes the workflow into the new project's directory. It
// returns the absolute path written, or an empty path plus the name of the
// workflow that already claimed the directory — the never-clobber rule. A
// caller that gets an empty path with an empty found value did not ask for CI.
func writeCIWorkflow(dir string, prof stackProfile) (written, found string, err error) {
	wfDir := filepath.Join(dir, ".github", "workflows")
	if found, err = existingWorkflow(wfDir); err != nil || found != "" {
		return "", found, err
	}
	if err = os.MkdirAll(wfDir, 0o755); err != nil {
		return "", "", err
	}
	path := filepath.Join(wfDir, "ci.yml")
	if err = os.WriteFile(path, []byte(ciWorkflow(prof)), 0o644); err != nil {
		return "", "", err
	}
	return path, "", nil
}

// existingWorkflow names the first workflow file already in the directory, if
// any. Both extensions count: GitHub reads .yml and .yaml alike, so checking
// only for our own ci.yml would let `dacli new` add a second, competing
// pipeline next to a hand-written ci.yaml. A missing directory is not an error
// — it is the ordinary greenfield case.
func existingWorkflow(wfDir string) (string, error) {
	entries, err := os.ReadDir(wfDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".yml", ".yaml":
			return e.Name(), nil
		}
	}
	return "", nil
}
