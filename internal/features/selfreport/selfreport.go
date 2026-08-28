// Package selfreport is the upstream-bug channel: when an agent hits a
// problem in dacli ITSELF (not the user's project), `dacli report` files it
// as an issue on the tool's own repo, with version and environment context.
// It is an EXPLICIT command — never automatic — so dacli never surprises a
// user by opening outbound issues on its own.
package selfreport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/buildinfo"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "report", Brief: "File a dacli-tool bug upstream via gh (explicit; never automatic)", Mutates: true, Usage: "dacli report \\", Run: cmdReport},
	{Path: "version", Brief: "Print binary identity or diagnose installed skill compatibility", JSON: true, Usage: "dacli version [--compatibility [requirements.json]] [--json]", Run: cmdVersion},
}

// Compatibility is supplied by the app layer, which alone can see the fully
// aggregated command and MCP registries without introducing a feature cycle.
var Compatibility func(ctx *clikit.Ctx, requirementsPath string) error

func cmdVersion(ctx *clikit.Ctx, args []string) error {
	// Compatibility is the only command-specific flag. Keep the allowlist
	// closed so a typo cannot turn a requested diagnosis into a plain version
	// report that looks successful.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject("compatibility"); err != nil {
		return err
	} else if _, ok := f.Alias("compatibility"); ok {
		if len(f.Pos) > 0 {
			return clikit.Usagef("version --compatibility accepts at most one requirement path")
		}
		if Compatibility == nil {
			return fmt.Errorf("compatibility registry is unavailable")
		}
		return Compatibility(ctx, f.Get("compatibility"))
	} else if len(f.Pos) > 0 {
		return clikit.Usagef("version takes no positional arguments")
	}
	if ctx.JSON {
		path, _ := os.Executable()
		return json.NewEncoder(ctx.Stdout).Encode(map[string]string{"name": "dacli", "version": buildinfo.Version, "path": filepath.Clean(path), "goos": runtime.GOOS, "goarch": runtime.GOARCH})
	}
	fmt.Fprint(ctx.Stdout, clikit.Banner())
	fmt.Fprintf(ctx.Stdout, "dacli %s (%s/%s)\n", buildinfo.Version, runtime.GOOS, runtime.GOARCH)
	return nil
}

func cmdReport(ctx *clikit.Ctx, args []string) error {
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("body", "run", "repo", "disclose", "dry-run"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli report \"<what went wrong>\" [--body detail] [--run <run-id>] [--repo owner/name] [--disclose]\n(files an issue on the dacli tool's own tracker — an explicit action, never automatic; --disclose opts in to attaching the workspace name + run transcript, withheld by default since the upstream is public)")
	}
	title := "[agent-report] " + strings.Join(f.Pos, " ")

	// A read-only agent may file a plain report to the tool's own tracker — that
	// is the feature. But the two abusable levers are gated on rw: an explicit
	// --repo is an exfiltration channel (a ro child could redirect the issue,
	// with its body, to any repo the operator's gh token can write), and
	// --disclose attaches workspace internals to a public upstream. Neither
	// belongs on the read-only surface.
	// Fail CLOSED: if the identity cannot be resolved (run outside a
	// workspace), the privileged levers are refused rather than skipped —
	// gating them only when a workspace happened to resolve left the
	// exfiltration channel wide open from any other directory (dacli 198).
	if f.Get("repo") != "" || f.Bool("disclose") {
		_, id, oerr := clikit.OpenWorkspace(ctx)
		if oerr != nil {
			return clikit.Refusedf("--repo/--disclose need a resolvable workspace identity to authorize; run inside a dacli workspace (plain `dacli report` works anywhere)")
		}
		if err := clikit.RequireRW(id, "reporting with --repo or --disclose"); err != nil {
			return err
		}
	}

	// The target is the TOOL's repo, not the user's project. Overridable, so
	// a fork can point its telemetry at its own tracker.
	repo := f.Get("repo")
	if repo == "" {
		repo = os.Getenv("DACLI_REPORT_REPO")
	}
	if repo == "" {
		repo = buildinfo.Repo
	}

	// The report is filed on the TOOL's own tracker, which is a PUBLIC repo by
	// default (mlnomadpy/dacli). The workspace NAME and a RAW transcript tail are
	// internal artifacts — a disclosure the same way a ghmirror push to a public
	// repo is — so they ride a gate: attached only when the operator opts in with
	// --disclose. Ungated, the report still carries the version/platform context
	// that makes a tool bug actionable, but withholds anything project-internal.
	disclose := f.Bool("disclose")

	// The context that makes a bug report actionable — gathered by dacli so
	// the agent does not have to know how.
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", f.Get("body"))
	fmt.Fprintf(&body, "---\n_Reported via `dacli report`._\n")
	fmt.Fprintf(&body, "- dacli: %s\n- platform: %s/%s\n", buildinfo.Version, runtime.GOOS, runtime.GOARCH)
	if w, _, err := clikit.OpenWorkspace(ctx); err == nil {
		if disclose {
			fmt.Fprintf(&body, "- workspace: %s\n", w.Name)
			// A run transcript excerpt, when the failure came from a spawned run.
			if runID := f.Get("run"); runID != "" {
				if excerpt := runExcerpt(w, runID); excerpt != "" {
					fmt.Fprintf(&body, "\n<details><summary>run %s (tail)</summary>\n\n```\n%s\n```\n</details>\n", runID, excerpt)
				}
			}
		} else {
			fmt.Fprintf(&body, "- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them\n")
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
	if _, err := ghOutput(ctx.Cwd, "auth", "status"); err != nil {
		return fmt.Errorf("gh is not authenticated: %w", err)
	}

	out, err := ghOutput(ctx.Cwd, "issue", "create", "--repo", repo,
		"--title", title, "--body", body.String())
	if err != nil {
		return fmt.Errorf("gh issue create failed: %w (%s)", err, out)
	}
	fmt.Fprintf(ctx.Stdout, "reported to %s: %s", repo, out)
	return nil
}

// ghOutput runs the GitHub CLI under a deadline. gh is network- and auth-bound
// (a dead network, an interactive credential prompt), so a bare exec.Command
// could hang indefinitely — and under `dacli mcp serve` a hung `dacli report`
// would block the entire stdio loop. Mirrors ghmirror's timeout wrapper.
func ghOutput(cwd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = cwd
	operation := "gh"
	if len(args) > 0 {
		operation += " " + args[0]
	}
	out, err := commandresult.Run(cmd, commandresult.RunOptions{
		Operation:     operation,
		WorkspaceRoot: cwd,
		TimedOut: func() bool {
			return ctx.Err() == context.DeadlineExceeded
		},
	})
	return strings.TrimSpace(string(out)), err
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
