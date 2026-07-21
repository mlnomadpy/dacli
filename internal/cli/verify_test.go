package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The panel: one refuter per runtime, refute-framed prompt delivered,
// verdicts derived from the log, majority rule, diversity warning.
func TestVerifyPanel(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Audit the writer paths", "--project", "p", "--accept", "a")

	// Two panel runtimes with opposite convictions, plus a silent one. Each
	// script proves it received the refute framing by echoing a marker.
	verifier := func(name, verdict string) {
		var report string
		if verdict != "" {
			report = bin + ` note add finding "verdict: ` + verdict + ` — scripted" --project p --about 001 --body "fixture evidence"`
		}
		script := filepath.Join(dir, name+".sh")
		body := "#!/bin/sh\ntee " + filepath.Join(dir, name+"_brief.md") + " > /dev/null\n" + report + "\n"
		if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		run(t, dir, 0, "runtime", "add", name, "--binary", "sh", "--mode", "stdin", "--arg", script, "--env", "PATH")
	}
	verifier("confirmer", "confirmed")
	verifier("refuter", "refuted")
	verifier("silent", "")

	// 1-of-3 confirmed against a required 2: KILLED, exit 1; the silent
	// panelist counts as unconfirmed, never as agreement.
	out := run(t, dir, 1, "verify", "--task", "001", "--panel", "confirmer,refuter,silent",
		"--claim", "the batch job bypasses the service layer", "--require", "2",
		"--grant", "ro", "--cooperative")
	for _, want := range []string{"confirmed 1/3 (required 2)", "KILLED", "no-verdict", "reported nothing"} {
		if !strings.Contains(out, want) {
			t.Errorf("panel output missing %q:\n%s", want, out)
		}
	}

	// Every seat received the refute framing and the claim.
	for _, name := range []string{"confirmer", "refuter", "silent"} {
		raw, err := os.ReadFile(filepath.Join(dir, name+"_brief.md"))
		if err != nil {
			t.Fatalf("%s never received its brief: %v", name, err)
		}
		for _, want := range []string{"## Adversarial verification", "REFUTE", "bypasses the service layer", "default to REFUTED"} {
			if !strings.Contains(string(raw), want) {
				t.Errorf("%s brief missing %q", name, want)
			}
		}
	}

	// Majority default (2-of-2 needs 2): confirmed+refuted → killed; with
	// require 1 the same tally survives, exit 0.
	out = run(t, dir, 0, "verify", "--task", "001", "--panel", "confirmer,refuter",
		"--claim", "still the same claim", "--require", "1", "--grant", "ro", "--cooperative")
	if !strings.Contains(out, "SURVIVES") {
		t.Errorf("require-1 panel should survive:\n%s", out)
	}

	// Single-runtime panel draws the diversity warning.
	out = run(t, dir, 0, "verify", "--task", "001", "--panel", "confirmer",
		"--claim", "x", "--grant", "ro", "--cooperative")
	if !strings.Contains(out, "single-runtime panel") {
		t.Errorf("diversity warning missing:\n%s", out)
	}

	// With no --claim, the newest finding about the task is the claim.
	out = run(t, dir, 0, "verify", "--task", "001", "--panel", "confirmer",
		"--grant", "ro", "--cooperative")
	if !strings.Contains(out, "SURVIVES") {
		t.Errorf("default-claim panel failed:\n%s", out)
	}
	// And a task with no findings refuses at usage level.
	run(t, dir, 0, "task", "add", "Second task no findings", "--project", "p", "--accept", "a")
	run(t, dir, 2, "verify", "--task", "002", "--panel", "confirmer", "--grant", "ro", "--cooperative")
}
