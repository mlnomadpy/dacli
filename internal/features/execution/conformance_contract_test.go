//go:build !windows

package execution

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCodingCLIConformanceContract is the one executable behavioral suite for
// every first-class adapter fixture. The stand-ins make no network calls and
// need no credentials; changing any adapter's transport or lifecycle behavior
// breaks the same assertions rather than a vendor-specific imitation of them.
func TestCodingCLIConformanceContract(t *testing.T) {
	for name, base := range contractFixtureRuntimes {
		t.Run(name, func(t *testing.T) {
			t.Run("transport model terminal result usage and exit", func(t *testing.T) {
				body := `
case " $* " in *" --contract-ro "*) exit 3;; esac
case " $* " in *" --contract-rw "*) : > contract-write;; esac
if [ "` + base.UsageFormat + `" = codex-jsonl ]; then
  printf '%s\n' '{"type":"thread.started","thread_id":"fixture-session"}' '{"type":"item.completed","item":{"type":"agent_message","text":"fixture-result"}}' '{"type":"turn.completed","usage":{"input_tokens":11,"output_tokens":7}}'
elif [ "` + base.UsageFormat + `" = gemini-stream-json ]; then
  printf '%s\n' '{"type":"init","session_id":"fixture-session","model":"fixture-model"}' '{"type":"message","role":"assistant","content":"fixture-result","delta":true}' '{"type":"result","status":"success","stats":{"input_tokens":11,"output_tokens":7}}'
else
  printf '%s\n' '{"type":"assistant","message":{"content":[{"type":"text","text":"fixture-result"}]}}' '{"type":"result","session_id":"fixture-session","result":"fixture-result","usage":{"input_tokens":11,"output_tokens":7},"num_turns":1,"total_cost_usd":0}'
fi`
				bin, capture := recorderBinary(t, body)
				rt := base
				rt.Binary = bin
				runDir := t.TempDir()
				extra := []string{rt.ModelFlag, "fixture-model"}
				if _, timedOut, err := execRuntime(runDir, filepath.Join(runDir, "transcript.log"), rt, "fixture-prompt", "tok", extra, 10, false, nil); err != nil || timedOut {
					t.Fatalf("run = (timedOut %v, err %v)", timedOut, err)
				}
				argv := strings.Join(captureArgv(t, capture), " ")
				if !strings.Contains(argv, "--model fixture-model") {
					t.Errorf("model selection missing from argv: %s", argv)
				}
				if rt.Mode == "stdin" {
					if got := strings.TrimSpace(readCapture(t, capture, "stdin")); got != "fixture-prompt" {
						t.Errorf("stdin prompt = %q", got)
					}
				} else if !strings.Contains(argv, rt.Flag+" fixture-prompt") {
					t.Errorf("argument prompt transport missing: %s", argv)
				}
				for file, wants := range map[string][]string{
					"usage.txt":  {"input_tokens: 11", "output_tokens: 7"},
					"result.txt": {"session_id: fixture-session", "exit_outcome: completed", "final_message: fixture-result"},
				} {
					raw, err := os.ReadFile(filepath.Join(runDir, file))
					if err != nil {
						t.Fatal(err)
					}
					for _, want := range wants {
						if !strings.Contains(string(raw), want) {
							t.Errorf("%s missing %q:\n%s", file, want, raw)
						}
					}
				}
				if _, err := os.Stat(filepath.Join(runDir, "contract-write")); err != nil {
					t.Errorf("workspace-write mode did not write in cwd: %v", err)
				}

				rt.Args, rt.SandboxRO = nil, []string{"--contract-ro"}
				_, _, roErr := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "ro.log"), rt, "fixture-prompt", "tok", rt.SandboxRO, 10, false, nil)
				var exitErr *exec.ExitError
				if !errors.As(roErr, &exitErr) || exitErr.ExitCode() != 3 {
					t.Errorf("read-only policy exit = %v, want executable refusal exit 3", roErr)
				}
			})

			t.Run("timeout and cancellation", func(t *testing.T) {
				bin, _ := recorderBinary(t, "sleep 60")
				rt := base
				rt.Binary = bin
				rt.UsageFormat = ""
				if _, timedOut, _ := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "timeout.log"), rt, "p", "tok", nil, 1, false, nil); !timedOut {
					t.Fatal("deadline was not classified as timeout")
				}
				started := make(chan int, 1)
				done := make(chan error, 1)
				go func() {
					_, timedOut, err := execRuntime(t.TempDir(), filepath.Join(t.TempDir(), "cancel.log"), rt, "p", "tok", nil, 30, false, func(_, pgid int) { started <- pgid })
					if timedOut {
						done <- errors.New("cancellation misclassified as timeout")
						return
					}
					done <- err
				}()
				pgid := <-started
				if err := killProcessGroup(pgid); err != nil {
					t.Fatal(err)
				}
				select {
				case err := <-done:
					if err == nil {
						t.Fatal("cancelled process reported success")
					}
				case <-time.After(10 * time.Second):
					t.Fatal("cancelled process group did not terminate")
				}
			})
		})
	}
}

func TestPublishedConformanceMatrixIsGeneratedFromExecutableFixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "RUNTIMES.md"))
	if err != nil {
		t.Fatal(err)
	}
	const start, end = "<!-- BEGIN GENERATED CONFORMANCE MATRIX -->\n", "<!-- END GENERATED CONFORMANCE MATRIX -->"
	body := string(raw)
	i, j := strings.Index(body, start), strings.Index(body, end)
	if i < 0 || j < i {
		t.Fatal("generated matrix markers missing")
	}
	got := body[i+len(start) : j]
	if want := conformanceMatrixMarkdown(); got != want {
		t.Errorf("published matrix drifted from executable fixtures; regenerate it\nwant:\n%s\ngot:\n%s", want, got)
	}
}
