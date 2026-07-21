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
	{"init", "Create a .dacli workspace (--template to seed a process)", cmdInit},

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

	{"context", "Assemble a scoped context brief for an agent (the main event)", cmdContext},
	{"status", "Tree-wide project state in one screen", cmdStatus},

	{"agent spawn", "Mint a child agent identity and print its token once", cmdAgentSpawn},
	{"agent tree", "Show agent lineage and write attribution", cmdAgentTree},
	{"agent retire", "Mark an agent retired, freeing its WIP slot", cmdAgentRetire},
	{"whoami", "Show the acting agent and its grant", cmdWhoami},

	// Team: roles, spawning, and escalation. See docs/TEAM.md.
	{"spawn", "Spawn a child into a role: identity, brief, skills, shortcuts", notImplemented},
	{"team", "Roster: roles, active agents, WIP headroom", cmdTeam},
	{"team route", "Who owns this path, and the chain to reach them", cmdTeamRoute},
	{"role add", "Define a role: skills, scope, shortcuts, escalation", cmdRoleAdd},
	{"role list", "List roles", cmdRoleList},
	// These were originally registered as "help ask" / "help answer" /
	// "help escalate" — all three were unreachable, because Main intercepts
	// args[0] == "help" for usage before dispatch. Renamed; a test now
	// guards the reserved word.
	{"ask", "Ask a blocking question; the asking task blocks until answered", cmdAsk},
	{"answer", "Answer a question; the answer becomes a durable note", cmdAnswer},
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
	{"run", "Expand and run a shortcut (--dry-run, --confirm, --list)", cmdRun},
	{"shortcut add", "Define a shortcut", cmdShortcutAdd},
	{"shortcut promote", "Turn a repeated ad-hoc command into a shortcut", notImplemented},

	// Skills: one canonical format, compiled per runtime. Spec only — see
	// docs/SKILLS.md.
	{"skill add", "Author a workspace skill", notImplemented},
	{"skill list", "Workspace skills with sizes and delivery floors", notImplemented},
	{"skill show", "One skill: body, resources, est. tokens", notImplemented},
	{"skill import", "Ingest a native skill tree losslessly", notImplemented},
	{"skill compile", "Materialize skills for a role on a runtime (--dry-run)", notImplemented},
	{"skill promote", "Owner-gated promotion of a lesson into a skill", notImplemented},

	{"project add", "Create a project", cmdProjectAdd},
	{"project list", "List projects", cmdProjectList},
	{"project show", "Show a project", cmdProjectShow},

	{"task add", "Create a task", cmdTaskAdd},
	{"task list", "List tasks, optionally by status", cmdTaskList},
	{"task show", "Show a task", cmdTaskShow},
	{"task claim", "Take ownership of a task", cmdTaskClaim},
	{"task check", "Check acceptance boxes (--n N or --all)", cmdTaskCheck},
	{"task done", "Move a task to done; verifies acceptance, refuses if unmet", cmdTaskDone},
	{"task block", "Mark a task blocked", cmdTaskBlock},

	{"note add", "Record a decision, finding, metric, or reference", cmdNoteAdd},
	{"glossary", "Show or edit the project term list", cmdGlossary},

	// The SPM layer. See docs/SPM.md for what each framework buys and which
	// ones deliberately do not port to agent work.
	{"lint", "Format, INVEST, requirements-quality, and ambiguity checks", cmdLint},
	{"estimate", "PERT three-point estimate widened by the Cone of Uncertainty", notImplemented},
	{"critical-path", "CPM: the zero-slack chain and per-task slack", notImplemented},
	{"next", "What to work on now: MoSCoW, then critical path (--parallel N)", cmdNext},
	{"wbs", "Work breakdown tree for a project", notImplemented},
	{"risk add", "Record a risk in the impact x likelihood matrix", cmdRiskAdd},
	{"risk list", "List risks by rank; rank 1 and 2 require an action plan", cmdRiskList},
	{"doctor", "Detect management anti-patterns in tasks, risks, and the log", cmdDoctor},
	{"burndown", "Points remaining against tokens spent", notImplemented},
	{"velocity", "Tasks completed per 100k tokens, trailing sessions", notImplemented},
	{"standup", "Per-agent roll-up: done, doing, impediments — derived, never filed", cmdStandup},
	{"retro", "Harvest a task/project: went well, didn't, improve", cmdRetro},

	{"queue add", "Create a queue of ordered steps", cmdQueueAdd},
	{"queue list", "List queues and their cursors", cmdQueueList},
	{"queue next", "Print the next step (dacli does not run it)", cmdQueueNext},
	{"queue advance", "Move the cursor past the current step (--fail halts)", cmdQueueAdvance},

	{"events tail", "Follow the append-only write log", cmdEventsTail},
	{"sync", "Apply pending child events to objects you own", cmdSync},

	{"mcp serve", "Serve the workspace as MCP tools over stdio", cmdMcpServe},
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
		// The exit-code contract (ARCHITECTURE § 4): 2 usage, 3 refused by
		// policy, 4 not found, 1 everything else. Agents branch on these
		// without parsing stderr — and must never retry a 3.
		return exitCode(err)
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
