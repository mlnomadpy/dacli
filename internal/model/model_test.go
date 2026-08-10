package model

import (
	"strings"
	"testing"
)

// An unrecognized status matched no folder and produced an EMPTY list with a
// nil error, so `dacli task list --status closed` (the user meant done)
// printed nothing and exited 0 — indistinguishable from an empty backlog.
// Flags.Reject already refuses an unknown flag NAME loudly; an unknown flag
// VALUE deserves the same answer rather than a plausible-but-wrong one
// (task 322, filed by the loop's own review agent against a flag I had added
// an hour earlier).
func TestParseStatusRefusesAnUnknownValue(t *testing.T) {
	for _, bad := range []string{"closed", "opne", "DONE", "in-progress"} {
		if _, err := ParseStatus(bad); err == nil {
			t.Errorf("ParseStatus(%q) was accepted; a typo must not read as an empty backlog", bad)
		} else if !strings.Contains(err.Error(), bad) {
			t.Errorf("ParseStatus(%q): the error must name the bad value, got %v", bad, err)
		}
	}

	// Empty means "every status" — that is how the list commands express
	// "no filter", and it must keep working.
	if got, err := ParseStatus(""); err != nil || got != "" {
		t.Errorf(`ParseStatus("") = %q, %v; want the no-filter sentinel and no error`, got, err)
	}

	// Every canonical status still round-trips.
	for _, st := range AllStatuses {
		if got, err := ParseStatus(string(st)); err != nil || got != st {
			t.Errorf("ParseStatus(%q) = %q, %v; want it accepted", st, got, err)
		}
	}
}

// The refusal must name the allowed set, or a caller who typo'd learns only
// that they were wrong, not what is right.
func TestParseStatusNamesTheAllowedSet(t *testing.T) {
	_, err := ParseStatus("closed")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, st := range AllStatuses {
		if !strings.Contains(err.Error(), string(st)) {
			t.Errorf("the refusal must list %q as an option: %v", st, err)
		}
	}
}
