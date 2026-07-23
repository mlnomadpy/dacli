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

// Regression for issue #76: a fresh claude-code adapter must opt into
// stream-json capture by default, or `agents --tail` and calibration are
// silently blind until someone knows to pass --usage-format by hand.
func TestRuntimeAddClaudeCodePresetDefaultsUsageFormatStreamJSON(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "runtime", "add", "claude-code", "--preset", "claude-code")

	raw, err := os.ReadFile(filepath.Join(dir, ".dacli", "runtimes", "claude-code.md"))
	if err != nil {
		t.Fatalf("read adapter file: %v", err)
	}
	if !strings.Contains(string(raw), "usage_format: stream-json") {
		t.Errorf("claude-code preset did not default usage_format to stream-json:\n%s", raw)
	}
}

// generic-exec has no fixed binary, so it declares no streaming shape to
// opt into — it must stay untouched, not inherit claude-code's default.
func TestRuntimeAddGenericExecPresetLeavesUsageFormatEmpty(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "runtime", "add", "generic-exec", "--preset", "generic-exec", "--binary", "mycli")

	raw, err := os.ReadFile(filepath.Join(dir, ".dacli", "runtimes", "generic-exec.md"))
	if err != nil {
		t.Fatalf("read adapter file: %v", err)
	}
	if strings.Contains(string(raw), "usage_format") {
		t.Errorf("generic-exec preset should not declare usage_format:\n%s", raw)
	}
}

// `runtime doctor` must call out a claude-family adapter with no
// usage_format by name, since that's exactly the silent-blind-spot the
// default above is meant to close for anyone who overrides it away.
func TestRuntimeDoctorWarnsOnClaudeBinaryWithoutUsageFormat(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")

	// A fake "claude" on PATH so doctor's LookPath probe succeeds.
	binDir := t.TempDir()
	fake := filepath.Join(binDir, "claude")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho fake-claude 1.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	run(t, dir, 0, "runtime", "add", "bare-claude", "--binary", "claude", "--mode", "arg", "--flag", "-p")
	out := run(t, dir, 0, "runtime", "doctor")
	if !strings.Contains(out, "usage_format") || !strings.Contains(out, "blind") {
		t.Errorf("doctor did not warn about missing usage_format on a claude binary:\n%s", out)
	}

	run(t, dir, 0, "runtime", "add", "streaming-claude", "--preset", "claude-code")
	out = run(t, dir, 0, "runtime", "doctor")
	if strings.Count(out, "will be blind") != 1 {
		t.Errorf("doctor should warn only for the adapter missing usage_format, not the preset default:\n%s", out)
	}
}
