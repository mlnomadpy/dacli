package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// An explicit --slug bypasses Slugify, so CreateProject must reject a slug that
// is not a safe single path segment — otherwise it writes a project file
// outside the workspace, which `project rm` would then RemoveAll (dacli 163).
func TestCreateProjectRejectsTraversingSlug(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	// A well-formed explicit slug still works.
	if _, err := CreateProject(w, "a-root", "Core", "core", "", ""); err != nil {
		t.Fatalf("CreateProject with clean slug: %v", err)
	}

	for _, evil := range []string{"../../../../escape/p", "..", "a/b", "/abs"} {
		if _, err := CreateProject(w, "a-root", "X", evil, "", ""); err == nil {
			t.Errorf("CreateProject(slug=%q) succeeded; want rejection", evil)
		}
	}

	// Nothing must have been written above the workspace root.
	up := filepath.Join(w.Root, "..", "..", "escape")
	if _, err := os.Stat(up); err == nil {
		t.Fatalf("a traversing slug wrote outside the workspace at %s", up)
	}
}
