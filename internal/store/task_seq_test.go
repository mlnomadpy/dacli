package store

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func seqLockFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.ProjectDir(project), ".seq.lock")
}

// seqLockBody renders a lock file exactly as a holder would have written it,
// dated age ago. Tests plant these instead of sleeping: the state an acquirer
// must react to (live / crashed / ancient) is written down, not raced for.
func seqLockBody(pid int, pidStart, host, token string, age time.Duration) string {
	return fmt.Sprintf("{\"pid\":%d,\"pid_start\":%q,\"host\":%q,\"token\":%q,\"ts\":%q}\n",
		pid, pidStart, host, token, time.Now().Add(-age).Format(time.RFC3339Nano))
}

// plantSeqLock writes body as the project's seq lock and back-dates its mtime,
// so content-age and file-age agree.
func plantSeqLock(t *testing.T, w *workspace.Workspace, project, body string, age time.Duration) string {
	t.Helper()
	path := seqLockFile(w, project)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("plant lock: %v", err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("backdate lock: %v", err)
	}
	return path
}

// deadPID returns the pid of a process that has already exited and been reaped
// — a crashed holder, without having to crash one.
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	return cmd.Process.Pid
}

// createConcurrently fires n CreateTask calls at once and returns what each got.
func createConcurrently(t *testing.T, w *workspace.Workspace, n int) ([]*Task, []error) {
	t.Helper()
	var wg sync.WaitGroup
	results := make([]*Task, n)
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = CreateTask(w, "a-root", "core", fmt.Sprintf("concurrent task %d", i), TaskOpts{})
		}(i)
	}
	close(start)
	wg.Wait()
	return results, errs
}

// assertDistinctSeqs is the invariant 209 bought and 247 defends: every task
// that was actually created owns its NNN alone, on disk and through FindTask.
func assertDistinctSeqs(t *testing.T, w *workspace.Workspace, tasks []*Task) {
	t.Helper()
	seen := map[int]string{}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if prev, ok := seen[task.Seq]; ok {
			t.Fatalf("seq %d allocated to both %q and %q", task.Seq, prev, task.Slug)
		}
		seen[task.Seq] = task.Slug
	}

	all, err := ListTasks(w, "core", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != len(seen) {
		t.Fatalf("expected %d tasks on disk, got %d (a seq collision let one file overwrite another)", len(seen), len(all))
	}

	// FindTask must resolve every seq unambiguously.
	for seq, slug := range seen {
		got, err := FindTask(w, fmt.Sprintf("%d", seq))
		if err != nil {
			t.Fatalf("FindTask(%d): %v", seq, err)
		}
		if got.Slug != slug {
			t.Fatalf("FindTask(%d) = %q, want %q", seq, got.Slug, slug)
		}
	}
}

// Two agents filing tasks at the same moment must never land on the same
// seq: a shared NNN makes both tasks unaddressable by number and trips
// FindTask's ambiguity check (dacli 209). The lock that serializes them must
// hold that line even when it has to break a lock left by a dead holder
// (dacli 247) — a steal that hands the "lock" to two waiters at once puts the
// duplicate seq straight back.
func TestCreateTaskConcurrentGetsDistinctSeqs(t *testing.T) {
	const n = 20

	t.Run("uncontended lock file", func(t *testing.T) {
		w := indexWorkspace(t)
		tasks, errs := createConcurrently(t, w, n)
		for i, err := range errs {
			if err != nil {
				t.Fatalf("CreateTask[%d]: %v", i, err)
			}
		}
		assertDistinctSeqs(t, w, tasks)
	})

	// A holder that crashed mid-CreateTask leaves its lock file behind. Every
	// waiter must still end up with its own seq: the steal has to elect one
	// winner, not release twenty.
	t.Run("crashed holder past seqLockTimeout", func(t *testing.T) {
		w := indexWorkspace(t)
		host, _ := os.Hostname()
		plantSeqLock(t, w, "core", seqLockBody(deadPID(t), "", host, "01AAAAAAAAAAAAAAAAAAAAAAAA", 2*seqLockTimeout), 2*seqLockTimeout)

		tasks, errs := createConcurrently(t, w, n)
		for i, err := range errs {
			if err != nil {
				t.Fatalf("CreateTask[%d] did not break the crashed holder's lock: %v", i, err)
			}
		}
		assertDistinctSeqs(t, w, tasks)
		if _, err := os.Stat(seqLockFile(w, "core")); !os.IsNotExist(err) {
			t.Fatalf("lock file survived the burst: %v", err)
		}
	})

	// An ancient lock from another machine: no pid we can probe, so age is all
	// we have. Same requirement — one winner.
	t.Run("ancient lock from another host", func(t *testing.T) {
		w := indexWorkspace(t)
		plantSeqLock(t, w, "core", seqLockBody(31337, "", "some-other-host.invalid", "01BBBBBBBBBBBBBBBBBBBBBBBB", 2*time.Hour), 2*time.Hour)

		tasks, errs := createConcurrently(t, w, n)
		for i, err := range errs {
			if err != nil {
				t.Fatalf("CreateTask[%d] did not break the ancient lock: %v", i, err)
			}
		}
		assertDistinctSeqs(t, w, tasks)
	})

	// The other half of the acceptance: a LIVE holder that outlives
	// seqLockTimeout is not stale. Waiters must fail rather than allocate a
	// seq behind its back, and the tasks that do exist stay distinct.
	t.Run("live holder past seqLockTimeout", func(t *testing.T) {
		w := indexWorkspace(t)
		mine, err := CreateTask(w, "a-root", "core", "held", TaskOpts{})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		unlock, err := acquireSeqLock(w, "core")
		if err != nil {
			t.Fatalf("acquire: %v", err)
		}
		tasks, errs := createConcurrently(t, w, 4)
		for i, err := range errs {
			if err == nil {
				t.Fatalf("CreateTask[%d] allocated seq %d while the lock was held by a live process", i, tasks[i].Seq)
			}
		}
		unlock()

		// Released cleanly, the next callers pick up where the holder left off.
		after, errs := createConcurrently(t, w, 4)
		for i, err := range errs {
			if err != nil {
				t.Fatalf("CreateTask[%d] after release: %v", i, err)
			}
		}
		assertDistinctSeqs(t, w, append(after, mine))
	})
}

// A waiter that cannot get the lock must back off and eventually give up. It
// must never take a lock whose owner is alive: that is the ownerless steal
// dacli 247 filed, and it hands the same seq to two callers.
func TestAcquireSeqLockNeverStealsFromALiveHolder(t *testing.T) {
	w := indexWorkspace(t)
	unlock, err := acquireSeqLock(w, "core")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer unlock()

	start := time.Now()
	second, err := acquireSeqLock(w, "core")
	if err == nil {
		second()
		t.Fatal("second acquire took a lock still held by this live process")
	}
	if waited := time.Since(start); waited < seqLockTimeout {
		t.Fatalf("gave up after %v, before seqLockTimeout (%v): the wait is not backing off", waited, seqLockTimeout)
	}
}

// Unlock must remove the lock it owns, not whatever file happens to sit at the
// path. If our lock was replaced while we held it, the replacement belongs to
// someone else and deleting it un-serializes them.
func TestSeqLockReleaseOnlyRemovesItsOwnLock(t *testing.T) {
	w := indexWorkspace(t)
	unlock, err := acquireSeqLock(w, "core")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	// Someone stole our lock and is now holding their own.
	path := seqLockFile(w, "core")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	host, _ := os.Hostname()
	plantSeqLock(t, w, "core", seqLockBody(os.Getpid(), "", host, "01CCCCCCCCCCCCCCCCCCCCCCCC", 0), 0)

	unlock()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("release deleted the current holder's lock: %v", err)
	}
	if want := "01CCCCCCCCCCCCCCCCCCCCCCCC"; !strings.Contains(string(body), want) {
		t.Fatalf("lock file is not the holder's any more: %q", body)
	}
}

// A lock file that cannot be read as a complete record is a holder mid-write
// until proven otherwise: unreadable is not stale. Only file age, well past the
// staleness horizon, makes an unreadable lock breakable.
func TestAcquireSeqLockTreatsAnUnreadableLockAsLive(t *testing.T) {
	w := indexWorkspace(t)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	var now time.Time
	clock := seqLockClock{
		now: func() time.Time { return now },
		sleep: func(d time.Duration) {
			now = now.Add(d)
		},
	}
	plantAt := func(body string, modified time.Time) string {
		t.Helper()
		path := plantSeqLock(t, w, "core", body, 0)
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatalf("set controlled lock time: %v", err)
		}
		return path
	}

	t.Run("partial write", func(t *testing.T) {
		now = base
		path := plantAt(`{"pid":4242,"ho`, base)
		defer func() { _ = os.Remove(seqLockFile(w, "core")) }()
		unlock, err := acquireFileLockWithClock(path, clock)
		if err == nil {
			unlock()
			t.Fatal("broke a lock that was mid-write")
		}
		if elapsed := now.Sub(base); elapsed < seqLockTimeout || elapsed >= seqLockStaleAfter {
			t.Fatalf("controlled waiter elapsed %v; want timeout reached before stale horizon", elapsed)
		}
	})

	t.Run("ancient contentless lock", func(t *testing.T) {
		now = base
		path := plantAt("", base.Add(-2*time.Hour))
		unlock, err := acquireFileLockWithClock(path, clock)
		if err != nil {
			t.Fatalf("an ancient contentless lock wedges the project forever: %v", err)
		}
		unlock()
	})
}
