package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Model tiering and seniority: a junior role on a cheap model mechanically
// cannot take the hard task, and the model flag reaches the child's argv.
func TestModelRoutingAndSeniority(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "The hard migration", "--project", "p",
		"--priority", "must", "--estimate", "4,8,16", "--accept", "a")
	run(t, dir, 0, "task", "add", "Small cleanup chore", "--project", "p",
		"--priority", "could", "--estimate", "1,2,3", "--accept", "a")
	run(t, dir, 0, "task", "add", "Unestimated mystery work", "--project", "p", "--accept", "a")

	// A runtime whose script echoes its argv, with a declared model flag.
	script := filepath.Join(dir, "echoargs.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat > /dev/null\necho ARGS: \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Dash-leading values need the = form (--key=--value): the space form
	// reads the value as the next flag. Filed as a workspace finding when it
	// mangled the cc adapter; this is the documented workaround.
	run(t, dir, 0, "runtime", "add", "mock", "--binary", "sh", "--mode", "stdin",
		"--arg", script, "--env", "PATH", "--model-flag=--model")

	// junior: cheap model, capped at 3 points, routed to the mock runtime.
	run(t, dir, 0, "role", "add", "junior", "--grant", "rw",
		"--runtime", "mock", "--model", "haiku", "--max-points", "3")

	// The hard task (Te 8.7) is refused for junior — exit 3, naming the cap.
	refusal := run(t, dir, 3, "spawn", "--task", "001", "--role", "junior")
	if !strings.Contains(refusal, "above role junior's cap of 3") || !strings.Contains(refusal, "heavier role") {
		t.Fatalf("seniority refusal wrong:\n%s", refusal)
	}
	// Unestimated work is refused too: capped roles take sized work only.
	if got := run(t, dir, 3, "spawn", "--task", "003", "--role", "junior"); !strings.Contains(got, "only estimated tasks") {
		t.Errorf("unestimated gate wrong:\n%s", got)
	}

	// The small chore runs — no --runtime flag needed (role routes it), and
	// the model flag lands in the child's argv.
	out := run(t, dir, 0, "spawn", "--task", "002", "--role", "junior")
	if !strings.Contains(out, "ok in") {
		t.Fatalf("junior spawn failed:\n%s", out)
	}
	list := run(t, dir, 0, "runs", "list")
	runID := strings.Fields(list)[0]
	detail := run(t, dir, 0, "runs", "show", runID)
	if !strings.Contains(detail, "ARGS: --model haiku") {
		t.Errorf("model flag missing from child argv:\n%s", detail)
	}

	// A runtime without model_flag announces inoperative routing.
	run(t, dir, 0, "runtime", "add", "plain", "--binary", "sh", "--mode", "stdin", "--arg", script, "--env", "PATH")
	warn := run(t, dir, 0, "spawn", "--task", "002", "--runtime", "plain", "--grant", "rw", "--model", "opus")
	if !strings.Contains(warn, "no model_flag") {
		t.Errorf("inoperative model routing not announced:\n%s", warn)
	}
}

// The workflow prompts: writers get git discipline (PR block only with
// --pr), reviewers get review discipline; ro children get neither branch
// instruction nor a silent PR push.
func TestWorkflowPromptsReachChildren(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Build the widget", "--project", "p", "--accept", "a")

	got := filepath.Join(dir, "got_brief.md")
	script := filepath.Join(dir, "save.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat > "+got+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, dir, 0, "runtime", "add", "mock", "--binary", "sh", "--mode", "stdin", "--arg", script, "--env", "PATH")

	// rw without --pr: branch + commit discipline, explicitly no PR.
	run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "rw")
	raw, _ := os.ReadFile(got)
	for _, want := range []string{"## Git discipline", "git checkout -b dacli/001-build-the-widget", "Do NOT push or open a pull request"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("rw prompt missing %q", want)
		}
	}
	if strings.Contains(string(raw), "pr --task") {
		t.Error("PR instructions present without --pr")
	}

	// rw with --pr: the push-and-PR flow through dacli (which records the PR).
	run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "rw", "--pr")
	raw, _ = os.ReadFile(got)
	if !strings.Contains(string(raw), "push --task 001") || !strings.Contains(string(raw), "pr --task 001") {
		t.Errorf("--pr prompt missing the dacli push/pr flow:\n%s", raw)
	}

	// --review: judge the diff, file findings twice, approval semantics.
	run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "ro", "--cooperative", "--review")
	raw, _ = os.ReadFile(got)
	for _, want := range []string{"## Review discipline", "gh pr diff", "not against taste", "--request-changes"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("review prompt missing %q", want)
		}
	}
	if strings.Contains(string(raw), "## Git discipline") {
		t.Error("ro reviewer must not receive branch/commit instructions")
	}
}
