package cli

import (
	"os/exec"
	"strings"
	"testing"
)

// forwardsUnknownFlags names the commands that legitimately accept flags this
// table cannot enumerate, because forwarding them IS the feature.
var forwardsUnknownFlags = map[string]bool{
	// `run` passes unknown flags to the shortcut as parameters.
	"run": true,
	// pull/sync forward their flags to the push/pull halves they drive.
	"github pull": true,
	"github sync": true,
}

// TestEveryCommandRejectsAnUnknownFlag drives the WHOLE surface with a flag
// that exists nowhere and asserts it is refused.
//
// This is the drift guard the flag kernel never had. `Flags.Reject` is opt-in
// per handler, and opt-in guards in this codebase drift without exception: it
// reached 4 handlers out of 112 once, was fixed, and by the 2026-08-06 audit
// had drifted back to ~25 handlers with none — including mutating ones. The
// live demo was `dacli task block 001 --whyy "the real reason"`, which exited
// 0, blocked the task, and recorded an EMPTY reason. The caller's intent was
// lost with a success exit.
//
// A per-handler test can only assert what someone remembered to write. This
// asserts it for every command that exists, so a new handler that forgets
// Reject fails here rather than in someone's workspace.
func TestEveryCommandRejectsAnUnknownFlag(t *testing.T) {
	for i := range commands {
		cmd := &commands[i]
		if forwardsUnknownFlags[cmd.Path] {
			continue
		}
		t.Run(cmd.Path, func(t *testing.T) {
			dir := t.TempDir()
			// A real repo: `github doctor` and `worktree list` shell to git
			// and would otherwise fail on the missing repository BEFORE
			// reaching flag validation, which would let them pass this test
			// without ever rejecting anything.
			gitInit(t, dir)
			run(t, dir, 0, "init", "--name", "x")

			var out, errb strings.Builder
			ctx := &Ctx{Stdout: &out, Stderr: &errb, Cwd: dir}
			err := invoke(ctx, cmd, []string{"--zz-no-such-flag", "value"})
			if err == nil {
				t.Fatalf("%s accepted --zz-no-such-flag and ran anyway; a typo'd flag must never be silently dropped", cmd.Path)
			}
			// The error must NAME the flag. Accepting any error would let a
			// command pass by failing on a missing positional first, which is
			// exactly the false confidence this test exists to remove: the
			// flag would still be dropped once the positional was supplied.
			if !strings.Contains(err.Error(), "zz-no-such-flag") {
				t.Fatalf("%s failed for some other reason (%v) — it never rejected the unknown flag itself", cmd.Path, err)
			}
			if code := exitCode(err); code != 2 {
				t.Errorf("%s: exit %d for an unknown flag, want 2 (usage)", cmd.Path, code)
			}
		})
	}
}

// Asking what a command does must never be the thing that does it. Before the
// dispatcher handled it, `--help` was silently dropped by any handler without
// Reject and the command RAN: `dacli task claim 001 --help` claimed the task.
func TestHelpNeverExecutesTheCommand(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a real goal for the project")
	run(t, dir, 0, "task", "add", "a task", "--project", "p", "--accept", "it works")

	out := run(t, dir, 0, "task", "claim", "001", "--help")
	if !strings.Contains(out, "task claim") {
		t.Errorf("--help must describe the command, got:\n%s", out)
	}
	// The task must be untouched: help is a read.
	if got := run(t, dir, 0, "task", "show", "001"); strings.Contains(got, "claimed") {
		t.Errorf("--help claimed the task instead of describing it:\n%s", got)
	}

	// -h is the same promise.
	if out := run(t, dir, 0, "kill", "-h"); !strings.Contains(out, "kill") {
		t.Errorf("-h must describe the command, got:\n%s", out)
	}
}

// gitInit makes dir a git repository with one commit, so commands that shell
// to git get far enough for their flag handling to be what fails.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "x@x"},
		{"config", "user.name", "x"},
		{"commit", "-q", "--allow-empty", "-m", "base"},
	} {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}
