// Fourth slice: queues (the last stubbed v0.1 object) and the MCP server.
package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mcp"
	"github.com/mlnomadpy/dacli/internal/store"
)

func cmdQueueAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 || len(f.all("step")) == 0 {
		return usagef("usage: dacli queue add <slug> --step 'cmd or instruction'... [--title t]")
	}
	q, err := store.CreateQueue(w, id.ID, f.pos[0], f.get("title"), f.all("step"))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "queue %s created: %d steps, owned by %s\n", q.Slug, len(q.Steps), q.Owner)
	return nil
}

func cmdQueueList(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
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

func cmdQueueNext(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli queue next <slug>")
	}
	q, err := store.LoadQueue(w, f.pos[0])
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

func cmdQueueAdvance(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli queue advance <slug> [--fail reason]")
	}
	q, err := store.LoadQueue(w, f.pos[0])
	if err != nil {
		return err
	}
	// The cursor is mutable state with exactly one writer — the C2 fix.
	if q.Owner != "" && q.Owner != id.ID {
		return refusedf("queue %s is owned by %s; ask them to advance it", q.Slug, q.Owner)
	}
	if !id.CanMutate(q.Owner) {
		return refusedf("advancing a queue rewrites its cursor, which needs an rw grant")
	}
	if err := store.Advance(q, f.get("fail")); err != nil {
		return err
	}
	if f.get("fail") != "" {
		fmt.Fprintf(ctx.Stdout, "queue halted: %s\n", f.get("fail"))
		return nil
	}
	if step, done := q.Next(); done {
		fmt.Fprintln(ctx.Stdout, "queue complete")
	} else {
		fmt.Fprintf(ctx.Stdout, "next → step %d/%d: %s\n", q.Cursor+1, len(q.Steps), step)
	}
	return nil
}

// dispatch indirects the command lookup so that the `commands` table can
// reference cmdMcpServe without a static initialization cycle: the table
// references this var (nil at init), and init() closes the loop at runtime.
var dispatch func(args []string) (*Command, []string)

func init() { dispatch = match }

// executor adapts the command table for the MCP server: same dispatch, same
// exit-code contract, buffered output. This closure is the entire coupling
// between the two front ends — mcp never imports cli.
func executor(cwd string) mcp.Executor {
	return func(argv []string, jsonMode bool) (string, string, int) {
		var out, errb bytes.Buffer
		c := &Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd, JSON: jsonMode}
		cmd, rest := dispatch(argv)
		if cmd == nil {
			return "", fmt.Sprintf("unknown command %q", strings.Join(argv, " ")), 2
		}
		err := cmd.Run(c, rest)
		msg := errb.String()
		if err != nil {
			if msg != "" && !strings.HasSuffix(msg, "\n") {
				msg += "\n"
			}
			msg += err.Error()
		}
		return out.String(), msg, exitCode(err)
	}
}

func cmdMcpServe(ctx *Ctx, args []string) error {
	// Identity binds at launch from the environment; Serve fails fast on a
	// bad token rather than erroring on the tenth tool call.
	fmt.Fprintln(ctx.Stderr, "dacli mcp: serving on stdio (identity from "+"DACLI_AGENT"+", root if unset)")
	return mcp.Serve(os.Stdin, ctx.Stdout, executor(ctx.Cwd))
}
