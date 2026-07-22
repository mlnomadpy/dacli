// Package wscore is the workspace-bootstrap slice: init and identity.
package wscore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "init", Brief: "Create a .dacli workspace (--template seeds a default process, --roster seeds roles)", Run: cmdInit},
	{Path: "whoami", Brief: "Show the acting agent and its grant", Run: cmdWhoami},
}

func cmdInit(ctx *clikit.Ctx, args []string) error {
	f, _ := clikit.ParseFlags(args)
	name := f.Get("name")
	if name == "" {
		name = filepath.Base(ctx.Cwd)
	}

	// Validate the advertised flags BEFORE creating anything: a typo in
	// --template or --roster should refuse loudly, not silently seed an empty
	// workspace (the bug this slice previously had — clikit.ParseFlags accepts
	// unknown flags, so an ignored --template exited 0 with no process seeded).
	tmpl := f.Get("template")
	roster := f.Get("roster")
	if tmpl != "" {
		if _, err := gates.Get(nil, tmpl); err != nil {
			return clikit.Usagef("unknown template %q — available: %s", tmpl, templateNames())
		}
	}
	if roster != "" {
		if _, ok := rosters[roster]; !ok {
			return clikit.Usagef("unknown roster %q — available: %s", roster, rosterNames())
		}
	}

	w, err := workspace.Init(ctx.Cwd, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "initialized workspace %q (%s) at %s\n", w.Name, w.ID, filepath.Join(w.Root, workspace.Dir))

	// --template records the workspace default process. `project add` with no
	// --template falls back to it, so a new user's first project is staged
	// exactly as WALKTHROUGH.md § 1 promises — instead of the flag being
	// silently dropped.
	if tmpl != "" {
		if err := setDefaultTemplate(w, tmpl); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "default template: %s (new projects seed this process; override with `project add --template`)\n", tmpl)
	}

	// --roster seeds a starting set of role files. Editing them is expected;
	// the roster only saves the first-run typing.
	if roster != "" {
		roles := rosters[roster]
		for _, r := range roles {
			if err := store.CreateRole(w, "a-root", r); err != nil {
				return err
			}
		}
		fmt.Fprintf(ctx.Stdout, "roster %s: seeded %d role(s): %s\n", roster, len(roles), roleNames(roles))
	}
	return nil
}

// setDefaultTemplate appends the chosen template to config.yml. open() parses
// it back into Workspace.DefaultTemplate, and planning's `project add` reads
// that as the fallback when no --template is passed.
func setDefaultTemplate(w *workspace.Workspace, tmpl string) error {
	raw, err := os.ReadFile(w.ConfigPath())
	if err != nil {
		return err
	}
	updated := strings.TrimRight(string(raw), "\n") + fmt.Sprintf("\ndefault_template: %s\n", tmpl)
	return os.WriteFile(w.ConfigPath(), []byte(updated), 0o644)
}

func templateNames() string {
	ts, _ := gates.Load(nil)
	var names []string
	for _, t := range ts {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func rosterNames() string {
	var names []string
	for n := range rosters {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func roleNames(roles []team.Role) string {
	var names []string
	for _, r := range roles {
		names = append(names, r.Name)
	}
	return strings.Join(names, ", ")
}

func cmdWhoami(ctx *clikit.Ctx, args []string) error {
	_, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.Role != "" {
		fmt.Fprintf(ctx.Stdout, "%s (grant: %s, role: %s)\n", id.ID, id.Grant, id.Role)
	} else {
		fmt.Fprintf(ctx.Stdout, "%s (grant: %s)\n", id.ID, id.Grant)
	}
	return nil
}
