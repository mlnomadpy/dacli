package cli

import (
	"strings"
	"testing"
)

// TestGatedHandlersRejectUnknownFlags locks dacli 175: command handlers across
// the feature slices now reject typo'd/unknown flags with a usage error
// (exit 2) that names the offender, instead of ParseFlags silently dropping
// them and the command returning exit 0 with the caller's intent lost. The
// dacli 143 fix covered `task add`; this widens the guarantee to the rest of
// the surface, spot-checking one handler per representative slice.
func TestGatedHandlersRejectUnknownFlags(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")

	for _, tc := range []struct {
		name string
		args []string
		bad  string
	}{
		{"task add", []string{"task", "add", "t", "--project", "p", "--acccept", "y"}, "acccept"},
		{"next", []string{"next", "--projectt", "p"}, "projectt"},
		{"context", []string{"context", "--budgett", "5"}, "budgett"},
	} {
		out := run(t, dir, 2, tc.args...)
		if !strings.Contains(out, "unknown flag") || !strings.Contains(out, tc.bad) {
			t.Errorf("%s: want a usage error naming --%s, got:\n%s", tc.name, tc.bad, out)
		}
	}

	// A correctly-spelled flag must still be accepted — Reject narrows the
	// allowlist to the handler's real flags, it does not reject everything.
	run(t, dir, 0, "task", "add", "a well-formed task", "--project", "p", "--accept", "criterion")
}
