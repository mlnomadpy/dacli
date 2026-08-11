// `github sync` runs pull then push over ONE argument list, so every flag
// either half accepts is seen by both. pull's unknown-flag guard listed only
// --dry-run, so `github sync <proj> --since 2h` — a documented push window —
// exited 2 at the inbound half and the outbound half never ran. The code
// carried a comment claiming pull was deliberately ungated for exactly this
// reason; the gate had been added above it and the comment left behind, which
// is why reading it was not enough to notice.
package ghmirror

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// syncEnv is a linked project plus a STUBBED gh. These tests are about which
// flags survive argument parsing, which happens before any remote call — and
// leaving gh real would put a network round-trip inside a unit test and make
// the result depend on whether the machine is authenticated.
func syncEnv(t *testing.T) *clikit.Ctx {
	t.Helper()
	w := mirrorWorkspace(t) // already carries project "core"
	linkRepo(t, w, "core", "owner/repo")
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) > 1 && args[0] == "issue" && args[1] == "list" {
			return "[]", nil // no inbound issues to adopt
		}
		return "", nil
	}
	ctx, _ := releaseCtx(t, w)
	return ctx
}

// TestSyncAcceptsPushOnlyFlags is the regression: --since is push's, and sync
// must reach push with it rather than being refused by pull.
func TestSyncAcceptsPushOnlyFlags(t *testing.T) {
	for _, flag := range []string{"--since", "--findings-as-issues", "--with-tasks"} {
		args := []string{"core", flag}
		if flag == "--since" {
			args = append(args, "2h")
		}
		err := cmdSync(syncEnv(t), args)
		if err != nil && clikit.ExitCode(err) == 2 {
			t.Fatalf("github sync must not refuse push's own flag %s as a usage error: %v", flag, err)
		}
	}
}

// TestSyncStillRefusesATypo is the other half. Widening pull's allowlist to
// every flag would have made sync accept anything, turning a loud exit-2 into
// a silently ignored argument — the failure mode the Reject guard exists to
// prevent.
func TestSyncStillRefusesATypo(t *testing.T) {
	err := cmdSync(syncEnv(t), []string{"core", "--sinse", "2h"})
	if err == nil {
		t.Fatal("github sync accepted a misspelled flag")
	}
	if clikit.ExitCode(err) != 2 {
		t.Fatalf("a misspelled flag must be a usage error (exit 2), got exit %d: %v", clikit.ExitCode(err), err)
	}
	if !strings.Contains(err.Error(), "sinse") {
		t.Fatalf("the refusal must name the flag the caller typed, got: %v", err)
	}
}

// TestPullAloneStillRefusesPushOnlyFlags: on the direct path the flag is a
// real mistake — pull has no window to apply, so accepting --since would let
// the caller believe one had been. The tolerance belongs to sync, not to pull.
func TestPullAloneStillRefusesPushOnlyFlags(t *testing.T) {
	err := cmdPull(syncEnv(t), []string{"core", "--since", "2h"})
	if err == nil {
		t.Fatal("github pull accepted --since, which it cannot honour")
	}
	if clikit.ExitCode(err) != 2 {
		t.Fatalf("expected a usage error (exit 2), got exit %d: %v", clikit.ExitCode(err), err)
	}
	if !strings.Contains(err.Error(), "since") {
		t.Fatalf("the refusal must name the flag, got: %v", err)
	}
}
