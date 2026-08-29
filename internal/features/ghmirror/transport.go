package ghmirror

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// githubTransport is the package-local port between projection policy and the
// GitHub CLI adapter. Tests may still replace gh directly for narrow fixtures;
// production construction stays explicit here (issue #901).
type githubTransport interface {
	Run(*workspace.Workspace, ...string) (string, error)
}

type githubCLITransport struct{}

func (githubCLITransport) Run(w *workspace.Workspace, args ...string) (string, error) {
	return ghExec(w, args...)
}

// gh runs the GitHub CLI in the workspace root. Credentials are gh's own —
// dacli never handles a token. The exact subcommands used here are
// assumptions until doctor probes them, per the standing doctrine.
// A package variable so a test can force gh to FAIL. The bugs this package has
// had are all in the failure path — an unchecked error becoming a silent wrong
// outcome — and that path is unreachable while gh keeps succeeding (dacli 208).
var defaultGitHubTransport githubTransport = githubCLITransport{}
var gh = defaultGitHubTransport.Run

var ghCommandTimeout = func([]string) time.Duration { return 120 * time.Second }

type synchronizedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func ghExec(w *workspace.Workspace, args ...string) (string, error) {
	// gh is network- and auth-bound; a deadline keeps a hung request (no
	// network, an interactive auth prompt) from blocking the caller — and,
	// under `dacli mcp serve`, the entire stdio loop.
	interruptCtx, stopInterrupt := signal.NotifyContext(context.Background(), ghInterruptSignals()...)
	defer stopInterrupt()
	ctx, cancel := context.WithTimeout(interruptCtx, ghCommandTimeout(args))
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = w.Root
	setGHProcessGroup(cmd)
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			procmon.KillTree(cmd.Process.Pid, 500*time.Millisecond)
		}
		return nil
	}
	var out synchronizedBuffer
	mutation := ghMutation(args)
	var streamDone func()
	if mutation {
		// A multi-task push can spend minutes in remote writes. CombinedOutput
		// withheld the child's progress until every write completed, which made
		// an active publisher look like its invoking `github push` had returned
		// after printing only the plan (issue #797). Tee writes, but keep the
		// complete transcript for callers that need to parse or report it.
		var streamErr error
		streamDone, streamErr = attachGHStreams(cmd, &out)
		if streamErr != nil {
			return "", streamErr
		}
	} else {
		cmd.Stdout = &out
		cmd.Stderr = &out
	}
	err := cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}
	// gh occasionally delegates work to a child and exits first. The repository
	// lease belongs to the whole publication tree, not just its leader: wait for
	// successful descendants, and terminate failed ones, before cmdPush can
	// release the lease and tell its caller publication is complete (issue #797).
	if cmd.Process != nil {
		pgid := cmd.Process.Pid
		if err != nil {
			procmon.KillTree(pgid, 500*time.Millisecond)
		} else {
			for procmon.GroupAlive(pgid) && ctx.Err() == nil {
				time.Sleep(10 * time.Millisecond)
			}
			if ctx.Err() != nil {
				procmon.KillTree(pgid, 500*time.Millisecond)
				err = ctx.Err()
			}
		}
	}
	if streamDone != nil {
		streamDone()
	}
	if err != nil {
		safe := commandresult.SanitizeTail(out.String(), w.Root)
		typed := commandresult.NewExternalError(cmd, commandresult.RunOptions{
			Operation: "GitHub mirror operation", WorkspaceRoot: w.Root,
		}, nil, []byte(out.String()), err, ctx.Err() == context.DeadlineExceeded)
		if ctx.Err() == context.DeadlineExceeded {
			return safe, fmt.Errorf("GitHub command timed out: %w", typed)
		}
		return safe, typed
	}
	return strings.TrimSpace(out.String()), nil
}

// attachGHStreams uses caller-owned OS pipes rather than os/exec's implicit
// writer goroutines. Cmd.Wait can therefore report the leader's exit even when
// a forked publisher still owns an inherited output descriptor; ghExec then
// reconciles that process group before waiting for both stream copies.
func attachGHStreams(cmd *exec.Cmd, out io.Writer) (func(), error) {
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("open gh stdout stream: %w", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		return nil, fmt.Errorf("open gh stderr stream: %w", err)
	}
	cmd.Stdout, cmd.Stderr = stdoutW, stderrW
	var copies sync.WaitGroup
	copies.Add(2)
	go func() {
		defer copies.Done()
		_, _ = io.Copy(io.MultiWriter(out, os.Stdout), stdoutR)
		_ = stdoutR.Close()
	}()
	go func() {
		defer copies.Done()
		_, _ = io.Copy(io.MultiWriter(out, os.Stderr), stderrR)
		_ = stderrR.Close()
	}()
	return func() {
		_ = stdoutW.Close()
		_ = stderrW.Close()
		copies.Wait()
	}, nil
}

// ghMutation identifies the gh subcommands that can make an outbound change.
// Read probes are deliberately kept quiet: streaming their JSON would make a
// push's progress unreadable, while mutating calls are the operator-relevant
// work that can take time and must remain visibly attached to the caller.
func ghMutation(args []string) bool {
	if len(args) < 2 {
		return false
	}
	switch args[0] {
	case "issue":
		switch args[1] {
		case "create", "comment", "close", "edit":
			return true
		}
	case "label":
		return args[1] == "create"
	case "api":
		for i := 1; i+1 < len(args); i++ {
			if args[i] == "--method" && strings.EqualFold(args[i+1], "POST") {
				return true
			}
		}
	}
	return false
}

// ghRepo runs a gh subcommand explicitly against the project's LINKED repo
// (github_repo), not whatever the workspace-root git remote happens to resolve
// to. cmd.Dir = w.Root means a bare gh call targets the checkout's own remote —
// but a dacli workspace can manage several projects, each `github link`ed to a
// DIFFERENT repo, while the root has exactly ONE remote. Without --repo, every
// project's issues would be created/edited/closed on that one cwd repo, so all
// but one project's mirror lands in the WRONG repository (dacli 221). Passing
// --repo makes the write target the repo the project is actually linked to.
//
// --repo is a per-command (cobra-inherited) flag, invalid at the root `gh`
// level, so it is APPENDED after the subcommand verb (the same placement
// selfreport and catalog use). It routes through the stubbable `gh` var so the
// failure-path tests still intercept it. An empty repo falls back to cwd
// resolution, so the pre-link discovery paths (doctor, link) keep working.
func ghRepo(w *workspace.Workspace, repo string, args ...string) (string, error) {
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	return gh(w, args...)
}

type repoInfo struct {
	NameWithOwner string `json:"nameWithOwner"`
	Visibility    string `json:"visibility"`
}

// repoView probes a repo's live visibility. A non-empty repo queries THAT repo
// explicitly so the disclosure gate judges the repo the push actually writes
// to, not the cwd remote (dacli 221, mirroring catalog's dacli 167 fix); an
// empty repo falls back to the cwd repo, which is how doctor/link discover the
// repo to report or store in the first place.
//
// `gh repo view` is the ONE call site here that cannot take --repo: the
// repository is its positional argument, and passing the flag makes gh exit 1
// with `unknown flag: --repo`. dacli 221 routed every gh call through ghRepo
// uniformly and broke `github push` at its first call — the disclosure probe —
// so the whole outbound mirror failed before it wrote anything (dacli 297).
// Verified against the installed gh: issue list/create/edit/close/comment/view,
// label create and release view all accept the inherited --repo; repo view does
// not. Uniformity was the bug; this one takes it positionally.
func repoView(w *workspace.Workspace, repo string) (repoInfo, error) {
	var info repoInfo
	args := []string{"repo", "view"}
	if repo != "" {
		args = append(args, repo)
	}
	args = append(args, "--json", "nameWithOwner,visibility")
	out, err := gh(w, args...)
	if err != nil {
		return info, fmt.Errorf("gh repo view failed: %w (%s)", err, out)
	}
	return info, json.Unmarshal([]byte(out), &info)
}
