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

	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		usage(ctx.Stdout)
		return 0
	}

	cmd, rest := match(args)
	if cmd == nil {
		fmt.Fprintf(ctx.Stderr, "dacli: unknown command %q\n\n", strings.Join(args, " "))
		usage(ctx.Stderr)
		return 2
	}

	if err := cmd.Run(ctx, rest); err != nil {
		fmt.Fprintf(ctx.Stderr, "dacli: %v\n", err)
		// The exit-code contract (ARCHITECTURE § 4): 2 usage, 3 refused by
		// policy, 4 not found, 1 everything else. Agents branch on these
		// without parsing stderr — and must never retry a 3.
		return exitCode(err)
	}
	return 0
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

func init() { dispatch = match }

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
		err := cmd.Run(c, rest)
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
	// Identity binds at launch from the environment; Serve fails fast on a
	// bad token rather than erroring on the tenth tool call.
	fmt.Fprintln(ctx.Stderr, "dacli mcp: serving on stdio (identity from DACLI_AGENT, root if unset)")
	return mcp.Serve(os.Stdin, ctx.Stdout, executor(ctx.Cwd))
}
