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

// --max-tokens is a launch contract, not merely a calibrated estimate. These
// public-command cases cross the dispatcher, adapter persistence, launch gate,
// and child argv boundary so no layer can silently downgrade a requested cap.
func TestMaxTokensRuntimeCeilingContract(t *testing.T) {
	setup := func(t *testing.T, tokenFlag string) (string, string) {
		t.Helper()
		dir := t.TempDir()
		run(t, dir, 0, "init", "--name", "x")
		run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
		run(t, dir, 0, "task", "add", "Do the thing", "--project", "p", "--estimate", "1,2,3", "--accept", "a")
		script := filepath.Join(dir, "capture.sh")
		if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > token-argv.txt\ncat >/dev/null\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		args := []string{"runtime", "add", "mock", "--binary", "sh", "--mode", "stdin", "--arg", script, "--env", "PATH"}
		if tokenFlag != "" {
			args = append(args, "--token-limit-flag", tokenFlag)
		}
		run(t, dir, 0, args...)
		return dir, script
	}
	invocation := func(t *testing.T, dir string) string {
		t.Helper()
		entries, err := os.ReadDir(filepath.Join(dir, ".dacli", "runs"))
		if err != nil || len(entries) != 1 {
			t.Fatalf("read single run record: entries=%d err=%v", len(entries), err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, ".dacli", "runs", entries[0].Name(), "invocation.txt"))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}

	t.Run("unsupported accounting refuses", func(t *testing.T) {
		dir, _ := setup(t, "")
		preview := run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "rw", "--max-tokens", "100", "--advise")
		if !strings.Contains(preview, "UNSUPPORTED") || !strings.Contains(preview, "no agent spawned") {
			t.Fatalf("preview did not distinguish unsupported enforcement without launching:\n%s", preview)
		}
		out := run(t, dir, 3, "spawn", "--task", "001", "--runtime", "mock", "--grant", "rw", "--max-tokens", "100")
		if !strings.Contains(out, "cannot enforce --max-tokens") || !strings.Contains(out, "--allow-advisory-tokens") {
			t.Fatalf("unsupported runtime refusal is not actionable:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(dir, "token-argv.txt")); !os.IsNotExist(err) {
			t.Fatalf("unsupported runtime launched despite refusal: %v", err)
		}
	})

	t.Run("explicit advisory override is loud", func(t *testing.T) {
		dir, _ := setup(t, "")
		out := run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "rw", "--max-tokens", "100", "--allow-advisory-tokens")
		if !strings.Contains(out, "ADVISORY ONLY") {
			t.Fatalf("override did not disclose the unenforced ceiling:\n%s", out)
		}
		if got := invocation(t, dir); !strings.Contains(got, "max_tokens_mode: advisory-only") {
			t.Fatalf("run record lost advisory downgrade:\n%s", got)
		}
	})

	t.Run("declared ceiling reaches child argv", func(t *testing.T) {
		dir, _ := setup(t, "--token-ceiling")
		out := run(t, dir, 0, "spawn", "--task", "001", "--runtime", "mock", "--grant", "rw", "--max-tokens", "100")
		if !strings.Contains(out, "ENFORCED") {
			t.Fatalf("launch did not disclose runtime enforcement:\n%s", out)
		}
		got, err := os.ReadFile(filepath.Join(dir, "token-argv.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "--token-ceiling\n100\n" {
			t.Fatalf("child argv = %q, want exact ceiling flag/value", got)
		}
		if got := invocation(t, dir); !strings.Contains(got, "max_tokens_mode: runtime-enforced") {
			t.Fatalf("run record lost enforcement mode:\n%s", got)
		}
	})
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

func TestRuntimeAddCodexPresets(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	for _, preset := range []string{"codex", "codex-rw"} {
		run(t, dir, 0, "runtime", "add", preset, "--preset", preset)
		raw, err := os.ReadFile(filepath.Join(dir, ".dacli", "runtimes", preset+".md"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, want := range []string{"global_args: \"[--ask-for-approval, never]\"", "invoke_args: \"[exec, --json, --ephemeral", "model_flag: --model", "usage_format: codex-jsonl"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s adapter missing %q:\n%s", preset, want, text)
			}
		}
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
