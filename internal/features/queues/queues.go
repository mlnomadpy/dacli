// Package queues is the checklist slice: ordered steps with an owned cursor.
// dacli never executes a step.
package queues

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "queue add", Brief: "Create a queue of ordered steps", Mutates: true, Usage: "dacli queue add <slug> --step 'cmd or instruction'... [--title t]", Run: cmdAdd},
	{Path: "queue rm", Brief: "Remove a queue", Mutates: true, Usage: "dacli queue rm <name>", Run: cmdQueueRm},
	{Path: "queue list", Brief: "List queues and their cursors", Usage: "dacli queue list", Run: cmdList},
	{Path: "queue next", Brief: "Print the next step (dacli does not run it)", Usage: "dacli queue next <slug>", Run: cmdNext},
	{Path: "queue advance", Brief: "Record an idempotent success, retryable failure, or terminal dead-letter transition", Mutates: true, Usage: "dacli queue advance <slug> [--key id] [--retry reason|--terminal reason|--fail reason]", Run: cmdAdvance},
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
	f, err := clikit.ParseFlags(args, "fail", "key", "retry", "terminal")
	if err != nil {
		return err
	}
	if err := f.Reject("fail", "key", "retry", "terminal"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli queue advance <slug> [--key id] [--retry reason|--terminal reason|--fail reason]")
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
	retryReason, terminalReason := f.Get("retry"), f.Get("terminal")
	if retryReason != "" && (terminalReason != "" || f.Get("fail") != "") {
		return clikit.Usagef("choose exactly one failure classification: --retry or --terminal/--fail")
	}
	if terminalReason == "" {
		terminalReason = f.Get("fail")
	}
	key := f.Get("key")
	if key == "" {
		outcome := "success"
		if retryReason != "" {
			outcome = "retryable"
		} else if terminalReason != "" {
			outcome = "terminal"
		}
		key = fmt.Sprintf("queue:%s:%d:%s", q.Slug, q.Cursor, outcome)
	}
	receipts := filepath.Join(w.QueuesDir(), q.Slug+".transitions")
	if transitionSeen(receipts, key) {
		fmt.Fprintf(ctx.Stdout, "transition %s already applied — no-op\n", key)
		return nil
	}
	if retryReason != "" {
		if err := recordQueueTransition(w, id.ID, q, receipts, key, "retryable", retryReason, false); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "queue retry recorded: %s\n", retryReason)
		return nil
	}
	if terminalReason != "" {
		if err := recordQueueTransition(w, id.ID, q, receipts, key, "terminal", terminalReason, true); err != nil {
			return err
		}
		if err := store.Advance(q, terminalReason); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "queue halted: %s\n", terminalReason)
		return nil
	}
	if err := store.Advance(q, ""); err != nil {
		return err
	}
	if err := recordQueueTransition(w, id.ID, q, receipts, key, "success", "", false); err != nil {
		return err
	}
	if step, done := q.Next(); done {
		fmt.Fprintln(ctx.Stdout, "queue complete")
	} else {
		fmt.Fprintf(ctx.Stdout, "next → step %d/%d: %s\n", q.Cursor+1, len(q.Steps), step)
	}
	return nil
}

func transitionPath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("%x.md", sha256.Sum256([]byte(key))))
}

func transitionSeen(dir, key string) bool {
	_, err := os.Stat(transitionPath(dir, key))
	return err == nil
}

func recordQueueTransition(w *workspace.Workspace, actor string, q *store.Queue, receipts, key, outcome, reason string, dead bool) error {
	body := fmt.Sprintf("queue transition key=%q outcome=%s cursor=%d", key, outcome, q.Cursor)
	if reason != "" {
		body += " reason=" + fmt.Sprintf("%q", reason)
	}
	if _, err := eventlog.Append(w, actor, model.EventRun, "q-"+q.Slug, "agent", body); err != nil {
		return err
	}
	dir := receipts
	if dead {
		dir = filepath.Join(w.QueuesDir(), q.Slug+".dead-letter")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(transitionPath(dir, key), []byte(body+"\n"), 0o644); err != nil {
		return err
	}
	if dead {
		if err := os.MkdirAll(receipts, 0o755); err != nil {
			return err
		}
		return os.WriteFile(transitionPath(receipts, key), []byte(body+"\n"), 0o644)
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
