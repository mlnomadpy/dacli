package cli

import (
	"strings"
	"testing"
)

// report is explicit and gathers context; --dry-run proves what it would
// file without touching gh, so the test needs no network.
func TestReportDryRun(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "myproj")

	// Ungated: the version/platform context is present, but the workspace name
	// and transcript are WITHHELD — the upstream is public, so they are a gated
	// disclosure.
	out := run(t, dir, 0, "report", "spawn hangs on the mock runtime",
		"--body", "the child never exits", "--repo", "someone/dacli", "--dry-run")
	for _, want := range []string{
		"would file to someone/dacli",
		"[agent-report] spawn hangs on the mock runtime",
		"the child never exits",
		"dacli: dev", // version stamped
		"withheld",   // workspace name gated behind --disclose
		"Reported via",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report dry-run missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "workspace: myproj") {
		t.Errorf("ungated report must NOT leak the workspace name to a public upstream:\n%s", out)
	}

	// --disclose opts in: the workspace name is attached.
	disc := run(t, dir, 0, "report", "spawn hangs", "--repo", "someone/dacli", "--disclose", "--dry-run")
	if !strings.Contains(disc, "workspace: myproj") {
		t.Errorf("--disclose report should attach the workspace name:\n%s", disc)
	}
	// The target defaults to the tool's own repo, not the user's project.
	def := run(t, dir, 0, "report", "x", "--dry-run")
	if !strings.Contains(def, "would file to mlnomadpy/dacli") {
		t.Errorf("default target should be the tool repo:\n%s", def)
	}
	// Env overrides the default.
	t.Setenv("DACLI_REPORT_REPO", "fork/dacli")
	if e := run(t, dir, 0, "report", "x", "--dry-run"); !strings.Contains(e, "fork/dacli") {
		t.Errorf("env repo override ignored:\n%s", e)
	}
}

func TestVersion(t *testing.T) {
	dir := t.TempDir()
	if out := run(t, dir, 0, "version"); !strings.Contains(out, "dacli dev") {
		t.Errorf("version: %s", out)
	}
}

func TestReportNeedsAMessage(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 2, "report")
}
