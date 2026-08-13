//go:build !windows

package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

// recorderBinary writes an executable stand-in for an agent CLI that records
// exactly what it was invoked with: argv (NUL-free, one per line), its whole
// environment, and everything it read from stdin. The capture directory is
// baked into the script text because execRuntime's env allowlist would strip
// any variable used to pass it in — which is itself the property under test.
//
// This is a stub, not an agent: it makes no network calls and spawns nothing.
func recorderBinary(t *testing.T, body string) (bin, capture string) {
	t.Helper()
	dir := t.TempDir()
	capture = filepath.Join(dir, "capture")
	if err := os.MkdirAll(capture, 0o755); err != nil {
		t.Fatal(err)
	}
	bin = filepath.Join(dir, "recorder")
	script := fmt.Sprintf("#!/bin/sh\nfor a in \"$@\"; do printf '%%s\\n' \"$a\" >> %[1]s/argv; done\nenv > %[1]s/env\ncat > %[1]s/stdin\n%s\n: > %[1]s/complete\n", capture, body)
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, capture
}

// awaitDetachedCompletion blocks until the recorder has closed its captures.
// It is the
// mandatory companion to every DETACHED execRuntime call in a test whose
// fixtures live under t.TempDir(): the child keeps writing into the recorder's
// capture dir after execRuntime has returned, and t.TempDir's RemoveAll then
// races those writes and fails the test with "directory not empty". That flake
// is load-dependent — it passes on an idle laptop and fails under CI load,
// which is the worst kind.
//
// The completion file is written after stdin has been fully captured. Process
// visibility is not a completion signal: an unreadable PID is no evidence that
// the child exited (task 384). procState is injected so the regression can
// force that denied-observation result while the recorder is still running.
func awaitDetachedCompletion(t *testing.T, capture string, pid int, procState func(int) (string, bool)) {
	t.Helper()
	if pid <= 0 {
		t.Fatalf("cannot wait on a detached child: onStart reported pid %d", pid)
	}
	const limit = 30 * time.Second
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(capture, "complete")); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	st, observable := procState(pid)
	t.Fatalf("detached recorder pid %d did not signal completion after %s (state %q, observable=%v)",
		pid, limit, st, observable)
}

// awaitGuardianExitFile waits for the detached guardian's final write, not
// merely the runtime recorder's completion marker. The recorder is the
// guardian's child: it can finish first while RunGuardian is still writing
// runtime-exit.txt beside the transcript. Returning from a test at that point
// lets t.TempDir cleanup race the guardian's last filesystem operation (issue
// #573). The exit file is a stronger and portable completion signal than PID
// visibility, which may be unavailable in a restricted sandbox.
func awaitGuardianExitFile(t *testing.T, runDir string) {
	t.Helper()
	const limit = 30 * time.Second
	path := filepath.Join(runDir, "runtime-exit.txt")
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(raw)) != "" {
			return
		} else if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("detached guardian did not persist %s after %s", path, limit)
}

func readCapture(t *testing.T, capture, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(capture, name))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	return string(raw)
}

func captureArgv(t *testing.T, capture string) []string {
	t.Helper()
	s := readCapture(t, capture, "argv")
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// Argv construction is the contract between dacli and every adapter. The order
// is load-bearing: adapter args, then the caller's extra args (sandbox flags,
// model routing), then the usage-format switches, and the prompt LAST behind
// the adapter's prompt flag — a prompt that drifted ahead of a flag would be
// consumed as that flag's value.
func TestExecRuntimeArgvConstruction(t *testing.T) {
	cases := []struct {
		name      string
		rt        store.Runtime
		extraArgs []string
		prompt    string
		wantArgv  []string
		wantStdin string
	}{
		{
			name:      "codex global flags precede exec flags and stdin prompt",
			rt:        store.Runtime{Mode: "stdin", GlobalArgs: []string{"-a", "never"}, Args: []string{"exec", "--json"}, UsageFormat: "codex-jsonl"},
			extraArgs: []string{"--model", "gpt-5"}, prompt: "brief",
			wantArgv: []string{"-a", "never", "exec", "--json", "--model", "gpt-5"}, wantStdin: "brief",
		},
		{
			name:      "arg mode: adapter args, extras, then flag+prompt",
			rt:        store.Runtime{Mode: "arg", Flag: "-p", Args: []string{"--adapter"}},
			extraArgs: []string{"--allowedTools", "Read", "--model", "opus"},
			prompt:    "do the thing",
			wantArgv:  []string{"--adapter", "--allowedTools", "Read", "--model", "opus", "-p", "do the thing"},
		},
		{
			name:     "arg mode with no flag appends the prompt directly",
			rt:       store.Runtime{Mode: "arg"},
			prompt:   "do the thing",
			wantArgv: []string{"do the thing"},
		},
		{
			name:      "stdin mode keeps the prompt OFF argv",
			rt:        store.Runtime{Mode: "stdin", Args: []string{"--adapter"}},
			extraArgs: []string{"--ro"},
			prompt:    "secret brief content",
			wantArgv:  []string{"--adapter", "--ro"},
			wantStdin: "secret brief content",
		},
		{
			name:      "stream-json adds the output-format switches before the prompt",
			rt:        store.Runtime{Mode: "arg", Flag: "-p", UsageFormat: "stream-json"},
			extraArgs: []string{"--ro"},
			prompt:    "brief",
			wantArgv:  []string{"--ro", "--output-format", "stream-json", "--verbose", "-p", "brief"},
		},
		{
			name:     "an empty usage_format leaves a text runtime's argv untouched",
			rt:       store.Runtime{Mode: "arg", Flag: "-p"},
			prompt:   "brief",
			wantArgv: []string{"-p", "brief"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin, capture := recorderBinary(t, "")
			rt := tc.rt
			rt.Binary = bin
			transcript := filepath.Join(t.TempDir(), "transcript.log")

			_, timedOut, err := execRuntime(t.TempDir(), transcript, rt, tc.prompt, "tok", tc.extraArgs, 30, false, nil)
			if err != nil || timedOut {
				t.Fatalf("execRuntime = (timedOut %v, err %v)", timedOut, err)
			}
			if got := captureArgv(t, capture); !equalStrings(got, tc.wantArgv) {
				t.Errorf("argv = %#v, want %#v", got, tc.wantArgv)
			}
			if got := strings.TrimSpace(readCapture(t, capture, "stdin")); got != tc.wantStdin {
				t.Errorf("stdin = %q, want %q", got, tc.wantStdin)
			}
			// The prompt must never appear in argv for a stdin adapter: argv is
			// visible in `ps` to every user on the box.
			if tc.rt.Mode == "stdin" {
				for _, a := range captureArgv(t, capture) {
					if strings.Contains(a, tc.prompt) {
						t.Errorf("stdin-mode prompt leaked into argv: %q", a)
					}
				}
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The child's environment is an ALLOWLIST by name: its own token, plus exactly
// what the adapter declares — never the parent's full environment. This is the
// boundary that stops an operator's credentials (API keys, cloud tokens,
// anything else in the parent's env) from reaching a spawned agent, so it is
// asserted positively AND negatively.
func TestExecRuntimeEnvIsAllowlistedByName(t *testing.T) {
	t.Setenv("DACLI_TEST_DECLARED", "declared-value")
	t.Setenv("DACLI_TEST_SECRET", "super-secret-do-not-leak")
	t.Setenv("ANTHROPIC_API_KEY", "sk-should-never-reach-a-child")

	bin, capture := recorderBinary(t, "")
	rt := store.Runtime{Binary: bin, Mode: "stdin", Env: []string{"DACLI_TEST_DECLARED", "DACLI_TEST_NOT_SET"}}

	if _, _, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"), rt, "brief", "the-token", nil, 30, false, nil); err != nil {
		t.Fatal(err)
	}
	env := readCapture(t, capture, "env")

	if !strings.Contains(env, agentid.EnvVar+"=the-token") {
		t.Errorf("the child did not receive its own token:\n%s", env)
	}
	if !strings.Contains(env, "DACLI_TEST_DECLARED=declared-value") {
		t.Errorf("a declared env name was not forwarded:\n%s", env)
	}
	for _, leaked := range []string{"DACLI_TEST_SECRET", "ANTHROPIC_API_KEY"} {
		if strings.Contains(env, leaked+"=") {
			t.Errorf("undeclared parent env %q reached the child — the allowlist leaked", leaked)
		}
	}
	// A declared-but-unset name must be omitted, not forwarded as an empty
	// string (which some CLIs read as "configured to empty").
	if strings.Contains(env, "DACLI_TEST_NOT_SET=") {
		t.Errorf("a declared-but-unset env was forwarded as empty:\n%s", env)
	}
}

// The child's stdout+stderr stream to a real file, so a detached child's output
// survives the parent's exit and `logs -f` has something to tail.
func TestExecRuntimeWritesTranscript(t *testing.T) {
	bin, _ := recorderBinary(t, "echo 'child stdout'\necho 'child stderr' >&2\n")
	transcript := filepath.Join(t.TempDir(), "transcript.log")
	rt := store.Runtime{Binary: bin, Mode: "stdin"}

	if _, _, err := execRuntime(t.TempDir(), transcript, rt, "brief", "tok", nil, 30, false, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(transcript)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"child stdout", "child stderr"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("transcript missing %q:\n%s", want, raw)
		}
	}
}

func TestFakeCodexJSONLNonzeroExitStillRecordsResult(t *testing.T) {
	body := "printf '%s\\n' '{\"type\":\"thread.started\",\"thread_id\":\"session-1\"}' '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"Could not finish.\"}}' '{\"type\":\"turn.failed\"}'\nexit 7\n"
	bin, _ := recorderBinary(t, body)
	runDir := t.TempDir()
	rt := store.Runtime{Binary: bin, Mode: "stdin", UsageFormat: "codex-jsonl"}
	_, _, err := execRuntime(t.TempDir(), filepath.Join(runDir, "transcript.log"), rt, "brief", "tok", nil, 30, false, nil)
	if err == nil {
		t.Fatal("nonzero fake Codex exit was reported as success")
	}
	raw, rerr := os.ReadFile(filepath.Join(runDir, "result.txt"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	for _, want := range []string{"session_id: session-1", "exit_outcome: failed", "final_message: Could not finish."} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("result missing %q:\n%s", want, raw)
		}
	}
}

// onStart hands the caller the live (pid, pgid) so a SEPARATE dacli invocation
// can find and reap the tree while this spawn blocks. Under Setpgid the child
// is its own group leader, so pgid == pid — the invariant `dacli kill` relies
// on when it signals the negative pgid.
func TestExecRuntimeReportsPIDAndGroupOnStart(t *testing.T) {
	bin, _ := recorderBinary(t, "")
	rt := store.Runtime{Binary: bin, Mode: "stdin"}

	var gotPID, gotPGID int
	calls := 0
	onStart := func(pid, pgid int) { gotPID, gotPGID, calls = pid, pgid, calls+1 }

	if _, _, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"), rt, "brief", "tok", nil, 30, false, onStart); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("onStart called %d time(s), want exactly 1", calls)
	}
	if gotPID <= 0 || gotPGID != gotPID {
		t.Errorf("onStart(pid=%d, pgid=%d); a Setpgid child leads its own group", gotPID, gotPGID)
	}
}

// A non-zero exit is reported, not swallowed — cmdSpawn classifies the outcome
// from it (failed vs partial).
func TestExecRuntimeSurfacesChildExitCode(t *testing.T) {
	bin, _ := recorderBinary(t, "exit 7\n")
	rt := store.Runtime{Binary: bin, Mode: "stdin"}
	_, timedOut, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"), rt, "brief", "tok", nil, 30, false, nil)
	if err == nil {
		t.Error("a child exiting 7 must surface an error")
	}
	if timedOut {
		t.Error("a clean non-zero exit is not a timeout")
	}
}

// A hung child must be reported as timedOut (cmdSpawn maps that to "stalled")
// rather than blocking forever, and the whole GROUP is killed — killing only
// the leader would orphan the subprocesses the agent forked, which is the
// runaway leak the group discipline exists to prevent.
func TestExecRuntimeTimesOutAndReapsTheGroup(t *testing.T) {
	bin, _ := recorderBinary(t, "sleep 60 &\nsleep 60\n")
	rt := store.Runtime{Binary: bin, Mode: "stdin"}

	var pgid int
	onStart := func(pid, pg int) { pgid = pg }

	start := time.Now()
	_, timedOut, _ := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"), rt, "brief", "tok", nil, 1, false, onStart)
	if !timedOut {
		t.Fatal("a child that outlives its timeout must be reported as timed out")
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("timeout took %s — WaitDelay is not bounding the hung tree", elapsed)
	}
	if pgid == 0 {
		t.Fatal("onStart never reported a pgid")
	}
	// The forked grandchild must be gone too, not just the leader.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && procmon.GroupAlive(pgid) {
		time.Sleep(100 * time.Millisecond)
	}
	if procmon.GroupAlive(pgid) {
		t.Errorf("process group %d survived the timeout — forked children were orphaned", pgid)
	}
}

// A detached spawn must hand the child the FULL prompt. Briefs routinely exceed
// the ~64KB pipe buffer, and the old strings.Reader stdin died with the parent
// mid-copy, silently truncating them. Backing stdin with a real file makes the
// fd inheritable so the child reads everything with no parent involvement.
func TestExecRuntimeDetachedDeliversAnOversizedPrompt(t *testing.T) {
	bin, capture := recorderBinary(t, "")
	rt := store.Runtime{Binary: bin, Mode: "stdin"}
	prompt := strings.Repeat("brief line that is long enough to matter\n", 4000) // ~160KB

	var pid int
	elapsed, timedOut, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"), rt, prompt, "tok", nil, 30, true, func(p, _ int) { pid = p })
	if err != nil || timedOut || elapsed != 0 {
		t.Fatalf("detached start = (%v, %v, %v); it must return immediately", elapsed, timedOut, err)
	}

	// Waiting for the recorder's completion marker (rather than polling the
	// capture file for a long-enough prefix) settles the t.TempDir cleanup race
	// and makes the assertion exact: once the writer is done, a short read is a real
	// truncation and not a read taken mid-write.
	awaitDetachedCompletion(t, capture, pid, procmon.ProcState)
	if got := readCapture(t, capture, "stdin"); got != prompt {
		t.Errorf("detached child read %d of %d prompt bytes — the oversized prompt was truncated", len(got), len(prompt))
	}
}

// A detached spawn reports its pid through onStart even though the parent never
// Waits on it — without that, `dacli agents` and `dacli kill` could not see the
// released process at all.
func TestExecRuntimeDetachedReportsPID(t *testing.T) {
	bin, capture := recorderBinary(t, "")
	rt := store.Runtime{Binary: bin, Mode: "arg"}
	runDir := t.TempDir()
	var pid int
	if _, _, err := execRuntime(t.TempDir(), filepath.Join(runDir, "t.log"), rt, "brief", "tok", nil, 30, true, func(p, _ int) { pid = p }); err != nil {
		t.Fatal(err)
	}
	if pid <= 0 {
		t.Errorf("detached spawn reported pid %d", pid)
	}
	// The child writes into the recorder's capture dir under t.TempDir() and
	// outlives this call by design; let it finish before cleanup runs.
	awaitDetachedCompletion(t, capture, pid, procmon.ProcState)
	awaitGuardianExitFile(t, runDir)
}

func TestDetachedCompletionDoesNotEquateUnobservablePIDWithExit(t *testing.T) {
	bin, capture := recorderBinary(t, "sleep 1")
	prompt := strings.Repeat("brief line that is long enough to matter\n", 4000)
	var pid int
	if _, _, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "t.log"),
		store.Runtime{Binary: bin, Mode: "stdin"}, prompt, "tok", nil, 30, true,
		func(p, _ int) { pid = p }); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	awaitDetachedCompletion(t, capture, pid, func(int) (string, bool) { return "", false })
	if elapsed := time.Since(started); elapsed < 500*time.Millisecond {
		t.Fatalf("unobservable ProcState was mistaken for exit after %s", elapsed)
	}
	if got := readCapture(t, capture, "stdin"); got != prompt {
		t.Errorf("detached child read %d of %d prompt bytes — the oversized prompt was truncated", len(got), len(prompt))
	}
}

// A runtime whose binary does not exist must return a start error rather than
// reporting a phantom successful run.
func TestExecRuntimeMissingBinaryIsAStartError(t *testing.T) {
	rt := store.Runtime{Binary: filepath.Join(t.TempDir(), "does-not-exist"), Mode: "stdin"}
	if _, _, err := execRuntime(t.TempDir(), "", rt, "brief", "tok", nil, 5, false, nil); err == nil {
		t.Error("a missing binary must surface a start error")
	}
	if _, _, err := execRuntime(t.TempDir(), "", rt, "brief", "tok", nil, 5, true, nil); err == nil {
		t.Error("a missing binary must surface a start error on the detached path too")
	}
}
