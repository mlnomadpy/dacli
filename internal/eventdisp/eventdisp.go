// Package eventdisp contains the dependency-neutral index of terminal event
// dispositions. eventlog and store both need the exact same answer, but
// eventlog already imports store for Sync, so putting the predicate in either
// package would create a cycle.
package eventdisp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
)

// EventSchemaVersion is the current checksummed event format. Keeping the
// checksum primitive below both eventlog and store lets disposition indexing
// use the same integrity rule as ordinary event reads.
const EventSchemaVersion = 1

// Payload is the immutable portion covered by an event checksum. Applied is
// intentionally absent: it is the one mutable lifecycle marker in the format.
type Payload struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	DocumentKind  model.Kind      `json:"kind"`
	Kind          model.EventKind `json:"event_kind"`
	Created       string          `json:"created"`
	Actor         string          `json:"created_by"`
	About         string          `json:"about,omitempty"`
	Origin        string          `json:"origin"`
	Against       string          `json:"against,omitempty"`
	Body          string          `json:"body,omitempty"`
}

// Checksum returns the canonical digest used by eventlog writers and readers.
func Checksum(payload Payload) string {
	b, _ := json.Marshal(payload) // closed scalar struct: JSON cannot fail
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validDismissal(doc *mdstore.Doc) (string, bool) {
	versionText, ok := doc.Front.Get("schema_version")
	if !ok {
		return "", false
	}
	version, err := strconv.Atoi(versionText)
	if err != nil || version != EventSchemaVersion {
		return "", false
	}
	id, _ := doc.Front.Get("id")
	documentKind, _ := doc.Front.Get("kind")
	eventKind, _ := doc.Front.Get("event_kind")
	created, _ := doc.Front.Get("created")
	actor, _ := doc.Front.Get("created_by")
	about, _ := doc.Front.Get("about")
	about = strings.TrimSuffix(strings.TrimPrefix(about, "[["), "]]")
	origin, _ := doc.Front.Get("origin")
	against, _ := doc.Front.Get("against")
	applied, _ := doc.Front.Get("applied")
	got, _ := doc.Front.Get("checksum")
	var body string
	for _, section := range doc.Sections {
		body += section.Content
	}
	body = strings.TrimSpace(body)
	want := Checksum(Payload{
		SchemaVersion: version,
		ID:            id,
		DocumentKind:  model.Kind(documentKind),
		Kind:          model.EventKind(eventKind),
		Created:       created,
		Actor:         actor,
		About:         about,
		Origin:        origin,
		Against:       against,
		Body:          body,
	})
	return about, documentKind == string(model.KindEvent) &&
		eventKind == string(model.EventDismissal) && applied == "true" &&
		id != "" && about != "" && got != "" && got == want
}

// DismissedIDs returns event IDs named by durable, integrity-valid dismissal
// records. Malformed or tampered records fail closed: they do not make an
// original event disappear.
func DismissedIDs(eventsDir string) map[string]bool {
	out := map[string]bool{}
	_ = filepath.WalkDir(eventsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".md") {
			//nolint:nilerr // an unreadable entry cannot authorize dismissal; keep walking
			return nil
		}
		doc, err := mdstore.ReadFile(path)
		if err != nil {
			//nolint:nilerr // malformed records fail closed and do not hide proposals
			return nil
		}
		about, valid := validDismissal(doc)
		if valid {
			out[about] = true
		}
		return nil
	})
	return out
}
