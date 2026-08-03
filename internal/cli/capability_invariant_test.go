package cli

import (
	"strings"
	"testing"
)

// TestMutatingCommandsRefuseReadOnlyAgents is an INVARIANT test, not a feature
// test, and it exists because of how this codebase's capability bugs have
// actually been found.
//
// Every one of them was the same shape: a rule applied to some call sites and
// silently missed at others. `RequireRW` guarded `shortcut add` but not
// `skill add` — which is strictly more dangerous, since a skill body is
// compiled into every future agent's context. `SafeSegment` guarded project
// slugs but not role, queue, or skill names, so `skill add ../../../pwned`
// wrote outside the workspace. `Flags.Reject` reached 4 handlers out of 112.
// Each was fixed individually, and each time the NEXT audit found another
// instance, because per-command tests can only assert what someone remembered
// to write a test for.
//
// This table is the memory instead. Adding a mutating command without a
// capability check fails here, whether or not its author thought about it.
// When you add a command that executes code, writes to the workspace, or
// writes to a remote, add it here — a deliberate omission needs a comment
// saying why it is safe, not silence.
func TestMutatingCommandsRefuseReadOnlyAgents(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a real goal for the project")
	run(t, dir, 0, "task", "add", "a task to act on", "--project", "p", "--accept", "it works")

	// Mint a read-only child and act as it for the rest of the test.
	out := run(t, dir, 0, "agent", "spawn", "--role", "junior", "--grant", "ro")
	token := strings.TrimSpace(strings.Split(strings.TrimSpace(out), "\n")[0])
	if len(token) != 48 {
		t.Fatalf("spawn token = %q, want 48 hex chars", token)
	}
	t.Setenv("DACLI_AGENT", token)

	// Each entry must refuse a read-only caller with exit 3 (refused by
	// policy — a caller must never retry it).
	for _, tc := range []struct {
		name string
		why  string
		args []string
	}{
		{
			"shortcut add", "defines executable code a later `run` executes as the operator",
			[]string{"shortcut", "add", "s", "--command", "echo hi", "--effect", "read"},
		},
		{
			"skill add", "a skill body is compiled into every future agent's context",
			[]string{"skill", "add", "s", "--desc", "a trigger description"},
		},
		{
			"skill fetch", "clones a third-party repo into the skill library",
			[]string{"skill", "fetch", "owner/repo"},
		},
		{
			"runtime add", "names the binary and env every child in it executes with",
			[]string{"runtime", "add", "rt", "--binary", "/bin/sh"},
		},
		{
			"project add", "creates workspace state and a filesystem path",
			[]string{"project", "add", "Another", "--slug", "q"},
		},
		{
			"kill", "terminates process groups from an on-disk record",
			[]string{"kill", "--all"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, dir, 3, tc.args...)
			if !strings.Contains(strings.ToLower(got), "grant") {
				t.Errorf("%s must refuse a ro caller because it %s; refusal should mention the grant, got:\n%s",
					tc.name, tc.why, got)
			}
		})
	}
}

// TestUserSuppliedNamesCannotEscapeTheWorkspace is the path-traversal half of
// the same invariant. Every name below becomes a filesystem path; a value
// carrying `..` or a separator must be rejected rather than resolved. The
// guard (workspace.SafeSegment) existed for a while but was applied only to
// project slugs, so `skill add ../../../pwned` wrote outside `.dacli`
// entirely. New name→path sinks belong in this table.
func TestUserSuppliedNamesCannotEscapeTheWorkspace(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"project slug", []string{"project", "add", "P", "--slug", "../../escaped"}},
		{"skill name", []string{"skill", "add", "../../escaped", "--desc", "a trigger description"}},
		{"queue slug", []string{"queue", "add", "../../escaped", "--step", "do a thing"}},
		{"role name", []string{"role", "add", "../../escaped"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb strings.Builder
			ctx := &Ctx{Stdout: &out, Stderr: &errb, Cwd: dir}
			cmd, rest := match(tc.args)
			if cmd == nil {
				t.Fatalf("no such command: %v", tc.args)
			}
			// The contract is only that it must FAIL — usage (2) or a plain
			// error (1) are both acceptable answers to a traversing name.
			// What must never happen is success.
			if err := cmd.Run(ctx, rest); err == nil {
				t.Fatalf("%s: a traversing name was accepted; it must be rejected", tc.name)
			}
		})
	}
}
