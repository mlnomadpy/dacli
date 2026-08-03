package model

import "testing"

// A `wont` priority is a recorded decision NOT to do the work. It used to fall
// through Rank's default arm to 3 — the same rank as an untriaged task — so
// `dacli next` handed "delete production (wont)" back as the top
// recommendation, the exact failure the priority exists to prevent (dacli 199).
func TestWontRanksBelowUntriagedAndIsUnschedulable(t *testing.T) {
	if got, want := PriorityWont.Rank(), Priority("").Rank(); got <= want {
		t.Errorf("PriorityWont.Rank() = %d, untriaged = %d; wont must rank strictly worse", got, want)
	}
	for _, p := range []Priority{PriorityMust, PriorityShould, PriorityCould, ""} {
		if !p.Schedulable() {
			t.Errorf("Priority(%q).Schedulable() = false; want true", p)
		}
	}
	if PriorityWont.Schedulable() {
		t.Error("PriorityWont.Schedulable() = true; a wont task must never be recommended or picked up")
	}
}

// An unrecognized note kind used to fall through to the "refs" folder, so a
// typo like `findings` wrote a note that never reached the brief's findings
// section and never got a trust grade — data loss reported as success.
func TestNoteKindValidIsAClosedSet(t *testing.T) {
	for _, k := range []NoteKind{NoteDecision, NoteFinding, NoteRef, NoteMetric} {
		if !k.Valid() {
			t.Errorf("NoteKind(%q).Valid() = false; want true", k)
		}
	}
	for _, k := range []NoteKind{"findings", "decisions", "", "Finding", "note"} {
		if NoteKind(k).Valid() {
			t.Errorf("NoteKind(%q).Valid() = true; want false (would silently misfile)", k)
		}
	}
}
