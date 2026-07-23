package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression for run 01KY2K8N4C: `runtime add ... --sandbox-ro-arg
// --allowedTools ...` must store the literal value, not silently drop it as
// a bare boolean — and that value must reach the spawned child's argv
// unmangled, since that's where the corruption actually bit.
func TestRuntimeAddDashLeadingValueReachesChildArgv(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Do the thing", "--project", "p", "--accept", "a")

	script := filepath.Join(dir, "echoargs.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat > /dev/null\necho ARGS: \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// The exact shape that corrupted run 01KY2K8N4C: a value-flag's value
	// starts with "--" and is passed via the space form.
	run(t, dir, 0, "runtime", "add", "mock", "--binary", "sh", "--mode", "stdin",
		"--arg", script, "--env", "PATH", "--sandbox-ro-arg", "--allowedTools", "--sandbox-ro-arg", "Bash")

	list := run(t, dir, 0, "runtime", "list")
	if !strings.Contains(list, "ro: --allowedTools Bash") {
		t.Fatalf("sandbox-ro-arg not captured verbatim:\n%s", list)
	}

	run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "ro", "--cooperative")

	runsList := run(t, dir, 0, "runs", "list")
	runID := strings.Fields(runsList)[0]
	detail := run(t, dir, 0, "runs", "show", runID)
	if !strings.Contains(detail, "ARGS: --allowedTools Bash") {
		t.Errorf("sandbox-ro-arg value corrupted in child argv:\n%s", detail)
	}
}

// A value-flag left with no value (--sandbox-ro-arg as the last token) is a
// usage mistake, not a silent "true".
func TestRuntimeAddValueFlagMissingValueFails(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	out := run(t, dir, 2, "runtime", "add", "mock", "--binary", "sh", "--sandbox-ro-arg")
	if !strings.Contains(out, "sandbox-ro-arg") || !strings.Contains(out, "requires a value") {
		t.Errorf("missing-value error unclear:\n%s", out)
	}
}
