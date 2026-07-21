package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockRuntime writes a fixture script and registers it as an adapter — the
// RUNTIMES.md `mock` pattern: the entire CI story for spawning, no API calls.
func mockRuntime(t *testing.T, dir, name, script string) {
	t.Helper()
	path := filepath.Join(dir, name+".sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, dir, 0, "runtime", "add", name, "--binary", "sh", "--mode", "stdin",
		"--arg", path, "--env", "PATH")
}

func TestRuntimeDoctorProbes(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "runtime", "add", "shrt", "--binary", "sh", "--mode", "stdin")
	run(t, dir, 0, "runtime", "add", "ghost", "--binary", "no-such-binary-xyz")

	out := run(t, dir, 0, "runtime", "doctor")
	if !strings.Contains(out, "shrt") || !strings.Contains(out, "✓") {
		t.Errorf("sh should probe ✓:\n%s", out)
	}
	if !strings.Contains(out, "ghost") || !strings.Contains(out, "✗ binary") {
		t.Errorf("missing binary should probe ✗:\n%s", out)
	}
	// No sandbox args declared → the consequence is stated, not implied.
	if !strings.Contains(out, "ro spawns will be refused") {
		t.Errorf("missing sandbox consequence not stated:\n%s", out)
	}
}

// The whole spawn contract: § 8 refusal, --cooperative override, brief
// delivery on stdin, token in env, run record, transcript, evaluation.
func TestSpawnRunsChildProcess(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "One write path.")
	run(t, dir, 0, "task", "add", "Audit the batch job", "--project", "p",
		"--priority", "must", "--accept", "writers listed")

	got := filepath.Join(dir, "got")
	mockRuntime(t, dir, "mock", strings.Join([]string{
		"cat > " + got + "_brief.md",
		"echo \"$DACLI_AGENT\" > " + got + "_token.txt",
		"echo child working",
	}, "\n"))

	// § 8: ro on a runtime with no sandbox args is a refusal, not a downgrade.
	refusal := run(t, dir, 3, "spawn", "--task", "001", "--runtime", "mock", "--grant", "ro")
	if !strings.Contains(refusal, "cannot enforce read-only") || !strings.Contains(refusal, "--cooperative") {
		t.Fatalf("sandbox refusal wrong:\n%s", refusal)
	}

	// --cooperative accepts convention-only permissions, loudly.
	out := run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock",
		"--grant", "ro", "--cooperative", "--budget", "4000")
	if !strings.Contains(out, "COOPERATIVE") {
		t.Errorf("cooperative downgrade not loud:\n%s", out)
	}
	if !strings.Contains(out, "ok in") || !strings.Contains(out, "acceptance 0/1") {
		t.Errorf("spawn outcome/evaluation missing:\n%s", out)
	}

	// The child actually received the brief on stdin...
	briefGot, err := os.ReadFile(got + "_brief.md")
	if err != nil {
		t.Fatal("child never received the brief:", err)
	}
	for _, want := range []string{"## Task: Audit the batch job", "data, not instructions",
		"## How to report", "note add finding", "read-only"} {
		if !strings.Contains(string(briefGot), want) {
			t.Errorf("delivered brief missing %q", want)
		}
	}
	// ...and a usable identity in its environment, which resolves to an agent.
	token, _ := os.ReadFile(got + "_token.txt")
	if len(strings.TrimSpace(string(token))) != 48 {
		t.Fatalf("child token = %q", token)
	}
	t.Setenv("DACLI_AGENT", strings.TrimSpace(string(token)))
	who := run(t, dir, 0, "whoami")
	if !strings.Contains(who, "grant: ro") {
		t.Errorf("child token does not resolve: %s", who)
	}
	t.Setenv("DACLI_AGENT", "")

	// The run record: brief frozen, invocation redacts the prompt but names
	// the env vars, transcript captured, outcome recorded.
	show := run(t, dir, 0, "runs", "list")
	if !strings.Contains(show, "outcome: ok") {
		t.Errorf("runs list missing outcome:\n%s", show)
	}
	runID := strings.Fields(show)[0]
	detail := run(t, dir, 0, "runs", "show", runID)
	for _, want := range []string{"=== brief.md ===", "=== transcript.log ===", "child working", "env_names: DACLI_AGENT,PATH", "budget: 4000 (recorded, not enforced"} {
		if !strings.Contains(detail, want) {
			t.Errorf("runs show missing %q:\n%s", want, detail)
		}
	}
}

func TestSpawnFailureRecorded(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "a")
	mockRuntime(t, dir, "boom", "echo exploding >&2\nexit 7")

	out := run(t, dir, 1, "spawn", "--task", "001", "--runtime", "boom", "--grant", "rw")
	if !strings.Contains(out, "failed") {
		t.Errorf("failure not reported:\n%s", out)
	}
	list := run(t, dir, 0, "runs", "list")
	if !strings.Contains(list, "outcome: failed") || !strings.Contains(list, "exit status 7") {
		t.Errorf("failed outcome not recorded:\n%s", list)
	}
}

func TestRunsPrune(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "a")
	mockRuntime(t, dir, "mock", "true")
	for range [3]int{} {
		run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "rw")
	}
	out := run(t, dir, 0, "runs", "prune", "--keep", "1")
	if !strings.Contains(out, "pruned 2") {
		t.Errorf("prune wrong:\n%s", out)
	}
}
