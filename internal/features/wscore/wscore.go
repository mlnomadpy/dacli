// Package wscore is the workspace-bootstrap slice: init and identity.
package wscore

import (
	"fmt"
	"path/filepath"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "init", Brief: "Create a .dacli workspace (--template to seed a process)", Run: cmdInit},
	{Path: "whoami", Brief: "Show the acting agent and its grant", Run: cmdWhoami},
}

func cmdInit(ctx *clikit.Ctx, args []string) error {
	f, _ := clikit.ParseFlags(args)
	name := f.Get("name")
	if name == "" {
		name = filepath.Base(ctx.Cwd)
	}
	w, err := workspace.Init(ctx.Cwd, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "initialized workspace %q (%s) at %s\n", w.Name, w.ID, filepath.Join(w.Root, workspace.Dir))
	return nil
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
