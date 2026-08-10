// The parallel-agent git lifecycle: isolated worktrees so agents work at once
// without clobbering each other, push, PR, and conflict-aware merge/integrate.
// This is what turns `dacli next --parallel N` from advice into real
// concurrent work.
package vcs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func init() {
	Commands = append(Commands,
		clikit.Command{Path: "worktree add", Brief: "Isolated worktree+branch for a task so parallel agents don't collide", Mutates: true, Run: cmdWorktreeAdd},
		clikit.Command{Path: "worktree list", Brief: "Active worktrees and their branches", Run: cmdWorktreeList},
		clikit.Command{Path: "worktree remove", Brief: "Tear down a task's worktree", Mutates: true, Run: cmdWorktreeRemove},
		clikit.Command{Path: "worktree prune", Brief: "Reclaim every worktree whose branch has merged or whose run is finished (--into <trunk>, default main; --dry-run to preview) — the loop runs this each cycle so checkouts don't pile up", Mutates: true, Run: cmdWorktreePrune},
		clikit.Command{Path: "push", Brief: "Push a task's branch to origin", Mutates: true, Run: cmdPush},
		clikit.Command{Path: "pr", Brief: "Open a PR for a task's branch (gh); body carries acceptance + findings + Fixes #issue. --with-verdicts leads the body and review with a loud trust-grade summary + per-finding verdict tally, plus the verify panel's per-seat verdicts, and posts each finding that names a file:line as a LINE COMMENT on the diff; --approve/--request-changes post a real review state instead of a bare comment; --auto queues GitHub auto-merge so the PR self-lands on green CI", Mutates: true, Run: cmdPR},
		clikit.Command{Path: "pr status", Brief: "Did this task's branch land? Checks gh PR state first (merged/landing/orphaned) and only falls back to a fresh trunk fetch if no PR is found — never a stale local branch-vs-main compare, which misread in-flight --auto merges as orphaned (see tasks 157, 160)", Run: cmdPRStatus},
		clikit.Command{Path: "merge", Brief: "Merge a task's branch; a conflict blocks the task, never half-merges", Mutates: true, Run: cmdMerge},
		clikit.Command{Path: "integrate", Brief: "Merge task branches (--tasks <refs> or all done) into --into <branch>; every named task must be DONE or the run is refused (--force overrides), so a merge never leaves the task open for `next` to re-rank; --pr opens a PR per branch and merges via gh (--auto sets GitHub auto-merge on CI green, default gates on gh pr checks, --no-merge stops for review), else a local merge", Mutates: true, Run: cmdIntegrate},
	)
}

// BranchFor is the task branch convention, shared with the git_workflow
// prompt. The convention itself lives in store so acceptance can check the
// same branch without importing this slice.
func BranchFor(t *store.Task) string { return store.TaskBranch(t) }

// runGH runs the GitHub CLI in dir under a network deadline and returns trimmed
// combined output. It is a package variable so a test can substitute an
// in-process stub — the PR-first integration path (push → pr → gh pr merge)
// must be exercisable without a live GitHub or a real `gh` binary.
var runGH = func(dir string, args ...string) (string, error) {
	pctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	c := exec.CommandContext(pctx, "gh", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if pctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), fmt.Errorf("gh %s timed out", strings.Join(args, " "))
	}
	return strings.TrimSpace(string(out)), err
}

// pushBranch pushes a task branch to origin. A package variable for the same
// reason as runGH — so a test can drive the fallback (a network failure at push
// falls back to a local merge) without a real remote.
var pushBranch = func(root, branch string) (string, error) {
	return gitx.Push(root, branch)
}

// isNetworkErr reports whether gh/git output names a GitHub-unreachable
// condition — the ONLY failure ship/integrate --pr falls back to a local merge
// on. A non-network failure (bad auth, protected branch, dirty tree) is a real
// error the operator must see, never silently local-merged.
// reachedGitHub reports whether gh's output is a SERVER ANSWER rather than a
// transport failure — a checks report, or a verdict about the pull request.
// Either is proof the request completed, which no network failure produces.
//
// It exists because isNetworkErr is a bare substring scan and runGH uses
// CombinedOutput, so gh's own reply is what gets scanned. A check named
// `integration-timeout`, or a mergeability verdict quoting a failing job with
// "timed out" in its name, made a REFUSAL look like an OUTAGE — and both
// call sites answer an outage by local-merging the branch into trunk. A gate
// saying no became a merge.
func reachedGitHub(out string) bool {
	s := strings.ToLower(out)
	for _, answer := range []string{
		// A checks table: one row per check, each carrying a state word.
		"pass", "fail", "pending", "skipping", "successful", "no checks reported",
		// A mergeability verdict about the pull request itself.
		"pull request", "not mergeable", "merge conflict", "conflict",
		"required status", "review", "base branch", "protected branch",
	} {
		if strings.Contains(s, answer) {
			return true
		}
	}
	return false
}

func isNetworkErr(s string) bool {
	s = strings.ToLower(s)
	for _, sig := range []string{
		"could not resolve host", "couldn't resolve host", "no such host",
		"network is unreachable", "could not connect", "failed to connect",
		"connection refused", "connection reset", "connection timed out",
		"operation timed out", "timed out", "timeout", "i/o timeout",
		"dial tcp", "temporary failure in name resolution", "tls handshake",
		"unreachable", "server misbehaving", "eof",
	} {
		if strings.Contains(s, sig) {
			return true
		}
	}
	return false
}

func resolveTaskFlag(w *workspace.Workspace, f *clikit.Flags) (*store.Task, error) {
	ref := f.Get("task")
	if ref == "" && len(f.Pos) > 0 {
		ref = f.Pos[0]
	}
	if ref == "" {
		return nil, clikit.Usagef("need a task: <ref> or --task <ref>")
	}
	return store.FindTask(w, ref)
}

func cmdWorktreeAdd(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if !gitx.Available() {
		return fmt.Errorf("git not on PATH")
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task"); err != nil {
		return err
	}
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	branch, path := BranchFor(t), w.WorktreePath(t.Project, t.Seq, t.Slug)
	if err := gitx.AddWorktree(w.Root, path, branch); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "worktree ready: %s (branch %s)\n", path, branch)
	fmt.Fprintf(ctx.Stdout, "an agent works here in isolation; commit with `dacli commit`, then `dacli push`/`dacli pr`\n")
	return nil
}

func cmdWorktreeList(ctx *clikit.Ctx, args []string) error {
	// This command takes no flags, so ANY flag is a typo. An empty allowlist
	// rejects every one — without it a mistyped flag was dropped and the
	// command ran as if nothing were wrong.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	wts, err := gitx.ListWorktrees(w.Root)
	if err != nil {
		return err
	}
	for _, wt := range wts {
		fmt.Fprintf(ctx.Stdout, "%-10s %s\n", clikit.OrDash(wt.Branch), wt.Path)
	}
	return nil
}

func cmdWorktreeRemove(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task"); err != nil {
		return err
	}
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	if err := gitx.RemoveWorktree(w.Root, w.WorktreePath(t.Project, t.Seq, t.Slug)); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "removed worktree for %03d-%s\n", t.Seq, t.Slug)
	return nil
}

// cmdWorktreePrune reclaims every worktree whose branch has merged or whose run
// is finished — the operator-runnable form of the loop's own reap, so a
// long-running project does not accumulate a gigabyte of dead checkouts (one
// per task ever spawned with --worktree; dacli 252). The safety predicate lives
// in store.ReclaimableWorktrees, shared with the loop so both reap identically.
func cmdWorktreePrune(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if !gitx.Available() {
		return fmt.Errorf("git not on PATH")
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("into", "dry-run"); err != nil {
		return err
	}
	into := clikit.OrDash(f.Get("into"), "main")
	// Never reclaim the worktree the operator is standing in.
	cwd := ctx.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	if f.Bool("dry-run") {
		cand, err := store.ReclaimableWorktrees(w, into, cwd)
		if err != nil {
			return err
		}
		for _, c := range cand {
			fmt.Fprintf(ctx.Stdout, "would prune %s (%s — %s)\n", c.Path, clikit.OrDash(c.Branch), c.Reason)
		}
		fmt.Fprintf(ctx.Stdout, "%d worktree(s) reclaimable\n", len(cand))
		return nil
	}

	removed, err := store.PruneWorktrees(w, into, cwd)
	if err != nil {
		return err
	}
	for _, c := range removed {
		fmt.Fprintf(ctx.Stdout, "pruned %s (%s — %s)\n", c.Path, clikit.OrDash(c.Branch), c.Reason)
	}
	fmt.Fprintf(ctx.Stdout, "reclaimed %d worktree(s)\n", len(removed))
	return nil
}

func cmdPush(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("pushing needs an rw grant (yours is %s)", id.Grant)
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task"); err != nil {
		return err
	}
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	branch := BranchFor(t)
	if !gitx.BranchExists(w.Root, branch) {
		return fmt.Errorf("no branch %s — `dacli worktree add --task %03d` and commit first", branch, t.Seq)
	}
	// PushSync retries a non-fast-forward rejection with a fetch+rebase, so a
	// stale local branch (a sibling branch's async auto-merge advanced trunk
	// since this checkout last synced) doesn't fail the push outright.
	out, err := gitx.PushSync(w.Root, branch)
	if err != nil {
		return fmt.Errorf("push failed: %s", out)
	}
	fmt.Fprintf(ctx.Stdout, "pushed %s\n", branch)
	return nil
}

func cmdPR(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	// Opening a PR is an outward-facing GitHub write — `gh pr create`, and with
	// --with-verdicts / --approve / --request-changes a review (summary body,
	// line-anchored finding comments, and a review STATE) posted to a possibly
	// public origin. Gate it behind rw like every other outward vcs command
	// (push/merge/integrate), so a read-only agent cannot leak internal findings
	// to GitHub, and cannot approve anything in the workspace's name (brief
	// rank-2 risk; dacli 194 widened what this gate covers).
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("opening a PR needs an rw grant (yours is %s)", id.Grant)
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task", "base", "with-verdicts", "auto", "approve", "request-changes", "draft"); err != nil {
		return err
	}
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh not on PATH — `dacli pr` opens the PR via the GitHub CLI")
	}
	event, err := reviewEventFor(f)
	if err != nil {
		return err
	}
	base := clikit.OrDash(f.Get("base"), "main")
	url, reused, err := openPR(ctx, w, id.ID, t, base, f.Bool("with-verdicts"), event, f.Bool("draft"))
	if err != nil {
		return err
	}
	if reused {
		fmt.Fprintf(ctx.Stdout, "PR already open: %s\n", url)
	} else {
		fmt.Fprintf(ctx.Stdout, "PR opened and recorded: %s\n", url)
	}

	// --auto: queue GitHub's native auto-merge so the PR lands the instant its
	// required checks go green — no operator merge, no ship follow-up. This is
	// what lets a spawned fixer's own PR self-land in the perpetual loop: the
	// agent that opens the PR also queues it.
	//
	// A queue that does NOT take (the repo has "Allow auto-merge" off, or GitHub
	// is unreachable) is FATAL, not a stderr aside: this used to print a note and
	// return nil — exit 0 — so a headless agent that keyed off the exit code
	// believed its PR would self-land and moved on, leaving the PR stranded open
	// forever with no one to merge it (task 290). The sibling path
	// integrate --pr --auto (prIntegrateTask) already treats the identical
	// failure as fatal; cmdPR now matches. The PR itself is already open and
	// recorded (printed above), so the caller still has the URL to merge by
	// hand — only the false "it landed" signal is removed.
	if f.Bool("auto") {
		if err := queueAutoMerge(w.Root, BranchFor(t)); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "auto-merge queued — GitHub merges %s when CI passes\n", url)
	}
	return nil
}

// queueAutoMerge asks GitHub to merge branch's PR the instant its required
// checks pass (gh pr merge --auto). A failure to queue is returned as an error,
// never swallowed: the caller promised the PR would self-land, so a queue that
// did not take must reach the exit status rather than only stderr — otherwise a
// headless caller reading exit 0 believes a stranded PR has landed (task 290).
// The network case carries its own message so the caller can tell "GitHub is
// unreachable, retry later" apart from "this repo has auto-merge disabled".
func queueAutoMerge(root, branch string) error {
	out, err := runGH(root, "pr", "merge", branch, "--auto", "--merge", "--delete-branch")
	if err == nil {
		return nil
	}
	if isNetworkErr(out) || isNetworkErr(err.Error()) {
		return fmt.Errorf("PR opened but auto-merge NOT queued for %s and GitHub is unreachable — the PR will NOT self-land; queue it once GitHub is reachable or merge it by hand: %s", branch, oneLine(out))
	}
	return fmt.Errorf("PR opened but auto-merge could NOT be queued for %s — the PR will NOT self-land (is \"Allow auto-merge\" enabled on the repo?); merge it by hand or re-run once fixed: %s", branch, oneLine(out))
}

// LandStatus classifies whether a task's branch has actually reached trunk.
// State is one of:
//   - "merged"   — the branch's work is on trunk now.
//   - "landing"  — a PR is open (whether or not --auto's auto-merge is
//     queued); GitHub may merge it any moment. Not a defect.
//   - "orphaned" — no open PR and the branch never merged: the work really is
//     stuck.
//   - "unknown"  — gh and a trunk fetch both failed to answer the question.
type LandStatus struct {
	State  string
	Detail string
}

type prListEntry struct {
	State            string `json:"state"`
	URL              string `json:"url"`
	AutoMergeRequest *struct {
		EnabledAt string `json:"enabledAt"`
	} `json:"autoMergeRequest"`
	// MergeStateStatus is what turns "this PR is not merging" into an
	// actionable answer. GitHub reports DIRTY for a real content conflict AND
	// BEHIND for a branch that is merely out of date — the second clears by
	// merging trunk in and needs no conflict resolution at all. Triaging one as
	// the other wastes the exact time this command exists to save (task 310).
	MergeStateStatus string `json:"mergeStateStatus"`
}

// mergeStateDetail renders GitHub's mergeStateStatus as the action it implies.
// An unrecognized value is passed through rather than swallowed: a state this
// build has not seen is still information.
func mergeStateDetail(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "DIRTY":
		return "conflicts with the base branch — resolve them (this is a real conflict, not staleness)"
	case "BEHIND":
		return "behind the base branch — merge or rebase trunk in; there is nothing to resolve"
	case "BLOCKED":
		return "blocked on a required check or review"
	case "UNSTABLE":
		return "mergeable, but a check is failing or still running"
	case "CLEAN", "HAS_HOOKS":
		return "mergeable now"
	case "UNKNOWN", "":
		return "merge state not yet computed by GitHub — re-check in a moment"
	default:
		return "merge state " + state
	}
}

// checkLanded answers "did this land?" for branch against into (e.g. "main").
//
// It exists because a bare local `git merge-base --is-ancestor <branch> main`
// check misclassified two just-opened `--auto` PRs as orphaned (tasks 157,
// 160): the branch commit wasn't yet an ancestor of the reviewer's checkout
// of main not because the work was abandoned, but because GitHub's async
// auto-merge simply hadn't gone green yet. GitHub's own PR state is
// authoritative for "did this land" — a raw branch-vs-current-main comparison
// at review time is not, because the reviewer's local main is a snapshot from
// whenever they last fetched, and an --auto PR lands on GitHub's own clock.
//
// So this checks gh first: MERGED is landed, OPEN is landing (queued
// auto-merge or not — GitHub hasn't rejected it), CLOSED-unmerged is really
// orphaned. Only when gh reports no PR at all does it fall back to a trunk
// check — and even then it fetches origin first, never trusting whatever a
// prior checkout happened to have on disk.
func checkLanded(w *workspace.Workspace, branch, into string) LandStatus {
	if out, err := runGH(w.Root, "pr", "list", "--head", branch, "--state", "all",
		"--json", "state,url,autoMergeRequest,mergeStateStatus", "--limit", "1"); err == nil {
		var prs []prListEntry
		if jerr := json.Unmarshal([]byte(out), &prs); jerr == nil && len(prs) > 0 {
			pr := prs[0]
			switch strings.ToUpper(pr.State) {
			case "MERGED":
				return LandStatus{"merged", fmt.Sprintf("PR %s merged", pr.URL)}
			case "OPEN":
				why := mergeStateDetail(pr.MergeStateStatus)
				if pr.AutoMergeRequest != nil {
					return LandStatus{"landing", fmt.Sprintf("PR %s open with auto-merge queued — landing, not orphaned; %s", pr.URL, why)}
				}
				return LandStatus{"landing", fmt.Sprintf("PR %s open awaiting merge — landing, not orphaned; %s", pr.URL, why)}
			case "CLOSED":
				return LandStatus{"orphaned", fmt.Sprintf("PR %s closed without merging", pr.URL)}
			}
		}
	}
	// No PR found (or gh unreachable/absent): re-fetch origin so the trunk
	// comparison is current, never a stale local checkout.
	// `--` terminates options: `into` is a caller-supplied flag value, and
	// without the separator a value like `--upload-pack=<cmd>` would run <cmd>
	// via git fetch. Everything after `--` is a refspec, never an option.
	if _, err := gitx.RunNetwork(w.Root, "fetch", "-q", "origin", "--", into); err != nil {
		return LandStatus{"unknown", fmt.Sprintf("no PR found and could not fetch origin/%s to check: %v", into, err)}
	}
	// A branch whose tip is already on trunk's first-parent mainline carries no
	// work of its own — a spawn that died before committing. Trunk is trivially
	// its own ancestor, so without this guard the IsAncestor check below would
	// call that dead spawn "merged" and force-accept an empty branch as a done
	// task (dacli 168, 241). prLandStatus has this same guard on the loop's own
	// merge-confirmation path; here it must be topological, not a bare
	// `rev-list --count == 0`, because checkLanded's fallback deliberately serves
	// no-PR local integrates too, and a branch merged locally is ALSO zero
	// commits ahead of trunk — but it enters via a --no-ff merge commit's SECOND
	// parent (gitx.Merge), so its tip is NOT on the first-parent line while a
	// dead spawn's tip is.
	if bare, berr := gitx.TipOnFirstParentMainline(w.Root, branch, "origin/"+into); berr != nil {
		return LandStatus{"unknown", fmt.Sprintf("no PR found and could not classify the branch against origin/%s: %v", into, berr)}
	} else if bare {
		return LandStatus{"orphaned", fmt.Sprintf("no PR found and the branch has no commits of its own beyond origin/%s — a spawn that died before committing, not landed work", into)}
	}
	ok, err := gitx.IsAncestor(w.Root, branch, "origin/"+into)
	if err != nil {
		return LandStatus{"unknown", fmt.Sprintf("no PR found and could not compare against origin/%s: %v", into, err)}
	}
	if ok {
		return LandStatus{"merged", fmt.Sprintf("no PR found, but the branch is an ancestor of origin/%s (merged without a tracked PR, e.g. a local `dacli integrate`)", into)}
	}
	return LandStatus{"orphaned", fmt.Sprintf("no PR found and the branch is not an ancestor of origin/%s after a fresh fetch", into)}
}

func cmdPRStatus(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task", "into"); err != nil {
		return err
	}
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	into := clikit.OrDash(f.Get("into"), "main")
	status := checkLanded(w, BranchFor(t), into)
	fmt.Fprintf(ctx.Stdout, "%03d-%s: %s — %s\n", t.Seq, t.Slug, status.State, status.Detail)
	return nil
}

// openPR opens (via gh) an enriched PR for the task's ALREADY-PUSHED branch,
// records the URL, and — when withVerdicts, or when the operator asked for a
// review state — posts a real PR review: the verify panel's verdicts as the
// summary, each locatable finding as a LINE COMMENT on the diff, and
// event as the review state. It returns the PR URL and, on failure, an error
// whose text carries gh's stderr so a caller can tell a network failure (fall
// back to a local merge) from a real one (surface it). It does not push: the
// branch must already be on origin (cmdPush / the --pr integrate path pushes
// first).
// openPRURL returns the URL of the branch's already-open PR, and whether there
// is one. Anything other than a clean OPEN answer — no PR, a closed or merged
// one, an unreachable GitHub, unparseable output — reports "none", so the
// caller falls through to `pr create` and handles that failure as it always
// did. This probe only ever REMOVES a spurious failure; it never invents a PR.
func openPRURL(root, branch string) (string, bool) {
	out, err := runGH(root, "pr", "view", branch, "--json", "url,state", "-q", ".state + \" \" + .url")
	if err != nil {
		return "", false
	}
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) != 2 || fields[0] != "OPEN" {
		return "", false
	}
	return fields[1], true
}

// openPR returns the PR's URL and whether it was REUSED rather than created.
// Callers need the distinction only for what they print — every path after it
// (auto-merge, the check gate, the merge itself) treats the two identically,
// which is the entire point.
func openPR(ctx *clikit.Ctx, w *workspace.Workspace, actor string, t *store.Task, base string, withVerdicts bool, event string, draft bool) (string, bool, error) {
	branch := BranchFor(t)
	body := prBody(w, t, withVerdicts)

	// A PR for this branch may already be open — the loop opens one per task as
	// it lands, and `integrate --pr` is exactly the command you then run to
	// merge it. `gh pr create` hard-fails on "already exists", and that failure
	// used to abort the whole run BEFORE the --auto queue and the check-gated
	// merge below, so the sanctioned merge path could not merge any PR the loop
	// had opened — which is every PR it would ever be pointed at (dacli 255).
	//
	// Reuse it and fall through to the merge gate. Deliberately no second
	// eventlog entry and no second review post: the create path already recorded
	// both, and re-posting would put a duplicate review on the PR every time an
	// integration run touched it.
	if url, ok := openPRURL(w.Root, branch); ok {
		return url, true, nil
	}

	// gh talks to GitHub over the network; runGH bounds it with a deadline so a
	// wedged request can never hang the caller (or, under `dacli mcp serve`, the
	// stdio loop).
	createArgs := []string{"pr", "create", "--head", branch, "--base", base,
		"--title", fmt.Sprintf("%03d: %s", t.Seq, t.Title), "--body", body}
	// --draft opens the PR as a draft (dacli 224): CI runs but the PR is not
	// mergeable and requests no review until marked ready — the planning-stage
	// PR a real project opens for work-in-progress. Integration never drafts
	// (it opens PRs to LAND), so only the operator-run `dacli pr` sets it.
	if draft {
		createArgs = append(createArgs, "--draft")
	}
	out, err := runGH(w.Root, createArgs...)
	if err != nil {
		return "", false, fmt.Errorf("gh pr create failed: %s", strings.TrimSpace(out))
	}
	url := strings.TrimSpace(out)
	// An unrecorded PR does not exist: record the URL so it enters the workspace
	// and every future brief for the task. A PR-opened event is an operational
	// fact, not a code defect — record it as a comment, NOT a finding. An
	// EventFinding syncs into a durable, never-graded NoteFinding, which drags
	// the task's brief trust-floor to `unverified` forever and consumes a
	// finding slot; a comment lands as a Log line and does neither.
	if _, err := eventlog.Append(w, actor, model.EventComment, t.ID, "", "PR opened: "+url); err != nil {
		return url, false, err
	}
	// Operator-triggered only: mirror the verify panel's recorded verdicts and
	// the task's findings onto the PR as a real review, so human review sees the
	// model's adversarial checks where review actually happens. An explicit
	// --approve/--request-changes posts even without --with-verdicts: the state
	// IS the message. A post failure is a note, not a hard error: the PR itself
	// already exists and is recorded.
	if withVerdicts || event != reviewComment {
		if err := postReview(ctx, w, t, branch, url, event); err != nil {
			fmt.Fprintf(ctx.Stderr, "note: review not posted: %v\n", err)
		}
	}
	return url, false, nil
}

// prBody assembles the PR description from what dacli already knows about the
// task: the acceptance criteria, the finding notes agents flagged, and a
// `Fixes #<issue>` line when the task is mirrored to a GitHub issue (so merging
// the PR closes it). It touches no network, so it is unit-testable on fixtures.
// withVerdicts (task 146) puts the trust-grade summary + per-finding verdict
// tally FIRST, right under the intro line, so it is the loudest, most visible
// thing a reviewer sees — not another bullet buried under Findings.
func prBody(w *workspace.Workspace, t *store.Task, withVerdicts bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Implements dacli task %03d-%s.\n", t.Seq, t.Slug)
	if withVerdicts {
		if grade := trustGradeSection(w, t); grade != "" {
			b.WriteString("\n" + grade)
		}
	}
	if fixes := taskFixesLine(t); fixes != "" {
		b.WriteString("\n" + fixes + "\n")
	}
	if acc := taskAcceptance(t); acc != "" {
		b.WriteString("\n" + acc)
	}
	if finds := taskFindings(w, t); finds != "" {
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n" + finds)
	}
	return b.String()
}

func taskAcceptance(t *store.Task) string {
	if s, ok := t.Doc.Section("Acceptance"); ok {
		return "### Acceptance\n" + s.Content
	}
	return ""
}

// taskFixesLine reads the task's mirrored issue number from its OWN `github:`
// frontmatter block — the mapping ghmirror writes at push — and returns a
// `Fixes #N` line so merging the PR closes the issue. Empty (skipped cleanly)
// when the task is not linked. We parse the block here rather than import the
// ghmirror slice (feature slices don't import each other).
func taskFixesLine(t *store.Task) string {
	block, ok := t.Doc.Front.GetBlock("github")
	if !ok {
		return ""
	}
	for _, line := range strings.Split(block, "\n") {
		if k, v, found := strings.Cut(strings.TrimSpace(line), ":"); found && strings.TrimSpace(k) == "issue" {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				return fmt.Sprintf("Fixes #%d", n)
			}
		}
	}
	return ""
}

// taskFindingNotes returns this task's finding notes with a non-empty body —
// only those whose `about` names this task (by id or by NNN sequence,
// matching how verify resolves the task's findings) are included. Shared by
// taskFindings and trustGradeSection so the two PR sections count the exact
// same set of findings.
func taskFindingNotes(w *workspace.Workspace, t *store.Task) []*mdstore.Doc {
	notes, _ := store.ListNotes(w, t.Project, model.NoteFinding)
	var out []*mdstore.Doc
	for _, n := range notes {
		about, _ := n.Front.Get("about")
		if !strings.Contains(about, t.ID) && !strings.Contains(about, fmt.Sprintf("%03d", t.Seq)) {
			continue
		}
		if noteBody(n) == "" {
			continue
		}
		out = append(out, n)
	}
	return out
}

// noteBody collects a note's body text. On disk the body lives inside its
// level-1 title section (content runs to the next heading), so every
// section's content must be collected — the same rule the brief assembler
// uses.
func noteBody(n *mdstore.Doc) string {
	var body strings.Builder
	for _, s := range n.Sections {
		body.WriteString(s.Content)
	}
	return strings.TrimSpace(body.String())
}

// noteTitle returns a note's level-1 section title — the claim text verify
// grades and panelists vote on, so it is also the key trustGradeSection joins
// a finding to its recorded panel votes.
func noteTitle(n *mdstore.Doc) string {
	for _, s := range n.Sections {
		if s.Level == 1 {
			return s.Title
		}
	}
	return ""
}

// findingTags renders a finding's severity/trust frontmatter as the bold tag
// prefix a reader scans first. Shared by the PR body bullet and the anchored
// review comment (dacli 194) so a finding reads identically in both places.
func findingTags(n *mdstore.Doc) string {
	var tags strings.Builder
	if sev, _ := n.Front.Get("severity"); sev != "" {
		fmt.Fprintf(&tags, "**%s** ", sev)
	}
	if trust, _ := n.Front.Get("trust"); trust != "" {
		fmt.Fprintf(&tags, "[trust: %s] ", trust)
	}
	return tags.String()
}

// taskFindings renders the task's finding notes into a PR section, so a human
// reviewer sees what the agents flagged.
func taskFindings(w *workspace.Workspace, t *store.Task) string {
	var b strings.Builder
	for _, n := range taskFindingNotes(w, t) {
		fmt.Fprintf(&b, "- %s%s\n", findingTags(n), noteBody(n))
	}
	if b.Len() == 0 {
		return ""
	}
	return "### Findings\n" + b.String()
}

// parseVerdictLine parses one verify-verdict: comment body — the exact shape
// execution.VerdictRecord emits (see verdictMarker above) — into the verdict
// it recorded and the claim it was voting on. ok is false for any event body
// that isn't a verdict record.
func parseVerdictLine(body string) (verdict, claim string, ok bool) {
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, verdictMarker) {
		return "", "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(body, verdictMarker))
	const sep = " on claim: "
	idx := strings.Index(rest, sep)
	if idx < 0 {
		return "", "", false
	}
	verdict = strings.TrimSpace(strings.SplitN(rest[:idx], "—", 2)[0])
	claim = rest[idx+len(sep):]
	if i := strings.Index(claim, " — "); i >= 0 {
		// VerdictRecord appends " — <why>" after the claim when a panelist gave
		// a reason; that reason is not part of the claim text.
		claim = claim[:i]
	}
	return verdict, strings.TrimSpace(claim), true
}

// verdictTally counts each claim's recorded confirmed/refuted panel votes,
// from the same verify-verdict: comment events verdictReview already reads —
// no new collection, per task 146's acceptance.
func verdictTally(w *workspace.Workspace, t *store.Task) map[string]map[string]int {
	events, _ := eventlog.List(w, eventlog.Query{About: t.ID, Kinds: []model.EventKind{model.EventComment}})
	tally := map[string]map[string]int{}
	for _, e := range events {
		verdict, claim, ok := parseVerdictLine(e.Body)
		if !ok {
			continue
		}
		if tally[claim] == nil {
			tally[claim] = map[string]int{}
		}
		tally[claim][verdict]++
	}
	return tally
}

// trustGradeSection renders the LOUD, first-class trust-grade summary and
// per-finding verdict tally `dacli pr --with-verdicts` adds to both the PR
// body and the PR review (task 146): an aggregate count of this task's
// findings by trust grade, the trust floor (the WORST grade — refuted <
// unverified < confirmed, the same D3 ordering internal/brief renders into a
// brief's trust-floor), and each finding's recorded panel vote tally. It reads
// only what's already collected — finding notes' `trust:` frontmatter
// (store.GradeFinding's output) and the verify-verdict: comment events verify
// already writes — no new collection. Empty when the task has no finding
// notes to grade.
func trustGradeSection(w *workspace.Workspace, t *store.Task) string {
	notes := taskFindingNotes(w, t)
	if len(notes) == 0 {
		return ""
	}
	tally := verdictTally(w, t)
	counts := map[string]int{"confirmed": 0, "unverified": 0, "refuted": 0}
	floorRank := brief.TrustRank("confirmed") // starts at the best grade; drops to the worst seen
	var rows strings.Builder
	for _, n := range notes {
		trust, _ := n.Front.Get("trust")
		grade := brief.TrustLabel(trust)
		counts[grade]++
		if r := brief.TrustRank(trust); r < floorRank {
			floorRank = r
		}
		votes := "no panel votes recorded"
		if v := tally[noteTitle(n)]; len(v) > 0 {
			votes = fmt.Sprintf("%d confirmed, %d refuted", v["confirmed"], v["refuted"])
		}
		fmt.Fprintf(&rows, "| %s | %s | %s |\n", clikit.OrDash(noteTitle(n)), grade, votes)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## \U0001F6A8 TRUST GRADE: %s \U0001F6A8\n\n", strings.ToUpper(brief.TrustLabel(brief.RankTrust(floorRank))))
	fmt.Fprintf(&b, "**%d confirmed · %d unverified · %d refuted** — floor is the WORST grade among this task's findings (refuted < unverified < confirmed); an unverified finding has not been checked, treat it as a lead, not a fact.\n\n",
		counts["confirmed"], counts["unverified"], counts["refuted"])
	b.WriteString("| Finding | Trust | Panel tally |\n|---|---|---|\n")
	b.WriteString(rows.String())
	return b.String()
}

// verdictMarker mirrors execution.VerdictMarker: the prefix verify writes onto
// the comment event that records a panel verdict. Feature slices don't import
// each other, so this convention string — not an import — is the contract
// between `dacli verify` (writer) and `dacli pr --with-verdicts` (reader).
const verdictMarker = "verify-verdict:"

// verdictReview renders the task's recorded verify verdicts into a PR review
// body. It reads the verify-verdict: comment events verify writes, independently
// of gh, so the assembly is unit-testable. Empty when the task has no recorded
// verdicts (nothing to post).
func verdictReview(w *workspace.Workspace, t *store.Task) string {
	events, _ := eventlog.List(w, eventlog.Query{About: t.ID, Kinds: []model.EventKind{model.EventComment}})
	// eventlog.List is newest-first; reverse to chronological so the review reads
	// in the order the panel voted.
	var lines []string
	for i := len(events) - 1; i >= 0; i-- {
		body := strings.TrimSpace(events[i].Body)
		if !strings.HasPrefix(body, verdictMarker) {
			continue
		}
		lines = append(lines, "- "+strings.TrimSpace(strings.TrimPrefix(body, verdictMarker)))
	}
	if len(lines) == 0 {
		return ""
	}
	// The trust-grade summary leads the review too (task 146) — loud, first
	// thing a human sees, ahead of the raw per-seat vote list. Empty (no
	// finding notes graded yet) leaves the review exactly as it read before.
	var b strings.Builder
	if grade := trustGradeSection(w, t); grade != "" {
		b.WriteString(grade + "\n")
	}
	b.WriteString("### dacli verify panel\n\nThe adversarial verification panel's verdicts on this task's claims:\n\n" + strings.Join(lines, "\n") + "\n")
	return b.String()
}

// The three GitHub review states dacli can post. COMMENT is the default: a
// review state is a claim about the change, and a routine `dacli pr` must never
// silently approve its own work (dacli 194).
const (
	reviewComment        = "COMMENT"
	reviewApprove        = "APPROVE"
	reviewRequestChanges = "REQUEST_CHANGES"
)

// reviewEventFor maps --approve / --request-changes onto the review state the
// API takes. Before dacli 194 every review dacli posted was a comment, so a run
// that had confirmed a real defect said so in a body nobody had to answer;
// REQUEST_CHANGES is a state GitHub itself enforces (dacli 194).
func reviewEventFor(f *clikit.Flags) (string, error) {
	approve, changes := f.Bool("approve"), f.Bool("request-changes")
	if approve && changes {
		return "", clikit.Usagef("--approve and --request-changes are opposites; pass one")
	}
	switch {
	case approve:
		return reviewApprove, nil
	case changes:
		return reviewRequestChanges, nil
	}
	return reviewComment, nil
}

// prComment is one line-anchored PR review comment: a finding pinned to the
// file and line it was actually filed against.
type prComment struct {
	Path string
	Line int
	Body string
}

// commentPrefix leads every anchored comment body. It is not decoration: gh's
// -F magic reads a value that starts with "@" as a FILENAME, so a finding whose
// text began with "@" would make gh try to read a file off disk and post its
// contents. A constant, non-magic prefix makes that impossible.
const commentPrefix = "**dacli finding** "

// findingLocation parses a finding note's `origin:` frontmatter into the (path,
// line) a GitHub review comment anchors to. Origin is the provenance field
// every event and note already carries — "agent", "external:<who>", or
// "file:<path>[:<line>]" — so a finding filed against a file has always known
// where it lives; dacli simply never told GitHub (dacli 194).
//
// Accepted: "file:internal/x.go:42", "internal/x.go:42", "file:internal/x.go#L42".
// Everything else — a bare file with no line, an absolute path (GitHub anchors
// on repo-relative paths and 422s the whole review otherwise), an agent or
// external origin — returns ok=false, and the caller keeps that finding in the
// review's summary body. An unanchorable finding is never dropped.
func findingLocation(origin string) (path string, line int, ok bool) {
	s := strings.TrimSpace(origin)
	if s == "" || s == "agent" || strings.HasPrefix(s, "external:") {
		return "", 0, false
	}
	s = strings.TrimPrefix(s, "file:")
	sep, width := strings.LastIndex(s, "#L"), 2
	if sep < 0 {
		sep, width = strings.LastIndex(s, ":"), 1
	}
	if sep <= 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(s[sep+width:]))
	if err != nil || n <= 0 {
		return "", 0, false
	}
	path = strings.TrimPrefix(strings.TrimSpace(s[:sep]), "./")
	if path == "" || strings.HasPrefix(path, "/") {
		return "", 0, false
	}
	return path, n, true
}

// findingComments splits a task's findings into those that can be anchored to a
// line of the diff and those that cannot. Both halves are returned because both
// must reach the PR: anchored ones as line comments, the rest as summary
// bullets. Dropping the unanchorable half would quietly lose exactly the
// findings that name no file — often the architectural ones.
func findingComments(w *workspace.Workspace, t *store.Task) (anchored []prComment, orphans []string) {
	for _, n := range taskFindingNotes(w, t) {
		origin, _ := n.Front.Get("origin")
		if path, line, ok := findingLocation(origin); ok {
			anchored = append(anchored, prComment{Path: path, Line: line, Body: fmt.Sprintf(
				"%s%s%s\n\n_dacli task %03d-%s_", commentPrefix, findingTags(n), noteBody(n), t.Seq, t.Slug)})
			continue
		}
		orphans = append(orphans, fmt.Sprintf("- %s%s", findingTags(n), noteBody(n)))
	}
	return anchored, orphans
}

// reviewPayload assembles what the review posts: the summary body (trust grade
// + per-seat verdicts, as before, plus every finding that could not be pinned
// to a line) and the line-anchored comments. Empty body with no comments means
// there is nothing to say — except for an explicit APPROVE/REQUEST_CHANGES,
// which is an operator act that must land regardless (and which GitHub rejects
// outright with an empty body).
func reviewPayload(w *workspace.Workspace, t *store.Task, event string) (string, []prComment) {
	anchored, orphans := findingComments(w, t)
	var b strings.Builder
	b.WriteString(verdictReview(w, t))
	if len(orphans) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("### Findings without a code location\n\nFiled with no `file:line` origin, so they cannot be pinned to a line of the diff — listed here rather than dropped:\n\n" +
			strings.Join(orphans, "\n") + "\n")
	}
	if len(anchored) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%d finding(s) are posted as line comments on the diff below.\n", len(anchored))
	}
	if b.Len() == 0 && event != reviewComment {
		fmt.Fprintf(&b, "dacli review of task %03d-%s: no findings filed against this change.\n", t.Seq, t.Slug)
	}
	return b.String(), anchored
}

// reviewArgs builds the `gh api ... /reviews` invocation that posts one review
// with its line comments.
//
// The flag choice is load-bearing. Every comments[][...] field MUST use -F: gh
// parses -f (raw) and -F (typed) in two separate passes, so mixing them inside
// one array scrambles which value lands in which object. Verified against a
// local echo server — `-f comments[][path]=a.go -F comments[][line]=12 -f
// comments[][body]=x` produced [{path:a.go, body:x}, {line:12}], two objects,
// both wrong, while all-`-F` produced the intended single object. -F also gives
// `line` the integer type the API requires. The event and summary body stay -f
// so no @-magic ever touches free text.
func reviewArgs(num int, event, body string, comments []prComment) []string {
	// {owner}/{repo} are gh placeholders, filled from the repo of the directory
	// runGH runs in — so this needs no remote parsing of its own.
	args := []string{"api", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/reviews", num),
		"-X", "POST", "-f", "event=" + event, "-f", "body=" + body}
	for _, c := range comments {
		args = append(args,
			"-F", "comments[][path]="+c.Path,
			"-F", fmt.Sprintf("comments[][line]=%d", c.Line),
			"-F", "comments[][body]="+c.Body)
	}
	return args
}

// inlineComments folds anchored comments back into summary bullets, for the
// retry below.
func inlineComments(comments []prComment) string {
	var b strings.Builder
	b.WriteString("### Findings (line anchors rejected)\n\n")
	for _, c := range comments {
		fmt.Fprintf(&b, "- `%s:%d` — %s\n", c.Path, c.Line, oneLine(strings.TrimPrefix(c.Body, commentPrefix)))
	}
	return b.String()
}

// numberFromURL pulls the PR number out of a PR URL (.../pull/7). 0 when the
// text is not a PR URL, so the caller falls back to asking gh.
func numberFromURL(url string) int {
	fields := strings.Fields(strings.TrimSpace(url))
	if len(fields) == 0 {
		return 0
	}
	last := fields[len(fields)-1]
	i := strings.LastIndex(last, "/")
	if i < 0 {
		return 0
	}
	n, err := strconv.Atoi(last[i+1:])
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// prNumber resolves the PR number the review endpoint needs. `gh pr review`
// took a branch; the REST reviews endpoint (the only way to post line-anchored
// comments) takes a number, so we resolve it — from the URL `gh pr create` just
// printed when we have it, otherwise by asking gh, which also covers a PR that
// already existed.
func prNumber(w *workspace.Workspace, branch, url string) (int, error) {
	if n := numberFromURL(url); n > 0 {
		return n, nil
	}
	out, err := runGH(w.Root, "pr", "view", branch, "--json", "number", "-q", ".number")
	if err != nil {
		return 0, fmt.Errorf("could not resolve the PR number for %s: %s", branch, oneLine(out))
	}
	n, aerr := strconv.Atoi(strings.TrimSpace(out))
	if aerr != nil || n <= 0 {
		return 0, fmt.Errorf("could not resolve the PR number for %s from %q", branch, oneLine(out))
	}
	return n, nil
}

// postReview posts ONE review carrying the task's verdicts, its findings as
// line comments on the diff, and a review state.
//
// This is dacli 194's whole point. Across 30 merged PRs in this repo the
// reviews endpoint returned zero reviews, zero line comments, zero issue
// comments: the review work was real but landed only in `.dacli/`, never on the
// artifact a reader inspects. gh runs under runGH's deadline — a wedged gh must
// never hang the caller (the selfreport/018 lesson).
func postReview(ctx *clikit.Ctx, w *workspace.Workspace, t *store.Task, branch, url, event string) error {
	body, comments := reviewPayload(w, t, event)
	if body == "" && len(comments) == 0 {
		fmt.Fprintln(ctx.Stdout, "no findings or recorded verdicts to post — run `dacli verify --task` first")
		return nil
	}
	num, err := prNumber(w, branch, url)
	if err != nil {
		return err
	}
	out, err := runGH(w.Root, reviewArgs(num, event, body, comments)...)
	if err != nil && len(comments) > 0 {
		// GitHub rejects the WHOLE review (422) if any single comment names a
		// line outside the PR's diff — a finding about a file this branch never
		// touched, or a line a later commit moved. Retry once with the findings
		// folded into the summary, so a stale line number costs the anchor, not
		// the review.
		fmt.Fprintf(ctx.Stderr, "note: line-anchored review rejected (%s) — re-posting with the findings in the summary\n", oneLine(out))
		body += "\n" + inlineComments(comments)
		comments = nil
		out, err = runGH(w.Root, reviewArgs(num, event, body, nil)...)
	}
	if err != nil {
		return fmt.Errorf("gh api pulls/%d/reviews failed: %s", num, strings.TrimSpace(out))
	}
	fmt.Fprintf(ctx.Stdout, "posted a %s review on PR #%d with %d line comment(s)\n", event, num, len(comments))
	return nil
}

// cmdMerge integrates one task's branch. A conflict does NOT half-merge: it
// aborts, blocks the task, and files a finding naming the conflicted files —
// because dacli cannot resolve conflicts and must not pretend to.
func cmdMerge(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("merging needs an rw grant")
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task", "into"); err != nil {
		return err
	}
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	return mergeTask(ctx, w, id.ID, t, clikit.OrDash(f.Get("into"), "main"))
}

func mergeTask(ctx *clikit.Ctx, w *workspace.Workspace, actor string, t *store.Task, into string) error {
	branch := BranchFor(t)
	if !gitx.BranchExists(w.Root, branch) {
		return fmt.Errorf("no branch %s to merge", branch)
	}
	if cur := gitx.CurrentBranch(w.Root); cur != into {
		return clikit.Refusedf("checkout %s before merging (currently on %s)", into, cur)
	}
	conflicts, err := gitx.Merge(w.Root, branch, fmt.Sprintf("merge %03d-%s", t.Seq, t.Slug))
	if err != nil {
		return err
	}
	if len(conflicts) > 0 {
		// Block the task and record why — the conflict is now visible work,
		// not a silently-broken tree.
		body := fmt.Sprintf("merge into %s conflicts in: %s — resolve on branch %s, then re-merge", into, strings.Join(conflicts, ", "), branch)
		if _, err := eventlog.Append(w, actor, model.EventBlock, t.ID, "", body); err != nil {
			return err
		}
		if t.Status != model.StatusBlocked {
			// The block must actually persist. If the save/move fails (full disk,
			// read-only tasks dir), reporting "blocked" while the task stays in
			// active/ would let `next` re-hand it out and a supervisor re-spawn
			// onto the conflicted tree (dacli 176) — so surface the write error
			// instead of a false refusal.
			store.AppendLog(t, "blocked on merge conflict")
			if err := store.SaveTask(t); err != nil {
				return fmt.Errorf("merge conflict in %s, but recording the block failed: %w", strings.Join(conflicts, ", "), err)
			}
			if err := store.MoveTask(w, t, model.StatusBlocked); err != nil {
				return fmt.Errorf("merge conflict in %s, but moving the task to blocked failed: %w", strings.Join(conflicts, ", "), err)
			}
		}
		return clikit.Refusedf("merge conflict in %s — task %03d blocked; resolve on %s and re-merge (nothing was half-merged)",
			strings.Join(conflicts, ", "), t.Seq, branch)
	}
	// Clean merge: the worktree's job is done and the branch is now fully
	// merged into `into`, so tear both down — the worktree first (a branch
	// checked out in a worktree cannot be deleted), then the branch, so the
	// merged work stops showing up as integratable. Branch deletion is
	// best-effort: a failed delete leaves a harmless already-merged branch,
	// never a half-merged tree.
	_ = gitx.RemoveWorktree(w.Root, w.WorktreePath(t.Project, t.Seq, t.Slug))
	if _, delErr := gitx.Run(w.Root, "branch", "-D", branch); delErr != nil {
		fmt.Fprintf(ctx.Stdout, "merged %s into %s (worktree removed; branch delete failed: %v)\n", branch, into, delErr)
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "merged %s into %s (worktree removed, branch deleted)\n", branch, into)
	return nil
}

// cmdIntegrate merges task branches into a target branch, SERIALIZED so a
// conflict surfaces one task at a time rather than as a pile-up. It stops at
// the first conflict (that task is now blocked) so a human resolves before the
// rest pile on top, and reports exactly which branches merged before the stop.
//
// Two modes:
//   - `--tasks <ref,ref,...>` integrates an explicit, ordered list (each ref
//     resolved via store.FindTask — seq, id, or slug).
//   - no `--tasks` scans every done task in the project (back-compat).
//
// `--into <branch>` picks the target (default main); the current-branch guard
// compares against it, so integration works into any branch, not just main.
// A clean merge removes the task's worktree and deletes the merged branch.
func cmdIntegrate(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("integrating needs an rw grant")
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("into", "pr", "no-merge", "auto", "merge", "tasks", "project", "force"); err != nil {
		return err
	}
	into := clikit.OrDash(f.Get("into"), "main")
	if cur := gitx.CurrentBranch(w.Root); cur != into {
		return clikit.Refusedf("checkout %s before integrating (currently on %s)", into, cur)
	}
	// PR-first mode: instead of a local `git merge`, push each branch, open an
	// enriched PR (acceptance + findings + Fixes #issue + verify verdicts), and
	// land it via `gh pr merge`. Three sub-modes decide HOW a PR lands:
	//   --auto     set GitHub's native auto-merge (gh pr merge --auto --merge
	//              --delete-branch): GitHub merges each PR the instant its
	//              required checks go green, so the operator never waits on CI.
	//   --no-merge open the PRs and stop for human review — nothing is merged.
	//   (default)  gate each merge on `gh pr checks`: merge only PRs whose checks
	//              already pass, leaving any red/pending PR open rather than
	//              blindly merging it.
	// --merge picks a merge commit over the default squash for the gated path.
	// gh is required up front so we refuse cleanly rather than fail per-task.
	pr := f.Bool("pr")
	noMerge := f.Bool("no-merge")
	auto := f.Bool("auto")
	squash := !f.Bool("merge")
	if pr {
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("gh not on PATH — `dacli integrate --pr` opens PRs via the GitHub CLI (omit --pr for a local merge)")
		}
	}
	tasks, err := integrationTasks(w, f)
	if err != nil {
		return err
	}
	// merged counts branches that landed on `into` NOW; open counts PRs left on
	// GitHub un-merged (--no-merge, --auto queued, or a check not yet passing).
	merged, open := 0, 0
	for _, t := range tasks {
		if !gitx.BranchExists(w.Root, BranchFor(t)) {
			fmt.Fprintf(ctx.Stdout, "%03d-%s: skipped (no branch %s)\n", t.Seq, t.Slug, BranchFor(t))
			continue
		}
		var landed bool
		step := func() error { landed = true; return mergeTask(ctx, w, id.ID, t, into) }
		if pr {
			step = func() (err error) {
				landed, err = prIntegrateTask(ctx, w, id.ID, t, into, noMerge, auto, squash)
				return err
			}
		}
		if err := step(); err != nil {
			// A merge conflict is a Refused (exit 3): mergeTask blocked exactly
			// this task (nothing half-merged) and returned why. Report which
			// branches landed, then stop so a human resolves before the rest
			// pile on top — exit 0, because the block is visible, recorded work.
			if clikit.ExitCode(err) == 3 {
				fmt.Fprintf(ctx.Stdout, "%03d-%s: conflict — %v\n", t.Seq, t.Slug, err)
				fmt.Fprintf(ctx.Stdout, "integrated %d branch(es) into %s before the conflict; resolve it, then re-run\n", merged, into)
				return nil
			}
			// A genuine NON-conflict failure (a dirty code tree, a missing
			// branch, unrelated histories, an index lock, a timeout). Do NOT
			// mislabel it a conflict and do NOT swallow it to exit 0 — that once
			// let `dacli ship` believe integrate succeeded and half-ship a
			// partial record. Report what landed first, then propagate the real
			// error so the caller sees a non-zero exit.
			fmt.Fprintf(ctx.Stdout, "integrated %d branch(es) into %s before the error\n", merged, into)
			return fmt.Errorf("%03d-%s: merge failed: %w", t.Seq, t.Slug, err)
		}
		if landed {
			merged++
		} else {
			open++
		}
	}
	switch {
	case pr && noMerge:
		// --no-merge opened the PRs and stopped: nothing landed on `into`, so say
		// so honestly rather than reporting the count as merged branches (which
		// ship parses into its record-commit message).
		fmt.Fprintf(ctx.Stdout, "opened %d PR(s) targeting %s, none merged (--no-merge) — review and merge them yourself\n", open, into)
	case pr && auto:
		// --auto queued GitHub's native auto-merge on each PR: nothing is merged
		// locally yet — GitHub lands each one when its checks pass. Report the
		// queued count, not a merged count.
		fmt.Fprintf(ctx.Stdout, "queued %d PR(s) for auto-merge targeting %s — GitHub merges each when CI passes (hands-off)\n", open, into)
	default:
		// Local merge, or the gated PR path. Report what actually landed (ship
		// parses this line) and, if the check gate left any PR open, say how many.
		fmt.Fprintf(ctx.Stdout, "integrated %d branch(es) into %s, no conflicts\n", merged, into)
		if open > 0 {
			fmt.Fprintf(ctx.Stdout, "left %d PR(s) open for a pending or failed check — merge them once CI is green (or re-run)\n", open)
		}
	}
	return nil
}

// prIntegrateTask lands one task through GitHub instead of a local merge: push
// its branch, open an enriched PR, then land it via `gh pr merge`. It returns
// landed=true only when the branch is actually merged into `into` NOW (a gh
// merge whose checks passed, or a local-merge fallback); landed=false means the
// PR is left on GitHub un-merged — --no-merge (human review), --auto (GitHub
// merges later when CI passes), or the check gate found a red/pending check.
//
// The documented fallback: if GitHub is UNREACHABLE at push or PR-open, it warns
// and falls back to a local `git merge` so a wave still lands offline — UNLESS
// noMerge or auto is set, in which case the operator explicitly asked GitHub to
// own the merge, so an offline failure is surfaced rather than silently
// local-merged behind their back.
func prIntegrateTask(ctx *clikit.Ctx, w *workspace.Workspace, actor string, t *store.Task, into string, noMerge, auto, squash bool) (bool, error) {
	branch := BranchFor(t)
	// mode names why an offline fallback is refused, so the surfaced error tells
	// the operator which flag asked GitHub to own the merge.
	mode := ""
	if noMerge {
		mode = "--no-merge"
	} else if auto {
		mode = "--auto"
	}
	fallback := func(stage, detail string) (bool, error) {
		if mode != "" {
			return false, fmt.Errorf("%03d-%s: %s failed and GitHub is unreachable; %s asked GitHub to own the merge, so nothing was merged: %s", t.Seq, t.Slug, stage, mode, detail)
		}
		fmt.Fprintf(ctx.Stderr, "warning: %s for %s failed (GitHub unreachable) — falling back to a local merge so the wave still lands: %s\n", stage, branch, detail)
		if err := mergeTask(ctx, w, actor, t, into); err != nil {
			return false, err
		}
		return true, nil
	}

	// 1. push the branch to origin so a PR has a head.
	if out, err := pushBranch(w.Root, branch); err != nil {
		if isNetworkErr(out) || isNetworkErr(err.Error()) {
			return fallback("push", oneLine(out))
		}
		return false, fmt.Errorf("%03d-%s: push %s failed: %s", t.Seq, t.Slug, branch, strings.TrimSpace(out))
	}
	fmt.Fprintf(ctx.Stdout, "%03d-%s: pushed %s\n", t.Seq, t.Slug, branch)

	// 2. open the enriched PR (body + verify verdicts + line-anchored findings).
	//    Base is `into`. The review state stays COMMENT: an integration run
	//    merges on its own gate (checks / --auto), so approving its own PR would
	//    be a rubber stamp, not a review.
	url, reused, err := openPR(ctx, w, actor, t, into, true, reviewComment, false)
	if err != nil {
		if isNetworkErr(err.Error()) {
			return fallback("opening a PR", err.Error())
		}
		return false, fmt.Errorf("%03d-%s: %w", t.Seq, t.Slug, err)
	}
	if reused {
		fmt.Fprintf(ctx.Stdout, "%03d-%s: PR already open, reusing %s\n", t.Seq, t.Slug, url)
	} else {
		fmt.Fprintf(ctx.Stdout, "%03d-%s: PR opened %s\n", t.Seq, t.Slug, url)
	}
	if noMerge {
		fmt.Fprintf(ctx.Stdout, "%03d-%s: left open for human review (--no-merge)\n", t.Seq, t.Slug)
		return false, nil
	}

	// 3a. --auto: set GitHub's native auto-merge and STOP. GitHub merges the PR
	//     the instant its required checks go green and deletes the branch, so the
	//     operator never waits on CI or merges by hand. The branch is NOT merged
	//     locally yet, so we keep the worktree/branch — GitHub owns the merge.
	if auto {
		out, err := runGH(w.Root, "pr", "merge", branch, "--auto", "--merge", "--delete-branch")
		if err != nil {
			if isNetworkErr(out) || isNetworkErr(err.Error()) {
				return false, fmt.Errorf("%03d-%s: gh pr merge --auto failed and GitHub is unreachable; --auto asked GitHub to own the merge, so nothing was merged: %s", t.Seq, t.Slug, oneLine(out))
			}
			return false, fmt.Errorf("%03d-%s: gh pr merge --auto failed: %s", t.Seq, t.Slug, strings.TrimSpace(out))
		}
		fmt.Fprintf(ctx.Stdout, "%03d-%s: auto-merge set — GitHub merges %s when CI passes\n", t.Seq, t.Slug, url)
		return false, nil
	}

	// 3b. default (gated): merge ONLY if the PR's checks already pass. A red or
	//     pending check leaves the PR open rather than blindly merging it —
	//     `dacli integrate --pr` never merges over a failing gate. A repo with NO
	//     checks configured is left open too, not silently treated as green: an
	//     absent gate is not a passed gate (dacli 216).
	pass, absent, detail, netErr := prChecksPass(w.Root, branch)
	if netErr {
		// Can't reach GitHub to read the checks; land locally so the wave still
		// completes offline (same philosophy as a push/PR-open network failure).
		fmt.Fprintf(ctx.Stderr, "warning: gh pr checks for %s failed (GitHub unreachable) — falling back to a local merge: %s\n", branch, detail)
		if err := mergeTask(ctx, w, actor, t, into); err != nil {
			return false, err
		}
		return true, nil
	}
	if absent {
		// "no checks reported" has two very different causes, and conflating them
		// gives dangerous advice. If the repo has NO workflow at all, merging by
		// hand is the honest answer — there is no CI to wait for. But if the repo
		// HAS a workflow and this PR's head SHA still got no run, that is the
		// silent pull_request-trigger race that left three of this session's
		// branches unmergeable with no signal (dacli 263): the branch is NOT a
		// CI-less repo, it is a verified-nothing PR that will sit forever, and
		// advising a blind self-merge there would land unverified code. Name it
		// loudly as needing attention, with the actual recovery — a fresh head SHA
		// (merge origin/main and push → a synchronize event) or a manual dispatch
		// of the workflow dacli now writes with workflow_dispatch.
		if repoHasCIWorkflows(w.Root) {
			fmt.Fprintf(ctx.Stdout, "%03d-%s: PR NEEDS ATTENTION — the repo has CI but %s got no check run at all (the pull_request trigger silently did not fire; dacli 263). It is unmergeable until a check runs — do NOT merge unverified. Recover with: merge origin/main into %s and push (fires a fresh run), or `gh workflow run ci.yml --ref %s` to dispatch it\n", t.Seq, t.Slug, url, branch, branch)
			return false, nil
		}
		fmt.Fprintf(ctx.Stdout, "%03d-%s: PR left open — no checks configured on this repo; merge %s yourself, or set up CI, or use --auto/--no-merge\n", t.Seq, t.Slug, url)
		return false, nil
	}
	if !pass {
		fmt.Fprintf(ctx.Stdout, "%03d-%s: PR left open — checks not passing (%s); merge %s once CI is green\n", t.Seq, t.Slug, detail, url)
		return false, nil
	}

	// 3c. checks pass: merge via gh. --delete-branch cleans up the remote branch;
	//     we tear the local worktree and branch down ourselves so the merged work
	//     stops showing up as integratable (mirroring the local mergeTask path).
	strategy := "--squash"
	if !squash {
		strategy = "--merge"
	}
	out, err := runGH(w.Root, "pr", "merge", branch, strategy, "--delete-branch")
	if err != nil {
		// Same discriminator as prChecksPass: gh REFUSING to merge (red checks,
		// a conflict, branch protection) is an answer, not an outage — and only
		// an outage may fall through to a local merge. Without this a refused
		// merge whose text happened to contain a network word landed the branch
		// on trunk anyway and reported it merged (task 317).
		if !reachedGitHub(out) && (isNetworkErr(out) || isNetworkErr(err.Error())) {
			// The PR is open on GitHub but unmergeable right now; land it locally so
			// the wave still completes. The already-open PR is a harmless duplicate
			// record of the same change.
			fmt.Fprintf(ctx.Stderr, "warning: gh pr merge for %s failed (GitHub unreachable) — falling back to a local merge: %s\n", branch, oneLine(out))
			if err := mergeTask(ctx, w, actor, t, into); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, fmt.Errorf("%03d-%s: gh pr merge failed: %s", t.Seq, t.Slug, strings.TrimSpace(out))
	}
	fmt.Fprintf(ctx.Stdout, "%03d-%s: merged via gh (%s) %s\n", t.Seq, t.Slug, strings.TrimPrefix(strategy, "--"), url)
	_ = gitx.RemoveWorktree(w.Root, w.WorktreePath(t.Project, t.Seq, t.Slug))
	_, _ = gitx.Run(w.Root, "branch", "-D", branch)
	// Fast-forward the local target to the merge gh just made on the remote, so a
	// subsequent record commit / push (dacli ship) sits on top of the merged
	// state instead of behind it. Best-effort: no remote (or a diverged local)
	// leaves a note, never a failure — the merge already landed on GitHub.
	if out, err := gitx.Run(w.Root, "pull", "--ff-only"); err != nil {
		fmt.Fprintf(ctx.Stderr, "note: local %s not fast-forwarded to the merged remote state: %s\n", into, oneLine(out))
	}
	return true, nil
}

// prChecksPass reports whether the PR for `branch` has all its checks passing,
// by the exit code of `gh pr checks`: exit 0 means every required check is
// green. A non-zero exit means a check is failing or still pending (the gate
// keeps the PR open). "no checks reported" — a repo with no CI configured —
// is reported as its own `absent` state, distinct from `pass`: a gate that has
// nothing to check is not the same as a gate that checked and passed, and
// treating the two alike let a repo with no CI merge every PR as if it were
// green (dacli 216). netErr is true when GitHub was unreachable, so the
// caller can fall back to a local merge rather than leave the PR open forever.
func prChecksPass(root, branch string) (pass, absent bool, detail string, netErr bool) {
	out, err := runGH(root, "pr", "checks", branch)
	if err == nil {
		return true, false, oneLine(out), false
	}
	// A network error means gh could not REACH GitHub. If gh printed a checks
	// table, it plainly did reach it — whatever the rows happen to say.
	//
	// This mattered because runGH uses CombinedOutput, so `out` is gh's own
	// checks TABLE, and isNetworkErr is a bare substring scan for tokens like
	// "timeout", "unreachable" and "eof". A check named `integration-timeout`,
	// or a failing job whose row text contains one of those words, made a RED
	// run look like an outage — and the caller answers an outage by
	// local-merging the branch into trunk (prIntegrateTask). Failing CI was
	// therefore a path to landing unverified code.
	if !reachedGitHub(out) && (isNetworkErr(out) || isNetworkErr(err.Error())) {
		return false, false, oneLine(out), true
	}
	if strings.Contains(strings.ToLower(out), "no checks reported") {
		return false, true, "no checks reported", false
	}
	return false, false, oneLine(out), false
}

// repoHasCIWorkflows reports whether the repo at root carries any GitHub Actions
// workflow (a .yml/.yaml under .github/workflows). It is how the check gate tells
// a repo that genuinely has no CI — merge by hand is honest there — from one that
// HAS CI yet whose PR got no run at all: the silent pull_request-trigger race
// (dacli 263) where a branch pushed and its PR opened seconds later lands the
// event with no new commit to run against, leaving the branch unmergeable with no
// signal. The onboard slice has an existingWorkflow twin; feature slices never
// import each other, so this is a deliberate small duplicate (the arch rule), not
// a shared import.
func repoHasCIWorkflows(root string) bool {
	entries, err := os.ReadDir(filepath.Join(root, ".github", "workflows"))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".yml", ".yaml":
			return true
		}
	}
	return false
}

// oneLine collapses multi-line command output to a single line for a warning.
func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// integrationTasks resolves which tasks a `dacli integrate` run should merge:
// an explicit `--tasks <ref,ref,...>` list (order preserved, resolved via
// store.FindTask) when given, otherwise every done task in the project.
//
// The done filter is the acceptance gate. Without --tasks it is structural —
// ListTasks only returns StatusDone — but a named list used to walk straight
// past it, and that is how fourteen tasks whose code had been merged for hours
// stayed in open/ while `dacli next` went on ranking them `must` (dacli 257).
// Naming a task is a statement about which BRANCH to merge, never a claim that
// the work was accepted.
func integrationTasks(w *workspace.Workspace, f *clikit.Flags) ([]*store.Task, error) {
	list := f.Get("tasks")
	if list == "" {
		return store.ListTasks(w, f.Get("project"), model.StatusDone)
	}
	var tasks []*store.Task
	var notDone []string
	for _, ref := range strings.Split(list, ",") {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		t, err := store.FindTask(w, ref)
		if err != nil {
			return nil, fmt.Errorf("resolve --tasks %q: %w", ref, err)
		}
		if t.Status != model.StatusDone {
			notDone = append(notDone, fmt.Sprintf("%03d-%s (%s)", t.Seq, t.Slug, clikit.OrDash(string(t.Status))))
		}
		tasks = append(tasks, t)
	}
	if len(tasks) == 0 {
		return nil, clikit.Usagef("--tasks was empty; give a comma-separated list of task refs")
	}
	// Refused, not usage: the command line is well-formed and the answer is
	// "not yet". Every named task is listed, so one run tells you the whole
	// set to close rather than one per attempt.
	if len(notDone) > 0 && !f.Bool("force") {
		return nil, clikit.Refusedf("not done: %s — merging leaves the task open and `dacli next` keeps ranking it. Close it with `dacli accept <ref> --verify \"<cmd>\"` (add --force to accept for a finished agent), or pass --force here to merge the branch anyway", strings.Join(notDone, ", "))
	}
	return tasks, nil
}
