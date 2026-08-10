// Package cli is the app layer of the feature-sliced design: it aggregates
// the feature slices' command tables, dispatches argv, and hosts the MCP
// front end's executor. It owns NO feature logic — a command body in this
// package is a layering bug.
//
// The FSD mapping for this repo (documented in ARCHITECTURE § 2b):
//
//	shared    ulid, mdstore, prompts, spm, shortcut, team, clikit
//	entities  model, workspace, store, eventlog, agentid, brief
//	features  internal/features/* — one slice per capability, and slices
//	          NEVER import each other (enforced by arch_test.go)
//	app       this package, and internal/mcp
package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/features/acceptance"
	"github.com/mlnomadpy/dacli/internal/features/briefing"
	"github.com/mlnomadpy/dacli/internal/features/catalog"
	"github.com/mlnomadpy/dacli/internal/features/collab"
	"github.com/mlnomadpy/dacli/internal/features/dashboard"
	"github.com/mlnomadpy/dacli/internal/features/execution"
	"github.com/mlnomadpy/dacli/internal/features/ghmirror"
	"github.com/mlnomadpy/dacli/internal/features/insight"
	"github.com/mlnomadpy/dacli/internal/features/knowledge"
	"github.com/mlnomadpy/dacli/internal/features/onboard"
	"github.com/mlnomadpy/dacli/internal/features/orchestration"
	"github.com/mlnomadpy/dacli/internal/features/planning"
	"github.com/mlnomadpy/dacli/internal/features/queues"
	"github.com/mlnomadpy/dacli/internal/features/selfreport"
	"github.com/mlnomadpy/dacli/internal/features/ship"
	"github.com/mlnomadpy/dacli/internal/features/shortcuts"
	"github.com/mlnomadpy/dacli/internal/features/skillforge"
	"github.com/mlnomadpy/dacli/internal/features/stagegate"
	"github.com/mlnomadpy/dacli/internal/features/teamops"
	"github.com/mlnomadpy/dacli/internal/features/vcs"
	"github.com/mlnomadpy/dacli/internal/features/wscore"
	"github.com/mlnomadpy/dacli/internal/mcp"
)

// Ctx and Command are re-exported from the kernel so tests and callers keep
// one import.
type (
	Ctx     = clikit.Ctx
	Command = clikit.Command
)

// commands is the whole surface: every slice's table plus the app layer's
// own (mcp serve, which needs the dispatch loop and so cannot live in a
// slice).
var commands = aggregate(
	wscore.Commands,
	onboard.Commands,
	planning.Commands,
	briefing.Commands,
	knowledge.Commands,
	collab.Commands,
	insight.Commands,
	teamops.Commands,
	shortcuts.Commands,
	queues.Commands,
	execution.Commands,
	stagegate.Commands,
	ghmirror.Commands,
	skillforge.Commands,
	vcs.Commands,
	selfreport.Commands,
	acceptance.Commands,
	ship.Commands,
	catalog.Commands,
	orchestration.Commands,
	dashboard.Commands,
	[]Command{
		{Path: "mcp serve", Brief: "Serve the workspace as MCP tools over stdio", Run: cmdMcpServe},
	},
)

func aggregate(tables ...[]Command) []Command {
	var out []Command
	for _, t := range tables {
		out = append(out, t...)
	}
	return out
}

// Test seams: the suite drives handlers through the same entry path users
// take, and needs the kernel's plumbing under the old package-local names.
var (
	openWorkspace = clikit.OpenWorkspace
	exitCode      = clikit.ExitCode
)

// Main dispatches argv and returns a process exit code.
func Main(argv []string) int {
	cwd, _ := os.Getwd()
	ctx := &Ctx{Stdout: os.Stdout, Stderr: os.Stderr, Cwd: cwd}

	args := make([]string, 0, len(argv))
	for _, a := range argv {
		if a == "--json" {
			ctx.JSON = true
			continue
		}
		args = append(args, a)
	}

	if len(args) == 0 {
		fmt.Fprint(ctx.Stdout, clikit.Banner())
		usage(ctx.Stdout)
		return 0
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		usage(ctx.Stdout)
		return 0
	}

	cmd, rest := match(args)
	if cmd == nil {
		fmt.Fprintf(ctx.Stderr, "dacli: unknown command %q\n\n", strings.Join(args, " "))
		usage(ctx.Stderr)
		return 2
	}

	if err := invoke(ctx, cmd, rest); err != nil {
		fmt.Fprintf(ctx.Stderr, "dacli: %v\n", err)
		// The exit-code contract (ARCHITECTURE § 4): 2 usage, 3 refused by
		// policy, 4 not found, 1 everything else. Agents branch on these
		// without parsing stderr — and must never retry a 3.
		return exitCode(err)
	}
	return 0
}

// invoke is the single path from a matched command to its handler: the
// dispatcher's gates, then the handler. Both front ends (argv and MCP) and the
// test suite go through it, so a gate cannot hold for users while being
// invisible to tests — which is exactly what happened before it existed: the
// suite called cmd.Run directly, so nothing covered Main's checks.
func invoke(ctx *Ctx, cmd *Command, rest []string) error {
	// --help BEFORE anything runs. It was not a flag anywhere: on a command
	// that rejects unknown flags it was a usage error, and on one that does
	// not it was silently dropped and THE COMMAND RAN — `dacli task claim 001
	// --help` claimed the task. Asking what a command does must never be the
	// thing that does it.
	if hasHelp(rest) {
		printCommandHelp(ctx, cmd)
		return nil
	}
	if err := refuseUnsupportedJSON(cmd, ctx.JSON); err != nil {
		return err
	}
	if err := refuseUngrantedMutation(ctx, cmd, rest); err != nil {
		return err
	}
	return cmd.Run(ctx, rest)
}

// refuseUnsupportedJSON returns a usage error (exit 2) when --json was
// requested for a command that does not honor it. This one dispatch-layer
// check is the whole fix for task 291: --json is parsed globally (Main strips
// it, the MCP front end passes it as jsonMode), so before this the ~110
// commands that never read ctx.JSON accepted the flag and silently emitted
// human prose — a machine caller got unparseable text under a flag that
// promised structure. Enforcing it here rather than per-command is deliberate:
// the same class of drift (a rule applied at some call sites and missed at
// others) is how Flags.Reject reached 4 handlers out of 112. A command now
// either declares Command.JSON and adapts, or the flag is refused loudly.
func refuseUnsupportedJSON(cmd *Command, jsonMode bool) error {
	if !jsonMode || cmd.JSON {
		return nil
	}
	return clikit.Usagef("%s does not support --json — machine-readable output is available from: %s",
		cmd.Path, strings.Join(jsonCmdList(), ", "))
}

// refuseUngrantedMutation enforces Command.Mutates at the dispatcher: a
// read-only agent invoking a state-changing command is refused (exit 3) before
// the handler runs.
//
// This is the grant-side twin of refuseUnsupportedJSON, and it exists because
// per-handler enforcement demonstrably does not hold. The 2026-08-06 audit
// found four live bypasses, each sitting beside a correctly-gated sibling:
// `shortcut promote` beside `shortcut add`, six `github` verbs beside `github
// release`, `agents --reap` beside `kill`, `worktree remove` beside `merge`.
// Every one of those is closed by the table declaring what it does.
//
// Two deliberate escapes. A --dry-run is a read (see Command.Mutates). And a
// workspace that cannot be opened yields no identity to judge, so the gate
// defers to the handler, which reports the real problem (no workspace, bad
// token) instead of a misleading grant refusal.
func refuseUngrantedMutation(ctx *Ctx, cmd *Command, args []string) error {
	if !cmd.Mutates || hasDryRun(args) {
		return nil
	}
	_, id, err := openWorkspace(ctx)
	if err != nil {
		//nolint:nilerr // deliberate: no workspace means no identity to judge, so
		// the gate defers to the handler, which reports the REAL problem (no
		// workspace, bad token) instead of a misleading grant refusal.
		return nil
	}
	return clikit.RequireRW(id, cmd.Path)
}

// hasDryRun reports whether --dry-run was passed. It scans argv directly
// rather than using ParseFlags because the dispatcher must not consume or
// validate a command's flags — that stays the handler's job.
func hasDryRun(args []string) bool {
	for _, a := range args {
		if a == "--dry-run" || a == "--dry-run=true" {
			return true
		}
	}
	return false
}

// hasHelp reports whether the caller asked for help rather than execution.
// Scanned from argv directly: the dispatcher must not consume or validate a
// command's flags, which stays the handler's job.
func hasHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// printCommandHelp prints what one command is for. The Brief is the whole
// documentation a command carries, so this is deliberately thin — the point is
// that asking is SAFE and answers something, not that it answers everything.
func printCommandHelp(ctx *Ctx, cmd *Command) {
	fmt.Fprintf(ctx.Stdout, "dacli %s\n\n%s\n", cmd.Path, cmd.Brief)
	if cmd.Usage != "" {
		fmt.Fprintf(ctx.Stdout, "\n%s\n", cmd.Usage)
	}
	if cmd.JSON {
		fmt.Fprintln(ctx.Stdout, "\nSupports --json.")
	}
	if cmd.Mutates {
		fmt.Fprintln(ctx.Stdout, "Needs an rw grant (a --dry-run, where supported, is a read).")
	}
}

// jsonCmdList indirects jsonCommands so that refuseUnsupportedJSON does not
// reference the `commands` table through the static initializer graph — the
// same cycle-breaking trick as dispatch (both are assigned in init below).
var jsonCmdList func() []string

// jsonCommands lists, sorted, the command paths that honor --json. It is built
// from the table so the refusal hint can never drift from the set of commands
// that actually implement the flag.
func jsonCommands() []string {
	var out []string
	for i := range commands {
		if commands[i].JSON {
			out = append(out, commands[i].Path)
		}
	}
	sort.Strings(out)
	return out
}

// match finds the longest command path first, so "task add" beats "task".
func match(args []string) (*Command, []string) {
	for n := 2; n >= 1; n-- {
		if len(args) < n {
			continue
		}
		path := strings.Join(args[:n], " ")
		for i := range commands {
			if commands[i].Path == path {
				return &commands[i], args[n:]
			}
		}
	}
	return nil, nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "dacli — context management for hierarchies of coding agents")
	fmt.Fprintln(w, "\nUsage: dacli <command> [args] [--json]")
	fmt.Fprintln(w, "\nCommands:")

	paths := make([]string, 0, len(commands))
	byPath := map[string]string{}
	width := 0
	for _, c := range commands {
		paths = append(paths, c.Path)
		byPath[c.Path] = c.Brief
		if len(c.Path) > width {
			width = len(c.Path)
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		fmt.Fprintf(w, "  %-*s  %s\n", width, p, byPath[p])
	}
	fmt.Fprintln(w, "\nEnvironment:")
	fmt.Fprintln(w, "  DACLI_AGENT  agent token; unset means the root agent")
}

// dispatch indirects the command lookup so that the commands table can
// reference cmdMcpServe without a static initialization cycle.
var dispatch func(args []string) (*Command, []string)

func init() {
	dispatch = match
	jsonCmdList = jsonCommands
}

// executor adapts the command table for the MCP server: same dispatch, same
// exit-code contract, buffered output. This closure is the entire coupling
// between the two front ends — mcp never imports cli.
func executor(cwd string) mcp.Executor {
	return func(argv []string, jsonMode bool) (string, string, int) {
		var out, errb bytes.Buffer
		c := &Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd, JSON: jsonMode}
		cmd, rest := dispatch(argv)
		if cmd == nil {
			return "", fmt.Sprintf("unknown command %q", strings.Join(argv, " ")), 2
		}
		// The MCP front end is a second door to the same table, so it goes
		// through the same gates — otherwise every bypass they close reopens
		// over MCP.
		err := invoke(c, cmd, rest)
		msg := errb.String()
		if err != nil {
			if msg != "" && !strings.HasSuffix(msg, "\n") {
				msg += "\n"
			}
			msg += err.Error()
		}
		return out.String(), msg, exitCode(err)
	}
}

func cmdMcpServe(ctx *Ctx, args []string) error {
	// This command takes no flags, so ANY flag is a typo. An empty allowlist
	// rejects every one — without it a mistyped flag was dropped and the
	// command ran as if nothing were wrong.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
	// Identity binds at launch from the environment; Serve fails fast on a
	// bad token rather than erroring on the tenth tool call.
	fmt.Fprintln(ctx.Stderr, "dacli mcp: serving on stdio (identity from DACLI_AGENT, root if unset)")
	return mcp.Serve(os.Stdin, ctx.Stdout, executor(ctx.Cwd))
}
