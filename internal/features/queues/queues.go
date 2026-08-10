// Package queues is the checklist slice: ordered steps with an owned cursor.
// dacli never executes a step.
package queues

import (
	"errors"
	"fmt"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "queue add", Brief: "Create a queue of ordered steps", Mutates: true, Usage: "dacli queue add <slug> --step 'cmd or instruction'... [--title t]", Run: cmdAdd},
	{Path: "queue rm", Brief: "Remove a queue", Mutates: true, Usage: "dacli queue rm <name>", Run: cmdQueueRm},
	{Path: "queue list", Brief: "List queues and their cursors", Usage: "dacli queue list", Run: cmdList},
	{Path: "queue next", Brief: "Print the next step (dacli does not run it)", Usage: "dacli queue next <slug>", Run: cmdNext},
	{Path: "queue advance", Brief: "Move the cursor past the current step (--fail halts)", Mutates: true, Usage: "dacli queue advance <slug> [--fail reason]", Run: cmdAdvance},
}

func cmdAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("step", "title"); err != nil {
		return err
	}
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
	// This command takes no flags, so ANY flag is a typo. An empty allowlist
	// rejects every one — without it a mistyped flag was dropped and the
	// command ran as if nothing were wrong.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
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
	if err := f.Reject("fail"); err != nil {
		return err
	}
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

// cmdQueueRm is the removal inverse of the corresponding add. Every creatable object
// used to need a text editor to undo, which made a mistake a command made into
// a mistake only a human could fix (task 293).
func cmdQueueRm(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject(); err != nil {
		return err
	}
	if len(f.Pos) < 1 {
		return clikit.Usagef("usage: dacli queue rm <name>")
	}
	name := f.Pos[0]
	if err := store.RemoveQueue(w, name); err != nil {
		var ref store.ErrReferenced
		if errors.As(err, &ref) {
			// A dangling reference is worse than the mistake being removed, so
			// name what still points here rather than deleting and letting it
			// fail later at spawn time.
			return clikit.Refusedf("%v — retire or repoint them first", ref)
		}
		return err
	}
	fmt.Fprintf(ctx.Stdout, "removed queue %s\n", name)
	return nil
}
