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
		clikit.Command{Path: "integrate", Brief: "Merge task branches (--tasks <refs> or all done) into --into <branch>; clean merges remove the worktree and delete the branch", Run: cmdIntegrate},
	)
}

// BranchFor is the task branch convention, shared with the git_workflow prompt.
func BranchFor(t *store.Task) string {
	return fmt.Sprintf("dacli/%03d-%s", t.Seq, t.Slug)
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
	branch := BranchFor(t)
	base := clikit.OrDash(f.Get("base"), "main")
	body := prBody(w, t)
	// gh talks to GitHub over the network; a deadline keeps a wedged request
	// from hanging the caller (and, under `dacli mcp serve`, the stdio loop).
	pctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := exec.CommandContext(pctx, "gh", "pr", "create", "--head", branch, "--base", base,
		"--title", fmt.Sprintf("%03d: %s", t.Seq, t.Title), "--body", body).Output()
	if pctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("gh pr create timed out")
	}
	if err != nil {
		return fmt.Errorf("gh pr create failed: %v", err)
	}
	url := strings.TrimSpace(string(out))
	// An unrecorded PR does not exist: record the URL so it enters the workspace
	// and every future brief for the task. A PR-opened event is an operational
	// fact, not a code defect — record it as a comment, NOT a finding. An
	// EventFinding syncs into a durable, never-graded NoteFinding, which drags
	// the task's brief trust-floor to `unverified` forever and consumes a
	// finding slot; a comment lands as a Log line and does neither.
	if _, err := eventlog.Append(w, id.ID, model.EventComment, t.ID, "", "PR opened: "+url); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "PR opened and recorded: %s\n", url)
	// Operator-triggered only: mirror the verify panel's recorded verdicts onto
	// the PR as a review comment so human review sees the model's adversarial
	// checks. Never automatic — posting to GitHub is a flag. A post failure is a
	// note, not a hard error: the PR itself already exists and is recorded.
	if f.Bool("with-verdicts") {
		if err := postVerdicts(ctx, w, t, branch); err != nil {
			fmt.Fprintf(ctx.Stderr, "note: verdicts not posted: %v\n", err)
		}
	}
	return nil
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
	pctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := exec.CommandContext(pctx, "gh", "pr", "review", branch, "--comment", "--body", body).CombinedOutput()
	if pctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("gh pr review timed out")
	}
	if err != nil {
		return fmt.Errorf("gh pr review failed: %s", strings.TrimSpace(string(out)))
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
	tasks, err := integrationTasks(w, f)
	if err != nil {
		return err
	}
	merged := 0
	for _, t := range tasks {
		if !gitx.BranchExists(w.Root, BranchFor(t)) {
			fmt.Fprintf(ctx.Stdout, "%03d-%s: skipped (no branch %s)\n", t.Seq, t.Slug, BranchFor(t))
			continue
		}
		if err := mergeTask(ctx, w, id.ID, t, into); err != nil {
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
		merged++
	}
	fmt.Fprintf(ctx.Stdout, "integrated %d branch(es) into %s, no conflicts\n", merged, into)
	return nil
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
