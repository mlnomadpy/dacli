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
	MaxCycles      int           // cycles allowed in THIS invocation; 0 = perpetual. Gates cyclesThisRun, never the persisted cumulative count, so a fresh bounded run always gets its full budget regardless of how many cycles a prior invocation already ran.
	StopFile       string        // absolute path; its existence halts the loop
	NoProgressHalt int           // halt after this many consecutive 0-landed cycles; 0 disables

	// State — mutated as the loop runs.
	windowStart   time.Time
	windowSpent   int64
	zeroStreak    int
	cycle         int  // cumulative across restarts (persisted) — for reporting/resume only
	cyclesThisRun int  // cycles completed by THIS process (never persisted/restored) — what --max-cycles gates
	stopSeen      bool // the stop file has been observed at least once; never un-set (see StopRequested)
}

// DefaultWindowDur is the rolling budget window a token ceiling falls back to
// when none usable was configured — see windowDur.
const DefaultWindowDur = 24 * time.Hour

// windowDur is the EFFECTIVE budget window. A zero WindowDur alongside a live
// WindowTokens made `now.Sub(windowStart) >= WindowDur` true on every call, so
// the window rolled — and windowSpent reset to 0 — before it was ever compared
// against the ceiling: the budget silently disabled itself. `--budget-window 0`
// parses cleanly, so an operator who asked for a token ceiling got none at all
// (dacli 218). cmdLoop refuses that flag combination up front; this defaults it
// for every other construction path, because a budget that quietly does nothing
// is worse than no budget at all.
func (g *Governor) windowDur() time.Duration {
	if g.WindowDur <= 0 {
		return DefaultWindowDur
	}
	return g.WindowDur
}

// StopRequested reports whether the operator's stop file is present — and it
// LATCHES: once seen, this Governor stays stopped even if the file later
// disappears. STOP lives inside the very repo the loop's own children edit, so
// a child agent (or a stray `git clean -fdx`) can delete the operator's only
// kill switch; without the latch that silently revokes it mid-run. Resuming is
// a restart — an operator affordance, never something a governed child can
// grant itself (dacli 207).
func (g *Governor) StopRequested() bool {
	if g.stopSeen {
		return true
	}
	if g.StopFile == "" {
		return false
	}
	if _, err := os.Stat(g.StopFile); err == nil {
		g.stopSeen = true
	}
	return g.stopSeen
}

// StopReason is the halt message for a stop-file halt, shared by every place
// that re-checks it — Before once per cycle is not often enough, since a whole
// wave of children runs between two Before calls (dacli 207).
func (g *Governor) StopReason() string {
	return fmt.Sprintf("stop file present (%s) — remove it to resume", g.StopFile)
}

// Cycle reports how many cycles have completed, cumulative across restarts.
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
// restarted loop resumes its cumulative cycle count (for reporting), budget
// window, and thrash streak instead of starting over. It deliberately does
// NOT touch cyclesThisRun: --max-cycles bounds what THIS invocation runs, so
// a fresh bounded run always gets its full budget no matter how many cycles
// a prior invocation already completed — see MaxCycles.
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
	if g.StopRequested() {
		return Halt, g.StopReason()
	}
	if g.MaxCycles > 0 && g.cyclesThisRun >= g.MaxCycles {
		return Halt, fmt.Sprintf("reached --max-cycles %d", g.MaxCycles)
	}
	if g.WindowTokens > 0 {
		if g.windowStart.IsZero() {
			g.windowStart = now
		}
		if now.Sub(g.windowStart) >= g.windowDur() {
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

// CountIdleCycle records that an idle tick occurred so it counts toward
// --max-cycles. An idle tick is not a sprint — it touches neither the
// persisted cycle count nor the thrash streak — but it IS one iteration of
// this invocation, and --max-cycles bounds iterations. Without this a bounded
// run on a permanently empty backlog (`loop --max-cycles 1`) would idle
// forever, spawning a review agent every interval (dacli 172).
func (g *Governor) CountIdleCycle() { g.cyclesThisRun++ }

// MaxCyclesReached reports whether this invocation has hit its --max-cycles
// bound. Lets the idle branch stop before sleeping, rather than sleeping a full
// interval only for the next Before() to halt.
func (g *Governor) MaxCyclesReached() bool {
	return g.MaxCycles > 0 && g.cyclesThisRun >= g.MaxCycles
}

// ChargeIdleTokens adds tokens spent by an Idle-branch review spawn to the
// current window. Unlike AfterCycle, it does not advance the cycle counter or
// touch the thrash streak — an idle tick regenerates backlog, it does not
// complete a sprint — but its spend must still count against --window-tokens,
// which otherwise never trips while the loop idles.
func (g *Governor) ChargeIdleTokens(tokens int64) {
	g.windowSpent += tokens
}

// AfterCycle records the outcome of a completed cycle (tasks landed on trunk,
// tokens spent) and decides whether the loop may continue. It advances the
// cycle counter, so it must be called exactly once per executed cycle.
func (g *Governor) AfterCycle(landed int, tokens int64) (Decision, string) {
	g.cycle++
	g.cyclesThisRun++
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

// AfterCycleUnmeasured records a completed cycle whose trunk advancement could
// not be MEASURED — git briefly unavailable, an index lock, a timed-out
// rev-list. It charges the tokens and advances the cycle counters exactly like
// AfterCycle, but deliberately leaves the thrash streak alone in both
// directions: it neither increments it (a failed measurement is not evidence of
// no progress, and feeding it in as a zero walks the loop toward a false halt)
// nor resets it (a failed measurement is not evidence of progress either, and
// resetting would let a genuinely stuck loop launder its streak away every time
// git hiccuped). "Could not measure" is its own answer — see dacli 212 and the
// caller in loop(), which also holds the previous marker rather than rebasing
// on a fabricated zero.
func (g *Governor) AfterCycleUnmeasured(tokens int64) (Decision, string) {
	g.cycle++
	g.cyclesThisRun++
	g.windowSpent += tokens
	return Proceed, ""
}

// WindowRemaining is how long until the current budget window resets. Zero when
// no window is configured.
func (g *Governor) WindowRemaining(now time.Time) time.Duration {
	if g.WindowTokens <= 0 || g.windowStart.IsZero() {
		return 0
	}
	rem := g.windowDur() - now.Sub(g.windowStart)
	if rem < 0 {
		return 0
	}
	return rem
}
