// Package queues is the checklist slice: ordered steps with an owned cursor.
// dacli never executes a step.
package queues

import (
	"fmt"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "queue add", Brief: "Create a queue of ordered steps", Run: cmdAdd},
	{Path: "queue list", Brief: "List queues and their cursors", Run: cmdList},
	{Path: "queue next", Brief: "Print the next step (dacli does not run it)", Run: cmdNext},
	{Path: "queue advance", Brief: "Move the cursor past the current step (--fail halts)", Run: cmdAdvance},
}

func cmdAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || len(f.All("step")) == 0 {
		return clikit.Usagef("usage: dacli queue add <slug> --step 'cmd or instruction'... [--title t]")
	}
	q, err := store.CreateQueue(w, id.ID, f.Pos[0], f.Get("title"), f.All("step"))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "queue %s created: %d steps, owned by %s\n", q.Slug, len(q.Steps), q.Owner)
	return nil
}

func cmdList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	qs, err := store.ListQueues(w)
	if err != nil {
		return err
	}
	for _, q := range qs {
		state := fmt.Sprintf("%d/%d", q.Cursor, len(q.Steps))
		if q.Halted != "" {
			state = "HALTED: " + q.Halted
		}
		fmt.Fprintf(ctx.Stdout, "%-20s %-24s %s\n", q.Slug, state, q.Title)
	}
	return nil
}

func cmdNext(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli queue next <slug>")
	}
	q, err := store.LoadQueue(w, f.Pos[0])
	if err != nil {
		return err
	}
	if q.Halted != "" {
		return fmt.Errorf("queue is halted: %s", q.Halted)
	}
	step, done := q.Next()
	if done {
		fmt.Fprintln(ctx.Stdout, "queue complete")
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "step %d/%d: %s\n", q.Cursor+1, len(q.Steps), step)
	return nil
}

func cmdAdvance(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli queue advance <slug> [--fail reason]")
	}
	q, err := store.LoadQueue(w, f.Pos[0])
	if err != nil {
		return err
	}
	// The cursor is mutable state with exactly one writer — the C2 fix.
	if q.Owner != "" && q.Owner != id.ID {
		return clikit.Refusedf("queue %s is owned by %s; ask them to advance it", q.Slug, q.Owner)
	}
	if !id.CanMutate(q.Owner) {
		return clikit.Refusedf("advancing a queue rewrites its cursor, which needs an rw grant")
	}
	if err := store.Advance(q, f.Get("fail")); err != nil {
		return err
	}
	if f.Get("fail") != "" {
		fmt.Fprintf(ctx.Stdout, "queue halted: %s\n", f.Get("fail"))
		return nil
	}
	if step, done := q.Next(); done {
		fmt.Fprintln(ctx.Stdout, "queue complete")
	} else {
		fmt.Fprintf(ctx.Stdout, "next → step %d/%d: %s\n", q.Cursor+1, len(q.Steps), step)
	}
	return nil
}
