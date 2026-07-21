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
	{"template list", "Available project templates and their stated cost", planned("template manifests and gate predicates", "docs/TEMPLATES.md")},
	{"template show", "Stages, required docs, and gates for a template", planned("template manifests and gate predicates", "docs/TEMPLATES.md")},
	{"template add", "Vendor a template into this workspace for editing", planned("template manifests and gate predicates", "docs/TEMPLATES.md")},
	{"stage", "Current stage and unmet exit conditions", planned("gate evaluation (filled-not-present checks)", "docs/TEMPLATES.md § 5")},
	{"stage advance", "Advance if the gate opens; else list what is missing", planned("gate evaluation (filled-not-present checks)", "docs/TEMPLATES.md § 5")},

	// GitHub projection. Spec only — see docs/GITHUB.md.
	{"github doctor", "Probe gh, auth, repo access, and Projects scope", planned("the issue/project mirror", "docs/GITHUB.md")},
	{"github link", "Bind a project to a repo and a GitHub Project", planned("the issue/project mirror", "docs/GITHUB.md")},
	{"github sync", "Sync with GitHub Issues and Projects (--dry-run)", planned("the issue/project mirror with marker-based idempotency", "docs/GITHUB.md § 4")},
	{"github pull", "Inbound only: fetch remote changes as events", planned("inbound humans-as-events", "docs/GITHUB.md § 3")},
	{"github push", "Outbound only: mirror local structure", planned("the issue/project mirror", "docs/GITHUB.md")},

	{"context", "Assemble a scoped context brief for an agent (the main event)", cmdContext},
	{"status", "Tree-wide project state in one screen", cmdStatus},

	{"agent spawn", "Mint a child agent identity and print its token once", cmdAgentSpawn},
	{"agent tree", "Show agent lineage and write attribution", cmdAgentTree},
	{"agent retire", "Mark an agent retired, freeing its WIP slot", cmdAgentRetire},
	{"whoami", "Show the acting agent and its grant", cmdWhoami},

	// Team: roles, spawning, and escalation. See docs/TEAM.md.
	{"spawn", "Launch a child agent on a runtime: identity, brief, sandbox, run record", cmdSpawn},
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
	{"escalate", "Escalate out of the tree to a human (--github files an issue)", cmdEscalate},
	{"threads", "Questions and their answers, open first", cmdThreads},

	// Runtimes: driving coding-agent CLIs. Spec only — see docs/RUNTIMES.md.
	{"runtime list", "Configured runtimes and their declared capabilities", cmdRuntimeList},
	{"runtime doctor", "Probe installs: binary, version; declared-vs-probed kept distinct", cmdRuntimeDoctor},
	{"runtime add", "Add a coding-agent CLI adapter (--preset claude-code|generic-exec)", cmdRuntimeAdd},
	{"supervise", "Spawn-evaluate-correct loop until accepted or --max-turns", cmdSupervise},
	{"verify", "Verification panel across multiple runtimes", planned("multi-runtime verdict panels need a second runtime worth polling", "docs/RUNTIMES.md § 10")},
	{"runs list", "Recorded agent runs, newest first", cmdRunsList},
	{"runs show", "Invocation, outcome, brief, and transcript for one run", cmdRunsShow},
	{"runs prune", "Bound transcript growth (--keep N, default 20)", cmdRunsPrune},

	// Shortcuts. See docs/SHORTCUTS.md.
	{"run", "Expand and run a shortcut (--dry-run, --confirm, --list)", cmdRun},
	{"shortcut add", "Define a shortcut", cmdShortcutAdd},
	{"shortcut promote", "Turn a repeated ad-hoc command into a shortcut", planned("ad-hoc command tracking — dacli only sees shortcut runs today, so there is nothing un-promoted to promote from", "docs/SHORTCUTS.md § promotion")},

	// Skills: one canonical format, compiled per runtime. Spec only — see
	// docs/SKILLS.md.
	{"skill add", "Author a workspace skill", planned("skill compilation", "docs/SKILLS.md")},
	{"skill list", "Workspace skills with sizes and delivery floors", planned("skill compilation", "docs/SKILLS.md")},
	{"skill show", "One skill: body, resources, est. tokens", planned("skill compilation", "docs/SKILLS.md")},
	{"skill import", "Ingest a native skill tree losslessly", planned("skill compilation", "docs/SKILLS.md")},
	{"skill compile", "Materialize skills for a role on a runtime (--dry-run)", planned("the fidelity ladder (native/context/inline)", "docs/SKILLS.md § 3")},
	{"skill promote", "Owner-gated promotion of a lesson into a skill", planned("lessons (PROPOSALS P1) landing first — nothing to promote yet", "docs/SKILLS.md § 6")},

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
	{"estimate", "PERT three-point estimate widened by the Cone of Uncertainty", cmdEstimate},
	{"critical-path", "CPM: full schedule with slack; star marks the critical path", cmdCriticalPath},
	{"next", "What to work on now: MoSCoW, then critical path (--parallel N)", cmdNext},
	{"wbs", "Work breakdown tree (task add --parent builds it)", cmdWBS},
	{"risk add", "Record a risk in the impact x likelihood matrix", cmdRiskAdd},
	{"risk list", "List risks by rank; rank 1 and 2 require an action plan", cmdRiskList},
	{"doctor", "Detect management anti-patterns in tasks, risks, and the log", cmdDoctor},
	{"burndown", "Points remaining vs done, per-day completions", cmdBurndown},
	{"velocity", "Completions per active day (time proxy until usage reporting)", cmdVelocity},
	{"standup", "Per-agent roll-up: done, doing, impediments — derived, never filed", cmdStandup},
	{"retro", "Harvest a task/project: went well, didn't, improve", cmdRetro},

	{"queue add", "Create a queue of ordered steps", cmdQueueAdd},
	{"queue list", "List queues and their cursors", cmdQueueList},
	{"queue next", "Print the next step (dacli does not run it)", cmdQueueNext},
	{"queue advance", "Move the cursor past the current step (--fail halts)", cmdQueueAdvance},

	{"events tail", "Follow the append-only write log", cmdEventsTail},
	{"prompt list", "The prompt registry; overrides marked", cmdPromptList},
	{"prompt show", "One prompt's resolved template", cmdPromptShow},
	{"sync", "Apply pending child events to objects you own", cmdSync},

	{"mcp serve", "Serve the workspace as MCP tools over stdio", cmdMcpServe},
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
