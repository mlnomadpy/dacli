package cli

import (
	"strings"
	"testing"
)

// Main intercepts "help" for usage before dispatch ever runs, so any command
// path beginning with "help" is silently unreachable. This shipped once —
// "help ask", "help answer", and "help escalate" were all dead on arrival,
// and the duplicate-path test could not see it.
func TestNoCommandShadowedByReservedWords(t *testing.T) {
	for _, c := range commands {
		first := strings.SplitN(c.Path, " ", 2)[0]
		switch first {
		case "help", "-h", "--help":
			t.Errorf("command %q is unreachable: %q is intercepted before dispatch", c.Path, first)
		}
	}
}

// A duplicate command silently shadows the later registration and prints
// twice in help. This shipped once already; it should not ship again.
func TestNoDuplicateCommandPaths(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range commands {
		if seen[c.Path] {
			t.Errorf("duplicate command path %q", c.Path)
		}
		seen[c.Path] = true
	}
}

func TestEveryCommandHasABrief(t *testing.T) {
	for _, c := range commands {
		if c.Path == "" {
			t.Error("command with an empty path")
		}
		if c.Brief == "" {
			t.Errorf("command %q has no description", c.Path)
		}
		if c.Run == nil {
			t.Errorf("command %q has no Run function", c.Path)
		}
	}
}

// Longest-path matching must prefer "task add" over any bare "task", so that
// adding a bare parent command later cannot hijack its subcommands.
func TestMatchPrefersLongestPath(t *testing.T) {
	cmd, rest := match([]string{"task", "add", "Audit the write paths"})
	if cmd == nil {
		t.Fatal("no match for \"task add\"")
	}
	if cmd.Path != "task add" {
		t.Errorf("matched %q, want \"task add\"", cmd.Path)
	}
	if len(rest) != 1 || rest[0] != "Audit the write paths" {
		t.Errorf("rest = %v, want the trailing argument only", rest)
	}
}

func TestMatchReturnsNilForUnknown(t *testing.T) {
	if cmd, _ := match([]string{"frobnicate"}); cmd != nil {
		t.Errorf("matched %q for an unknown command", cmd.Path)
	}
}

func TestUnknownCommandExitsNonZero(t *testing.T) {
	if code := Main([]string{"frobnicate"}); code == 0 {
		t.Error("unknown command should exit non-zero")
	}
}

func TestHelpExitsZero(t *testing.T) {
	if code := Main([]string{"help"}); code != 0 {
		t.Errorf("help exit code = %d, want 0", code)
	}
}
