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
	if err := f.Reject("all", "verify", "force", "require-verify", "require-independent", "allow-unverified", "allow-unlanded"); err != nil {
		return err
	}
	requireVerify := f.Bool("require-verify")
	requireIndependent := f.Bool("require-independent")
	allowUnlanded := f.Bool("allow-unlanded")
	allowUnverified := f.Bool("allow-unverified")

	// --all: accept every task an agent has proposed for acceptance, in one
	// pass. This is the "owner sets policy instead of hand-closing every spawn"
	// surface — the verify command, if given, now runs PER TASK (dacli 185).
	if f.Bool("all") {
		return acceptAll(ctx, w, id, f.Get("verify"), f.Bool("force"), requireVerify, requireIndependent, allowUnverified, allowUnlanded)
	}

	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli accept <ref> [--verify \"cmd\"] [--require-verify] [--force] | dacli accept --all [--verify \"cmd\"] [--require-verify] [--force]\n(--verify runs per task and its result is recorded on the task; --require-verify refuses to close anything unverified)")
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
			return acceptOne(ctx, w, id, t, f.Get("verify"), requireVerify, requireIndependent, allowUnverified, allowUnlanded)
		}
		return propose(ctx, w, id, t)
	}

	return acceptOne(ctx, w, id, t, f.Get("verify"), requireVerify, requireIndependent, allowUnverified, allowUnlanded)
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
func acceptOne(ctx *clikit.Ctx, w *workspace.Workspace, id *agentid.Identity, t *store.Task, verify string, requireVerify, requireIndependent, allowUnverified, allowUnlanded bool) error {
	// A task with no acceptance criteria checks zero boxes and reports success,
	// so zero boxes read as all boxes and the close certifies nothing (dacli
	// 289). Refuse unless the owner explicitly opts into an unverified close —
	// the same rule task done and the propose→sync path enforce.
	if !store.HasAcceptanceCriteria(t) && !allowUnverified {
		return emptyAcceptanceRefusal(t.Seq)
	}
	if verify == "" && requireVerify {
		return requireVerifyRefusal(t.Seq)
	}
	if err := independenceCheck(id, t, requireIndependent); err != nil {
		return err
	}
	if verify != "" {
		if err := runVerify(ctx, w, verify); err != nil {
			// A failed check is a RESULT, reported operationally (exit 1): the
			// verification ran and the task did not pass it, so it stays open.
			return fmt.Errorf("verification failed — task %03d NOT accepted: %w", t.Seq, err)
		}
		fmt.Fprintf(ctx.Stderr, "verification passed: %s\n", verify)
	}

	// A passing verify proves the TREE is healthy, not that THIS task's work is
	// in it — the gap issue #382 called its most serious finding (done:15/21
	// reported while the commands did not exist, because the PRs had failed to
	// merge). Ask the question a build cannot: did this task's branch reach
	// trunk? Loud by default because a close that outruns its deliverable is
	// exactly what nobody notices; a refusal under --require-verify, where the
	// operator has already said the record matters.
	landing, branch := checkLanded(w, t, trunkBranch(w))
	if landing == landingUnlanded && !allowUnlanded {
		if requireVerify {
			return unlandedRefusal(t.Seq, branch, trunkBranch(w))
		}
		// --allow-unlanded silences this as well as the refusal: the flag says
		// the caller has accounted for the gap. The loop's local path passes
		// it because it accepts BEFORE integrating (integrate cannot merge a
		// task that is not done), so the warning would fire on every task of
		// every cycle for a landing about to happen — and a warning that is
		// always wrong is one nobody reads when it is right.
		fmt.Fprintf(ctx.Stderr, "warning: %s has commits that are NOT in trunk — this close records work the trunk has not received. Merge the branch, or re-run with --require-verify to make this a refusal.\n", branch)
	}

	// Read the pending proposals now but do NOT consume them yet: they are the
	// owner's acknowledgement of this close, and marking them applied before the
	// close is durable would orphan the work if CloseTask fails (dacli 210).
	proposals := pendingProposals(w, t)

	newly := store.CheckAllAcceptance(t)
	line := fmt.Sprintf("accepted by %s", id.ID)
	if len(proposals) > 0 {
		line += fmt.Sprintf(" (applied %d proposal(s))", len(proposals))
	}
	store.AppendLog(t, line)
	// Record the evidence — or its absence — on the task itself, so the
	// trajectory never implies a verification that did not happen.
	store.AppendLog(t, verificationEvidence(verify))
	// State what was known about the deliverable, so the trajectory never
	// implies a landing that was never confirmed.
	store.AppendLog(t, landingEvidence(landing, branch))
	if !store.HasAcceptanceCriteria(t) {
		store.AppendLog(t, emptyAcceptanceEvidence)
	}
	// CloseTask stamps "completed by" (the actuals capture field) and moves to
	// done — the same canonical close `task done` uses. Without it a
	// single-accept closed a task with no actuals, silently breaking calibration
	// (E1). The "accepted by" line above is flushed by CloseTask's SaveTask.
	if err := store.CloseTask(w, t, id.ID); err != nil {
		return err
	}
	// The close is durable — only now consume the proposals. If CloseTask failed
	// above, they stay pending so a retry re-finds the task (dacli 210).
	markProposalsApplied(proposals)

	fmt.Fprintf(ctx.Stdout, "accepted: %03d-%s — checked %d acceptance box(es), moved to done\n", t.Seq, t.Slug, newly)
	return nil
}

// acceptAll accepts every task carrying at least one pending proposal. The
// verify command, if given, runs PER TASK: gating the whole batch once meant a
// single passing build closed every proposed task, including ones whose work
// was unrelated or absent (dacli 185). A task whose verification fails is
// skipped and left open. force mirrors the
// single-ref override (cmdAccept): when the acting identity is root, a task
// owned by another (finished, orphaning) agent is adopted and reconciled
// instead of skipped — so a wave-ending `ship` can auto-close every task a
// now-dead spawned agent proposed, not just the ones root itself owns.
func acceptAll(ctx *clikit.Ctx, w *workspace.Workspace, id *agentid.Identity, verify string, force, requireVerify, requireIndependent, allowUnverified, allowUnlanded bool) error {
	trunk := trunkBranch(w)
	proposed, err := proposedTasks(w)
	if err != nil {
		return err
	}
	if len(proposed) == 0 {
		fmt.Fprintln(ctx.Stdout, "no tasks proposed for acceptance")
		return nil
	}
	if verify == "" && requireVerify {
		return clikit.Refusedf("--require-verify is set and no --verify command was given: %d task(s) cannot be closed on unverified assertions", len(proposed))
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
		// A task with no acceptance criteria certifies nothing when closed
		// (dacli 289): skip it and leave it open unless the owner opted into an
		// unverified batch close, mirroring the single-ref refusal.
		if !store.HasAcceptanceCriteria(t) && !allowUnverified {
			fmt.Fprintf(ctx.Stderr, "skipped %03d-%s: no acceptance criteria — nothing to verify (pass --allow-unverified to close it explicitly UNVERIFIED)\n", t.Seq, t.Slug)
			continue
		}
		if err := independenceCheck(id, t, requireIndependent); err != nil {
			fmt.Fprintf(ctx.Stderr, "skipped %03d-%s: %v\n", t.Seq, t.Slug, err)
			continue
		}
		// Verify PER TASK. A single workspace-wide command run once told you
		// the tree was healthy after all the work — it never established that
		// task N specifically was done, yet it closed every proposed task
		// (dacli 185). Re-running costs more; a true record is worth it.
		if verify != "" {
			if err := runVerify(ctx, w, verify); err != nil {
				fmt.Fprintf(ctx.Stderr, "skipped %03d-%s: verification failed — %v\n", t.Seq, t.Slug, err)
				continue
			}
		}
		// Read but do not consume the proposals until the close is durable
		// (dacli 210): a CloseTask failure below returns before the mark, so the
		// proposals stay pending and the task is re-found on the next accept.
		proposals := pendingProposals(w, t)
		newly := store.CheckAllAcceptance(t)
		store.AppendLog(t, fmt.Sprintf("accepted by %s (applied %d proposal(s))", id.ID, len(proposals)))
		store.AppendLog(t, verificationEvidence(verify))
		// Same deliverable question the single-task path asks: did THIS task's
		// work reach trunk? --all is the batch path the loop uses, so a silent
		// close here is the one most likely to go unnoticed.
		landing, branch := checkLanded(w, t, trunk)
		if landing == landingUnlanded && !allowUnlanded {
			if requireVerify {
				return unlandedRefusal(t.Seq, branch, trunk)
			}
			fmt.Fprintf(ctx.Stderr, "warning: %s has commits that are NOT in trunk — task %03d is being closed over work the trunk has not received\n", branch, t.Seq)
		}
		store.AppendLog(t, landingEvidence(landing, branch))
		if !store.HasAcceptanceCriteria(t) {
			store.AppendLog(t, emptyAcceptanceEvidence)
		}
		// CloseTask stamps "completed by" (the actuals capture field) and moves to
		// done — calibration pairs it with the spawn-time "claimed by" (E3) to size
		// the run. One canonical close for every path; no task closes without it.
		if err := store.CloseTask(w, t, id.ID); err != nil {
			return err
		}
		markProposalsApplied(proposals)
		fmt.Fprintf(ctx.Stdout, "accepted: %03d-%s — checked %d box(es)\n", t.Seq, t.Slug, newly)
		accepted++
	}
	fmt.Fprintf(ctx.Stdout, "accepted %d task(s)\n", accepted)
	return nil
}

// pendingProposals returns every unconsumed box-check proposal event for this
// task. It only READS: consuming a proposal (MarkApplied) is deferred to
// markProposalsApplied, which runs after the close is durable — see dacli 210.
func pendingProposals(w *workspace.Workspace, t *store.Task) []*eventlog.Event {
	events, err := eventlog.List(w, eventlog.Query{About: t.ID, Pending: true})
	if err != nil {
		return nil
	}
	var out []*eventlog.Event
	for _, e := range events {
		if isCloseRequest(e) {
			out = append(out, e)
		}
	}
	return out
}

// isCloseRequest reports whether an event is an agent asking the owner to
// close this task — through EITHER channel.
//
// There are two, and an agent cannot be expected to know the difference:
// `dacli accept` files an accept-propose COMMENT, while `dacli task done`
// files a propose-status EVENT. Only the comment channel was ever consumed
// here, so an agent that used `task done` — which is what the protocol
// preamble tells it to do — produced a request nothing acted on.
//
// That was one leg of a three-way deadlock (task 312): the agent's claim is
// not applied until sync, so it cannot check its own acceptance boxes; sync
// then refuses its propose:done because those boxes are unmet; and
// `accept --all` could not see the request at all. Both channels mean the
// same thing — "I believe this is finished, please verify and close" — so
// both are answered here, by the owner, who is the only one allowed to check
// the boxes.
func isCloseRequest(e *eventlog.Event) bool {
	switch e.Kind {
	case model.EventComment:
		return isProposal(e)
	case model.EventProposeStatus:
		return strings.TrimSpace(strings.TrimPrefix(e.Body, "propose:")) == string(model.StatusDone)
	}
	return false
}

// markProposalsApplied marks each proposal event as applied, returning how many
// were acknowledged. Only the owner reaches here, so marking applied is the
// owner's decision recorded — mirroring eventlog.Sync's contract. It MUST be
// called only after store.CloseTask succeeds: a proposal consumed before the
// close is durable would vanish from pending while the task stayed open, so the
// next accept could no longer re-find the task and the completed work would be
// permanently invisible (dacli 210).
func markProposalsApplied(proposals []*eventlog.Event) int {
	n := 0
	for _, e := range proposals {
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
	events, err := eventlog.List(w, eventlog.Query{Pending: true})
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
		if !isCloseRequest(e) {
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

// verificationEvidence renders what actually certified a close, for the task
// log. The record must distinguish "a command ran and passed" from "nobody
// checked" — a close whose evidence is absent used to be indistinguishable
// from a verified one, which is what made every `done` label an unverified
// assertion (dacli 184).
func verificationEvidence(cmd string) string {
	if cmd == "" {
		return "closed WITHOUT verification — no --verify command was given"
	}
	return fmt.Sprintf("verified by `%s` (exit 0)", cmd)
}

// independenceCheck refuses a close where the certifier is the same agent that
// did the work. The implementer marking its own work complete is the oldest
// failure in the review literature and the one dacli's whole role split exists
// to prevent — yet nothing enforced it: the task owner (the claimant) accepted
// its own task. Opt-in, because the common operator flow (root closes work its
// children did) is already independent and must keep working (dacli 188).
func independenceCheck(id *agentid.Identity, t *store.Task, required bool) error {
	if !required {
		return nil
	}
	claimant := store.ClaimedBy(t)
	if claimant != "" && claimant == id.ID {
		return clikit.Refusedf("--require-independent is set: %s claimed task %03d and cannot also certify it; a different agent (or root) must accept", id.ID, t.Seq)
	}
	return nil
}

// emptyAcceptanceEvidence is the Log stamp for an --allow-unverified close of a
// task that stated no acceptance criteria — the record must say plainly that
// nothing was checked, never let the "done" imply otherwise (dacli 289).
const emptyAcceptanceEvidence = "closed with NO acceptance criteria — UNVERIFIED (--allow-unverified)"

// emptyAcceptanceRefusal is the empty-acceptance refusal (exit 3): a task with
// no acceptance boxes checks zero boxes and reports success, so closing it
// certifies nothing. Refuse rather than silently close, the same rule every
// close path enforces (dacli 289).
func emptyAcceptanceRefusal(seq int) error {
	return clikit.Refusedf("task %03d has no acceptance criteria: closing it would verify nothing. Add at least one criterion, or pass --allow-unverified to close it as explicitly UNVERIFIED — do not retry", seq)
}

// requireVerifyRefusal is the strict-mode refusal. Generating repositories
// whose trajectory is the product needs every close backed by a command that
// actually ran; --require-verify makes an unverified close impossible rather
// than merely visible.
func requireVerifyRefusal(seq int) error {
	return clikit.Refusedf("--require-verify is set and no --verify command was given: task %03d cannot be closed on an unverified assertion", seq)
}
