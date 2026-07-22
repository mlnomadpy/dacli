package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// buildDacli compiles the real binary once per test run, so fixture children
// can call dacli exactly the way real children do — through the same entry
// path as users. The about-filter bug survived unit tests precisely because
// they bypassed that path.
var dacliBin struct {
	once sync.Once
	path string
	err  error
}

func buildDacli(t *testing.T) string {
	t.Helper()
	dacliBin.once.Do(func() {
		dir, err := os.MkdirTemp("", "dacli-bin-*")
		if err != nil {
			dacliBin.err = err
			return
		}
		dacliBin.path = filepath.Join(dir, "dacli")
		out, err := exec.Command("go", "build", "-o", dacliBin.path, "../../cmd/dacli").CombinedOutput()
		if err != nil {
			dacliBin.err = err
			dacliBin.path = string(out)
		}
	})
	if dacliBin.err != nil {
		t.Fatalf("building dacli: %v\n%s", dacliBin.err, dacliBin.path)
	}
	return dacliBin.path
}

// The § 7 loop end to end with a real child process that behaves like a real
// (cooperative) agent: turn 1 it claims and reports a finding; the
// supervisor syncs, making it the owner; turn 2 it checks its boxes and
// finishes. Accepted after 2 turns — the loop terminated on the external
// criterion, not on anyone's impression of the answer.
func TestSuperviseLoopConverges(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "One write path.")
	run(t, dir, 0, "task", "add", "Audit the batch job", "--project", "p",
		"--priority", "must", "--accept", "writers listed")

	counter := filepath.Join(dir, "turn")
	script := strings.Join([]string{
		"cat > /dev/null",              // consume the brief
		"echo t >> " + counter,         // count invocations
		"N=$(wc -l < " + counter + ")", // which turn is this
		"if [ \"$N\" -le 1 ]; then",
		"  " + bin + " task claim 001",
		"  " + bin + " note add finding \"Found the writer\" --project p --about 001 --body \"settle.go:112\"",
		"else",
		"  " + bin + " task check 001 --all",
		"  " + bin + " task done 001",
		"fi",
	}, "\n")
	mockRuntime(t, dir, "coop", script)

	out := run(t, dir, 0, "supervise", "--task", "001", "--runtime", "coop",
		"--grant", "rw", "--max-turns", "3")
	if !strings.Contains(out, "accepted after 2 turn(s)") {
		t.Fatalf("supervise did not converge on turn 2:\n%s", out)
	}
	// The correction turn told the child exactly what was unmet.
	if !strings.Contains(out, "applied") {
		t.Errorf("child events not applied between turns:\n%s", out)
	}
	// Turn 2's recorded brief carries the supervisor's correction.
	runsDir := filepath.Join(dir, ".dacli", "runs")
	entries, _ := os.ReadDir(runsDir)
	foundCorrection := false
	for _, e := range entries {
		if raw, err := os.ReadFile(filepath.Join(runsDir, e.Name(), "brief.md")); err == nil {
			if strings.Contains(string(raw), "supervisor: turn 2") && strings.Contains(string(raw), "writers listed") {
				foundCorrection = true
			}
		}
	}
	if !foundCorrection {
		t.Error("turn-2 brief missing the correction preamble naming the unmet criterion")
	}
	// And the child's finding survived into the workspace, attributed.
	st := run(t, dir, 0, "status")
	if !strings.Contains(st, "done:1") {
		t.Errorf("task not done after acceptance:\n%s", st)
	}
}

// A child that never satisfies the criteria stalls at max-turns with the
// unmet list and a do-not-just-re-run instruction — never an infinite loop.
func TestSuperviseStalls(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "never met")
	mockRuntime(t, dir, "lazy", "cat > /dev/null\necho did nothing")

	out := run(t, dir, 1, "supervise", "--task", "001", "--runtime", "lazy",
		"--grant", "rw", "--max-turns", "2")
	if !strings.Contains(out, "stalled after 2 turns") || !strings.Contains(out, "never met") {
		t.Errorf("stall verdict wrong:\n%s", out)
	}
	if !strings.Contains(out, "decompose") {
		t.Errorf("stall should point at decomposition, not retry:\n%s", out)
	}
}

func TestDisplayCommands(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g", "--stage", "elicitation")
	run(t, dir, 0, "task", "add", "Parent epic task", "--project", "p", "--priority", "must",
		"--estimate", "2,5,14", "--accept", "a")
	run(t, dir, 0, "task", "add", "Child work item", "--project", "p", "--priority", "must",
		"--estimate", "1,2,3", "--accept", "a", "--parent", "001", "--depends-on", "001")

	// estimate: PERT + the cone for the project's stage (6 at elicitation → 3–12).
	out := run(t, dir, 0, "estimate", "001")
	for _, want := range []string{"Te 6.0", "cone (elicitation): 3.0–12.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("estimate missing %q:\n%s", want, out)
		}
	}
	run(t, dir, 4, "estimate", "no-such")

	// critical-path: both on the chain, star-marked, duration = 6+2.
	out = run(t, dir, 0, "critical-path", "p")
	// Two starred rows plus the legend's star.
	if !strings.Contains(out, "project duration: 8.0") || strings.Count(out, "★") != 3 {
		t.Errorf("critical path wrong:\n%s", out)
	}

	// wbs: the child nests under the parent.
	out = run(t, dir, 0, "wbs", "p")
	if !strings.Contains(out, "\n  002-child-work-item") {
		t.Errorf("wbs tree not nested:\n%s", out)
	}

	// burndown before/after completion; velocity from the same stamps.
	out = run(t, dir, 0, "burndown")
	if !strings.Contains(out, "remaining: 8.0") {
		t.Errorf("burndown remaining wrong:\n%s", out)
	}
	run(t, dir, 0, "task", "claim", "001")
	run(t, dir, 0, "task", "check", "001", "--all")
	run(t, dir, 0, "task", "done", "001")
	out = run(t, dir, 0, "burndown")
	if !strings.Contains(out, "remaining: 2.0") || !strings.Contains(out, "done: 6.0") {
		t.Errorf("burndown after completion wrong:\n%s", out)
	}
	out = run(t, dir, 0, "velocity")
	if !strings.Contains(out, "1 task(s)") || !strings.Contains(out, "proxy") {
		t.Errorf("velocity wrong or not labeled a proxy:\n%s", out)
	}

	// threads: open, then answered.
	run(t, dir, 0, "ask", "Is the schema frozen?", "--about", "002")
	out = run(t, dir, 0, "threads")
	if !strings.Contains(out, "[OPEN]") || !strings.Contains(out, "schema frozen") {
		t.Errorf("threads open state wrong:\n%s", out)
	}
	qid := strings.Fields(out)[0]
	run(t, dir, 0, "answer", qid, "Yes, until Q3.")
	out = run(t, dir, 0, "threads")
	if !strings.Contains(out, "answered by a-root") {
		t.Errorf("threads answered state wrong:\n%s", out)
	}

	// escalate (event path only — no gh in CI): records and reports.
	out = run(t, dir, 0, "escalate", "Nobody owns the infra directory", "--about", "002")
	if !strings.Contains(out, "a human does now") {
		t.Errorf("escalate wrong:\n%s", out)
	}

	// planned stubs say what they are waiting on, not "not implemented".
	// (github sync/pull are now implemented — G4; shortcut promote is still a
	// planned stub, so it carries the honest "what I'm waiting on" message.)
	out = run(t, dir, 1, "shortcut", "promote")
	if !strings.Contains(out, "docs/SHORTCUTS.md") {
		t.Errorf("planned stub message unhelpful:\n%s", out)
	}
	// github sync is now a real command: with no project it is a usage error
	// (exit 2), not a planned-stub exit-1 message.
	out = run(t, dir, 2, "github", "sync")
	if !strings.Contains(out, "github pull <project>") {
		t.Errorf("github sync should now require a project, not stub:\n%s", out)
	}
}
