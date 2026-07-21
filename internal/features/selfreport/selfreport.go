// Package selfreport is the upstream-bug channel: when an agent hits a
// problem in dacli ITSELF (not the user's project), `dacli report` files it
// as an issue on the tool's own repo, with version and environment context.
// It is an EXPLICIT command — never automatic — so dacli never surprises a
// user by opening outbound issues on its own.
package selfreport

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mlnomadpy/dacli/internal/buildinfo"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "report", Brief: "File a dacli-tool bug upstream via gh (explicit; never automatic)", Run: cmdReport},
	{Path: "version", Brief: "Print the dacli version", Run: cmdVersion},
}

func cmdVersion(ctx *clikit.Ctx, args []string) error {
	fmt.Fprintf(ctx.Stdout, "dacli %s (%s/%s)\n", buildinfo.Version, runtime.GOOS, runtime.GOARCH)
	return nil
}

func cmdReport(ctx *clikit.Ctx, args []string) error {
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli report \"<what went wrong>\" [--body detail] [--run <run-id>] [--repo owner/name]\n(files an issue on the dacli tool's own tracker — an explicit action, never automatic)")
	}
	title := "[agent-report] " + strings.Join(f.Pos, " ")

	// The target is the TOOL's repo, not the user's project. Overridable, so
	// a fork can point its telemetry at its own tracker.
	repo := f.Get("repo")
	if repo == "" {
		repo = os.Getenv("DACLI_REPORT_REPO")
	}
	if repo == "" {
		repo = buildinfo.Repo
	}

	// The context that makes a bug report actionable — gathered by dacli so
	// the agent does not have to know how.
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", f.Get("body"))
	fmt.Fprintf(&body, "---\n_Reported via `dacli report`._\n")
	fmt.Fprintf(&body, "- dacli: %s\n- platform: %s/%s\n", buildinfo.Version, runtime.GOOS, runtime.GOARCH)
	if w, _, err := clikit.OpenWorkspace(ctx); err == nil {
		fmt.Fprintf(&body, "- workspace: %s\n", w.Name)
		// A run transcript excerpt, when the failure came from a spawned run.
		if runID := f.Get("run"); runID != "" {
			if excerpt := runExcerpt(w, runID); excerpt != "" {
				fmt.Fprintf(&body, "\n<details><summary>run %s (tail)</summary>\n\n```\n%s\n```\n</details>\n", runID, excerpt)
			}
		}
	}

	if f.Bool("dry-run") {
		// --dry-run touches no network: it shows exactly what would be filed.
		fmt.Fprintf(ctx.Stdout, "would file to %s:\ntitle: %s\n\n%s", repo, title, body.String())
		return nil
	}

	// Real filing needs gh, and it is an outward-facing action — so the
	// checks live here, not on the dry-run path.
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh not on PATH — dacli report files via the GitHub CLI")
	}
	if out, err := exec.Command("gh", "auth", "status").CombinedOutput(); err != nil {
		return fmt.Errorf("gh is not authenticated: %s", strings.TrimSpace(string(out)))
	}

	out, err := exec.Command("gh", "issue", "create", "--repo", repo,
		"--title", title, "--body", body.String()).Output()
	if err != nil {
		return fmt.Errorf("gh issue create failed: %v", err)
	}
	fmt.Fprintf(ctx.Stdout, "reported to %s: %s", repo, string(out))
	return nil
}

func runExcerpt(w *workspace.Workspace, runID string) string {
	raw, err := os.ReadFile(w.RunDir(runID) + "/transcript.log")
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) > 30 {
		lines = lines[len(lines)-30:]
	}
	return strings.Join(lines, "\n")
}
