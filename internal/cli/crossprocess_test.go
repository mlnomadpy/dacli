package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// CROSS-PROCESS safety, which the race detector cannot observe by construction.
//
// ci.yml has carried an honest scope note about this since the race job was
// added: -race instruments goroutines within ONE process, while dacli's
// sharpest concurrency risk is several dacli invocations read-modify-writing
// the same markdown at once. A green -race run is not evidence about that, and
// nothing else was evidence either.
//
// This is the test that produces it: N REAL binaries against ONE workspace,
// concurrently, asserting the invariant the seq lock exists to hold — every
// task gets a distinct number and none is lost. The failure it guards is not
// hypothetical; it is task 209, where two agents filing at once computed the
// same NNN and FindTask reported the ref ambiguous.
func TestConcurrentProcessesNeverShareOrLoseASeq(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	gitInit(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a real goal for this project")

	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c := exec.Command(bin, "task", "add", fmt.Sprintf("concurrent task %d", n),
				"--project", "p", "--accept", "it exists")
			c.Dir = dir
			if out, err := c.CombinedOutput(); err != nil {
				errs <- fmt.Errorf("writer %d: %w\n%s", n, err, out)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("a concurrent writer failed: %v", err)
	}

	// EVERY task survived. A lost write is the quieter half of the defect and
	// the one an aggregate count would hide.
	listing := run(t, dir, 0, "task", "list", "--project", "p")
	for i := 0; i < writers; i++ {
		if !strings.Contains(listing, fmt.Sprintf("concurrent task %d", i)) {
			t.Errorf("task %d was lost under concurrent creation:\n%s", i, listing)
		}
	}

	// And every seq is DISTINCT. Two tasks sharing a number are unaddressable
	// by ref — the exact symptom of task 209.
	seqs := map[string]string{}
	for _, line := range strings.Split(listing, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || !strings.Contains(f[1], "-") {
			continue
		}
		seq := strings.SplitN(f[1], "-", 2)[0]
		if prev, dup := seqs[seq]; dup {
			t.Errorf("seq %s was handed out twice: %q and %q", seq, prev, f[1])
		}
		seqs[seq] = f[1]
	}
	if len(seqs) != writers {
		t.Errorf("got %d distinct seqs from %d concurrent writers:\n%s", len(seqs), writers, listing)
	}
}

// The same question for the NOTE store, where several processes write into
// one directory at once and each note's filename is derived from its title.
//
// Notes rather than events, deliberately: as the ROOT identity `note add`
// writes the durable note directly — the propose-then-sync event path is what
// a non-owner takes — so asserting on the event log here would be asserting on
// a channel this caller never uses. Measured before writing the assertion, not
// assumed.
func TestConcurrentProcessesNeverLoseANote(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	gitInit(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "a real goal for this project")

	const writers = 8
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c := exec.Command(bin, "note", "add", "finding",
				fmt.Sprintf("concurrent finding %d", n), "--project", "p",
				"--origin", "x.go:1", "--body", "a body")
			c.Dir = dir
			_, _ = c.CombinedOutput()
		}(i)
	}
	wg.Wait()

	entries, err := os.ReadDir(filepath.Join(dir, ".dacli", "projects", "p", "notes", "findings"))
	if err != nil {
		t.Fatalf("reading the notes directory: %v", err)
	}
	names := make(map[string]bool, len(entries))
	for _, e := range entries {
		names[e.Name()] = true
	}
	missing := 0
	for i := 0; i < writers; i++ {
		want := fmt.Sprintf("concurrent-finding-%d.md", i)
		if !names[want] {
			missing++
			t.Errorf("note %q is missing after concurrent creation", want)
		}
	}
	if missing == 0 && len(entries) != writers {
		t.Errorf("got %d note files from %d writers — a write was duplicated", len(entries), writers)
	}
}
