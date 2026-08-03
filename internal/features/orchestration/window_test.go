package orchestration

import (
	"testing"
	"time"
)

// WindowRemaining decides how long the loop sleeps once its token budget is
// spent — and nothing asserted it. Mutation testing found the guard
// `WindowTokens <= 0` unprotected: flipping it to `< 0` survived the suite,
// which would make an UNBUDGETED loop (WindowTokens == 0, the default) compute
// a real sleep duration instead of zero, so a loop with no budget could park
// itself for the length of a window it never opted into (dacli 205).
func TestWindowRemaining(t *testing.T) {
	base := time.Unix(1_000_000, 0)

	t.Run("no budget configured means no wait", func(t *testing.T) {
		g := &Governor{WindowDur: time.Hour} // WindowTokens deliberately 0
		g.Before(1, base)                    // would start a window if one were configured
		if got := g.WindowRemaining(base); got != 0 {
			t.Errorf("WindowRemaining with WindowTokens=0 = %s, want 0 — an unbudgeted loop must never sleep on a window", got)
		}
	})

	t.Run("negative budget also means no wait", func(t *testing.T) {
		g := &Governor{WindowDur: time.Hour, WindowTokens: -1}
		if got := g.WindowRemaining(base); got != 0 {
			t.Errorf("WindowRemaining with a negative budget = %s, want 0", got)
		}
	})

	t.Run("no window started yet means no wait", func(t *testing.T) {
		g := &Governor{WindowDur: time.Hour, WindowTokens: 100}
		if got := g.WindowRemaining(base); got != 0 {
			t.Errorf("WindowRemaining before any window opened = %s, want 0", got)
		}
	})

	t.Run("counts down from the window start", func(t *testing.T) {
		g := &Governor{WindowDur: time.Hour, WindowTokens: 100}
		g.Before(1, base) // opens the window at base
		if got := g.WindowRemaining(base.Add(20 * time.Minute)); got != 40*time.Minute {
			t.Errorf("WindowRemaining 20m into a 1h window = %s, want 40m", got)
		}
	})

	t.Run("never negative once the window has elapsed", func(t *testing.T) {
		g := &Governor{WindowDur: time.Hour, WindowTokens: 100}
		g.Before(1, base)
		if got := g.WindowRemaining(base.Add(3 * time.Hour)); got != 0 {
			t.Errorf("WindowRemaining past the window = %s, want 0 (a negative sleep is not a sleep)", got)
		}
	})
}
