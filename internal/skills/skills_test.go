package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "skills-test")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// writeSkill materializes a skill directory. mainName lets a test choose the
// manifest's exact case, which is the whole point of the lossless-import rule.
func writeSkill(t *testing.T, dir, mainName, body string, files map[string]fileSpec) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, mainName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, spec := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(spec.content), spec.mode); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

type fileSpec struct {
	content string
	mode    os.FileMode
}

const pdfSkill = `---
name: pdf
description: Extract text and tables from PDF files
min_delivery: native
---

# pdf

## Usage

Call the extractor with a path.
`

// The manifest is found case-insensitively because import NEVER renames: a
// native SKILL.md and a dacli-authored skill.md must both load, and the
// canonical name returned must be the name as it exists ON DISK. On macOS's
// case-insensitive filesystem an os.Stat("skill.md") probe matches SKILL.md
// and returns the WRONG name, which then fails every equality check
// downstream — that is the bug this walks the directory listing to avoid.
func TestMainFileKeepsTheOnDiskCase(t *testing.T) {
	for _, name := range []string{"SKILL.md", "skill.md", "Skill.md"} {
		dir := writeSkill(t, filepath.Join(t.TempDir(), "s"), name, pdfSkill, nil)
		got, entries := mainFile(dir)
		if got != name {
			t.Errorf("mainFile returned %q, want the on-disk name %q", got, name)
		}
		if len(entries) == 0 {
			t.Errorf("mainFile must also return the directory listing it read")
		}
	}
	// A directory with no manifest is not a skill.
	empty := t.TempDir()
	if got, _ := mainFile(empty); got != "" {
		t.Errorf("mainFile on a manifest-less dir = %q, want empty", got)
	}
	if got, _ := mainFile(filepath.Join(empty, "nope")); got != "" {
		t.Errorf("mainFile on a missing dir = %q, want empty", got)
	}
}

// load() reads the canonical shape: name/description/min_delivery from the
// frontmatter, the body from every non-H1 section, and the payload files split
// into plain resources vs SCRIPTS — scripts are what cannot ride a context or
// inline target, so the classification drives the delivery decision.
func TestLoadParsesTheCanonicalSkill(t *testing.T) {
	dir := writeSkill(t, filepath.Join(t.TempDir(), "pdf"), "SKILL.md", pdfSkill, map[string]fileSpec{
		"reference.md": {"a table of flags\n", 0o644},
		"extract.py":   {"print('hi')\n", 0o644}, // script by extension, not by mode
		"run.sh":       {"echo hi\n", 0o644},
		"tool":         {"#!/bin/sh\n", 0o755}, // script by executable bit
	})
	s, err := load(dir, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "pdf" {
		t.Errorf("Name = %q, want pdf (from the frontmatter, not the dir)", s.Name)
	}
	if s.Desc != "Extract text and tables from PDF files" {
		t.Errorf("Desc = %q", s.Desc)
	}
	if s.MinDelivery != Native {
		t.Errorf("MinDelivery = %q, want native", s.MinDelivery)
	}
	// The H1 title is the skill's own heading, not instruction content.
	if strings.Contains(s.Body, "# pdf") {
		t.Errorf("body swallowed the H1 title:\n%s", s.Body)
	}
	if !strings.Contains(s.Body, "Call the extractor with a path.") {
		t.Errorf("body lost the instructions:\n%s", s.Body)
	}
	if s.EstTokens <= 0 {
		t.Errorf("EstTokens = %d; a context/inline target's per-turn tax must be estimated", s.EstTokens)
	}

	if len(s.Resources) != 4 {
		t.Errorf("Resources = %v, want all four payload files (the manifest excluded)", s.Resources)
	}
	for _, r := range s.Resources {
		if strings.EqualFold(r, "SKILL.md") {
			t.Errorf("the manifest must not be listed as a resource")
		}
	}
	scripts := map[string]bool{}
	for _, sc := range s.Scripts {
		scripts[sc] = true
	}
	for _, want := range []string{"extract.py", "run.sh", "tool"} {
		if !scripts[want] {
			t.Errorf("%q not classified as a script (it cannot ride a context file)", want)
		}
	}
	if scripts["reference.md"] {
		t.Errorf("a plain markdown resource must not be classified as a script")
	}
}

// Defaults matter: a skill with no name falls back to its directory name (so
// an imported native skill without a `name:` still resolves), and no
// min_delivery means INLINE — the floor that always works, never a silent
// requirement that gets it omitted.
func TestLoadDefaults(t *testing.T) {
	body := "---\ndescription: no name here\n---\n\n## Usage\n\nDo it.\n"
	dir := writeSkill(t, filepath.Join(t.TempDir(), "on-disk-name"), "SKILL.md", body, nil)
	s, err := load(dir, "on-disk-name")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "on-disk-name" {
		t.Errorf("Name = %q, want the directory-name fallback", s.Name)
	}
	if s.MinDelivery != Inline {
		t.Errorf("MinDelivery = %q, want inline (the always-works floor)", s.MinDelivery)
	}
	if _, err := load(t.TempDir(), "x"); err == nil {
		t.Error("a directory with no manifest must not load as a skill")
	}
}

func TestLoadSkillsAndLoadSkill(t *testing.T) {
	w := newWS(t)
	writeSkill(t, filepath.Join(w.SkillsLibDir(), "zebra"), "SKILL.md", "---\nname: zebra\n---\n\n## Use\n\nz\n", nil)
	writeSkill(t, filepath.Join(w.SkillsLibDir(), "alpha"), "skill.md", "---\nname: alpha\n---\n\n## Use\n\na\n", nil)
	// A stray directory with no manifest is skipped, not an error.
	if err := os.MkdirAll(filepath.Join(w.SkillsLibDir(), "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	list, err := LoadSkills(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "alpha" || list[1].Name != "zebra" {
		t.Fatalf("LoadSkills = %v, want [alpha zebra] sorted by name", names(list))
	}

	if s, err := LoadSkill(w, "zebra"); err != nil || s.Name != "zebra" {
		t.Errorf("LoadSkill(zebra) = (%+v, %v)", s, err)
	}
	// An unknown skill is a not-found, not a zero Skill silently compiled into
	// a role's toolkit.
	if _, err := LoadSkill(w, "nope"); err == nil {
		t.Error("LoadSkill on a missing skill must error")
	} else {
		var nf store.ErrNotFound
		if !errors.As(err, &nf) {
			t.Errorf("LoadSkill error = %T, want store.ErrNotFound", err)
		}
	}
}

func names(list []Skill) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.Name
	}
	return out
}

// Plan is the fidelity ladder: the best mode the RUNTIME supports, floored by
// each SKILL's min_delivery. Below the floor the skill is OMITTED AND
// ANNOUNCED — a silently absent skill is a role lying about its competence, so
// every omission must carry a reason.
func TestPlanDeliveryLadder(t *testing.T) {
	native := store.Runtime{Name: "claude-code", SkillsNativeDir: ".claude/skills"}
	context := store.Runtime{Name: "ctx-cli", SkillsContextFile: "AGENTS.md"}
	inline := store.Runtime{Name: "plain-cli"}
	// A runtime declaring BOTH takes the richer one.
	both := store.Runtime{Name: "rich", SkillsNativeDir: ".x/skills", SkillsContextFile: "AGENTS.md"}

	cases := []struct {
		name   string
		rt     store.Runtime
		min    Delivery
		want   Delivery
		reason bool
	}{
		{"native runtime, native floor", native, Native, Native, false},
		{"native runtime, inline floor", native, Inline, Native, false},
		{"context runtime, inline floor", context, Inline, Context, false},
		{"context runtime, context floor", context, Context, Context, false},
		{"context runtime CANNOT meet a native floor", context, Native, Omitted, true},
		{"inline runtime, inline floor", inline, Inline, Inline, false},
		{"inline runtime CANNOT meet a context floor", inline, Context, Omitted, true},
		{"inline runtime CANNOT meet a native floor", inline, Native, Omitted, true},
		{"native beats context when both are declared", both, Native, Native, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := Plan([]Skill{{Name: "s", MinDelivery: tc.min}}, tc.rt)
			if len(items) != 1 {
				t.Fatalf("Plan returned %d items", len(items))
			}
			if items[0].Mode != tc.want {
				t.Errorf("mode = %q, want %q", items[0].Mode, tc.want)
			}
			if tc.reason && items[0].Reason == "" {
				t.Error("an omitted skill must be ANNOUNCED with a reason, never silently dropped")
			}
			if tc.reason && !strings.Contains(items[0].Reason, string(tc.min)) {
				t.Errorf("the omission reason %q does not name the unmet floor %q", items[0].Reason, tc.min)
			}
		})
	}
}

// A skill carrying scripts delivered to a non-native target keeps its mode but
// must announce that the scripts cannot ride along — otherwise a role gets a
// skill whose tools silently do not exist.
func TestPlanFlagsScriptsOnNonNativeTargets(t *testing.T) {
	withScript := Skill{Name: "pdf", MinDelivery: Inline, Scripts: []string{"extract.py"}}

	native := Plan([]Skill{withScript}, store.Runtime{Name: "cc", SkillsNativeDir: ".claude/skills"})[0]
	if native.Mode != Native || native.Reason != "" {
		t.Errorf("a native target carries scripts fine; got mode %q reason %q", native.Mode, native.Reason)
	}

	for _, rt := range []store.Runtime{{Name: "ctx", SkillsContextFile: "AGENTS.md"}, {Name: "plain"}} {
		it := Plan([]Skill{withScript}, rt)[0]
		if it.Reason == "" {
			t.Errorf("%s: script deferral not announced", rt.Name)
		}
		if !strings.Contains(it.Reason, "script") {
			t.Errorf("%s: reason %q does not mention scripts", rt.Name, it.Reason)
		}
	}
}

// Compile is a REGENERABLE PROJECTION: the whole role dir is deleted and
// rebuilt, so output from a previous compile can never survive into the next
// one and be delivered as if it were current.
func TestCompileIsARegenerableProjection(t *testing.T) {
	w := newWS(t)
	dir := writeSkill(t, filepath.Join(w.SkillsLibDir(), "pdf"), "SKILL.md", pdfSkill, map[string]fileSpec{
		"extract.py": {"print('hi')\n", 0o755},
	})
	s, err := load(dir, "pdf")
	if err != nil {
		t.Fatal(err)
	}
	rt := store.Runtime{Name: "cc", SkillsNativeDir: ".claude/skills"}

	out, tax, err := Compile(w, "implementer", rt, Plan([]Skill{s}, rt))
	if err != nil {
		t.Fatal(err)
	}
	// Native delivery is lazy-loaded: it costs nothing per turn.
	if tax != 0 {
		t.Errorf("native delivery turn tax = %d, want 0 (it is lazy-loaded)", tax)
	}
	if _, err := os.Stat(filepath.Join(out, "pdf", "SKILL.md")); err != nil {
		t.Errorf("native skill not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "pdf", "extract.py")); err != nil {
		t.Errorf("native skill's script resource not copied: %v", err)
	}

	// A leftover from a previous compile must not survive the next one.
	stale := filepath.Join(out, "stale.md")
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Compile(w, "implementer", rt, Plan([]Skill{s}, rt)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale compile output survived a rebuild (err %v)", err)
	}
}

// Context and inline targets carry the FULL BODY, so their per-turn token tax
// is real and must be reported; the two land in different files because one is
// a startup file and the other is prepended to the brief.
func TestCompileContextAndInlineTargets(t *testing.T) {
	w := newWS(t)
	dir := writeSkill(t, filepath.Join(w.SkillsLibDir(), "pdf"), "SKILL.md", pdfSkill, nil)
	s, err := load(dir, "pdf")
	if err != nil {
		t.Fatal(err)
	}
	s.MinDelivery = Inline // let it ride a weaker target

	ctxRT := store.Runtime{Name: "ctx", SkillsContextFile: "AGENTS.md"}
	out, tax, err := Compile(w, "r", ctxRT, Plan([]Skill{s}, ctxRT))
	if err != nil {
		t.Fatal(err)
	}
	if tax != s.EstTokens {
		t.Errorf("context turn tax = %d, want the skill's %d — an unreported tax is an invisible cost", tax, s.EstTokens)
	}
	raw, err := os.ReadFile(filepath.Join(out, "AGENTS.md"))
	if err != nil {
		t.Fatalf("context file not written: %v", err)
	}
	for _, want := range []string{"dacli:skill:pdf begin", "## Skill: pdf", "Call the extractor with a path.", "dacli:skill:pdf end"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("context file missing %q:\n%s", want, raw)
		}
	}

	inlineRT := store.Runtime{Name: "plain"}
	out2, tax2, err := Compile(w, "r", inlineRT, Plan([]Skill{s}, inlineRT))
	if err != nil {
		t.Fatal(err)
	}
	if tax2 != s.EstTokens {
		t.Errorf("inline turn tax = %d, want %d", tax2, s.EstTokens)
	}
	if _, err := os.ReadFile(filepath.Join(out2, "inline.md")); err != nil {
		t.Errorf("inline payload not written: %v", err)
	}

	// An OMITTED skill contributes no payload and no tax.
	s.MinDelivery = Native
	out3, tax3, err := Compile(w, "r", inlineRT, Plan([]Skill{s}, inlineRT))
	if err != nil {
		t.Fatal(err)
	}
	if tax3 != 0 {
		t.Errorf("an omitted skill charged %d tokens", tax3)
	}
	entries, _ := os.ReadDir(out3)
	if len(entries) != 0 {
		t.Errorf("an omitted skill wrote %d file(s)", len(entries))
	}
}

// Import is LOSSLESS — byte copies, never rewrites — because the anti-fifteenth
// -standard mitigation is that a dacli skill IS a valid native skill. File
// contents and the executable bit both have to survive, or an imported skill's
// scripts stop working.
func TestImportIsLossless(t *testing.T) {
	w := newWS(t)
	src := t.TempDir()
	writeSkill(t, filepath.Join(src, "pdf"), "SKILL.md", pdfSkill, map[string]fileSpec{
		"extract.py":          {"#!/usr/bin/env python3\nprint('hi')\n", 0o755},
		"refs/reference.md":   {"nested payload\n", 0o644},
		"weird name & chars!": {"verbatim\n", 0o644},
	})
	// A sibling directory that is not a skill must be skipped silently.
	if err := os.MkdirAll(filepath.Join(src, "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	imported, err := Import(w, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 1 || imported[0] != "pdf" {
		t.Fatalf("Import = %v, want [pdf]", imported)
	}

	dst := filepath.Join(w.SkillsLibDir(), "pdf")
	for name, want := range map[string]string{
		"SKILL.md":            pdfSkill,
		"extract.py":          "#!/usr/bin/env python3\nprint('hi')\n",
		"refs/reference.md":   "nested payload\n",
		"weird name & chars!": "verbatim\n",
	} {
		got, err := os.ReadFile(filepath.Join(dst, name))
		if err != nil {
			t.Errorf("%s not imported: %v", name, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s was rewritten, not copied:\n got %q\nwant %q", name, got, want)
		}
	}
	// The manifest keeps its original case: a rename would break the "still a
	// valid native skill" contract.
	if _, err := os.Stat(filepath.Join(dst, "SKILL.md")); err != nil {
		t.Errorf("the manifest was renamed on import: %v", err)
	}
	info, err := os.Stat(filepath.Join(dst, "extract.py"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("the executable bit was lost on import (mode %v) — the script no longer runs", info.Mode())
	}

	// Re-importing must refuse rather than clobber a skill already in the
	// library: an overwrite would silently discard local edits.
	if _, err := Import(w, src); err == nil {
		t.Error("importing over an existing skill must refuse")
	}
}

func TestImportMissingSourceErrors(t *testing.T) {
	w := newWS(t)
	if _, err := Import(w, filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("importing from a missing directory must error")
	}
}

// Fetch's contract is the registry's own owner/repo convention. A malformed ref
// must be rejected BEFORE any git clone is attempted — these tests never touch
// the network, which is exactly the property being asserted.
func TestFetchRejectsMalformedRefs(t *testing.T) {
	w := newWS(t)
	for _, bad := range []string{"", "justaname", "too/many/slashes", "a/b/c/d"} {
		if _, err := Fetch(w, bad); err == nil {
			t.Errorf("Fetch(%q) was accepted; owner/repo is the whole contract", bad)
		} else if !strings.Contains(err.Error(), "owner/repo") {
			t.Errorf("Fetch(%q) error %q does not explain the expected form", bad, err)
		}
	}
}

func TestOwnerRepoLeaf(t *testing.T) {
	if got := ownerRepoLeaf("mattpocock/skills"); got != "skills" {
		t.Errorf("ownerRepoLeaf = %q, want skills", got)
	}
}

// rank orders the ladder; Plan's floor comparison is meaningless if it drifts.
func TestDeliveryRank(t *testing.T) {
	if !(rank(Native) > rank(Context) && rank(Context) > rank(Inline) && rank(Inline) > rank(Omitted)) {
		t.Errorf("the fidelity ladder is out of order: native=%d context=%d inline=%d omitted=%d",
			rank(Native), rank(Context), rank(Inline), rank(Omitted))
	}
	if rank("nonsense") != 0 {
		t.Errorf("an unknown delivery must rank below every real one")
	}
}
