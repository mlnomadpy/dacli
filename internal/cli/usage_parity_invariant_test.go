package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/mcp"
)

// handlerUsageVariants are intentional table extensions: a missing positional
// cannot exercise their optional flags or explanatory continuation. Keep each
// exception here rather than weakening the invariant for every command.
var handlerUsageVariants = map[string]string{
	"new":               "dacli new \"<product name>\" --goal \"<what it is>\" [--slug s] [--stack go|python|typescript|rust|auto] [--template solo|standard|product] [--out-of-scope x]... [--success \"criterion\"]... [--no-ci] [--gitignore-workspace]",
	"project add":       "dacli project add <title> [--slug s] [--goal g] [--stage definition|elicitation|approach|design]",
	"project show":      "dacli project show <slug>",
	"ask":               "dacli ask \"question\" --about <task-ref> [--need path]",
	"escalate":          "dacli escalate \"summary\" [--about task] [--github]",
	"role add":          "dacli role add <name> [--summary s] [--kind researcher|planner|designer|implementer|reviewer] [--skill s]... [--scope glob]... [--shortcut n]... [--escalate-to role]... [--grant ro|rw] [--wip N] [--runtime rt] [--model m] [--max-points N]",
	"shortcut add":      "dacli shortcut add <name> --command 'tmpl {{p}}' --effect read|write|destructive [--summary s] [--param name=default]... [--role r]... [--why text]",
	"runtime add":       "dacli runtime add <name> [--preset claude-code|claude-code-rw|codex|codex-rw|gemini|gemini-rw|copilot|copilot-rw|generic-exec] [--binary b] [--mode stdin|arg] [--flag -p] [--arg a]... [--sandbox-ro-arg a]... [--env NAME]... [--model-flag f]\n(--flag/--arg/--sandbox-ro-arg/--model-flag take their value verbatim, even one starting with -, e.g. --model-flag --model)",
	"verify":            "dacli verify --task <ref> --panel rt1,rt2[,rt3] [--claim text] [--require N] [--grant ro|rw] [--budget N] [--timeout sec] [--cooperative]",
	"template show":     "dacli template show <name>",
	"github sync":       "dacli github pull <project>",
	"github codeowners": "dacli github codeowners <project> | dacli github codeowners --owner <org>",
	"commit":            "dacli commit \"<message>\" [--task ref] [--no-add] [--force]",
	"report":            "dacli report \"<what went wrong>\" [--body detail] [--run <run-id>] [--repo owner/name] [--disclose]\n(files an issue on the dacli tool's own tracker — an explicit action, never automatic; --disclose opts in to attaching the workspace name + run transcript, withheld by default since the upstream is public)",
	"accept":            "dacli accept <ref> [--verify \"cmd\"] [--require-verify] [--force] | dacli accept --all [--verify \"cmd\"] [--require-verify] [--force]\n(--verify runs per task and its result is recorded on the task; --require-verify refuses to close anything unverified)",
}

// TestCommandUsageMatchesHandlerUsage keeps the command table — the shared
// CLI/MCP contract — from advertising a copied synopsis. A handler that
// reports a missing argument supplies the executable contract; commands whose
// omission is valid or whose error is not a synopsis are intentionally skipped.
func TestCommandUsageMatchesHandlerUsage(t *testing.T) {
	originalToken, hadToken := os.LookupEnv(agentid.EnvVar)
	if err := os.Unsetenv(agentid.EnvVar); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if hadToken {
			_ = os.Setenv(agentid.EnvVar, originalToken)
		}
	})
	dir := t.TempDir()
	gitInit(t, dir)
	run(t, dir, 0, "init", "--name", "usage-parity")
	mcpDescription := mcp.CommandDescription(commandUsages())

	checked := 0
	for i := range commands {
		cmd := &commands[i]
		help := run(t, dir, 0, append(strings.Fields(cmd.Path), "--help")...)
		if !strings.Contains(help, cmd.Usage) {
			t.Errorf("%s --help omitted declared Usage %q", cmd.Path, cmd.Usage)
		}
		if !strings.Contains(mcpDescription, cmd.Usage) {
			t.Errorf("MCP cli description omitted %s Usage %q", cmd.Path, cmd.Usage)
		}
		ctx := &Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: dir}
		err := cmd.Run(ctx, nil)
		if err == nil || exitCode(err) != 2 {
			continue
		}
		got, ok := strings.CutPrefix(err.Error(), "usage: ")
		if !ok {
			continue
		}
		checked++
		if got != cmd.Usage && handlerUsageVariants[cmd.Path] != got {
			t.Errorf("%s Usage = %q, handler missing-argument usage = %q", cmd.Path, cmd.Usage, got)
		}
	}
	if checked == 0 {
		t.Fatal("no command emitted a handler usage error — this invariant measured nothing")
	}
}
