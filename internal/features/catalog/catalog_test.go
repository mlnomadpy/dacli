package catalog

import (
	"path/filepath"
	"strings"
	"testing"
)

// --out must resolve against the CALLER's cwd, not the workspace root, so a
// worktree agent's catalog lands in its own tree rather than the shared main
// checkout. An absolute path is honored verbatim; an empty flag defaults.
func TestResolveOut(t *testing.T) {
	cwd := filepath.Join("home", "agent", "worktree")
	if got, want := resolveOut(cwd, ""), filepath.Join(cwd, defaultOut); got != want {
		t.Errorf("resolveOut(cwd, \"\") = %q, want %q (default relative to caller)", got, want)
	}
	if got, want := resolveOut(cwd, "out/R.md"), filepath.Join(cwd, "out", "R.md"); got != want {
		t.Errorf("resolveOut relative = %q, want %q", got, want)
	}
	abs := filepath.Join("etc", "roster.md")
	if !filepath.IsAbs(abs) {
		abs = string(filepath.Separator) + abs
	}
	if got := resolveOut(cwd, abs); got != abs {
		t.Errorf("resolveOut(abs) = %q, want %q (absolute honored verbatim)", got, abs)
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
		"plain":       "plain",
		"a | b":       "a \\| b",
		"line1\nline2": "line1 line2",
		"  spaced  ":  "spaced",
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

func TestFirstLine(t *testing.T) {
	if got := firstLine("one\ntwo\nthree"); got != "one" {
		t.Errorf("firstLine = %q, want %q", got, "one")
	}
	if got := firstLine("  solo  "); got != "solo" {
		t.Errorf("firstLine trimmed = %q, want %q", got, "solo")
	}
}
