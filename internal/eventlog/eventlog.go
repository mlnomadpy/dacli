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
	"errors"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Append writes a new event. Never fails on contention, because there is none.
func Append(w *workspace.Workspace, actor string, kind model.EventKind, about model.Ref, body string) (*model.Event, error) {
	// TODO: generate a ULID, mkdir -p events/YYYY/MM/DD, write atomically.
	return nil, errors.New("not implemented")
}

// Query filters the log. Reads walk the date-partitioned tree newest-first and
// stop once Since is passed, so a long-lived workspace does not pay for its
// whole history on every call.
type Query struct {
	About   model.Ref
	Kinds   []model.EventKind
	Actor   string
	Since   time.Time
	Pending bool // only events with applied: false
	Limit   int
}

// List returns matching events, newest first.
func List(w *workspace.Workspace, q Query) ([]model.Event, error) {
	return nil, errors.New("not implemented")
}

// Sync materializes pending events into the objects they reference. Only the
// owner of an object may sync it.
//
// Note what does NOT require a sync: reads. `dacli status` and `dacli context`
// fold pending events in on the fly, so a child's finding is visible across
// the whole agent tree the moment it is written. Sync is only about promoting
// an event into the durable object — moving a task folder on an accepted
// propose-status, appending a finding to the task's ## Log.
//
// Events are never deleted, only marked applied. `dacli events compact`
// archives old ones.
func Sync(w *workspace.Workspace, actor string, about model.Ref) (applied int, err error) {
	return 0, errors.New("not implemented")
}
