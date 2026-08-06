package catalog

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
)

// A FAILED `git status` must surface as an error, never as a silent clean tree.
// Before this fix the call site discarded git's error and read the empty stdout
// of a failed status as "nothing to commit", so a wiki that was never pushed was
// reported as already up to date (219).
func TestWikiCleanFailedStatusIsError(t *testing.T) {
	// Empty output, no error: a genuinely clean tree.
	if clean, err := wikiClean("", nil); err != nil || !clean {
		t.Errorf("empty output with no error must be clean: clean=%v err=%v", clean, err)
	}
	// Non-empty output, no error: a dirty tree with changes to commit.
	if clean, err := wikiClean(" M Roster.md", nil); err != nil || clean {
		t.Errorf("non-empty status must be dirty: clean=%v err=%v", clean, err)
	}
	// A git error MUST propagate, even though its stdout is empty — the empty
	// string is the whole trap: it looks identical to a clean tree. This is the
	// case the pre-fix call site got wrong (it discarded err and read out=="" as
	// clean); the error must reach the caller so publish fails loudly (219).
	if clean, err := wikiClean("", errors.New("fatal: not a git repository")); err == nil || clean {
		t.Errorf("a failed status must be an error, not a clean tree: clean=%v err=%v", clean, err)
	}
}

// --out must resolve against the CALLER's cwd, not the workspace root, so a
// worktree agent's catalog lands in its own tree rather than the shared main
// checkout — and must never resolve OUTSIDE that tree. An absolute or
// dot-dot --out was an arbitrary-file write for anyone who could run the
// command (2026-08-06 audit); both are refused now.
func TestResolveOut(t *testing.T) {
	cwd := t.TempDir()
	got, err := resolveOut(cwd, "")
	if err != nil {
		t.Fatalf("resolveOut(cwd, \"\"): %v", err)
	}
	if want := filepath.Join(cwd, defaultOut); got != want {
		t.Errorf("resolveOut(cwd, \"\") = %q, want %q (default relative to caller)", got, want)
	}
	if got, err := resolveOut(cwd, "out/R.md"); err != nil || got != filepath.Join(cwd, "out", "R.md") {
		t.Errorf("resolveOut relative = %q, %v — want the path under cwd and no error", got, err)
	}

	// Escapes: an absolute path elsewhere, and a traversal out of the tree.
	// Both must be REFUSED (exit 3), not silently honored or clamped.
	for _, esc := range []string{
		filepath.Join(string(filepath.Separator), "etc", "roster.md"),
		filepath.Join("..", "..", "escaped.md"),
	} {
		out, err := resolveOut(cwd, esc)
		if err == nil {
			t.Errorf("resolveOut(%q) = %q with no error — an out-of-tree write must be refused", esc, out)
			continue
		}
		if code := clikit.ExitCode(err); code != 3 {
			t.Errorf("resolveOut(%q) exit code = %d, want 3 (policy refusal)", esc, code)
		}
	}

	// A path under cwd that merely LOOKS absolute-adjacent still works, and a
	// sibling directory sharing the cwd prefix is not mistaken for inside it.
	if _, err := resolveOut(cwd, "docs/deep/nested/R.md"); err != nil {
		t.Errorf("a nested path under cwd must be allowed: %v", err)
	}
	if _, err := resolveOut(cwd, filepath.Join("..", filepath.Base(cwd)+"-sibling", "R.md")); err == nil {
		t.Error("a sibling dir sharing the cwd prefix must not count as inside it")
	}
}

// renderCatalog is the load-bearing pure projection: it must be deterministic,
// scannable, and injection-proof (a pipe in a purpose must not break the table).
func TestRenderCatalog(t *testing.T) {
	roles := []roleRow{
		{Name: "implementer", Version: "v2", Grant: "rw", Kind: "implementer", Model: "opus",
			Purpose: "writes code", LastChanged: "3 days ago · 069: catalog", Skills: []string{"go", "git"}},
		{Name: "reviewer", Version: "v1", Grant: "ro", Purpose: "reviews | audits code"},
	}
	skls := []skillRow{
		{Name: "verify", Version: "v1", Purpose: "drive the flow", EstTokens: 512, LastChanged: "1 week ago · seed"},
	}
	md := renderCatalog(roles, skls)

	for _, want := range []string{
		"# Team Roster",
		"one-way read view", // the no-edit provenance banner (paraphrase check below)
		"## Roles (2)",
		"## Skills (1)",
		"| implementer | v2 | rw | implementer | opus | go, git | writes code | 3 days ago · 069: catalog |",
		"| verify | v1 | 512 | drive the flow | 1 week ago · seed |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("catalog missing %q\n---\n%s", want, md)
		}
	}
	// The banner must state the one-way rule so a reader never edits the catalog.
	if !strings.Contains(md, "do **not** edit") {
		t.Errorf("catalog must warn against editing the generated view:\n%s", md)
	}
	// The preamble must state that grant and runtime have to agree and how to
	// check, so a reader never files a ro role onto a runtime that cannot back it.
	// It must also say that the rw direction is the UNENFORCED one — a reader
	// told only about the ro refusal concludes the coupling is handled, and
	// then routes implementation work to a role whose runtime cannot write.
	for _, want := range []string{"must agree with its runtime", "dacli runtime doctor", "is **not** checked"} {
		if !strings.Contains(md, want) {
			t.Errorf("catalog preamble missing grant/runtime note %q:\n%s", want, md)
		}
	}
	// A pipe inside a cell must be escaped, never left to split the row.
	if !strings.Contains(md, "reviews \\| audits code") {
		t.Errorf("pipe in a cell was not escaped:\n%s", md)
	}
	// An empty optional field renders as a dash, not a blank cell.
	if !strings.Contains(md, "| reviewer | v1 | ro | — | — |") {
		t.Errorf("empty role fields should render as em dashes:\n%s", md)
	}
}

// An empty roster is still a valid, honest page — not a crash or a blank file.
func TestRenderCatalogEmpty(t *testing.T) {
	md := renderCatalog(nil, nil)
	for _, want := range []string{"## Roles (0)", "_No roles defined._", "## Skills (0)", "_No skills in the library._"} {
		if !strings.Contains(md, want) {
			t.Errorf("empty catalog missing %q\n---\n%s", want, md)
		}
	}
}

func TestCell(t *testing.T) {
	cases := map[string]string{
		"plain":        "plain",
		"a | b":        "a \\| b",
		"line1\nline2": "line1 line2",
		"  spaced  ":   "spaced",
	}
	for in, want := range cases {
		if got := cell(in); got != want {
			t.Errorf("cell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDash(t *testing.T) {
	if got := dash(""); got != "—" {
		t.Errorf("dash(empty) = %q, want em dash", got)
	}
	if got := dash("   "); got != "—" {
		t.Errorf("dash(blank) = %q, want em dash", got)
	}
	if got := dash("x"); got != "x" {
		t.Errorf("dash(%q) = %q, want passthrough", "x", got)
	}
}

// The disclosure gate reuses ghmirror's scoped-consent semantics: consent is the
// exact repo, never a bare boolean and never a different repo.
func TestConsentCoversRepo(t *testing.T) {
	if !consentCoversRepo("owner/repo", "owner/repo") {
		t.Error("consent for a repo must cover that same repo")
	}
	if !consentCoversRepo("Owner/Repo", "owner/repo") {
		t.Error("consent match must be case-insensitive")
	}
	if consentCoversRepo("owner/repo", "owner/other") {
		t.Error("consent for one repo must NOT cover a different repo")
	}
	if consentCoversRepo("true", "owner/repo") {
		t.Error("a legacy bare-boolean consent must fail closed")
	}
	if consentCoversRepo("", "owner/repo") {
		t.Error("absent consent must fail closed")
	}
}
