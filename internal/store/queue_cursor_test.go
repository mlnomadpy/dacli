package store

import "testing"

// Queue files are hand-editable by design — `dacli queue advance` even tells the
// operator to edit one to resume — so Next() is exposed to a cursor it did not
// write. It guarded the upper bound but not a negative one, so `cursor: -1`
// indexed Steps[-1] and panicked the command outright (dacli 199).
func TestQueueNextToleratesAnOutOfRangeCursor(t *testing.T) {
	q := &Queue{Steps: []string{"first", "second"}}

	for _, tc := range []struct {
		cursor   int
		wantStep string
		wantDone bool
		scenario string
	}{
		{0, "first", false, "start"},
		{1, "second", false, "middle"},
		{2, "", true, "past the end is done"},
		{99, "", true, "far past the end is done"},
		{-1, "first", false, "a negative cursor must clamp to the start, not panic"},
		{-99, "first", false, "a wildly negative cursor must clamp too"},
	} {
		q.Cursor = tc.cursor
		step, done := q.Next() // must not panic
		if step != tc.wantStep || done != tc.wantDone {
			t.Errorf("%s: cursor=%d Next() = (%q, %v), want (%q, %v)",
				tc.scenario, tc.cursor, step, done, tc.wantStep, tc.wantDone)
		}
	}
}

// An empty queue is complete, not a crash.
func TestQueueNextOnEmptySteps(t *testing.T) {
	q := &Queue{}
	for _, c := range []int{-1, 0, 1} {
		q.Cursor = c
		if step, done := q.Next(); step != "" || !done {
			t.Errorf("empty queue at cursor %d = (%q, %v), want (\"\", true)", c, step, done)
		}
	}
}
