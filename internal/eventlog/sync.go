package eventlog

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Result summarizes one sync pass.
type Result struct {
	Applied int
	Skipped int      // events for objects the caller does not own
	Notes   []string // human-readable line per applied event
}

// Sync materializes pending events into the objects they reference. Only the
// owner of an object applies its events — canMutate is the caller's identity
// check, passed in because L3 has no notion of identity by layering.
//
// Reads never needed this: status and context fold pending events in on the
// fly. Sync is only about promoting an event into the durable object — the
// folder move, the note file, the Log line.
func Sync(w *workspace.Workspace, actor string, canMutate func(owner string) bool) (*Result, error) {
	pending, err := List(w, Query{Pending: true})
	if err != nil {
		return nil, err
	}
	res := &Result{}

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

		applied, note, err := apply(w, e, t)
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
		logOnce(t, e.ID, fmt.Sprintf("status %s proposed by %s, applied", target, e.Actor))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		if err := store.MoveTask(w, t, target); err != nil {
			return false, "", err
		}
		return true, fmt.Sprintf("status: %s → %s (proposed by %s)", label, target, e.Actor), nil

	case model.EventComment:
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
// wrongly suppressed. t.Doc reflects the durable Log because Sync loads the
// task fresh each pass.
func logOnce(t *store.Task, eventID, line string) {
	tag := "(event " + eventID + ")"
	if s, ok := t.Doc.Section("Log"); ok && strings.Contains(s.Content, tag) {
		return
	}
	store.AppendLog(t, line+" "+tag)
}
