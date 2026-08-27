package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLogsTailAcceptsDocumentedSeparatedValue exercises the public command
// table, rather than cmdLogs directly: its --help advertises `--tail N`, so
// that exact invocation is an executable CLI contract.
func TestLogsTailAcceptsDocumentedSeparatedValue(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "logs-tail")

	runID := "01LOGSTAILTEST00000000000000"
	path := filepath.Join(dir, ".dacli", "runs", runID, "transcript.log")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"logs", runID, "--tail", "8"},
		{"logs", runID, "--tail=8"},
	} {
		out := run(t, dir, 0, args...)
		if got, want := out, "three\nfour\nfive\nsix\nseven\neight\nnine\nten\n"; got != want {
			t.Errorf("%v output = %q, want %q", args, got, want)
		}
	}
}

func TestLogsTailRejectsInvalidValues(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "logs-tail-invalid")
	runID := "01LOGSTAILTEST00000000000000"
	path := filepath.Join(dir, ".dacli", "runs", runID, "transcript.log")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"logs", runID, "--tail"},
		{"logs", runID, "--tail", "0"},
		{"logs", runID, "--tail", "-1"},
		{"logs", runID, "--tail", "nope"},
	} {
		out := run(t, dir, 2, args...)
		if !strings.Contains(out, "--tail") {
			t.Errorf("%v usage error = %q, want it to name --tail", args, out)
		}
	}
}
