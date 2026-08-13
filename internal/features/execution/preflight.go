// dacli 272: one preflight for whether a role can actually do what its
// prompt asks. Three checks already existed as separate, uncoordinated
// signals — sandboxFor's grant-vs-write-capability refusal (dacli 250),
// warnExeAllowlist's binary-path-vs-allowlist warning (dacli 267) — plus a
// third that did not exist yet: whether the tools a role's prompt names are
// tools the runtime's --allowedTools allowlist actually grants. Run
// separately, the first one refusing hid whatever the other two would have
// said. preflightIssues runs all three every time and returns every
// mismatch, so a caller can report them together before deciding whether to
// refuse or warn per class.
package execution

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

func init() {
	Commands = append(Commands, clikit.Command{
		Path: "preflight", Brief: "Check whether a role/runtime/grant can do what the role's prompt asks: grant vs write capability, binary path vs allowlist, prompt's named tools vs runtime — every mismatch, one pass", Usage: "dacli preflight --role <name> [--runtime name] [--grant ro|rw] [--cooperative]", Run: cmdPreflight,
	})
}

// preflightIssue is one mismatch between what a role/runtime/grant claims and
// what it can actually deliver. refuse follows each check's own existing
// convention (grant-write refuses per dacli 250; binary-allowlist and
// prompt-tools warn — both are signals from a partial view, not certainties,
// the same reasoning dacli 267's decision note gives for binary-allowlist).
type preflightIssue struct {
	class   string
	refuse  bool
	message string
}

// sandboxArgsFor derives the sandbox args a launch with this grant would
// apply, without judging whether that is honest — sandboxFor still owns that
// judgment and its refusal. Needed here only to build the effective
// --allowedTools view the binary-allowlist and prompt-tools checks read.
func sandboxArgsFor(rt store.Runtime, grant model.Grant) []string {
	if grant == model.GrantRO && store.RuntimeEnforcesRO(rt) {
		return rt.SandboxRO
	}
	return nil
}

// preflightIssues runs dacli 272's three checks against rt/role/grant and
// returns every mismatch found — never stopping at the first, so a
// grant-write refusal never hides a binary-allowlist or prompt-tools warning
// a caller would otherwise never see. exe is the dacli binary path the child
// preamble will name (os.Executable(), or "" to skip the binary-allowlist
// check when that call failed).
func preflightIssues(rt store.Runtime, role team.Role, hasRole bool, grant model.Grant, cooperative bool, exe string) []preflightIssue {
	var issues []preflightIssue

	// Class 1: grant vs runtime write capability (dacli 250's rw refusal;
	// the ro-enforcement half of § 8 stays a sandboxFor-only concern — it is
	// not one of the three classes this task names).
	if grant != model.GrantRO && !store.RuntimeWritable(rt) {
		issues = append(issues, preflightIssue{
			class:  "grant-write",
			refuse: !cooperative,
			message: fmt.Sprintf("runtime %s grants no write tool (its --allowedTools list has no Edit/Write), so an rw child cannot modify the workspace and would burn the run finding out",
				rt.Name),
		})
	}

	sandbox := sandboxArgsFor(rt, grant)
	args := append(append([]string{}, rt.Args...), sandbox...)

	// Class 2: dacli binary path vs allowlist (dacli 267).
	if exe != "" {
		if msg, ok := exeAllowlistWarning(rt, sandbox, exe); ok {
			issues = append(issues, preflightIssue{
				class:   "binary-allowlist",
				refuse:  false,
				message: strings.TrimSuffix(strings.TrimPrefix(msg, "warning: "), "\n"),
			})
		}
	}

	// Class 3: the role prompt's named tools vs what the runtime permits.
	if hasRole {
		for _, tool := range store.NamedTools(role.Prompt) {
			if !store.RuntimeAllowsTool(args, tool) {
				issues = append(issues, preflightIssue{
					class:  "prompt-tools",
					refuse: false,
					message: fmt.Sprintf("role %s's prompt names %s, but runtime %s's --allowedTools does not grant it",
						role.Name, tool, rt.Name),
				})
			}
		}
	}

	return issues
}

// cmdPreflight is the standalone form of dacli 272's check: given a
// role/runtime/grant (resolved the same way spawn resolves them, minus the
// task), report every mismatch in one pass and exit 3 if any of them refuse.
func cmdPreflight(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, ferr := clikit.ParseFlags(args)
	if ferr != nil {
		return ferr
	}
	if err := f.Reject("role", "runtime", "grant", "cooperative"); err != nil {
		return err
	}

	roleName := f.Get("role")
	var role team.Role
	hasRole := false
	if roleName != "" {
		var ok bool
		role, ok = store.LoadRole(w, roleName)
		if !ok {
			return store.ErrNotFound{Ref: "role " + roleName}
		}
		hasRole = true
	}

	rtName := f.Get("runtime")
	if rtName == "" && hasRole {
		rtName = role.Runtime
	}
	if rtName == "" {
		return clikit.Usagef("usage: dacli preflight --role <name> [--runtime name] [--grant ro|rw] [--cooperative]")
	}
	rt, err := store.LoadRuntime(w, rtName)
	if err != nil {
		return err
	}
	path, err := exec.LookPath(rt.Binary)
	if err != nil {
		return fmt.Errorf("runtime %s: binary %q not on PATH", rt.Name, rt.Binary)
	}
	rt = store.HydrateRuntimeROProbe(w, rt, path)

	grant := model.Grant(f.Get("grant"))
	if grant == "" && hasRole && role.Grant != "" {
		grant = model.Grant(role.Grant)
	}
	if grant == "" {
		grant = model.GrantRO
	}

	exe, _ := os.Executable()
	issues := preflightIssues(rt, role, hasRole, grant, f.Bool("cooperative"), exe)

	if len(issues) == 0 {
		fmt.Fprintf(ctx.Stdout, "preflight %s on %s (%s): no mismatches\n", clikit.OrDash(roleName), rt.Name, grant)
		return nil
	}
	var refused []string
	for _, iss := range issues {
		verdict := "warn  "
		if iss.refuse {
			verdict = "refuse"
			refused = append(refused, iss.message)
		}
		fmt.Fprintf(ctx.Stdout, "%s  %-16s %s\n", verdict, iss.class, iss.message)
	}
	if len(refused) > 0 {
		return clikit.Refusedf("%s", strings.Join(refused, "; "))
	}
	return nil
}
