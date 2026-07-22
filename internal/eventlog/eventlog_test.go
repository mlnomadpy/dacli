package eventlog

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// TestListSurfacesMalformedEvent proves a corrupt event file is not silently
// dropped: the readable event still lists, and the parse failure is logged
// rather than hidden — the append-only log must never erase an event without
// signal.
func TestListSurfacesMalformedEvent(t *testing.T) {
	root := t.TempDir()
	w, err := workspace.Init(root, "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	// One well-formed event via the real append path.
	if _, err := Append(w, "a-x", model.EventComment, "", "", "hello"); err != nil {
		t.Fatalf("append: %v", err)
	}

	// One malformed event file dropped straight into the events tree — an
	// unterminated frontmatter block mdstore.Parse rejects.
	bad := filepath.Join(w.EventsDir(), "2026", "01", "01", "01ZZZ-a-y-comment.md")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bad, []byte("---\nid: broken\nno-closing-fence\n"), 0o644); err != nil {
		t.Fatalf("write bad: %v", err)
	}

	var logbuf bytes.Buffer
	log.SetOutput(&logbuf)
	defer log.SetOutput(os.Stderr)

	events, err := List(w, Query{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// The good event still comes through.
	if len(events) != 1 || events[0].Body != "hello" {
		t.Fatalf("expected the readable event to survive, got %+v", events)
	}
	// The bad event is surfaced, not hidden.
	if !strings.Contains(logbuf.String(), bad) {
		t.Fatalf("malformed event was dropped silently; log = %q", logbuf.String())
	}
}
