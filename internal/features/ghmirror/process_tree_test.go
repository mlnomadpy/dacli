//go:build !windows

package ghmirror

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
)

// A timeout is terminal for the complete publisher tree. In particular, the
// forked worker must not outlive cmdPush and retain either execution or the
// repository sequence lease (issue #797).
func TestGitHubPushTimeoutKillsPublisherTreeBeforeReleasingLock(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	for _, title := range []string{"publish first", "publish second"} {
		if _, err := store.CreateTask(w, "a-root", "core", title, store.TaskOpts{}); err != nil {
			t.Fatal(err)
		}
	}

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "publisher.pid")
	script := `#!/bin/sh
case "$1 $2" in
  "repo view") echo '{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}' ;;
  "issue list") echo '[]' ;;
  "issue create")
    (trap 'exit 0' TERM INT; while :; do sleep 1; done) &
    echo $! > "$DACLI_GH_PID"
    wait
    ;;
esac
`
	ghPath := filepath.Join(dir, "gh")
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DACLI_GH_PID", pidFile)
	origGH, origTimeout := gh, ghCommandTimeout
	gh = ghExec
	ghCommandTimeout = func(args []string) time.Duration {
		if len(args) >= 2 && args[0] == "issue" && args[1] == "create" {
			return 150 * time.Millisecond
		}
		return 10 * time.Second
	}
	t.Cleanup(func() { gh, ghCommandTimeout = origGH, origTimeout })

	ctx, _ := releaseCtx(t, w)
	err := cmdPush(ctx, []string{"core"})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout result = %v, want terminal timeout", err)
	}
	if _, statErr := os.Stat(githubPushLockPath(w, "owner/repo")); !os.IsNotExist(statErr) {
		t.Fatalf("timed-out push returned with lock present: %v", statErr)
	}
	raw, readErr := os.ReadFile(pidFile)
	if readErr != nil {
		t.Fatalf("publisher did not record child pid: %v", readErr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	if procmon.Alive(pid) {
		t.Fatalf("publisher child %d survived timeout", pid)
	}
}
