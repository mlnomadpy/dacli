package orchestration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGovernorProceedsWithBacklog(t *testing.T) {
	g := &Governor{}
	d, _ := g.Before(3, time.Unix(0, 0))
	if d != Proceed {
		t.Fatalf("want Proceed with a non-empty backlog, got %s", d)
	}
}

func TestGovernorIdlesOnEmptyBacklog(t *testing.T) {
	g := &Governor{Idle: time.Minute}
	d, why := g.Before(0, time.Unix(0, 0))
	if d != Idle {
		t.Fatalf("want Idle on empty backlog, got %s", d)
	}
	if why == "" {
		t.Fatal("Idle decision must explain itself")
	}
}

func TestGovernorHaltsAtMaxCycles(t *testing.T) {
	g := &Governor{MaxCycles: 2}
	// Two cycles complete, then Before must halt.
	for i := 0; i < 2; i++ {
		if d, _ := g.Before(1, time.Unix(0, 0)); d != Proceed {
			t.Fatalf("cycle %d: want Proceed, got %s", i, d)
		}
		g.AfterCycle(1, 0)
	}
	if d, _ := g.Before(1, time.Unix(0, 0)); d != Halt {
		t.Fatalf("want Halt after max cycles, got %s", d)
	}
}

func TestGovernorStopFileHalts(t *testing.T) {
	dir := t.TempDir()
	stop := filepath.Join(dir, "STOP")
	g := &Governor{StopFile: stop}
	if d, _ := g.Before(1, time.Unix(0, 0)); d != Proceed {
		t.Fatalf("want Proceed before stop file, got %s", d)
	}
	if err := os.WriteFile(stop, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d, _ := g.Before(1, time.Unix(0, 0)); d != Halt {
		t.Fatalf("want Halt with stop file present, got %s", d)
	}
}

func TestGovernorTokenWindowSleepsThenResets(t *testing.T) {
	base := time.Unix(1_000_000, 0)
	g := &Governor{WindowDur: time.Hour, WindowTokens: 100}
	// Spend the whole window in one cycle.
	if d, _ := g.Before(1, base); d != Proceed {
		t.Fatalf("first cycle should proceed, got %s", d)
	}
	g.AfterCycle(1, 120) // overspend

	// Same window: must sleep.
	if d, _ := g.Before(1, base.Add(time.Minute)); d != SleepWindow {
		t.Fatalf("want SleepWindow after exhausting budget, got %s", d)
	}
	// After the window elapses: fresh allowance, proceed again.
	if d, _ := g.Before(1, base.Add(2*time.Hour)); d != Proceed {
		t.Fatalf("want Proceed after window reset, got %s", d)
	}
	if g.WindowSpent() != 0 {
		t.Fatalf("window spend should reset to 0, got %d", g.WindowSpent())
	}
}

func TestGovernorThrashGuardHalts(t *testing.T) {
	g := &Governor{NoProgressHalt: 2}
	// First zero-progress cycle: still allowed.
	if d, _ := g.AfterCycle(0, 10); d != Proceed {
		t.Fatalf("one zero cycle should not halt, got %s", d)
	}
	// Second consecutive zero-progress cycle: halt.
	if d, why := g.AfterCycle(0, 10); d != Halt {
		t.Fatalf("want Halt after 2 zero cycles, got %s (%s)", d, why)
	}
}

func TestGovernorThrashStreakResetsOnProgress(t *testing.T) {
	g := &Governor{NoProgressHalt: 2}
	g.AfterCycle(0, 10) // zero
	g.AfterCycle(3, 10) // progress resets the streak
	if d, _ := g.AfterCycle(0, 10); d != Proceed {
		t.Fatalf("streak should have reset; want Proceed, got %s", d)
	}
}
