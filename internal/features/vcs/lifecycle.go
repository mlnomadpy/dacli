// The parallel-agent git lifecycle: isolated worktrees so agents work at once
// without clobbering each other, push, PR, and conflict-aware merge/integrate.
// This is what turns `dacli next --parallel N` from advice into real
// concurrent work.
package vcs

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func init() {
	Commands = append(Commands,
		clikit.Command{Path: "worktree add", Brief: "Isolated worktree+branch for a task so parallel agents don't collide", Run: cmdWorktreeAdd},
		clikit.Command{Path: "worktree list", Brief: "Active worktrees and their branches", Run: cmdWorktreeList},
		clikit.Command{Path: "worktree remove", Brief: "Tear down a task's worktree", Run: cmdWorktreeRemove},
		clikit.Command{Path: "push", Brief: "Push a task's branch to origin", Run: cmdPush},
		clikit.Command{Path: "pr", Brief: "Open a PR for a task's branch (gh); body carries acceptance + findings + Fixes #issue. --with-verdicts posts the verify panel's verdicts as a PR review", Run: cmdPR},
		clikit.Command{Path: "merge", Brief: "Merge a task's branch; a conflict blocks the task, never half-merges", Run: cmdMerge},
		clikit.Command{Path: "integrate", Brief: "Merge task branches (--tasks <refs> or all done) into --into <branch>; --pr opens a PR per branch and merges via gh (--auto sets GitHub auto-merge on CI green, default gates on gh pr checks, --no-merge stops for review), else a local merge", Run: cmdIntegrate},
	)
}

// BranchFor is the task branch convention, shared with the git_workflow prompt.
func BranchFor(t *store.Task) string {
	return fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug)
}

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
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	branch, path := BranchFor(t), w.WorktreePath(t.Slug)
	if err := gitx.AddWorktree(w.Root, path, branch); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "worktree ready: %s (branch %s)\n", path, branch)
	fmt.Fprintf(ctx.Stdout, "an agent works here in isolation; commit with `dacli commit`, then `dacli push`/`dacli pr`\n")
	return nil
}

func cmdWorktreeList(ctx *clikit.Ctx, args []string) error {
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
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	if err := gitx.RemoveWorktree(w.Root, w.WorktreePath(t.Slug)); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "removed worktree for %03d-%s\n", t.Seq, t.Slug)
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
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	branch := BranchFor(t)
	if !gitx.BranchExists(w.Root, branch) {
		return fmt.Errorf("no branch %s — `dacli worktree add --task %03d` and commit first", branch, t.Seq)
	}
	out, err := gitx.Push(w.Root, branch)
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
	// --with-verdicts a `gh pr review` that posts the task's finding notes and
	// verify verdicts to (possibly public) origin. Gate it behind rw like every
	// other outward vcs command (push/merge/integrate), so a read-only agent
	// cannot leak internal findings to GitHub (brief rank-2 risk).
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("opening a PR needs an rw grant (yours is %s)", id.Grant)
	}
	f, _ := clikit.ParseFlags(args)
	t, err := resolveTaskFlag(w, f)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh not on PATH — `dacli pr` opens the PR via the GitHub CLI")
	}
	base := clikit.OrDash(f.Get("base"), "main")
	url, err := openPR(ctx, w, id.ID, t, base, f.Bool("with-verdicts"))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "PR opened and recorded: %s\n", url)
	return nil
}

// openPR opens (via gh) an enriched PR for the task's ALREADY-PUSHED branch,
// records the URL, and — when withVerdicts — posts the verify panel's verdicts
// as a review comment. It returns the PR URL and, on failure, an error whose
// text carries gh's stderr so a caller can tell a network failure (fall back to
// a local merge) from a real one (surface it). It does not push: the branch
// must already be on origin (cmdPush / the --pr integrate path pushes first).
func openPR(ctx *clikit.Ctx, w *workspace.Workspace, actor string, t *store.Task, base string, withVerdicts bool) (string, error) {
	branch := BranchFor(t)
	body := prBody(w, t)
	// gh talks to GitHub over the network; runGH bounds it with a deadline so a
	// wedged request can never hang the caller (or, under `dacli mcp serve`, the
	// stdio loop).
	out, err := runGH(w.Root, "pr", "create", "--head", branch, "--base", base,
		"--title", fmt.Sprintf("%03d: %s", t.Seq, t.Title), "--body", body)
	if err != nil {
		return "", fmt.Errorf("gh pr create failed: %s", strings.TrimSpace(out))
	}
	url := strings.TrimSpace(out)
	// An unrecorded PR does not exist: record the URL so it enters the workspace
	// and every future brief for the task. A PR-opened event is an operational
	// fact, not a code defect — record it as a comment, NOT a finding. An
	// EventFinding syncs into a durable, never-graded NoteFinding, which drags
	// the task's brief trust-floor to `unverified` forever and consumes a
	// finding slot; a comment lands as a Log line and does neither.
	if _, err := eventlog.Append(w, actor, model.EventComment, t.ID, "", "PR opened: "+url); err != nil {
		return url, err
	}
	// Operator-triggered only: mirror the verify panel's recorded verdicts onto
	// the PR as a review comment so human review sees the model's adversarial
	// checks. A post failure is a note, not a hard error: the PR itself already
	// exists and is recorded.
	if withVerdicts {
		if err := postVerdicts(ctx, w, t, branch); err != nil {
			fmt.Fprintf(ctx.Stderr, "note: verdicts not posted: %v\n", err)
		}
	}
	return url, nil
}

// prBody assembles the PR description from what dacli already knows about the
// task: the acceptance criteria, the finding notes agents flagged, and a
// `Fixes #<issue>` line when the task is mirrored to a GitHub issue (so merging
// the PR closes it). It touches no network, so it is unit-testable on fixtures.
func prBody(w *workspace.Workspace, t *store.Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Implements dacli task %03d-%s.\n", t.Seq, t.Slug)
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

// taskFindings renders the task's finding notes into a PR section, so a human
// reviewer sees what the agents flagged. Only findings whose `about` names this
// task are included (by id or by NNN sequence, matching how verify resolves the
// task's findings).
func taskFindings(w *workspace.Workspace, t *store.Task) string {
	notes, _ := store.ListNotes(w, t.Project, model.NoteFinding)
	var b strings.Builder
	for _, n := range notes {
		about, _ := n.Front.Get("about")
		if !strings.Contains(about, t.ID) && !strings.Contains(about, fmt.Sprintf("%03d", t.Seq)) {
			continue
		}
		// The note body lives inside its level-1 title section (content runs to
		// the next heading), so collect every section's content — the same rule
		// the brief assembler uses.
		var body strings.Builder
		for _, s := range n.Sections {
			body.WriteString(s.Content)
		}
		text := strings.TrimSpace(body.String())
		if text == "" {
			continue
		}
		var tags strings.Builder
		if sev, _ := n.Front.Get("severity"); sev != "" {
			fmt.Fprintf(&tags, "**%s** ", sev)
		}
		if trust, _ := n.Front.Get("trust"); trust != "" {
			fmt.Fprintf(&tags, "[trust: %s] ", trust)
		}
		fmt.Fprintf(&b, "- %s%s\n", tags.String(), text)
	}
	if b.Len() == 0 {
		return ""
	}
	return "### Findings\n" + b.String()
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
	return "### dacli verify panel\n\nThe adversarial verification panel's verdicts on this task's claims:\n\n" + strings.Join(lines, "\n") + "\n"
}

// postVerdicts posts the task's recorded panel verdicts as a single PR review
// comment (gh pr review --comment). gh runs under a deadline — a wedged gh must
// never hang the caller (the selfreport/018 lesson). The branch resolves the PR,
// so no PR number is needed.
func postVerdicts(ctx *clikit.Ctx, w *workspace.Workspace, t *store.Task, branch string) error {
	body := verdictReview(w, t)
	if body == "" {
		fmt.Fprintln(ctx.Stdout, "no recorded verify verdicts to post — run `dacli verify --task` first")
		return nil
	}
	out, err := runGH(w.Root, "pr", "review", branch, "--comment", "--body", body)
	if err != nil {
		return fmt.Errorf("gh pr review failed: %s", strings.TrimSpace(out))
	}
	fmt.Fprintln(ctx.Stdout, "posted verify verdicts as a PR review comment")
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
			store.AppendLog(t, "blocked on merge conflict")
			_ = store.SaveTask(t)
			_ = store.MoveTask(w, t, model.StatusBlocked)
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
	_ = gitx.RemoveWorktree(w.Root, w.WorktreePath(t.Slug))
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

	// 2. open the enriched PR (body + verify verdicts). Base is `into`.
	url, err := openPR(ctx, w, actor, t, into, true)
	if err != nil {
		if isNetworkErr(err.Error()) {
			return fallback("opening a PR", err.Error())
		}
		return false, fmt.Errorf("%03d-%s: %w", t.Seq, t.Slug, err)
	}
	fmt.Fprintf(ctx.Stdout, "%03d-%s: PR opened %s\n", t.Seq, t.Slug, url)
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
	//     `dacli integrate --pr` never merges over a failing gate.
	pass, detail, netErr := prChecksPass(w.Root, branch)
	if netErr {
		// Can't reach GitHub to read the checks; land locally so the wave still
		// completes offline (same philosophy as a push/PR-open network failure).
		fmt.Fprintf(ctx.Stderr, "warning: gh pr checks for %s failed (GitHub unreachable) — falling back to a local merge: %s\n", branch, detail)
		if err := mergeTask(ctx, w, actor, t, into); err != nil {
			return false, err
		}
		return true, nil
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
		if isNetworkErr(out) || isNetworkErr(err.Error()) {
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
	_ = gitx.RemoveWorktree(w.Root, w.WorktreePath(t.Slug))
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
// keeps the PR open) — EXCEPT "no checks reported", which means nothing gates
// the merge and is treated as passing. netErr is true when GitHub was
// unreachable, so the caller can fall back to a local merge rather than leave
// the PR open forever.
func prChecksPass(root, branch string) (pass bool, detail string, netErr bool) {
	out, err := runGH(root, "pr", "checks", branch)
	if err == nil {
		return true, oneLine(out), false
	}
	if isNetworkErr(out) || isNetworkErr(err.Error()) {
		return false, oneLine(out), true
	}
	if strings.Contains(strings.ToLower(out), "no checks reported") {
		return true, "no checks reported", false
	}
	return false, oneLine(out), false
}

// oneLine collapses multi-line command output to a single line for a warning.
func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// integrationTasks resolves which tasks a `dacli integrate` run should merge:
// an explicit `--tasks <ref,ref,...>` list (order preserved, resolved via
// store.FindTask) when given, otherwise every done task in the project.
func integrationTasks(w *workspace.Workspace, f *clikit.Flags) ([]*store.Task, error) {
	list := f.Get("tasks")
	if list == "" {
		return store.ListTasks(w, f.Get("project"), model.StatusDone)
	}
	var tasks []*store.Task
	for _, ref := range strings.Split(list, ",") {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		t, err := store.FindTask(w, ref)
		if err != nil {
			return nil, fmt.Errorf("resolve --tasks %q: %w", ref, err)
		}
		tasks = append(tasks, t)
	}
	if len(tasks) == 0 {
		return nil, clikit.Usagef("--tasks was empty; give a comma-separated list of task refs")
	}
	return tasks, nil
}
