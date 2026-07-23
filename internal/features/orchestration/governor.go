package orchestration

import (
	"fmt"
	"os"
	"time"
)

// Decision is the governor's verdict at a checkpoint. The whole point of a
// self-governed perpetual loop is that "keep going" is a *derived* answer, not
// the default — every cycle passes through the governor, and the governor is
// the only thing allowed to say Proceed.
type Decision int

const (
	// Proceed: run the cycle (or continue after one).
	Proceed Decision = iota
	// Idle: nothing evidence-based to do; sleep the idle interval and re-scan.
	// The loop does NOT invent work to fill an empty backlog — that is the
	// difference between a maintenance team and a runaway refactor.
	Idle
	// SleepWindow: the rolling token budget for this window is spent; sleep
	// until the window resets, then resume.
	SleepWindow
	// Halt: stop the loop entirely. A halt is terminal — the operator restarts.
	Halt
)

func (d Decision) String() string {
	switch d {
	case Proceed:
		return "proceed"
	case Idle:
		return "idle"
	case SleepWindow:
		return "sleep-window"
	case Halt:
		return "halt"
	default:
		return "unknown"
	}
}

// Governor holds the loop's policy knobs and its running state. It is a pure
// decision engine: it never spawns, never sleeps, never touches the network —
// it is handed facts (backlog size, tokens spent, wall clock) and returns a
// Decision. That purity is what makes the perpetual machine testable without
// burning a single token.
type Governor struct {
	// Policy — set once from flags.
	WindowDur      time.Duration // rolling budget window; 0 disables the window
	WindowTokens   int64         // tokens allowed per window; 0 = unlimited
	Idle           time.Duration // how long to sleep when the backlog is empty
	MaxCycles      int           // 0 = perpetual
	StopFile       string        // absolute path; its existence halts the loop
	NoProgressHalt int           // halt after this many consecutive 0-landed cycles; 0 disables

	// State — mutated as the loop runs.
	windowStart time.Time
	windowSpent int64
	zeroStreak  int
	cycle       int
}

// Cycle reports how many cycles have completed.
func (g *Governor) Cycle() int { return g.cycle }

// WindowSpent reports tokens charged against the current window.
func (g *Governor) WindowSpent() int64 { return g.windowSpent }

// WindowStart reports when the current budget window began (zero if none has
// started yet).
func (g *Governor) WindowStart() time.Time { return g.windowStart }

// ZeroStreak reports the number of consecutive zero-progress cycles seen so
// far — the thrash guard's running counter.
func (g *Governor) ZeroStreak() int { return g.zeroStreak }

// governorState is a persistable snapshot of a Governor's mutable counters —
// the state a restart must resume rather than reset.
type governorState struct {
	Cycle       int
	WindowStart time.Time
	WindowSpent int64
	ZeroStreak  int
}

// State returns a persistable snapshot of the governor's running counters.
func (g *Governor) State() governorState {
	return governorState{
		Cycle:       g.cycle,
		WindowStart: g.windowStart,
		WindowSpent: g.windowSpent,
		ZeroStreak:  g.zeroStreak,
	}
}

// Restore loads a previously persisted snapshot into the governor, so a
// restarted loop resumes its cycle count, budget window, and thrash streak
// instead of starting over.
func (g *Governor) Restore(st governorState) {
	g.cycle = st.Cycle
	g.windowStart = st.WindowStart
	g.windowSpent = st.WindowSpent
	g.zeroStreak = st.ZeroStreak
}

// Before decides whether to start a cycle, given the ready backlog size and the
// current time. It may reset the budget window as a side effect (a window that
// has elapsed rolls over to a fresh allowance).
func (g *Governor) Before(backlog int, now time.Time) (Decision, string) {
	if g.StopFile != "" {
		if _, err := os.Stat(g.StopFile); err == nil {
			return Halt, fmt.Sprintf("stop file present (%s) — remove it to resume", g.StopFile)
		}
	}
	if g.MaxCycles > 0 && g.cycle >= g.MaxCycles {
		return Halt, fmt.Sprintf("reached --max-cycles %d", g.MaxCycles)
	}
	if g.WindowTokens > 0 {
		if g.windowStart.IsZero() {
			g.windowStart = now
		}
		if now.Sub(g.windowStart) >= g.WindowDur {
			// Window elapsed: roll to a fresh allowance.
			g.windowStart = now
			g.windowSpent = 0
		}
		if g.windowSpent >= g.WindowTokens {
			return SleepWindow, fmt.Sprintf("token window exhausted (%d/%d) — sleeping until it resets", g.windowSpent, g.WindowTokens)
		}
	}
	if backlog == 0 {
		return Idle, "backlog empty — no evidence-based work; idling rather than inventing work"
	}
	return Proceed, ""
}

// AfterCycle records the outcome of a completed cycle (tasks landed on trunk,
// tokens spent) and decides whether the loop may continue. It advances the
// cycle counter, so it must be called exactly once per executed cycle.
func (g *Governor) AfterCycle(landed int, tokens int64) (Decision, string) {
	g.cycle++
	g.windowSpent += tokens
	if landed == 0 {
		g.zeroStreak++
	} else {
		g.zeroStreak = 0
	}
	if g.NoProgressHalt > 0 && g.zeroStreak >= g.NoProgressHalt {
		return Halt, fmt.Sprintf("no net progress for %d consecutive cycles — thrash guard tripped", g.zeroStreak)
	}
	return Proceed, ""
}

// WindowRemaining is how long until the current budget window resets. Zero when
// no window is configured.
func (g *Governor) WindowRemaining(now time.Time) time.Duration {
	if g.WindowTokens <= 0 || g.windowStart.IsZero() {
		return 0
	}
	rem := g.WindowDur - now.Sub(g.windowStart)
	if rem < 0 {
		return 0
	}
	return rem
}
