package cli

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
)

// github push is a public process boundary: the publisher must remain its
// child until the last remote mutation finishes. A handler-only test cannot
// catch the regression where the process printed its plan and exited while a
// publisher continued holding the repository lease (issue #797).
func TestGitHubPushStreamsAndReleasesPublisherLockBeforeExit(t *testing.T) {
	bin := buildDacli(t)
	dir := t.TempDir()
	ghDir := filepath.Join(dir, "fake-gh")
	if err := os.MkdirAll(ghDir, 0o755); err != nil {
		t.Fatal(err)
	}
	release := filepath.Join(dir, "release-publisher")
	writeFakeGHPublisher(t, filepath.Join(ghDir, "gh"))
	env := append(os.Environ(), "PATH="+ghDir+string(os.PathListSeparator)+os.Getenv("PATH"), "DACLI_GH_RELEASE="+release)

	runExternal(t, bin, dir, env, "init", "--name", "x")
	runExternal(t, bin, dir, env, "project", "add", "P", "--slug", "p", "--goal", "publish through one owner")
	runExternal(t, bin, dir, env, "github", "link", "p")
	runExternal(t, bin, dir, env, "task", "add", "Publish first record", "--project", "p")
	runExternal(t, bin, dir, env, "task", "add", "Reconcile archival ledger", "--project", "p")

	cmd := exec.Command(bin, "github", "push", "p")
	cmd.Dir = dir
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	lines := make(chan string, 16)
	readDone := make(chan error, 1)
	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			lines <- s.Text()
		}
		readDone <- s.Err()
	}()
	waitForPushLine(t, lines, "plan: will create 2 task")
	waitForPushLine(t, lines, "publisher: creating issue")

	// The stream tells us the publisher has reached its deliberate pause. The
	// public command must still be alive and own the lock at that instant.
	if !procmon.Alive(cmd.Process.Pid) {
		t.Fatalf("github push returned while publisher was active\nstderr: %s", stderr.String())
	}
	lock := testGitHubPushLock(dir, "owner/repo")
	if _, err := os.Stat(lock); err != nil {
		t.Fatalf("active publisher did not own github-push lock: %v", err)
	}
	if err := os.WriteFile(release, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("github push failed after publisher release: %v\nstderr: %s", err, stderr.String())
	}
	if err := <-readDone; err != nil {
		t.Fatalf("read github push stream: %v", err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("github push returned with publisher lock still present: %v", err)
	}

	// A completed push leaves neither a live publisher nor its lease behind;
	// the immediate idempotent retry must therefore return without waiting.
	second := exec.Command(bin, "github", "push", "p")
	second.Dir = dir
	second.Env = env
	if out, err := second.CombinedOutput(); err != nil {
		t.Fatalf("immediate idempotent retry: %v\n%s", err, out)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("retry returned with publisher lock still present: %v", err)
	}
}

func TestGitHubPushFailureAndCancellationDrainPublisherBeforeExit(t *testing.T) {
	bin := buildDacli(t)
	for _, tc := range []struct {
		name   string
		mode   string
		cancel bool
	}{
		{name: "failure", mode: "failure"},
		{name: "cancellation", mode: "cancel", cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ghDir := filepath.Join(dir, "fake-gh")
			if err := os.MkdirAll(ghDir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeFakeGHPublisher(t, filepath.Join(ghDir, "gh"))
			pidFile := filepath.Join(dir, "publisher.pid")
			env := append(os.Environ(),
				"PATH="+ghDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"DACLI_GH_MODE="+tc.mode,
				"DACLI_GH_PID="+pidFile,
			)
			runExternal(t, bin, dir, env, "init", "--name", "x")
			runExternal(t, bin, dir, env, "project", "add", "P", "--slug", "p", "--goal", "publisher lifecycle")
			runExternal(t, bin, dir, env, "github", "link", "p")
			runExternal(t, bin, dir, env, "task", "add", "Publish record", "--project", "p")

			cmd := exec.Command(bin, "github", "push", "p")
			cmd.Dir, cmd.Env = dir, env
			var output strings.Builder
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			waitForFile(t, pidFile)
			if tc.cancel {
				if err := cmd.Process.Signal(os.Interrupt); err != nil {
					t.Fatal(err)
				}
			}
			if err := cmd.Wait(); err == nil {
				t.Fatalf("%s push exited successfully:\n%s", tc.name, output.String())
			}
			if _, err := os.Stat(testGitHubPushLock(dir, "owner/repo")); !os.IsNotExist(err) {
				t.Fatalf("%s push returned with lock present: %v", tc.name, err)
			}
			raw, err := os.ReadFile(pidFile)
			if err != nil {
				t.Fatal(err)
			}
			pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
			if procmon.Alive(pid) {
				t.Fatalf("publisher process %d survived %s", pid, tc.name)
			}
		})
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func waitForPushLine(t *testing.T, lines <-chan string, want string) {
	t.Helper()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case line := <-lines:
			if strings.Contains(line, want) {
				return
			}
		case <-timeout.C:
			t.Fatalf("github push did not stream %q", want)
		}
	}
}

func runExternal(t *testing.T, bin, dir string, env []string, args ...string) {
	t.Helper()
	c := exec.Command(bin, args...)
	c.Dir = dir
	c.Env = env
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("dacli %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func testGitHubPushLock(dir, repo string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(repo)))
	return filepath.Join(dir, ".dacli", "locks", fmt.Sprintf("github-push-%x.lock", sum[:16]))
}

func writeFakeGHPublisher(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
case "$1 $2" in
  "repo view") echo '{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}' ;;
  "issue list") echo '[]' ;;
  "issue create")
    echo 'publisher: creating issue'
    case "$DACLI_GH_MODE" in
      failure)
        (trap 'exit 0' TERM INT; while :; do sleep 1; done) &
        echo $! > "$DACLI_GH_PID"
        echo 'publisher failed' >&2
        exit 9
        ;;
      cancel)
        (trap 'exit 0' TERM INT; while :; do sleep 1; done) &
        echo $! > "$DACLI_GH_PID"
        wait
        ;;
      *) while [ ! -f "$DACLI_GH_RELEASE" ]; do sleep 0.01; done; echo 'https://github.com/owner/repo/issues/1' ;;
    esac
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
