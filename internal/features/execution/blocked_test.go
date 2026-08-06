package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// writeBlocked is the child's half of the break-glass channel: a plain file
// write into the run directory, exactly what a child with no working dacli does.
func writeBlocked(t *testing.T, dir, reason string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, blockedFileName), []byte(reason), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readBlocked reports "not blocked" when the file is absent and returns the
// trimmed reason when present; firstLine reduces a multiline reason to its head.
func TestReadBlockedAndFirstLine(t *testing.T) {
	w := newExecWS(t)
	id := runID(1)
	if got := readBlocked(w, id); got != "" {
		t.Errorf("no blocked file must read as not-blocked, got %q", got)
	}
	writeBlocked(t, w.RunDir(id), "  dacli exec failed: unknown flag\nran: dacli note add ...\n\n")
	got := readBlocked(w, id)
	if got != "dacli exec failed: unknown flag\nran: dacli note add ..." {
		t.Errorf("readBlocked did not trim surrounding whitespace: %q", got)
	}
	if fl := firstLine(got); fl != "dacli exec failed: unknown flag" {
		t.Errorf("firstLine = %q, want the head line only", fl)
	}
	if fl := firstLine("single line only"); fl != "single line only" {
		t.Errorf("firstLine of a single line = %q", fl)
	}
}

// The wait half of the acceptance: a finished run that raised the channel is
// finalized as a DISTINCT blocked state, never as "done" or "no visible result".
// Before the fix, finalizeRun ignored the file and reported "no visible result".
func TestFinalizeRunReportsBlocked(t *testing.T) {
	w := newExecWS(t)
	id := runID(2)
	writeBlocked(t, w.RunDir(id), "cannot run dacli: sandbox denied the binary\nran: dacli task done 269")
	rec := procmon.Record{RunID: id, Child: "a-child-1", Task: "", Started: time.Now().Add(-time.Minute)}

	line := finalizeRun(w, rec)
	if !strings.Contains(line, "BLOCKED") {
		t.Fatalf("finalizeRun must surface BLOCKED as a distinct state, got %q", line)
	}
	if !strings.Contains(line, "sandbox denied the binary") {
		t.Errorf("finalizeRun must carry the child's reason, got %q", line)
	}
	if strings.Contains(line, "no visible result") || strings.Contains(line, "done") {
		t.Errorf("a blocked run must not read as a normal completion, got %q", line)
	}
	// The persisted outcome must match, so `runs show` reports blocked too.
	raw, err := os.ReadFile(filepath.Join(w.RunDir(id), "outcome.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "outcome: blocked") {
		t.Errorf("outcome.md must record the blocked state, got %q", raw)
	}
}

// The agents half of the acceptance: a live agent that raised the channel is
// tagged BLOCKED with its reason, and an ordinary live agent is not. Uses the
// test process itself as the live process so AliveRecord is genuinely true.
func TestAgentsSurfacesBlocked(t *testing.T) {
	w := newExecWS(t)
	pid := os.Getpid()
	rec := procmon.Record{
		RunID: runID(3), Child: "a-child-2", Task: "t1", Runtime: "cc",
		PID: pid, PGID: pid, PIDStart: pidStart(pid), Started: time.Now(),
	}
	if err := os.MkdirAll(w.RunDir(rec.RunID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := procmon.WriteRecord(filepath.Join(w.RunDir(rec.RunID), "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}

	// Control: no blocked file -> no BLOCKED tag.
	ctx, out, _ := newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "BLOCKED") {
		t.Fatalf("a normal live agent must not be tagged BLOCKED:\n%s", out.String())
	}

	// Raise the channel; agents must now tag it and print the reason.
	writeBlocked(t, w.RunDir(rec.RunID), "dacli unavailable: exec format error\nran: dacli note add finding")
	ctx, out, _ = newCtx(w.Root)
	if err := cmdAgents(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "BLOCKED") {
		t.Errorf("agents must tag a blocked live agent BLOCKED:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "exec format error") {
		t.Errorf("agents must print the blocked reason:\n%s", out.String())
	}
}

// The prompt half of the acceptance: the child's brief names the exact channel
// path and tells it this is a plain file write with no dacli, for when blocked.
func TestBlockedChannelPreambleNamesPathAndUsage(t *testing.T) {
	path := "/ws/.dacli/runs/01ABC/blocked.txt"
	p := blockedChannelPreamble(path)
	for _, want := range []string{path, "BLOCKED", "plain text file", "no command", "last resort"} {
		if !strings.Contains(p, want) {
			t.Errorf("blocked-channel preamble missing %q:\n%s", want, p)
		}
	}
}
