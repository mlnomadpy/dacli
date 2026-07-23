package mdstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const canonical = `---
id: t-002
kind: task
created: 2026-07-21T14:06:20Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 2, probable: 5, pessimistic: 14}
tags: [billing, urgent]
# a full-line comment that must survive
mystery_key_from_the_future: some value
---
# Add the ledger write shim

## Context
Why this task exists.

## Acceptance
- [ ] Shim covers the nightly batch path
- [x] Reconciliation suite green
`

func TestParseFrontmatter(t *testing.T) {
	d, err := Parse(canonical)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"id": "t-002", "kind": "task", "owner": "a-root", "priority": "must",
	} {
		if got, _ := d.Front.Get(k); got != want {
			t.Errorf("Get(%q) = %q, want %q", k, got, want)
		}
	}
	if got := d.Front.GetList("tags"); len(got) != 2 || got[0] != "billing" || got[1] != "urgent" {
		t.Errorf("GetList(tags) = %v", got)
	}
	m := d.Front.GetMap("estimate")
	if m["optimistic"] != "2" || m["probable"] != "5" || m["pessimistic"] != "14" {
		t.Errorf("GetMap(estimate) = %v", m)
	}
}

// Invariant 1: round-trip is byte-exact on canonical files, and unknown keys
// and comments survive.
func TestRoundTripByteExact(t *testing.T) {
	d, err := Parse(canonical)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(d)
	if out != canonical {
		t.Errorf("round-trip not byte-exact:\n--- got ---\n%s\n--- want ---\n%s", out, canonical)
	}
	if _, ok := d.Front.Get("mystery_key_from_the_future"); !ok {
		t.Error("unknown key dropped")
	}
}

// Real native skills write `description: |` — the literal-block indicator
// must parse as a block, round-trip byte-exactly, and read back as text.
func TestLiteralBlockScalars(t *testing.T) {
	const in = "---\nname: audit\ndescription: |\n  Referee-grade audit of a paper draft —\n  find proof gaps, trace assumptions.\n---\nbody\n"
	d, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if Render(d) != in {
		t.Errorf("literal block not byte-exact:\n%s", Render(d))
	}
	text, ok := d.Front.GetText("description")
	if !ok || !strings.Contains(text, "Referee-grade audit") || !strings.Contains(text, "trace assumptions.") {
		t.Errorf("GetText = %q", text)
	}
	if strings.Contains(text, "  Referee") {
		t.Error("GetText should dedent")
	}
}

func TestBlockValuesPreservedVerbatim(t *testing.T) {
	const in = "---\nid: x\ngithub:\n  issue: 42\n  node_id: I_abc\n---\nbody\n"
	d, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	blk, ok := d.Front.GetBlock("github")
	if !ok || !strings.Contains(blk, "issue: 42") {
		t.Fatalf("block not captured: %q", blk)
	}
	if Render(d) != in {
		t.Errorf("block round-trip:\n%s", Render(d))
	}
}

func TestSections(t *testing.T) {
	d, _ := Parse(canonical)
	s, ok := d.Section("Acceptance")
	if !ok {
		t.Fatal("Acceptance section missing")
	}
	boxes := Checkboxes(s.Content)
	if len(boxes) != 2 || boxes[0].Done || !boxes[1].Done {
		t.Errorf("checkboxes = %+v", boxes)
	}
	if _, ok := d.Section("context"); !ok {
		t.Error("section lookup should be case-insensitive")
	}
}

// A # inside a code fence is not a heading.
func TestHeadingsInsideFencesIgnored(t *testing.T) {
	const in = "## Real\ntext\n```bash\n# not a heading\n```\nmore\n## Also real\nx\n"
	d, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Sections) != 2 {
		t.Fatalf("sections = %d, want 2: %+v", len(d.Sections), d.Sections)
	}
	if !strings.Contains(d.Sections[0].Content, "# not a heading") {
		t.Error("fenced content lost")
	}
}

func TestValueCleaning(t *testing.T) {
	const in = "---\npriority: must   # must | should | could\nname: \"quoted name\"\n---\n"
	d, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := d.Front.Get("priority"); v != "must" {
		t.Errorf("trailing comment not stripped: %q", v)
	}
	if v, _ := d.Front.Get("name"); v != "quoted name" {
		t.Errorf("quotes not stripped: %q", v)
	}
	// But the raw value (with comment) survives rendering.
	if !strings.Contains(Render(d), "# must | should") {
		t.Error("inline comment lost on render")
	}
}

func TestUnterminatedFrontmatterIsError(t *testing.T) {
	if _, err := Parse("---\nid: x\nno terminator"); err == nil {
		t.Error("unterminated frontmatter should be an error, not a silent partial parse")
	}
}

func TestNoFrontmatter(t *testing.T) {
	d, err := Parse("# Just a doc\n\nbody\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Front.Keys()) != 0 {
		t.Error("phantom frontmatter")
	}
	if d.Sections[0].Title != "Just a doc" {
		t.Errorf("title = %q", d.Sections[0].Title)
	}
}

// Invariant 2: a write is all-or-nothing.
func TestWriteFileAtomicAndReadBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "x.md")
	d, _ := Parse(canonical)
	if err := WriteFile(path, d); err != nil {
		t.Fatal(err)
	}
	back, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if Render(back) != canonical {
		t.Error("read-back differs")
	}
	// No temp litter.
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".dacli-tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

// A rename fault must not orphan the temp file. Renaming over an existing
// non-empty directory fails, exercising the os.Rename error branch.
func TestWriteFileCleansTempOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target")
	// Make `path` a non-empty directory so os.Rename(tmp, path) fails.
	if err := os.MkdirAll(filepath.Join(path, "occupied"), 0o755); err != nil {
		t.Fatal(err)
	}
	d, _ := Parse(canonical)
	if err := WriteFile(path, d); err == nil {
		t.Fatal("expected rename to fail when target is a non-empty directory")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".dacli-tmp-") {
			t.Errorf("temp file leaked on rename failure: %s", e.Name())
		}
	}
}

func TestLinks(t *testing.T) {
	got := Links("see [[t-001]] and [[d-sync|the decision]] but not [broken")
	if len(got) != 2 || got[0] != "t-001" || got[1] != "d-sync" {
		t.Errorf("Links = %v", got)
	}
}

func TestBullets(t *testing.T) {
	got := Bullets("- one\n- [ ] not me\n- two\ntext\n")
	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("Bullets = %v", got)
	}
}

// SetList must be the exact inverse of GetList: every element that would
// otherwise confuse splitTop/clean (a comma, brackets/braces, quotes, or
// leading/trailing whitespace) round-trips losslessly.
func TestSetListRoundTrip(t *testing.T) {
	cases := [][]string{
		{"billing", "urgent"},
		{"Read,Grep,Glob,LS,Bash(dacli:*)"},
		{"a [bracketed] value", "a {braced} value"},
		{`has "double" quotes`},
		{"has 'single' quotes"},
		{"  leading and trailing spaces  "},
		{"plain", "with,comma", "with[bracket]", "with{brace}", `with"quote`, "with'apos", "  padded  "},
		{},
		nil,
	}
	for _, elems := range cases {
		var f Front
		f.SetList("k", elems)
		got := f.GetList("k")
		if len(got) != len(elems) {
			t.Fatalf("SetList(%q) round-trip length = %d, want %d (rendered %q)", elems, len(got), len(elems), f.entries)
		}
		for i := range elems {
			if got[i] != elems[i] {
				t.Errorf("SetList(%q) round-trip[%d] = %q, want %q", elems, i, got[i], elems[i])
			}
		}
	}
}
