package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
)

func TestEmitErrorJSONPreservesTypedExternalDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	root := t.TempDir()
	cmd := exec.Command("/bin/sh", "-c", "printf 'authentication failed: login required\\n' >&2; exit 4")
	cmd.Dir = root
	_, processErr := commandresult.Run(cmd, commandresult.RunOptions{Operation: "gh auth", WorkspaceRoot: root})
	err := fmt.Errorf("github check failed: %w", processErr)

	var stderr bytes.Buffer
	emitError(&Ctx{Stderr: &stderr, JSON: true}, err)
	var got clikit.ErrorDetails
	if decodeErr := json.Unmarshal(stderr.Bytes(), &got); decodeErr != nil {
		t.Fatalf("CLI JSON error is not a document: %v\n%s", decodeErr, stderr.String())
	}
	if got.ExitCode != 1 || got.Diagnostic == nil {
		t.Fatalf("CLI error details = %#v", got)
	}
	if got.Diagnostic.Kind != "authentication" || got.Diagnostic.ExitCode == nil || *got.Diagnostic.ExitCode != 4 {
		t.Fatalf("typed root cause was collapsed: %#v", got.Diagnostic)
	}
}

func TestMainEmitsJSONErrorDocument(t *testing.T) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stderr
	os.Stderr = writeEnd
	t.Cleanup(func() { os.Stderr = original })
	code := Main([]string{"--json", "frobnicate"})
	if err := writeEnd.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(readEnd)
	if err != nil {
		t.Fatal(err)
	}
	if code != 2 {
		t.Fatalf("unknown command exit = %d, want usage 2", code)
	}
	var details clikit.ErrorDetails
	if err := json.Unmarshal(raw, &details); err != nil {
		t.Fatalf("Main stderr is not one JSON document: %v\n%s", err, raw)
	}
	if details.ExitCode != 2 || details.Message != `unknown command "frobnicate"` {
		t.Fatalf("Main JSON error = %#v", details)
	}
}

func TestMCPExecutorReturnsTypedErrorDetails(t *testing.T) {
	_, msg, code := executor(t.TempDir())([]string{"frobnicate"}, false)
	if code != 2 {
		t.Fatalf("executor exit = %d, want 2", code)
	}
	var details clikit.ErrorDetails
	if err := json.Unmarshal([]byte(msg), &details); err != nil {
		t.Fatalf("MCP executor collapsed error to prose: %v\n%s", err, msg)
	}
	if details.ExitCode != 2 || details.Message != `unknown command "frobnicate"` {
		t.Fatalf("MCP executor details = %#v", details)
	}
}

func TestEmitErrorHumanRemainsActionableProse(t *testing.T) {
	var stderr bytes.Buffer
	emitError(&Ctx{Stderr: &stderr}, clikit.Refusedf("operator decision required"))
	if got := stderr.String(); got != "dacli: operator decision required\n" {
		t.Fatalf("human error changed unexpectedly: %q", got)
	}
}
