// Package acceptance is the supervisor-native close: `dacli accept` verifies an
// agent's completion and closes the task in one owner-set-policy step, replacing
// the per-spawn manual `task check --all` + `task done` ritual.
//
// It is a feature slice (ARCHITECTURE § 2b) and imports NO other slice — the
// close is expressed purely over store primitives (CheckAllAcceptance, SaveTask,
// MoveTask) and the event log, so the arch_test's isolation rule holds.
//
// Two paths, chosen by grant exactly as `task done` does:
//
//   - The OWNER (rw) runs `dacli accept <ref> [--verify "cmd"]`: an optional
//     verification command gates the close (a non-zero exit REFUSES the accept,
//     exit 1 — never close a task whose checks fail), then every acceptance box
//     is checked and the task moves to done. `--all` accepts, in one pass, every
//     task an agent has proposed for acceptance.
//   - A read-only AGENT runs the same command and, unable to rewrite the task,
//     records a box-check PROPOSAL as an event. The owner's accept applies it —
//     the child proposes, the owner decides.
package acceptance

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Commands is this slice's table, aggregated by the app layer (cli.go).
var Commands = []clikit.Command{
	{Path: "accept", Brief: "Verify an agent's completion and close the task (box-checks + done) in one owner step; --force lets root reconcile a task (or, with --all, every proposed task) orphaned by a finished agent", Run: cmdAccept},
}

// proposePrefix is the body convention that marks an EventComment as a
// box-check proposal. A comment carrying this prefix is the minimal-but-real
// event an agent emits and `accept` applies. It is NOT a finding: a proposed
// close is an intention, not a discovered fact, so it must not create a durable
// finding note. The convention is defined in eventlog (eventlog.ProposePrefix)
// because eventlog.Sync must recognize it too — Sync leaves proposals pending
// instead of consuming them as generic comments, so this consumer and Sync do
// not race on the same event.
const proposePrefix = eventlog.ProposePrefix

func cmdAccept(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)

	// --all: accept every task an agent has proposed for acceptance, in one
	// pass. This is the "owner sets policy instead of hand-closing every spawn"
	// surface — the verify command (if any) gates the whole batch once.
	if f.Bool("all") {
		return acceptAll(ctx, w, id, f.Get("verify"), f.Bool("force"))
	}

	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli accept <ref> [--verify \"cmd\"] [--force] | dacli accept --all [--verify \"cmd\"] [--force]")
	}
	t, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}

	// The grant decides the path, exactly as `task done` does: a read-only
	// agent cannot rewrite the task, so it proposes the close as an event.
	if !id.CanMutate(t.Owner()) {
		// Operator override: root can reconcile a task whose owner is a spawned
		// agent that has since finished — that owner will never run `sync` again,
		// so its proposed close would sit pending forever, orphan-locking the
		// backlog. --force makes the override explicit; without it the close stays
		// a proposal for a live owner to apply, preserving peer concurrency safety.
		if id.ID == agentid.RootID && f.Bool("force") {
			prev := t.Owner()
			t.Doc.Front.Set("owner", id.ID)
			store.AppendLog(t, fmt.Sprintf("adopted by %s (owner %s orphaned)", id.ID, clikit.OrDash(prev)))
			return acceptOne(ctx, w, id, t, f.Get("verify"))
		}
		return propose(ctx, w, id, t)
	}

	return acceptOne(ctx, w, id, t, f.Get("verify"))
}

// propose records a box-check proposal as an event. The owner applies it on the
// next `dacli accept` for this task — the decision stays with the owner.
func propose(ctx *clikit.Ctx, w *workspace.Workspace, id *agentid.Identity, t *store.Task) error {
	body := fmt.Sprintf("%s %s completed; proposing all acceptance boxes checked", proposePrefix, id.ID)
	if _, err := eventlog.Append(w, id.ID, model.EventComment, t.ID, "", body); err != nil {
		return err
	}
	reason := id.MutateRefusal()
	if id.Grant == model.GrantRW {
		reason = "not the owner — root can `accept --force` to reconcile"
	}
	fmt.Fprintf(ctx.Stdout, "acceptance proposed as event (%s); the owner applies it with `dacli accept %03d`\n", reason, t.Seq)
	return nil
}

// acceptOne runs the optional verification gate, then checks every acceptance
// box and moves the task to done. Any pending proposals for the task are
// acknowledged (marked applied) as part of the close.
func acceptOne(ctx *clikit.Ctx, w *workspace.Workspace, id *agentid.Identity, t *store.Task, verify string) error {
	if verify != "" {
		if err := runVerify(ctx, w, verify); err != nil {
			// A failed check is a RESULT, reported operationally (exit 1): the
			// verification ran and the task did not pass it, so it stays open.
			return fmt.Errorf("verification failed — task %03d NOT accepted: %w", t.Seq, err)
		}
		fmt.Fprintf(ctx.Stderr, "verification passed: %s\n", verify)
	}

	applied := applyProposals(w, id, t)

	newly := store.CheckAllAcceptance(t)
	line := fmt.Sprintf("accepted by %s", id.ID)
	if applied > 0 {
		line += fmt.Sprintf(" (applied %d proposal(s))", applied)
	}
	store.AppendLog(t, line)
	// CloseTask stamps "completed by" (the actuals capture field) and moves to
	// done — the same canonical close `task done` uses. Without it a
	// single-accept closed a task with no actuals, silently breaking calibration
	// (E1). The "accepted by" line above is flushed by CloseTask's SaveTask.
	if err := store.CloseTask(w, t, id.ID); err != nil {
		return err
	}

	fmt.Fprintf(ctx.Stdout, "accepted: %03d-%s — checked %d acceptance box(es), moved to done\n", t.Seq, t.Slug, newly)
	return nil
}

// acceptAll accepts every task carrying at least one pending proposal. The
// verify command, if given, gates the whole batch once — a workspace-wide
// build/test hook applies to every task being closed. force mirrors the
// single-ref override (cmdAccept): when the acting identity is root, a task
// owned by another (finished, orphaning) agent is adopted and reconciled
// instead of skipped — so a wave-ending `ship` can auto-close every task a
// now-dead spawned agent proposed, not just the ones root itself owns.
func acceptAll(ctx *clikit.Ctx, w *workspace.Workspace, id *agentid.Identity, verify string, force bool) error {
	proposed, err := proposedTasks(w)
	if err != nil {
		return err
	}
	if len(proposed) == 0 {
		fmt.Fprintln(ctx.Stdout, "no tasks proposed for acceptance")
		return nil
	}
	if verify != "" {
		if err := runVerify(ctx, w, verify); err != nil {
			return fmt.Errorf("verification failed — accepted nothing: %w", err)
		}
		fmt.Fprintf(ctx.Stderr, "verification passed: %s\n", verify)
	}

	accepted := 0
	for _, t := range proposed {
		if !id.CanMutate(t.Owner()) {
			if id.ID != agentid.RootID || !force {
				fmt.Fprintf(ctx.Stderr, "skipped %03d-%s: owned by %s\n", t.Seq, t.Slug, clikit.OrDash(t.Owner()))
				continue
			}
			prev := t.Owner()
			t.Doc.Front.Set("owner", id.ID)
			store.AppendLog(t, fmt.Sprintf("adopted by %s (owner %s orphaned)", id.ID, clikit.OrDash(prev)))
		}
		applied := applyProposals(w, id, t)
		newly := store.CheckAllAcceptance(t)
		store.AppendLog(t, fmt.Sprintf("accepted by %s (applied %d proposal(s))", id.ID, applied))
		// CloseTask stamps "completed by" (the actuals capture field) and moves to
		// done — calibration pairs it with the spawn-time "claimed by" (E3) to size
		// the run. One canonical close for every path; no task closes without it.
		if err := store.CloseTask(w, t, id.ID); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "accepted: %03d-%s — checked %d box(es)\n", t.Seq, t.Slug, newly)
		accepted++
	}
	fmt.Fprintf(ctx.Stdout, "accepted %d task(s)\n", accepted)
	return nil
}

// applyProposals marks every pending proposal event for this task as applied,
// returning how many were acknowledged. Only the owner reaches here, so marking
// applied is the owner's decision recorded — mirroring eventlog.Sync's contract.
func applyProposals(w *workspace.Workspace, id *agentid.Identity, t *store.Task) int {
	events, err := eventlog.List(w, eventlog.Query{About: t.ID, Kinds: []model.EventKind{model.EventComment}, Pending: true})
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range events {
		if !isProposal(e) {
			continue
		}
		if err := eventlog.MarkApplied(e.Path); err == nil {
			n++
		}
	}
	return n
}

// proposedTasks returns every task with at least one pending acceptance
// proposal, resolved via a single task-index build (FindTask per event would be
// O(events×tasks)).
func proposedTasks(w *workspace.Workspace) ([]*store.Task, error) {
	events, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventComment}, Pending: true})
	if err != nil {
		return nil, err
	}
	idx, err := store.BuildTaskIndex(w)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []*store.Task
	for _, e := range events {
		if !isProposal(e) {
			continue
		}
		t, err := idx.Find(e.About)
		if err != nil || seen[t.ID] {
			continue
		}
		seen[t.ID] = true
		out = append(out, t)
	}
	return out, nil
}

func isProposal(e *eventlog.Event) bool {
	return strings.HasPrefix(strings.TrimSpace(e.Body), proposePrefix)
}

// runVerify executes the verification command from the workspace root and
// returns its error (with combined output) on a non-zero exit.
func runVerify(ctx *clikit.Ctx, w *workspace.Workspace, cmd string) error {
	fmt.Fprintf(ctx.Stderr, "verifying: %s\n", cmd)
	c := exec.Command("sh", "-c", cmd)
	c.Dir = w.Root
	out, err := c.CombinedOutput()
	if err != nil {
		fmt.Fprint(ctx.Stderr, string(out))
		return fmt.Errorf("`%s` exited non-zero: %v", cmd, err)
	}
	return nil
}
