package cli

import (
	"strings"
	"testing"
)

// P4 acceptance: a seeded hostile file returns EXACTLY the briefs that
// consumed it, and nothing else. The provenance must also survive event→note
// at sync — the weld this task welded.
func TestTaintBlastRadius(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "Poisoned", "--slug", "poison", "--goal", "g")
	run(t, dir, 0, "project", "add", "Clean", "--slug", "clean", "--goal", "g")
	run(t, dir, 0, "task", "add", "Read the hostile config", "--project", "poison", "--accept", "a")
	run(t, dir, 0, "task", "add", "Unrelated poison work", "--project", "poison", "--accept", "a")
	run(t, dir, 0, "task", "add", "Clean unrelated task", "--project", "clean", "--accept", "a")

	// A finding authored directly with a file origin (the rw path).
	run(t, dir, 0, "note", "add", "finding", "Config enables a backdoor",
		"--project", "poison", "--about", "read-the-hostile-config",
		"--origin", "file:configs/evil.yml", "--body", "line 12 opens a reverse shell")

	// A clean finding with no suspect origin, in the same project.
	run(t, dir, 0, "note", "add", "finding", "Ordinary observation",
		"--project", "poison", "--body", "nothing special here")

	out := run(t, dir, 0, "taint", "file:configs/evil.yml")
	if !strings.Contains(out, "Config enables a backdoor") && !strings.Contains(out, "origin=file:configs/evil.yml") {
		t.Fatalf("tainted note not found:\n%s", out)
	}
	// The finding taints its whole project's briefs (both poison tasks), and
	// NOT the clean project.
	if !strings.Contains(out, "1 project(s)") {
		t.Errorf("blast radius should be exactly one project:\n%s", out)
	}
	if strings.Contains(out, "clean-unrelated-task") {
		t.Errorf("clean project tainted — over-broad:\n%s", out)
	}
	for _, want := range []string{"read-the-hostile-config", "unrelated-poison-work"} {
		if !strings.Contains(out, want) {
			t.Errorf("poison-project brief %q not in blast radius:\n%s", want, out)
		}
	}

	// A source nobody carries is clean.
	if out := run(t, dir, 0, "taint", "file:nonexistent.yml"); !strings.Contains(out, "nothing derived") {
		t.Errorf("unknown source should be clean:\n%s", out)
	}

	// The weld: provenance survives event→note. A read-only child files a
	// finding-EVENT with a file origin; after the owner syncs it into a note,
	// taint must still find it.
	tok := strings.TrimSpace(strings.Split(run(t, dir, 0, "agent", "spawn", "--grant", "ro"), "\n")[0])
	t.Setenv("DACLI_AGENT", tok)
	run(t, dir, 0, "note", "add", "finding", "Second-hand poison",
		"--project", "poison", "--about", "unrelated-poison-work",
		"--origin", "external:attacker", "--body", "from a hostile PR comment")
	t.Setenv("DACLI_AGENT", "")

	// Before sync it is a pending event; taint sees events too.
	if out := run(t, dir, 0, "taint", "external:attacker"); !strings.Contains(out, "event") {
		t.Errorf("tainted event not found pre-sync:\n%s", out)
	}
	run(t, dir, 0, "sync")
	// After sync the origin must have crossed the weld into the note.
	out = run(t, dir, 0, "taint", "external:attacker")
	if !strings.Contains(out, "note") || !strings.Contains(out, "origin=external:attacker") {
		t.Fatalf("provenance lost across event→note sync — the weld failed:\n%s", out)
	}

	// The catch-all: `file:` matches every file-origin artifact.
	if out := run(t, dir, 0, "taint", "file:"); !strings.Contains(out, "configs/evil.yml") {
		t.Errorf("prefix match failed:\n%s", out)
	}
}

// The opus reviewer's findings, as regression tests. A real spawned agent
// found these; they must not come back.
func TestTaintReviewerFindings(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "Home", "--slug", "home", "--goal", "g")
	run(t, dir, 0, "project", "add", "Other", "--slug", "other", "--goal", "g")
	run(t, dir, 0, "task", "add", "Home task", "--project", "home", "--accept", "a")
	run(t, dir, 0, "task", "add", "Other task", "--project", "other", "--accept", "a")

	// F1 (major): a workspace-scoped poisoned note reaches EVERY project's
	// briefs, so taint must report tree-wide — not blast radius 1.
	run(t, dir, 0, "note", "add", "finding", "Tree-wide poison lesson",
		"--project", "home", "--scope", "workspace",
		"--origin", "file:shared/poison.md", "--body", "reaches all projects")
	out := run(t, dir, 0, "taint", "file:shared/poison.md")
	if !strings.Contains(out, "TREE-WIDE") {
		t.Errorf("F1: workspace-scoped hit not reported tree-wide:\n%s", out)
	}
	if !strings.Contains(out, "other-task") {
		t.Errorf("F1: the other project's brief escaped the blast radius:\n%s", out)
	}

	// F3: case-insensitive match.
	if out := run(t, dir, 0, "taint", "FILE:Shared/POISON.md"); !strings.Contains(out, "TREE-WIDE") {
		t.Errorf("F3: case-sensitive match let a spelling evade:\n%s", out)
	}

	// F2: metric notes are scanned.
	run(t, dir, 0, "note", "add", "metric", "Poisoned metric", "--project", "home",
		"--origin", "file:evil.csv", "--body", "goal:x question:y metric:z")
	if out := run(t, dir, 0, "taint", "file:evil.csv"); !strings.Contains(out, "note") {
		t.Errorf("F2: metric note not scanned:\n%s", out)
	}

	// F4: output states it is a lower bound.
	if !strings.Contains(out, "LOWER BOUND") {
		t.Errorf("F4: blast radius not labeled a lower bound:\n%s", out)
	}
}
