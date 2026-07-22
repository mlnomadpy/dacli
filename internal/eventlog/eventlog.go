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
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Event is a parsed log entry.
type Event struct {
	ID      string
	Kind    model.EventKind
	Actor   string
	About   string // wikilink target, brackets stripped
	Origin  string // agent | file:<path> | external:<who> — the taint field
	Against string // an agent id this event's finding concerns — the review field
	Applied bool   // whether the owner has synced this event onto its object
	Body    string
	Path    string
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
	d.Front.Set("event_kind", string(kind))
	d.Front.Set("created", now.Format(time.RFC3339))
	d.Front.Set("created_by", actor)
	if about != "" {
		d.Front.Set("about", "[["+about+"]]")
	}
	d.Front.Set("origin", origin)
	if against != "" {
		d.Front.Set("against", against)
	}
	d.Front.Set("applied", "false")
	if body != "" {
		d.Sections = []mdstore.Section{{Level: 0, Content: body + "\n"}}
	}

	path := w.EventPath(now.Format("2006/01/02"), id, actor, kind)
	if err := mdstore.WriteFile(path, d); err != nil {
		return nil, err
	}
	return &Event{ID: id, Kind: kind, Actor: actor, About: about, Origin: origin, Against: against, Body: body, Path: path}, nil
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
	var paths []string
	root := w.EventsDir()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// A directory we cannot walk is not the same as an empty log:
			// this is an append-only, lossless log, so a read fault must be
			// surfaced rather than presented as "no events". Keep walking the
			// rest of the tree so one bad subtree does not hide the whole log.
			log.Printf("eventlog: walking %s: %v", path, err)
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
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
			continue
		}
		e := &Event{Path: p}
		e.ID, _ = doc.Front.Get("id")
		if k, ok := doc.Front.Get("event_kind"); ok {
			e.Kind = model.EventKind(k)
		}
		e.Actor, _ = doc.Front.Get("created_by")
		e.Origin, _ = doc.Front.Get("origin")
		e.Against, _ = doc.Front.Get("against")
		if a, ok := doc.Front.Get("about"); ok {
			e.About = strings.TrimSuffix(strings.TrimPrefix(a, "[["), "]]")
		}
		applied, _ := doc.Front.Get("applied")
		e.Applied = applied == "true"
		if q.Pending && applied != "false" {
			continue
		}
		if !kindOK(e.Kind) || (q.Actor != "" && e.Actor != q.Actor) || (q.About != "" && e.About != q.About) {
			continue
		}
		for _, s := range doc.Sections {
			e.Body += s.Content
		}
		e.Body = strings.TrimSpace(e.Body)
		out = append(out, e)
	}
	return out, nil
}

// MarkApplied flips the one mutable field in the format. Only the owner of
// the referenced object may call this (the caller enforces that — this
// package has no identity, by layering).
func MarkApplied(path string) error {
	d, err := mdstore.ReadFile(path)
	if err != nil {
		return err
	}
	d.Front.Set("applied", "true")
	return mdstore.WriteFile(path, d)
}
