// Package ship is the one-command wave tail: `dacli ship` closes the loop the
// operator still runs by hand after a wave of agents finish — accept the done
// tasks, integrate their branches, commit the workspace record SAFELY, and
// (optionally) push. Every step the operator used to type is now one command
// that stops honestly at the first failure and never half-ships.
//
// It is a feature slice (ARCHITECTURE § 2b) and imports NO other slice. Because
// slices cannot call each other, ship ORCHESTRATES by shelling out to its own
// binary (os.Executable) — `dacli accept`, `dacli integrate` — exactly as the
// prompt templates tell agents to invoke dacli. The record commit and push are
// git operations done directly through the shared gitx layer (an entity
// package, so importing it is allowed): the record commit lands on the
// integration branch, where `dacli commit` refuses to run.
//
// The pipeline, stopping at the first non-zero step so nothing is left
// half-shipped:
//
//  1. accept   — shell `dacli accept --all --force [--verify "cmd"]`: verify-
//     then-close every task an agent proposed for acceptance. --force is
//     always passed — `accept` only honors it for root, so it reconciles a
//     wave's tasks left owned by an agent that already finished (and will
//     never sync to apply its own proposal) instead of stalling on the orphan.
//  2. integrate— shell `dacli integrate --tasks <done seqs> --into <branch>`:
//     merge each done task's branch. A conflict blocks that task; ship
//     detects the block and stops before committing or pushing.
//  3. record   — stage ONLY .dacli (NEVER `git add -A`, the footgun that once
//     tracked a worktree gitlink) and commit the workspace state.
//  4. push     — with --push, push the integration branch; without it, print the
//     push command so the operator stays in control.
//
// `--dry-run` prints each step it would run and executes nothing.
package ship

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Commands is this slice's table, aggregated by the app layer (cli.go).
var Commands = []clikit.Command{
	{Path: "ship", Brief: "One wave tail: accept done tasks, integrate their branches (--pr opens PRs via gh — --auto sets GitHub auto-merge on CI green, default merges only checks-passing PRs, --no-merge stops for review), commit the .dacli record, optionally push", Run: cmdShip},
}

// shellDacli runs a dacli subcommand by shelling this binary, so ship
// orchestrates the accept/integrate steps without importing sibling slices. It
// runs from the workspace root (where the integration branch is checked out and
// .dacli lives) and inherits the environment, so the child resolves the same
// workspace and carries the operator's DACLI_AGENT for attribution. It is a
// package variable so a test can substitute an in-process runner.
var shellDacli = func(ctx *clikit.Ctx, w *workspace.Workspace, args ...string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = "dacli"
	}
	c := exec.Command(exe, args...)
	c.Dir = w.Root
	out, err := c.CombinedOutput()
	// Stream the child's output so the operator sees each step's per-task result.
	fmt.Fprint(ctx.Stdout, string(out))
	return string(out), err
}

func cmdShip(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	into := clikit.OrDash(f.Get("into"), "main")
	dry := f.Bool("dry-run")

	if !gitx.Available() {
		return fmt.Errorf("git not on PATH")
	}
	if dry {
		return printPlan(ctx, w, f, into)
	}
	// The real pipeline writes to the repo (integrate/commit/push).
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("ship integrates, commits and pushes; that needs an rw grant (yours is %s)", id.Grant)
	}
	// Guard the branch UP FRONT: shipping onto a branch other than `into` is
	// never intended, and catching it here means accept never runs against a
	// pipeline that integrate would only refuse a step later (never half-ship).
	if cur := gitx.CurrentBranch(w.Root); cur != into {
		return clikit.Refusedf("checkout %s before shipping (currently on %s) — ship integrates and records onto --into", into, cur)
	}

	// 1. accept — verify-then-close every proposed task. A failed verify (or any
	//    non-zero exit) stops the pipeline here: nothing has been integrated,
	//    committed or pushed yet.
	if !f.Bool("no-accept") {
		// --force is always forwarded: `dacli accept` only honors it for the
		// root identity, so this is a no-op unless ship itself is running as
		// root — but when it is, a wave's orphaned tasks (owned by a spawned
		// agent that has since finished and will never sync) get reconciled
		// and closed instead of sitting as a pending proposal forever.
		acceptArgs := []string{"accept", "--all", "--force"}
		if v := f.Get("verify"); v != "" {
			acceptArgs = append(acceptArgs, "--verify", v)
		}
		if _, err := shellDacli(ctx, w, acceptArgs...); err != nil {
			return fmt.Errorf("ship stopped at accept (nothing integrated, committed or pushed): %w", err)
		}
	}

	// 2. integrate — merge each done task's branch into `into`. Resolve the done
	//    set AFTER accept so freshly-closed tasks are included.
	done, err := store.ListTasks(w, f.Get("project"), model.StatusDone)
	if err != nil {
		return err
	}
	// merged counts the branches integrate ACTUALLY merged, so the record commit
	// message reports what really landed — not the raw done-task count, which
	// overstates it whenever a task has no branch (skipped) or a merge fails.
	merged := 0
	if !f.Bool("no-integrate") {
		if len(done) == 0 {
			fmt.Fprintln(ctx.Stdout, "integrate: no done tasks to integrate")
		} else {
			iargs := []string{"integrate", "--tasks", strings.Join(doneRefs(done), ","), "--into", into}
			if p := f.Get("project"); p != "" {
				iargs = append(iargs, "--project", p)
			}
			// PR-first integration: pass the mode through to `dacli integrate`,
			// which pushes each branch, opens an enriched PR, and merges via gh
			// (falling back to a local merge if GitHub is unreachable). --no-merge
			// opens the PRs and stops for human review; --merge picks a merge commit
			// over the default squash. Default (no --pr) keeps the local-merge path.
			iargs = append(iargs, prFlags(f)...)
			out, err := shellDacli(ctx, w, iargs...)
			if err != nil {
				// integrate now propagates a genuine (non-conflict) merge failure
				// as a non-zero exit — a dirty code tree, a missing branch,
				// unrelated histories. Stop here: nothing has been recorded or
				// pushed, so a hard integrate failure can never half-ship.
				return fmt.Errorf("ship stopped at integrate (workspace record not committed, nothing pushed): %w", err)
			}
			// integrate exits 0 even on a conflict (it prints the conflict and
			// blocks that one task). Detect the block SEMANTICALLY — a task from
			// our done set now sitting in blocked — and stop before recording or
			// pushing, so a conflict never half-ships.
			if b := blockedAmong(w, done); len(b) > 0 {
				return clikit.Refusedf("ship stopped: task(s) %s blocked on a merge conflict — resolve on the branch, then re-run ship (nothing committed or pushed)", strings.Join(b, ", "))
			}
			merged = integratedCount(out)
		}
	}

	// 3. record — commit the .dacli workspace state, staging ONLY .dacli. The
	//    message reports branches ACTUALLY merged, never the done-task count.
	if err := commitRecord(ctx, w, id, merged); err != nil {
		return err
	}

	// 4. push — opt-in, so the operator decides when work leaves the machine.
	branch := gitx.CurrentBranch(w.Root)
	if f.Bool("push") {
		out, err := gitx.Push(w.Root, branch)
		if err != nil {
			return fmt.Errorf("push failed: %s", out)
		}
		fmt.Fprintf(ctx.Stdout, "pushed %s to origin\n", branch)
	} else {
		fmt.Fprintf(ctx.Stdout, "not pushed (no --push). To push: git push -u origin %s\n", branch)
	}
	return nil
}

// prFlags forwards the PR-first integration flags ship accepts to the
// `dacli integrate` child, so `dacli ship --pr [--auto] [--no-merge] [--merge]`
// behaves exactly like the same flags on integrate. --auto sets GitHub's native
// auto-merge (merge on CI green, hands-off); absent --pr, it returns nothing and
// the local-merge path is unchanged.
func prFlags(f *clikit.Flags) []string {
	if !f.Bool("pr") {
		return nil
	}
	out := []string{"--pr"}
	if f.Bool("auto") {
		out = append(out, "--auto")
	}
	if f.Bool("no-merge") {
		out = append(out, "--no-merge")
	}
	if f.Bool("merge") {
		out = append(out, "--merge")
	}
	return out
}

// commitRecord stages ONLY the .dacli record and commits it, attributed to the
// acting agent. `git add -- .dacli` is the whole safety property: .dacli/.gitignore
// already excludes runs/build/worktrees, so nothing regenerable or code is
// swept — and we NEVER `git add -A`, the operator footgun that tracked a
// worktree gitlink this session. A belt-and-suspenders check refuses if anything
// outside .dacli somehow landed staged.
func commitRecord(ctx *clikit.Ctx, w *workspace.Workspace, id *agentid.Identity, integrated int) error {
	if out, err := gitx.Run(w.Root, "add", "--", ".dacli"); err != nil {
		return fmt.Errorf("git add .dacli: %s", out)
	}
	staged, _ := gitx.Run(w.Root, "diff", "--cached", "--name-only")
	staged = strings.TrimSpace(staged)
	if staged == "" {
		fmt.Fprintln(ctx.Stdout, "workspace record: nothing to commit (.dacli unchanged)")
		return nil
	}
	for _, p := range strings.Split(staged, "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p != workspace.Dir && !strings.HasPrefix(p, workspace.Dir+"/") {
			return fmt.Errorf("refusing to commit: %s is staged outside %s — ship records the workspace only, never code/worktrees/build", p, workspace.Dir)
		}
	}

	name := authorName(id.ID, id.Role)
	email := id.ID + "@agent.dacli"
	msg := fmt.Sprintf("ship: record workspace after integrating %d task(s)", integrated)
	trailers := "\n\nDacli-Agent: " + id.ID
	if id.Role != "" {
		trailers += "\nDacli-Role: " + id.Role
	}
	out, err := gitx.Run(w.Root,
		"-c", "user.name="+name, "-c", "user.email="+email,
		"commit", "--author", fmt.Sprintf("%s <%s>", name, email), "-m", msg+trailers)
	if err != nil {
		return fmt.Errorf("record commit failed: %s", out)
	}
	sha, _ := gitx.Run(w.Root, "rev-parse", "--short", "HEAD")
	fmt.Fprintf(ctx.Stdout, "workspace record committed %s (staged only %s)\n", sha, workspace.Dir)
	return nil
}

// printPlan renders every step ship WOULD run, executing nothing (--dry-run).
func printPlan(ctx *clikit.Ctx, w *workspace.Workspace, f *clikit.Flags, into string) error {
	done, _ := store.ListTasks(w, f.Get("project"), model.StatusDone)
	fmt.Fprintln(ctx.Stdout, "dry-run: dacli ship would run these steps (nothing executed)")

	switch {
	case f.Bool("no-accept"):
		fmt.Fprintln(ctx.Stdout, "  1. accept:    (skipped: --no-accept)")
	default:
		line := "dacli accept --all --force"
		if v := f.Get("verify"); v != "" {
			line += fmt.Sprintf(" --verify %q", v)
		}
		fmt.Fprintf(ctx.Stdout, "  1. accept:    %s\n", line)
	}

	switch {
	case f.Bool("no-integrate"):
		fmt.Fprintln(ctx.Stdout, "  2. integrate: (skipped: --no-integrate)")
	case len(done) == 0:
		fmt.Fprintln(ctx.Stdout, "  2. integrate: (nothing: no done tasks)")
	default:
		mode := "local merge"
		if f.Bool("pr") {
			mode = "PR-first via gh, merge only checks-passing PRs"
			switch {
			case f.Bool("auto"):
				mode = "open PRs + set GitHub auto-merge on CI green (--auto)"
			case f.Bool("no-merge"):
				mode = "open PRs, stop for review (--no-merge)"
			}
		}
		fmt.Fprintf(ctx.Stdout, "  2. integrate: dacli integrate --tasks %s --into %s %s  (%d done: %s) [%s]\n",
			strings.Join(doneRefs(done), ","), into, strings.Join(prFlags(f), " "), len(done), doneLabels(done), mode)
	}

	fmt.Fprintf(ctx.Stdout, "  3. record:    git add %s && git commit   (stages ONLY %s — never git add -A)\n", workspace.Dir, workspace.Dir)

	if f.Bool("push") {
		fmt.Fprintf(ctx.Stdout, "  4. push:      git push -u origin %s\n", gitx.CurrentBranch(w.Root))
	} else {
		fmt.Fprintln(ctx.Stdout, "  4. push:      (skipped: pass --push to push, else run git push yourself)")
	}
	return nil
}

// blockedAmong returns the labels of tasks from the given set that are now in
// blocked status — the signal that integrate hit a merge conflict on one.
func blockedAmong(w *workspace.Workspace, set []*store.Task) []string {
	blocked, err := store.ListTasks(w, "", model.StatusBlocked)
	if err != nil {
		return nil
	}
	inSet := map[string]bool{}
	for _, t := range set {
		inSet[t.ID] = true
	}
	var out []string
	for _, t := range blocked {
		if inSet[t.ID] {
			out = append(out, fmt.Sprintf("%03d-%s", t.Seq, t.Slug))
		}
	}
	return out
}

// doneRefs renders each task as a ref integrate resolves via store.FindTask.
// It uses the task's ULID id, which is GLOBALLY unique — a bare seq is only
// unique within a project, so across a multi-project done list two projects'
// task 5 both resolve as "5" and integrate aborts with "ref 5 is ambiguous".
// The ULID resolves to exactly one task regardless of how many projects the
// workspace holds. (A task predating ULID ids falls back to the qualified
// %03d-slug form — still not a bare seq.)
func doneRefs(tasks []*store.Task) []string {
	refs := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if t.ID != "" {
			refs = append(refs, t.ID)
			continue
		}
		refs = append(refs, fmt.Sprintf("%03d-%s", t.Seq, t.Slug))
	}
	return refs
}

// integratedCount reads the branch count integrate reports on its
// "integrated N branch(es) ..." line, so the record commit message states what
// ACTUALLY merged. On any parse miss it returns 0 rather than guessing a count.
func integratedCount(out string) int {
	n := 0
	for _, line := range strings.Split(out, "\n") {
		var c int
		if _, err := fmt.Sscanf(strings.TrimSpace(line), "integrated %d branch(es)", &c); err == nil {
			n = c
		}
	}
	return n
}

func doneLabels(tasks []*store.Task) string {
	labels := make([]string, 0, len(tasks))
	for _, t := range tasks {
		labels = append(labels, fmt.Sprintf("%03d-%s", t.Seq, t.Slug))
	}
	return strings.Join(labels, ", ")
}

// authorName encodes the role into the git identity so plain `git blame` and
// `git log` stay legible — a local copy of the vcs slice's rule (slices cannot
// import each other, and the record commit is attributed the same way).
func authorName(id, role string) string {
	if role != "" && role != "root" {
		return fmt.Sprintf("%s (%s)", id, role)
	}
	return id
}
