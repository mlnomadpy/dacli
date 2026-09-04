package execution

import (
	"fmt"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

func cmdClaimExpand(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task", "run", "add", "reason"); err != nil {
		return err
	}
	if len(f.Pos) != 0 || f.Get("task") == "" || f.Get("run") == "" || len(f.All("add")) == 0 || strings.TrimSpace(f.Get("reason")) == "" {
		return clikit.Usagef("usage: dacli claim expand --task <ref> --run <id> --add <path>... --reason <text> [--json]")
	}
	if id.ID != agentid.RootID {
		return clikit.Refusedf("claim expansion is owner authority; current agent %s cannot widen a worker scope", id.ID)
	}
	task, err := store.FindTask(w, f.Get("task"))
	if err != nil {
		return err
	}
	plan, err := store.ExpandTaskClaims(w, task, f.Get("run"), id.ID, f.Get("reason"), splitClaimValues(f.All("add")), time.Now())
	if err != nil {
		return clikit.Refusedf("claim expansion refused: %v", err)
	}
	ctx.Result = plan
	if ctx.JSON {
		return clikit.EmitJSON(ctx, plan)
	}
	fmt.Fprintf(ctx.Stdout, "claim expansion %s applied to %s for next relaunch\nold: %s\nnew: %s\nreason: %s\n", clikit.Short(plan.ID, 19), task.ID, strings.Join(plan.OldClaims, ", "), strings.Join(plan.NewClaims, ", "), plan.Reason)
	return nil
}
