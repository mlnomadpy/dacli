package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadmeDoesNotCallPromoteStubbed guards against the status drift filed as
// task 283: `skill promote` (skillforge cmdPromote) and `shortcut promote`
// (shortcuts cmdPromote) are both shipped and tested, and clikit.Planned() has
// zero callers, so no doc may keep describing them as unimplemented stubs. The
// README is the front door; a stale "honest stubs that refuse" line there
// teaches a reader (or a supervising agent) that a working command is a no-op.
func TestReadmeDoesNotCallPromoteStubbed(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/cli -> repo root.
	readme := filepath.Join(wd, "..", "..", "README.md")
	raw, err := os.ReadFile(readme)
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := string(raw)

	// The exact stale phrasings task 283 removed. Their return would mean the
	// README again claims an implemented command refuses.
	for _, banned := range []string{
		"honest stubs that refuse",
		"Still stubbed",
	} {
		if strings.Contains(text, banned) {
			t.Errorf("README.md still contains %q — skill/shortcut promote are shipped, not stubs (task 283)", banned)
		}
	}

	// A defense against paraphrase: no line may pair a promote command with a
	// stub/planned/unimplemented claim.
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "promote") {
			continue
		}
		low := strings.ToLower(line)
		for _, bad := range []string{"stub", "planned", "unimplemented"} {
			if strings.Contains(low, bad) {
				t.Errorf("README.md line describes a promote command as %q, but both promote commands are implemented (task 283): %q", bad, strings.TrimSpace(line))
			}
		}
	}
}
