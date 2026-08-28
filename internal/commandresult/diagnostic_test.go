package commandresult

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestRunRetainsTypedFailureInsteadOfBareExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	root := t.TempDir()
	cmd := exec.Command("/bin/sh", "-c", "printf 'partial result\\n'; printf 'fatal: index.lock is held\\n' >&2; exit 23")
	cmd.Dir = root
	out, err := Run(cmd, RunOptions{Operation: "git add", WorkspaceRoot: root})
	if err == nil {
		t.Fatal("expected command failure")
	}
	if !strings.Contains(string(out), "partial result") || !strings.Contains(string(out), "index.lock") {
		t.Fatalf("compatibility output lost a stream: %q", out)
	}
	diagnostic, ok := AsDiagnostic(fmt.Errorf("feature failed: %w", err))
	if !ok {
		t.Fatalf("wrapped error lost typed diagnostic: %T: %v", err, err)
	}
	if diagnostic.ExitCode == nil || *diagnostic.ExitCode != 23 {
		t.Fatalf("exit code = %v, want 23", diagnostic.ExitCode)
	}
	if diagnostic.StdoutTail != "partial result" || diagnostic.StderrTail != "fatal: index.lock is held" {
		t.Fatalf("stream tails collapsed: %#v", diagnostic)
	}
	if diagnostic.Kind != "contention" || !diagnostic.Retryable || !strings.Contains(diagnostic.NextAction, "stale lock") {
		t.Fatalf("contention classification = %#v", diagnostic)
	}
	if got := err.Error(); !strings.Contains(got, "fatal: index.lock is held") || strings.Contains(got, "exit status 23") {
		t.Fatalf("human error must lead with the actionable cause, not bare status: %q", got)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("typed diagnostic did not retain the original process cause")
	}
}

func TestRunRedactsSecretsAndOutsideWorkspacePathsFromEverySurface(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	const secret = "super-secret-token-value"
	t.Setenv("DACLI_TEST_API_TOKEN", secret)
	root := t.TempDir()
	inside := root + "/safe/file.txt"
	outside := "/private/operator/credentials.txt"
	cmd := exec.Command("/bin/sh", "-c", "printf '%s\\n' \"$DACLI_TEST_API_TOKEN\"; printf '%s %s\\n' \"$1\" \"$2\" >&2; exit 9", "fixture", inside, outside)
	cmd.Dir = root
	out, err := Run(cmd, RunOptions{
		Operation:     "provider authenticate " + secret,
		WorkspaceRoot: root,
	})
	if err == nil {
		t.Fatal("expected command failure")
	}
	diagnostic, ok := AsDiagnostic(err)
	if !ok {
		t.Fatalf("missing diagnostic: %v", err)
	}
	encoded, marshalErr := json.Marshal(diagnostic)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for surface, value := range map[string]string{
		"human":  err.Error(),
		"json":   strings.ReplaceAll(strings.ReplaceAll(string(encoded), `\u003c`, "<"), `\u003e`, ">"),
		"output": string(out),
	} {
		if strings.Contains(value, secret) || strings.Contains(value, outside) {
			t.Fatalf("%s surface leaked protected material: %s", surface, value)
		}
		if !strings.Contains(value, "<redacted>") || !strings.Contains(value, "<outside-workspace>") {
			t.Fatalf("%s surface omitted redaction markers: %s", surface, value)
		}
	}
	if !strings.Contains(diagnostic.StderrTail, "<workspace>/safe/file.txt") {
		t.Fatalf("workspace path was not safely scoped: %q", diagnostic.StderrTail)
	}
	if diagnostic.CwdScope != "." {
		t.Fatalf("cwd scope = %q, want .", diagnostic.CwdScope)
	}
}

func TestRunBoundsDiagnosticTails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	cmd := exec.Command("/bin/sh", "-c", "i=0; while [ $i -lt 6000 ]; do printf x >&2; i=$((i+1)); done; printf ' decisive-tail' >&2; exit 1")
	cmd.Dir = t.TempDir()
	_, err := Run(cmd, RunOptions{Operation: "fixture", WorkspaceRoot: cmd.Dir})
	diagnostic, ok := AsDiagnostic(err)
	if !ok {
		t.Fatalf("missing diagnostic: %v", err)
	}
	if len(diagnostic.StderrTail) > diagnosticTailBytes+len("[truncated] ") || !strings.HasSuffix(diagnostic.StderrTail, "decisive-tail") {
		t.Fatalf("stderr tail is not bounded from the actionable end: len=%d suffix=%q", len(diagnostic.StderrTail), actionableLine(diagnostic.StderrTail))
	}
}

func TestRunClassifiesSignalTermination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX signal fixture")
	}
	cmd := exec.Command("/bin/sh", "-c", "kill -TERM $$")
	cmd.Dir = t.TempDir()
	_, err := Run(cmd, RunOptions{Operation: "worker", WorkspaceRoot: cmd.Dir})
	diagnostic, ok := AsDiagnostic(err)
	if !ok || diagnostic.Kind != "signal" || diagnostic.Signal == "" {
		t.Fatalf("signal diagnostic = %#v, present=%v", diagnostic, ok)
	}
}

func TestCaptureReturnsQuietMutationIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	cmd := exec.Command("/bin/sh", "-c", `printf '{"merged":1,"open":0}' > "$DACLI_COMMAND_RESULT"`)
	cmd.Dir = t.TempDir()
	var result Integration
	out, err := Capture(cmd, &result)
	if err != nil {
		t.Fatalf("quiet mutation capture: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("fixture must be quiet; got presentation output %q", out)
	}
	if result.Merged != 1 || result.Open != 0 {
		t.Fatalf("quiet mutation lost its typed identity: %#v", result)
	}
	if identity, ok := IdentityOf(&result); !ok || identity != "integration merged=1 open=0" {
		t.Fatalf("quiet mutation identity = %q, %v", identity, ok)
	}
}

func TestCaptureRejectsQuietSuccessWithoutInventoriedIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	type untrackedMutation struct {
		Object string `json:"object"`
	}
	cmd := exec.Command("/bin/sh", "-c", `printf '  \n'; printf '{"object":"changed"}' > "$DACLI_COMMAND_RESULT"`)
	cmd.Dir = t.TempDir()
	var result untrackedMutation
	out, err := Capture(cmd, &result)
	if err == nil || !strings.Contains(err.Error(), "no stable result identity") {
		t.Fatalf("uninventoried quiet mutation = output %q, result %#v, error %v", out, result, err)
	}
}

func TestStructuredMutationIdentityInventory(t *testing.T) {
	for name, result := range map[string]any{
		"spawn":       &Spawn{RunID: "01RUN"},
		"integration": &Integration{Merged: 2, Open: 1},
		"wait":        &Wait{Runs: []WaitRun{{RunID: "01RUN"}}},
	} {
		if identity, ok := IdentityOf(result); !ok || strings.TrimSpace(identity) == "" {
			t.Errorf("%s result has no stable identity: %q, %v", name, identity, ok)
		}
	}
}

func TestCapturePreservesTypedCommandFailureWhenResultIsMalformed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	cmd := exec.Command("/bin/sh", "-c", `printf '{malformed' > "$DACLI_COMMAND_RESULT"; printf 'authentication failed decisively\n' >&2; exit 17`)
	cmd.Dir = t.TempDir()
	var result Integration
	_, err := Capture(cmd, &result)
	if err == nil || !strings.Contains(err.Error(), "decode command result") {
		t.Fatalf("Capture error = %v, want malformed-result context", err)
	}
	diagnostic, ok := AsDiagnostic(err)
	if !ok {
		t.Fatalf("malformed result collapsed the governed process failure: %T: %v", err, err)
	}
	if diagnostic.ExitCode == nil || *diagnostic.ExitCode != 17 || diagnostic.Kind != "authentication" {
		t.Fatalf("typed command facts = %#v, want authentication exit 17", diagnostic)
	}
	if !strings.Contains(diagnostic.StderrTail, "authentication failed decisively") {
		t.Fatalf("actionable stderr was lost: %#v", diagnostic)
	}
}
