// Package cli is the command-line front end.
//
// It holds no logic. Every command resolves the workspace and the acting
// agent, calls into internal/, and formats the result. The MCP server
// (internal/mcp) calls the same functions with no CLI involvement — which is
// only cheap because neither front end owns any behavior.
package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Command is one subcommand. Path is the space-separated invocation, e.g.
// "task add".
type Command struct {
	Path  string
	Brief string
	Run   func(ctx *Ctx, args []string) error
}

// Ctx carries the process-wide state a command needs: where to write, and
// whether the caller wants machine-readable output.
type Ctx struct {
	Stdout io.Writer
	Stderr io.Writer
	Cwd    string
	JSON   bool
}

var commands = []Command{
	{"init", "Create a .dacli workspace (--template to seed a process)", notImplemented},

	// Templates and stage gates. Spec only — see docs/TEMPLATES.md.
	{"template list", "Available project templates and their stated cost", notImplemented},
	{"template show", "Stages, required docs, and gates for a template", notImplemented},
	{"template add", "Vendor a template into this workspace for editing", notImplemented},
	{"stage", "Current stage and unmet exit conditions", notImplemented},
	{"stage advance", "Advance if the gate opens; else list what is missing", notImplemented},

	// GitHub projection. Spec only — see docs/GITHUB.md.
	{"github doctor", "Probe gh, auth, repo access, and Projects scope", notImplemented},
	{"github link", "Bind a project to a repo and a GitHub Project", notImplemented},
	{"github sync", "Sync with GitHub Issues and Projects (--dry-run)", notImplemented},
	{"github pull", "Inbound only: fetch remote changes as events", notImplemented},
	{"github push", "Outbound only: mirror local structure", notImplemented},

	{"context", "Assemble a scoped context brief for an agent (the main event)", notImplemented},
	{"status", "Tree-wide project state in one screen", notImplemented},

	{"agent spawn", "Mint a child agent identity and print its token once", notImplemented},
	{"agent tree", "Show agent lineage and write attribution", notImplemented},
	{"whoami", "Show the acting agent and its grant", notImplemented},

	// Team: roles, spawning, and escalation. See docs/TEAM.md.
	{"spawn", "Spawn a child into a role: identity, brief, skills, shortcuts", notImplemented},
	{"team", "Roster: roles, active agents, WIP headroom", notImplemented},
	{"team route", "Who owns this path, and the chain to reach them", notImplemented},
	{"role add", "Define a role: skills, scope, shortcuts, escalation", notImplemented},
	{"role list", "List roles", notImplemented},
	// These were originally registered as "help ask" / "help answer" /
	// "help escalate" — all three were unreachable, because Main intercepts
	// args[0] == "help" for usage before dispatch. Renamed; a test now
	// guards the reserved word.
	{"ask", "Ask a blocking question, routed to the role that owns it", notImplemented},
	{"answer", "Answer a request; the answer becomes a durable note", notImplemented},
	{"escalate", "Escalate out of the tree to a human (optionally a GitHub issue)", notImplemented},
	{"threads", "Open help requests and their answers", notImplemented},

	// Runtimes: driving coding-agent CLIs. Spec only — see docs/RUNTIMES.md.
	{"runtime list", "Configured runtimes and their probed capabilities", notImplemented},
	{"runtime doctor", "Probe installs and verify adapter assumptions", notImplemented},
	{"runtime add", "Add a coding-agent CLI adapter", notImplemented},
	{"supervise", "Run a task's spawn-evaluate-correct loop to completion or budget", notImplemented},
	{"verify", "Verification panel across multiple runtimes", notImplemented},
	{"runs list", "Recorded agent runs", notImplemented},
	{"runs show", "Invocation, transcript, usage, and outcome for one run", notImplemented},
	{"runs prune", "Bound transcript growth", notImplemented},

	// Shortcuts. See docs/SHORTCUTS.md.
	{"run", "Expand and run a shortcut (--dry-run to print only)", notImplemented},
	{"shortcut add", "Define a shortcut", notImplemented},
	{"shortcut promote", "Turn a repeated ad-hoc command into a shortcut", notImplemented},

	// Skills: one canonical format, compiled per runtime. Spec only — see
	// docs/SKILLS.md.
	{"skill add", "Author a workspace skill", notImplemented},
	{"skill list", "Workspace skills with sizes and delivery floors", notImplemented},
	{"skill show", "One skill: body, resources, est. tokens", notImplemented},
	{"skill import", "Ingest a native skill tree losslessly", notImplemented},
	{"skill compile", "Materialize skills for a role on a runtime (--dry-run)", notImplemented},
	{"skill promote", "Owner-gated promotion of a lesson into a skill", notImplemented},

	{"project add", "Create a project", notImplemented},
	{"project list", "List projects", notImplemented},
	{"project show", "Show a project", notImplemented},

	{"task add", "Create a task", notImplemented},
	{"task list", "List tasks, optionally by status", notImplemented},
	{"task show", "Show a task", notImplemented},
	{"task claim", "Take ownership of a task", notImplemented},
	{"task done", "Move a task to done", notImplemented},
	{"task block", "Mark a task blocked", notImplemented},

	{"note add", "Record a decision, finding, metric, or reference", notImplemented},
	{"glossary", "Show or edit the project term list", notImplemented},

	// The SPM layer. See docs/SPM.md for what each framework buys and which
	// ones deliberately do not port to agent work.
	{"lint", "Format, INVEST, requirements-quality, and ambiguity checks", notImplemented},
	{"estimate", "PERT three-point estimate widened by the Cone of Uncertainty", notImplemented},
	{"critical-path", "CPM: the zero-slack chain and per-task slack", notImplemented},
	{"next", "What to work on now: MoSCoW, then risk-value, then critical path", notImplemented},
	{"wbs", "Work breakdown tree for a project", notImplemented},
	{"risk add", "Record a risk in the impact x likelihood matrix", notImplemented},
	{"risk list", "List risks by rank; rank 1 and 2 require an action plan", notImplemented},
	{"doctor", "Detect management anti-patterns in the event log", notImplemented},
	{"burndown", "Points remaining against tokens spent", notImplemented},
	{"velocity", "Tasks completed per 100k tokens, trailing sessions", notImplemented},
	{"standup", "Per-agent roll-up: done, next, impediments", notImplemented},
	{"retro", "Harvest findings: went well, did not, improve", notImplemented},

	{"queue add", "Create a queue of ordered steps", notImplemented},
	{"queue list", "List queues and their cursors", notImplemented},
	{"queue next", "Print the next step (dacli does not run it)", notImplemented},
	{"queue advance", "Move the cursor past the current step", notImplemented},

	{"events tail", "Follow the append-only write log", notImplemented},
	{"sync", "Apply pending child events to objects you own", notImplemented},

	{"mcp serve", "Serve the same operations as MCP tools over stdio", notImplemented},
}

func notImplemented(ctx *Ctx, args []string) error {
	return fmt.Errorf("not implemented — this is a skeleton; see DESIGN.md")
}

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

	// Match the longest command path first, so "task add" beats "task".
	cmd, rest := match(args)
	if cmd == nil {
		fmt.Fprintf(ctx.Stderr, "dacli: unknown command %q\n\n", strings.Join(args, " "))
		usage(ctx.Stderr)
		return 2
	}

	if err := cmd.Run(ctx, rest); err != nil {
		fmt.Fprintf(ctx.Stderr, "dacli: %v\n", err)
		return 1
	}
	return 0
}

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
