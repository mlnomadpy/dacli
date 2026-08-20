package eventlog

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// ProposePrefix marks an EventComment body as an acceptance box-check proposal
// (a read-only agent's `dacli accept` records one; the owner's `dacli accept`
// applies it). It lives here, not in the acceptance slice, because two
// consumers race on the same pending event: eventlog.Sync must NOT treat a
// proposal like a generic comment (log it + mark applied), or the proposal is
// consumed before `dacli accept` can see it as pending — the task then never
// closes, boxes never checked, with no signal. Sync leaves proposals pending;
// acceptance owns them. The acceptance slice references this constant so the
// convention has exactly one definition.
const ProposePrefix = "accept-propose:"

// Result summarizes one sync pass.
type Result struct {
	Applied int
	Skipped int // events for objects the caller does not own
	// Retired counts journal events (commit, run) that an older dacli left
	// pending and this sync stamped applied. They are reported separately from
	// Applied because nothing was materialized — a record was simply corrected.
	Retired int
	Notes   []string // human-readable line per applied event
	// Unreadable is the pending events sync could NOT parse. A sync that
	// applies three of four proposals because one file is corrupt has done a
	// partial job, and reporting only "applied 3" makes that indistinguishable
	// from there having been three. The caller must surface these (dacli 350).
	Unreadable []string
}

// Sync materializes pending events into the objects they reference. Only the
// owner of an object applies its events — canMutate is the caller's identity
// check, passed in because L3 has no notion of identity by layering.
//
// Reads never needed this: status and context fold pending events in on the
// fly. Sync is only about promoting an event into the durable object — the
// folder move, the note file, the Log line.
func Sync(w *workspace.Workspace, actor string, canMutate func(owner string) bool) (*Result, error) {
	return syncWithCheckpointHook(w, actor, canMutate, nil)
}

// syncWithCheckpointHook exposes the durable boundary to failure-injection
// tests. The hook runs only after an event's applied marker is on disk, which
// is the replay checkpoint a restarted Sync consults.
func syncWithCheckpointHook(w *workspace.Workspace, actor string, canMutate func(owner string) bool, checkpointed func(*Event) error) (*Result, error) {
	pending, unreadable, err := ListReport(w, Query{Pending: true})
	if err != nil {
		return nil, err
	}
	res := &Result{Unreadable: unreadable}

	// Retire journal events left pending by an older dacli. Before the
	// journal/mailbox split, Append stamped every event `applied: false` and
	// apply() had no case for commit or run, so they accumulated forever: this
	// workspace held 203 of them, every commit event ever written, and every
	// `dacli status` told the operator to run a sync that could not clear
	// them. Marking them applied is not a lie — a journal event is complete
	// when written — and without it an upgraded workspace keeps the false
	// backlog it was upgraded to fix.
	for _, e := range pending {
		if !e.Kind.IsJournal() {
			continue
		}
		if err := MarkApplied(e.Path); err == nil {
			res.Retired++
			if checkpointed != nil {
				if err := checkpointed(e); err != nil {
					return res, err
				}
			}
		}
	}

	// Build the task index once: FindTask re-reads the whole task tree per
	// call, so resolving one per pending event was O(events×tasks) full
	// re-reads. One read up front, O(1) per lookup.
	idx, err := store.BuildTaskIndex(w)
	if err != nil {
		return nil, err
	}

	// Oldest first: a claim followed by a propose-status must apply in order.
	for i := len(pending) - 1; i >= 0; i-- {
		e := pending[i]

		t, err := idx.Find(e.About)
		if err != nil {
			// An event about nothing we can resolve stays pending — it may
			// reference an object another sync will create. Never applied
			// silently, never dropped.
			res.Skipped++
			continue
		}
		if !canMutate(t.Owner()) {
			res.Skipped++
			continue
		}

		// Serialize the read-modify-write against any other process touching
		// this task, and act on the copy re-read under that lock. Without it,
		// two syncs running as the same identity (the loop's auto-sync and an
		// operator's `dacli sync` — a routine pairing) interleave and one
		// silently drops the other's Log line, while its event is already
		// marked applied so logOnce can never restore it. See store.WithTask.
		var applied bool
		var note string
		err = store.WithTask(w, t, func(fresh *store.Task) error {
			var aerr error
			applied, note, aerr = apply(w, e, fresh)
			return aerr
		})
		if err != nil {
			return res, fmt.Errorf("applying %s: %w", e.ID, err)
		}
		if !applied {
			res.Skipped++
			continue
		}
		if err := MarkApplied(e.Path); err != nil {
			return res, err
		}
		res.Applied++
		res.Notes = append(res.Notes, note)
		if checkpointed != nil {
			if err := checkpointed(e); err != nil {
				return res, err
			}
		}
	}
	return res, nil
}

// apply is idempotent: Sync flips `applied` only after apply() returns, so a
// mid-apply failure leaves the event pending and re-runs it from the top on
// the next Sync. Every side effect here must therefore be safe to repeat —
// log lines are tagged with the event id and appended once (logOnce), notes
// carry the event id and dedupe on it (NoteOpts.SourceEvent), and MoveTask to
// a status the task already holds is a no-op rename. Without this a re-run
// would append a second "claimed by" line or write a duplicate finding note.
func apply(w *workspace.Workspace, e *Event, t *store.Task) (bool, string, error) {
	label := fmt.Sprintf("%03d-%s", t.Seq, t.Slug)

	switch e.Kind {
	case model.EventClaim:
		t.Doc.Front.Set("owner", e.Actor)
		logOnce(t, e.ID, fmt.Sprintf("claimed by %s", e.Actor))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		if t.Status == model.StatusOpen {
			if err := store.MoveTask(w, t, model.StatusActive); err != nil {
				return false, "", err
			}
		}
		return true, fmt.Sprintf("claim: %s → %s", label, e.Actor), nil

	case model.EventRelease:
		t.Doc.Front.Set("owner", "")
		logOnce(t, e.ID, fmt.Sprintf("released by %s", e.Actor))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		if err := store.MoveTask(w, t, model.StatusOpen); err != nil {
			return false, "", err
		}
		return true, "release: " + label, nil

	case model.EventFinding:
		// The event's first line becomes the note title, the rest the body.
		// Attribution is the event's actor, not the syncing owner — the
		// finding belongs to whoever found it.
		title, body, _ := strings.Cut(e.Body, "\n")
		if strings.TrimSpace(title) == "" {
			title = "finding from " + e.Actor
		}
		// SourceEvent makes CreateNote idempotent: a re-run finds this event's
		// existing note instead of writing a duplicate under a fresh suffix.
		if _, err := store.CreateNote(w, e.Actor, t.Project, model.NoteFinding, strings.TrimSpace(title), store.NoteOpts{
			About:       e.About,
			Origin:      e.Origin,  // carry provenance across the weld (P4)
			Against:     e.Against, // and the reviewed-agent attribution
			Body:        strings.TrimSpace(body),
			SourceEvent: e.ID,
		}); err != nil {
			return false, "", err
		}
		logOnce(t, e.ID, fmt.Sprintf("finding by %s: %s", e.Actor, strings.TrimSpace(title)))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		return true, fmt.Sprintf("finding on %s (by %s)", label, e.Actor), nil

	case model.EventProposeStatus:
		want := strings.TrimSpace(strings.TrimPrefix(e.Body, "propose:"))
		var target model.Status
		for _, s := range model.AllStatuses {
			if string(s) == want {
				target = s
			}
		}
		if target == "" {
			return false, "", nil // malformed proposal stays pending for a human
		}

		// A propose:done must close the task exactly like the owner's direct
		// `dacli task done`, not slip into done/ via a bare MoveTask. The owner
		// path (planning.go cmdTaskDone) VERIFIES acceptance, then routes through
		// store.CloseTask so the "completed by" actuals stamp lands — the
		// canonical close from task 037 that calibration.logSpan and doctor read.
		// A non-owner's proposal returned before that check (planning.go filed
		// EventProposeStatus), so Sync auto-applying it here bypassed both the
		// verification and the stamp, dropping an unstamped — possibly unmet —
		// task into done (task 284). Mirror the owner path for the done target.
		if target == model.StatusDone {
			// A propose→done for a task with no acceptance criteria would close
			// it with nothing verified — the zero-boxes-read-as-all-boxes gap
			// that task done and accept refuse (dacli 289). The box loop below
			// passes vacuously on an empty list, so guard explicitly: leave it
			// pending for the owner to close deliberately.
			if !store.HasAcceptanceCriteria(t) {
				return false, "", nil
			}
			// VERIFY first: unmet acceptance is a refusal, not a silent move.
			// Leave the event pending — the same "no is an answer" the owner
			// path returns (planning.go cmdTaskDone) — so a human resolves it
			// and no 'done' lands that no check supports.
			for _, box := range t.Acceptance() {
				if !box.Done {
					return false, "", nil
				}
			}
			logOnce(t, e.ID, fmt.Sprintf("status %s proposed by %s, applied", target, e.Actor))
			// CloseTask stamps "completed by" then moves to done. Guard the
			// stamp against a mid-apply re-run (apply is idempotent by
			// contract): if the stamp is already present — a crash after
			// CloseTask but before MarkApplied — do not append a second one,
			// just re-assert the move (an idempotent no-op rename).
			if store.LogHasStamp(t, "completed by") {
				if err := store.SaveTask(t); err != nil {
					return false, "", err
				}
				if err := store.MoveTask(w, t, model.StatusDone); err != nil {
					return false, "", err
				}
			} else if err := store.CloseTask(w, t, e.Actor); err != nil {
				return false, "", err
			}
			return true, fmt.Sprintf("status: %s → %s (proposed by %s)", label, target, e.Actor), nil
		}

		logOnce(t, e.ID, fmt.Sprintf("status %s proposed by %s, applied", target, e.Actor))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		if err := store.MoveTask(w, t, target); err != nil {
			return false, "", err
		}
		return true, fmt.Sprintf("status: %s → %s (proposed by %s)", label, target, e.Actor), nil

	case model.EventComment:
		// An acceptance proposal is an EventComment by body convention, but it
		// is NOT a generic comment: `dacli accept` consumes it (marks applied)
		// when the owner decides to close the task. If Sync logged+applied it
		// here, the two consumers would race and whichever ran first would
		// silently drop the proposal — proposedTasks (Pending:true) would then
		// report "no tasks proposed" and the task would never close. Leave it
		// pending for accept; accept is its only consumer.
		if strings.HasPrefix(strings.TrimSpace(e.Body), ProposePrefix) {
			return false, "", nil
		}
		logOnce(t, e.ID, fmt.Sprintf("%s: %s", e.Actor, e.Body))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		return true, "comment on " + label, nil

	case model.EventBlock:
		logOnce(t, e.ID, fmt.Sprintf("blocked by %s: %s", e.Actor, e.Body))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		if err := store.MoveTask(w, t, model.StatusBlocked); err != nil {
			return false, "", err
		}
		return true, "block: " + label, nil

	case model.EventDependency:
		change, err := store.DecodeDependencyChange(e.Body)
		if err != nil {
			//nolint:nilerr // invalid proposals stay pending for audited dismissal
			return false, "", nil // malformed proposals remain pending for dismissal
		}
		if err := store.ApplyDependencyChange(w, t, change); err != nil {
			//nolint:nilerr // graph drift is a proposal refusal, not a partial sync failure
			return false, "", nil // stale/invalid graph proposals fail closed
		}
		logOnce(t, e.ID, fmt.Sprintf("dependency edit proposed by %s, applied", e.Actor))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		return true, fmt.Sprintf("dependencies: %s (proposed by %s)", label, e.Actor), nil

	default:
		// help/answer/run materialize in later wedges; leaving them pending
		// is honest — an event silently marked applied is an event lost.
		return false, "", nil
	}
}

// logOnce appends a Log line tagged with the event id, but only if a line for
// this event is not already present — so re-running a partially-applied event
// does not duplicate it. The tag uses the FULL event id, not a prefix: a
// ULID's first 10 chars are purely its millisecond timestamp, so two events
// on one task in the same millisecond would share a prefix and one would be
// wrongly suppressed. t.Doc reflects the durable Log not because Sync reloads
// each task per event — it does not; it builds one store.BuildTaskIndex up
// front (Sync above) and every event in the pass mutates the same shared
// *Task pointer in place. Each Sync invocation rebuilds that index from disk,
// so across invocations the dedupe still sees the durable Log.
func logOnce(t *store.Task, eventID, line string) {
	tag := "(event " + eventID + ")"
	if s, ok := t.Doc.Section("Log"); ok && strings.Contains(s.Content, tag) {
		return
	}
	store.AppendLog(t, line+" "+tag)
}
