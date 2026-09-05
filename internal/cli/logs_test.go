package cli

import (
	"bytes"
	"encoding/json"
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

func TestLogsJSONSupportsIncrementalBoundedCursor(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "logs-cursor")
	runID := "01LOGSCURSORTEST000000000000"
	path := filepath.Join(dir, ".dacli", "runs", runID, "transcript.log")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonLogs := func(args ...string) string {
		t.Helper()
		var out, errOut bytes.Buffer
		ctx := &Ctx{Stdout: &out, Stderr: &errOut, Cwd: dir, JSON: true}
		cmd, rest := match(args)
		if cmd == nil {
			t.Fatalf("no command for %v", args)
		}
		if err := invoke(ctx, cmd, rest); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, errOut.String())
		}
		return out.String()
	}
	firstRaw := jsonLogs("logs", runID, "--limit", "8")
	var first struct {
		Schema     string `json:"schema"`
		Cursor     int    `json:"cursor"`
		NextCursor int    `json:"next_cursor"`
		EOF        bool   `json:"eof"`
		Output     string `json:"output"`
	}
	if err := json.Unmarshal([]byte(firstRaw), &first); err != nil {
		t.Fatalf("decode first chunk: %v\n%s", err, firstRaw)
	}
	if first.Schema != "transcript-chunk/v1" || first.Output != "one\ntwo\n" || first.NextCursor != 8 || first.EOF {
		t.Fatalf("first chunk = %+v", first)
	}
	secondRaw := jsonLogs("logs", runID, "--cursor", "8", "--limit", "8")
	var second struct {
		Cursor     int    `json:"cursor"`
		NextCursor int    `json:"next_cursor"`
		EOF        bool   `json:"eof"`
		Output     string `json:"output"`
	}
	if err := json.Unmarshal([]byte(secondRaw), &second); err != nil {
		t.Fatal(err)
	}
	if second.Cursor != first.NextCursor || second.Output != "three\n" || second.NextCursor != 14 || !second.EOF {
		t.Fatalf("second chunk = %+v", second)
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
