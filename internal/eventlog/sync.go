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

	// Oldest first: a claim followed by a propose-status must apply in order.
	for i := len(pending) - 1; i >= 0; i-- {
		e := pending[i]

		t, err := store.FindTask(w, e.About)
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

func apply(w *workspace.Workspace, e *Event, t *store.Task) (bool, string, error) {
	label := fmt.Sprintf("%03d-%s", t.Seq, t.Slug)

	switch e.Kind {
	case model.EventClaim:
		t.Doc.Front.Set("owner", e.Actor)
		store.AppendLog(t, fmt.Sprintf("claimed by %s (event %s)", e.Actor, e.ID[:10]))
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
		store.AppendLog(t, fmt.Sprintf("released by %s", e.Actor))
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
		if _, err := store.CreateNote(w, e.Actor, t.Project, model.NoteFinding, strings.TrimSpace(title), store.NoteOpts{
			About: e.About,
			Body:  strings.TrimSpace(body),
		}); err != nil {
			return false, "", err
		}
		store.AppendLog(t, fmt.Sprintf("finding by %s: %s", e.Actor, strings.TrimSpace(title)))
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
		store.AppendLog(t, fmt.Sprintf("status %s proposed by %s, applied", target, e.Actor))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		if err := store.MoveTask(w, t, target); err != nil {
			return false, "", err
		}
		return true, fmt.Sprintf("status: %s → %s (proposed by %s)", label, target, e.Actor), nil

	case model.EventComment:
		store.AppendLog(t, fmt.Sprintf("%s: %s", e.Actor, e.Body))
		if err := store.SaveTask(t); err != nil {
			return false, "", err
		}
		return true, "comment on " + label, nil

	case model.EventBlock:
		store.AppendLog(t, fmt.Sprintf("blocked by %s: %s", e.Actor, e.Body))
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
