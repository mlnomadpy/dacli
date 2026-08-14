// Package eventdisp contains the dependency-neutral index of terminal event
// dispositions. eventlog and store both need the exact same answer, but
// eventlog already imports store for Sync, so putting the predicate in either
// package would create a cycle.
package eventdisp

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
)

// DismissedIDs returns event IDs named by durable dismissal records.
// Malformed records fail closed: they do not make an original event disappear.
func DismissedIDs(eventsDir string) map[string]bool {
	out := map[string]bool{}
	_ = filepath.WalkDir(eventsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".md") {
			// A missing/unreadable disposition must fail closed: it cannot make
			// another event disappear, so keep indexing the remaining records.
			//nolint:nilerr // WalkDir callback: nil continues the walk.
			return nil
		}
		doc, err := mdstore.ReadFile(path)
		if err != nil {
			//nolint:nilerr // malformed records are deliberately ignored.
			return nil
		}
		kind, _ := doc.Front.Get("kind")
		eventKind, _ := doc.Front.Get("event_kind")
		applied, _ := doc.Front.Get("applied")
		about, _ := doc.Front.Get("about")
		about = strings.TrimSuffix(strings.TrimPrefix(about, "[["), "]]")
		if kind == "event" && eventKind == "dismissal" && applied == "true" && about != "" {
			out[about] = true
		}
		return nil
	})
	return out
}
