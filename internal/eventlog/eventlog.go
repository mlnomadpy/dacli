// Package eventlog implements the append-only cross-agent write path.
//
// The concurrency strategy is to make contention impossible rather than to
// manage it. Two agents editing one markdown file will corrupt it, so no two
// agents ever write the same file: each object is rewritten only by its owner,
// and every cross-agent write becomes a NEW file named by ULID. Simultaneous
// writers produce two different paths. There is no shared mutable state, so
// there is no race and no lock.
//
// ULIDs sort lexicographically by creation time, so the directory listing IS
// the ordered log.
package eventlog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/eventdisp"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// EventSchemaVersion is the current durable event payload schema. Version 0
// is reserved for legacy events written before checksums were introduced.
const EventSchemaVersion = eventdisp.EventSchemaVersion

// Event is a parsed log entry.
type Event struct {
	SchemaVersion int
	Checksum      string
	ID            string
	Kind          model.EventKind
	Actor         string
	About         string // wikilink target, brackets stripped
	Origin        string // agent | file:<path> | external:<who> — the taint field
	Against       string // an agent id this event's finding concerns — the review field
	Applied       bool   // whether the owner has synced this event onto its object
	Dismissed     bool   // whether an audited terminal disposition names this event
	Body          string
	Path          string

	// Pending mirrors Query.Pending's test EXACTLY — the `applied:` field reads
	// literally "false" — so a caller that takes one unfiltered walk and filters
	// in memory selects the same set as Query{Pending: true} (dacli 246). It is
	// deliberately NOT !Applied: an event whose applied field is missing or
	// malformed is neither applied nor pending, and a reader must not promote it
	// to pending by accident.
	Pending bool
}

// Append writes a new event. Never fails on contention, because there is none.
//
// origin records provenance for taint tracing (docs/PROPOSALS.md P4): where
// the content of this event actually came from. Empty defaults to "agent" —
// the actor speaking for itself.
func Append(w *workspace.Workspace, actor string, kind model.EventKind, about, origin, body string) (*Event, error) {
	return AppendFinding(w, actor, kind, about, origin, "", body)
}

// AppendFinding is Append plus `against`: the agent id a review finding
// concerns. This is how a reviewer's verdict names the agent behind a defect,
// so the self-evolving-team rollup (dacli contrib) can attribute findings to
// the role that produced them.
func AppendFinding(w *workspace.Workspace, actor string, kind model.EventKind, about, origin, against, body string) (*Event, error) {
	id := ulid.New()
	now := time.Now().UTC()
	if origin == "" {
		origin = "agent"
	}

	d := &mdstore.Doc{}
	d.Front.Set("id", id)
	d.Front.Set("kind", string(model.KindEvent))
	d.Front.Set("schema_version", strconv.Itoa(EventSchemaVersion))
	d.Front.Set("event_kind", string(kind))
	created := now.Format(time.RFC3339)
	d.Front.Set("created", created)
	d.Front.Set("created_by", actor)
	if about != "" {
		d.Front.Set("about", "[["+about+"]]")
	}
	d.Front.Set("origin", origin)
	if against != "" {
		d.Front.Set("against", against)
	}
	// A journal event (commit, run) is complete when written — nothing
	// consumes it — so it is born terminal. Only mailbox events wait for a
	// consumer, and only they should ever count as pending. See
	// model.EventKind.IsJournal for why this split exists.
	d.Front.Set("applied", strconv.FormatBool(kind.IsJournal()))
	checksum := eventdisp.Checksum(eventdisp.Payload{
		SchemaVersion: EventSchemaVersion,
		ID:            id,
		DocumentKind:  model.KindEvent,
		Kind:          kind,
		Created:       created,
		Actor:         actor,
		About:         about,
		Origin:        origin,
		Against:       against,
		Body:          strings.TrimSpace(body),
	})
	d.Front.Set("checksum", checksum)
	if body != "" {
		d.Sections = []mdstore.Section{{Level: 0, Content: body + "\n"}}
	}

	path := w.EventPath(now.Format("2006/01/02"), id, actor, kind)
	if err := mdstore.WriteFile(path, d); err != nil {
		return nil, err
	}
	applied := kind.IsJournal()
	return &Event{SchemaVersion: EventSchemaVersion, Checksum: checksum, ID: id, Kind: kind, Actor: actor, About: about, Origin: origin, Against: against, Applied: applied, Pending: !applied, Body: body, Path: path}, nil
}

// Query filters the log.
type Query struct {
	About   string
	Kinds   []model.EventKind
	Actor   string
	Pending bool // only events with applied: false
	Limit   int
}

// List returns matching events, newest first. It walks the date-partitioned
// tree so a long-lived workspace does not pay for its whole history on every
// call — though v0.1 walks everything; partition pruning comes with Since.
func List(w *workspace.Workspace, q Query) ([]*Event, error) {
	events, _, err := ListReport(w, q)
	return events, err
}

var walkEventTree = filepath.WalkDir

// ListReport is List plus the paths it could NOT read.
//
// List logs an unreadable event and keeps going, which is the right shape —
// one corrupt file must not blind a reader to the whole log — but the caller
// got a shorter slice and no way to tell. A `dacli sync` over four pending
// proposals with one corrupt file applied three and reported success, which is
// the silent-partial-success class this project treats as its most expensive
// bug (dacli 350).
//
// The second return is the holes. A caller that is merely displaying can
// ignore it; a caller that is APPLYING must not.
func ListReport(w *workspace.Workspace, q Query) ([]*Event, []string, error) {
	dismissed := eventdisp.DismissedIDs(w.EventsDir())
	var paths, unreadable []string
	for _, root := range []string{w.EventsDir(), filepath.Join(w.Root, workspace.Dir, "events-archive")} {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		_ = walkEventTree(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// A directory we cannot walk is not the same as an empty log:
				// this is an append-only, lossless log, so a read fault must be
				// surfaced rather than presented as "no events". Keep walking the
				// rest of the tree so one bad subtree does not hide the whole log.
				log.Printf("eventlog: walking %s: %v", path, err)
				unreadable = append(unreadable, path)
				return nil
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
	}
	// ULID filenames: lexical sort is time order; reverse for newest-first.
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	kindOK := func(k model.EventKind) bool {
		if len(q.Kinds) == 0 {
			return true
		}
		for _, want := range q.Kinds {
			if k == want {
				return true
			}
		}
		return false
	}

	var out []*Event
	for _, p := range paths {
		if q.Limit > 0 && len(out) >= q.Limit {
			break
		}
		doc, err := mdstore.ReadFile(p)
		if err != nil {
			// A malformed or unreadable event (half-written frontmatter, bad
			// permissions) is a hole in the durable log, not a non-event.
			// Surface it — dropping it silently would erase a claim/finding/
			// propose with no signal — but keep listing the rest so a single
			// corrupt file does not blind every reader to the whole log.
			log.Printf("eventlog: skipping unreadable event %s: %v", p, err)
			unreadable = append(unreadable, p)
			continue
		}
		e, err := parseEvent(p, doc)
		if err != nil {
			log.Printf("eventlog: skipping corrupt event %s: %v", p, err)
			unreadable = append(unreadable, p)
			continue
		}
		e.Dismissed = dismissed[e.ID]
		if e.Dismissed {
			e.Pending = false
		}
		applied, _ := doc.Front.Get("applied")
		if q.Pending && (applied != "false" || e.Dismissed) {
			continue
		}
		if !kindOK(e.Kind) || (q.Actor != "" && e.Actor != q.Actor) || (q.About != "" && e.About != q.About) {
			continue
		}
		out = append(out, e)
	}
	return out, unreadable, nil
}

// Find resolves a full or unambiguous prefix event ID without changing the
// append-only record.
func Find(w *workspace.Workspace, ref string) (*Event, error) {
	events, err := List(w, Query{})
	if err != nil {
		return nil, err
	}
	var match *Event
	for _, event := range events {
		if event.ID != ref && !strings.HasPrefix(event.ID, ref) {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("event id prefix %q is ambiguous", ref)
		}
		match = event
	}
	if match == nil {
		return nil, store.ErrNotFound{Ref: "event/" + ref}
	}
	return match, nil
}

// Dismiss appends one terminal audit record. Repeated calls are idempotent and
// return the existing disposition instead of writing a duplicate.
func Dismiss(w *workspace.Workspace, actor string, original *Event, reason string) (*Event, bool, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, false, fmt.Errorf("a dismissal needs a reason")
	}
	if original.Applied && !original.Dismissed {
		return nil, false, fmt.Errorf("applied event %s cannot be dismissed; append a compensating event for its target instead", original.ID)
	}
	events, err := List(w, Query{Kinds: []model.EventKind{model.EventDismissal}})
	if err != nil {
		return nil, false, err
	}
	for _, event := range events {
		if event.About == original.ID {
			return event, false, nil
		}
	}
	event, err := Append(w, actor, model.EventDismissal, original.ID, "", strings.TrimSpace(reason))
	return event, err == nil, err
}

func parseEvent(path string, doc *mdstore.Doc) (*Event, error) {
	e := &Event{Path: path}
	e.ID, _ = doc.Front.Get("id")
	documentKind, _ := doc.Front.Get("kind")
	if k, ok := doc.Front.Get("event_kind"); ok {
		e.Kind = model.EventKind(k)
	}
	created, _ := doc.Front.Get("created")
	e.Actor, _ = doc.Front.Get("created_by")
	e.Origin, _ = doc.Front.Get("origin")
	e.Against, _ = doc.Front.Get("against")
	if a, ok := doc.Front.Get("about"); ok {
		e.About = strings.TrimSuffix(strings.TrimPrefix(a, "[["), "]]")
	}
	applied, _ := doc.Front.Get("applied")
	e.Applied = applied == "true"
	e.Pending = applied == "false"
	for _, s := range doc.Sections {
		e.Body += s.Content
	}
	e.Body = strings.TrimSpace(e.Body)

	versionText, hasVersion := doc.Front.Get("schema_version")
	e.Checksum, _ = doc.Front.Get("checksum")
	if !hasVersion && e.Checksum == "" {
		return e, nil // legacy unversioned format
	}
	if !hasVersion || e.Checksum == "" {
		return nil, fmt.Errorf("incomplete event integrity metadata")
	}
	version, err := strconv.Atoi(versionText)
	if err != nil || version != EventSchemaVersion {
		return nil, fmt.Errorf("unsupported event schema version %q", versionText)
	}
	e.SchemaVersion = version
	want := eventdisp.Checksum(eventdisp.Payload{
		SchemaVersion: version,
		ID:            e.ID,
		DocumentKind:  model.Kind(documentKind),
		Kind:          e.Kind,
		Created:       created,
		Actor:         e.Actor,
		About:         e.About,
		Origin:        e.Origin,
		Against:       e.Against,
		Body:          e.Body,
	})
	if e.Checksum != want {
		return nil, fmt.Errorf("checksum mismatch: got %s, want %s", e.Checksum, want)
	}
	return e, nil
}

// MarkApplied flips the one mutable field in the format. Only the owner of
// the referenced object may call this (the caller enforces that — this
// package has no identity, by layering).
func MarkApplied(path string) error {
	d, err := mdstore.ReadFile(path)
	if err != nil {
		return err
	}
	if _, err := parseEvent(path, d); err != nil {
		return err
	}
	d.Front.Set("applied", "true")
	return mdstore.WriteFile(path, d)
}
