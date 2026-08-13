package vcs

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
)

// TestPRStatusMergedViaGH: gh reports the PR MERGED — landed, no fallback
// trunk check needed.
func TestPRStatusMergedViaGH(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	calls := stubGH(t, func(d string, args ...string) (string, error) {
		return `[{"state":"MERGED","url":"https://github.com/x/y/pull/1","autoMergeRequest":null}]`, nil
	})
	status := checkLanded(w, BranchFor(tk), "main")
	if status.State != "merged" {
		t.Fatalf("state = %q, want merged (%s)", status.State, status.Detail)
	}
	if len(*calls) == 0 || (*calls)[0][0] != "pr" {
		t.Fatalf("expected a gh pr call, got %v", *calls)
	}
	_ = dir
}

// GitHub commonly deletes a PR's head branch after a squash merge. The task
// log still carries the immutable PR URL, so status must resolve that identity
// before asking for a now-impossible head-branch match.
func TestPRStatusUsesRecordedURLAfterMergedHeadDeletion(t *testing.T) {
	_, w, tk := prIntegrateEnv(t)
	const prURL = "https://github.com/x/y/pull/544"
	store.AppendLog(tk, "PR opened: "+prURL)
	if err := store.SaveTask(tk); err != nil {
		t.Fatal(err)
	}
	calls := stubGH(t, func(_ string, args ...string) (string, error) {
		if len(args) >= 3 && args[0] == "pr" && args[1] == "view" && args[2] == prURL {
			return `{"state":"MERGED","url":"https://github.com/x/y/pull/544","autoMergeRequest":null}`, nil
		}
		return "[]", nil // the deleted head branch can no longer be listed
	})

	status := checkTaskLanded(w, tk, "main")
	if status.State != "merged" {
		t.Fatalf("state = %q, want merged (%s)", status.State, status.Detail)
	}
	if len(*calls) != 1 || (*calls)[0][1] != "view" {
		t.Fatalf("recorded PR must be resolved before head lookup, calls=%v", *calls)
	}
}

// TestPRStatusLandingWithAutoMergeQueued is the exact false-positive shape
// behind tasks 157/160: a just-opened --auto PR is OPEN with auto-merge
// queued. It must read as "landing", never "orphaned" — even though the
// branch commit is not yet an ancestor of local main.
func TestPRStatusLandingWithAutoMergeQueued(t *testing.T) {
	_, w, tk := prIntegrateEnv(t)
	stubGH(t, func(d string, args ...string) (string, error) {
		return `[{"state":"OPEN","url":"https://github.com/x/y/pull/2","autoMergeRequest":{"enabledAt":"2026-07-26T00:00:00Z"}}]`, nil
	})
	status := checkLanded(w, BranchFor(tk), "main")
	if status.State != "landing" {
		t.Fatalf("state = %q, want landing (%s)", status.State, status.Detail)
	}
	if !strings.Contains(status.Detail, "not orphaned") {
		t.Fatalf("detail should call out landing vs orphaned explicitly: %s", status.Detail)
	}
}

// An OPEN PR with no auto-merge queued yet is still landing, not orphaned —
// GitHub hasn't rejected or closed it.
func TestPRStatusLandingOpenNoAutoMerge(t *testing.T) {
	_, w, tk := prIntegrateEnv(t)
	stubGH(t, func(d string, args ...string) (string, error) {
		return `[{"state":"OPEN","url":"https://github.com/x/y/pull/3","autoMergeRequest":null}]`, nil
	})
	status := checkLanded(w, BranchFor(tk), "main")
	if status.State != "landing" {
		t.Fatalf("state = %q, want landing (%s)", status.State, status.Detail)
	}
}

// A CLOSED-without-merging PR is a genuine orphan.
func TestPRStatusOrphanedClosedUnmerged(t *testing.T) {
	_, w, tk := prIntegrateEnv(t)
	stubGH(t, func(d string, args ...string) (string, error) {
		return `[{"state":"CLOSED","url":"https://github.com/x/y/pull/4","autoMergeRequest":null}]`, nil
	})
	status := checkLanded(w, BranchFor(tk), "main")
	if status.State != "orphaned" {
		t.Fatalf("state = %q, want orphaned (%s)", status.State, status.Detail)
	}
}

// addBareOrigin turns dir's repo into one with a real "origin" remote (a bare
// repo cloned from dir's current state), so the no-PR fallback path has
// something to fetch.
func addBareOrigin(t *testing.T, dir string) string {
	t.Helper()
	bare := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", bare).CombinedOutput(); err != nil {
		t.Fatalf("remote add: %v\n%s", err, out)
	}
	return bare
}

// When gh finds no PR at all, checkLanded falls back to a FRESH trunk
// fetch+merge-base — never a stale local branch-vs-main compare. Here the
// branch's commit truly is not on origin/main after the fetch, so the
// fallback correctly reports orphaned.
func TestPRStatusFallbackOrphanedWhenNoPRAndNotMerged(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	addBareOrigin(t, dir)
	gitAt(t, dir, "push", "-q", "origin", "main")
	stubGH(t, func(d string, args ...string) (string, error) {
		return "[]", nil
	})
	status := checkLanded(w, BranchFor(tk), "main")
	if status.State != "orphaned" {
		t.Fatalf("state = %q, want orphaned (%s)", status.State, status.Detail)
	}
}

// Same no-PR fallback, but the branch WAS actually merged into origin/main
// (e.g. a local `dacli integrate`, never went through a PR at all) — the
// fresh fetch must report merged, not orphaned.
func TestPRStatusFallbackMergedWhenNoPRButAncestorOnOrigin(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	branch := BranchFor(tk)
	gitAt(t, dir, "merge", "-q", "--no-ff", "-m", "merge "+branch, branch)
	addBareOrigin(t, dir)
	gitAt(t, dir, "push", "-q", "origin", "main")
	stubGH(t, func(d string, args ...string) (string, error) {
		return "[]", nil
	})
	status := checkLanded(w, branch, "main")
	if status.State != "merged" {
		t.Fatalf("state = %q, want merged (%s)", status.State, status.Detail)
	}
}

// A spawn that died before committing leaves a branch pointing exactly at
// trunk — zero commits of its own. It is trivially an ancestor of origin/main,
// which once made checkLanded call it "merged" and force-accept an empty branch
// as a done task (dacli 168, 241). It must read as orphaned, never merged.
func TestPRStatusFallbackOrphanedWhenZeroCommitDeadSpawn(t *testing.T) {
	dir, w, _ := prIntegrateEnv(t)
	deadBranch := "dacli/999-dead-spawn"
	gitAt(t, dir, "branch", deadBranch, "main") // no commit: tip == main's tip
	addBareOrigin(t, dir)
	gitAt(t, dir, "push", "-q", "origin", "main")
	stubGH(t, func(d string, args ...string) (string, error) {
		return "[]", nil
	})
	status := checkLanded(w, deadBranch, "main")
	if status.State != "orphaned" {
		t.Fatalf("state = %q, want orphaned — a zero-commit dead spawn must never read as merged (%s)", status.State, status.Detail)
	}
}

// cmdPRStatus prints the classification for the operator/reviewer.
func TestCmdPRStatusPrintsClassification(t *testing.T) {
	_, w, tk := prIntegrateEnv(t)
	stubGH(t, func(d string, args ...string) (string, error) {
		return `[{"state":"OPEN","url":"https://github.com/x/y/pull/5","autoMergeRequest":{"enabledAt":"2026-07-26T00:00:00Z"}}]`, nil
	})
	ctx, out := prCtx(w.Root)
	if err := cmdPRStatus(ctx, []string{fmt.Sprintf("%d", tk.Seq)}); err != nil {
		t.Fatalf("cmdPRStatus: %v", err)
	}
	if !strings.Contains(out.String(), "landing") {
		t.Fatalf("output should name the landing state: %s", out.String())
	}
}
