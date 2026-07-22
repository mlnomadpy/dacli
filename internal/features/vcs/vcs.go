// Package vcs is the version-control slice: agents commit AS THEMSELVES with
// their role, so git blame and git log answer "who implemented this, and in
// what role" — the audit trail that lets a reviewer target findings at the
// responsible agent and lets the team improve over time, the way a human
// team uses blame.
//
// dacli owns the commit so attribution is guaranteed, not left to a prompt an
// agent might get wrong. It still runs one named command (git), consistent
// with the amended non-goal (DESIGN § 2): dacli runs agents, not work — and a
// commit is the agent recording its own work, attributed.
package vcs

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "commit", Brief: "Commit as yourself: author = agent (role), with dacli trailers", Run: cmdCommit},
	{Path: "blame", Brief: "Who wrote each line — agent and role — for a file", Run: cmdBlame},
	{Path: "contrib", Brief: "Per-role / per-agent contribution rollup from commit events", Run: cmdContrib},
}

const emailDomain = "@agent.dacli"

// authorName encodes the role into the git identity so plain `git blame` and
// `git log --format=%an` are already legible without any dacli tooling.
func authorName(id, role string) string {
	if role != "" && role != "root" {
		return fmt.Sprintf("%s (%s)", id, role)
	}
	return id
}

// git runs in the ACTUAL working directory (ctx.Cwd), not w.Root: an agent
// committing from an isolated worktree must commit in that worktree, on its
// own branch — while the .dacli workspace stays the shared one found by
// walking up. Using w.Root would send every worktree's commit to main and
// trip the branch guard (found by the parallel-lifecycle test).
func gitIn(dir string, args ...string) (string, error) {
	// A deadline so a git child blocked on a credential prompt cannot hang the
	// caller — critical under `dacli mcp serve`, where it would freeze the
	// stdio loop. These are all local plumbing (add/commit/blame/rev-parse).
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), fmt.Errorf("git %s timed out", strings.Join(args, " "))
	}
	return strings.TrimSpace(string(out)), err
}

func cmdCommit(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli commit \"<message>\" [--task ref] [--no-add]")
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("committing writes to the repo; that needs an rw grant (yours is %s)", id.Grant)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not on PATH")
	}
	// Commit in the agent's ACTUAL directory (its worktree, if isolated),
	// not the shared workspace root.
	gitDir := ctx.Cwd

	// Never commit to the default branch — the same rule the git_workflow
	// prompt gives agents, enforced here so it cannot be skipped.
	// branch --show-current names an unborn branch too, where rev-parse
	// --abbrev-ref returns a useless "HEAD".
	branch, _ := gitIn(gitDir, "branch", "--show-current")
	if branch == "main" || branch == "master" {
		return clikit.Refusedf("refusing to commit on %s — branch first (git checkout -b dacli/<task>-<slug>)", branch)
	}

	if !f.Bool("no-add") {
		if _, err := gitIn(gitDir, "add", "-A"); err != nil {
			return fmt.Errorf("git add: %w", err)
		}
	}
	if staged, _ := gitIn(gitDir, "diff", "--cached", "--name-only"); staged == "" {
		return clikit.Usagef("nothing staged to commit")
	}

	name := authorName(id.ID, id.Role)
	email := id.ID + emailDomain
	msg := f.Pos[0]

	// Trailers: machine-parseable provenance alongside the human author.
	trailers := fmt.Sprintf("\n\nDacli-Agent: %s", id.ID)
	if id.Role != "" {
		trailers += "\nDacli-Role: " + id.Role
	}
	taskRef := f.Get("task")
	if taskRef != "" {
		if t, err := store.FindTask(w, taskRef); err == nil {
			trailers += fmt.Sprintf("\nDacli-Task: %03d-%s", t.Seq, t.Slug)
			taskRef = t.ID
		}
	}

	out, err := gitIn(gitDir,
		"-c", "user.name="+name, "-c", "user.email="+email,
		"commit", "--author", fmt.Sprintf("%s <%s>", name, email),
		"-m", msg+trailers)
	if err != nil {
		return fmt.Errorf("git commit failed: %s", out)
	}
	sha, _ := gitIn(gitDir, "rev-parse", "--short", "HEAD")

	// The commit becomes a first-class workspace event — attributed, so the
	// team's whole read surface (standup, replay, contrib) sees it.
	body := fmt.Sprintf("%s %s\nrole: %s", sha, msg, clikit.OrDash(id.Role))
	if _, evErr := eventlog.Append(w, id.ID, model.EventCommit, taskRef, "", body); evErr != nil {
		return evErr
	}
	fmt.Fprintf(ctx.Stdout, "committed %s as %s\n", sha, name)
	return nil
}

// cmdBlame answers "who wrote each line, in what role" — the reviewer's tool.
// Author names already carry the role, so a summary over `git blame` is
// enough; no trailer parsing needed for the common case.
func cmdBlame(ctx *clikit.Ctx, args []string) error {
	if _, _, err := clikit.OpenWorkspace(ctx); err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli blame <file>")
	}
	out, err := gitIn(ctx.Cwd, "blame", "--line-porcelain", f.Pos[0])
	if err != nil {
		return fmt.Errorf("git blame: %s", out)
	}
	lines := map[string]int{}
	agents := map[string]bool{}
	total := 0
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "author ") {
			who := strings.TrimPrefix(l, "author ")
			lines[who]++
			total++
			if strings.Contains(who, "(") { // "id (role)" = a dacli agent
				agents[who] = true
			}
		}
	}
	type row struct {
		who   string
		count int
	}
	var rows []row
	for who, n := range lines {
		rows = append(rows, row{who, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].count > rows[j].count })
	for _, r := range rows {
		mark := " "
		if strings.Contains(r.who, "(") {
			mark = "*"
		}
		fmt.Fprintf(ctx.Stdout, "%s %5d lines (%4.1f%%)  %s\n", mark, r.count, 100*float64(r.count)/float64(total), r.who)
	}
	fmt.Fprintf(ctx.Stdout, "%d lines · %d dacli agent(s) touched this file (* = agent-authored)\n", total, len(agents))
	return nil
}

// cmdContrib is the self-evolving-team surface: which roles and agents did
// how much, read from commit events (no git needed). Pair it with the
// findings each agent's work drew to see which role needs improving.
func cmdContrib(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	commits, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventCommit}})
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		fmt.Fprintln(ctx.Stdout, "no attributed commits yet — agents commit with `dacli commit`")
		return nil
	}
	// Role of every agent — from the agent files, so an agent with findings
	// against it but no commits still resolves to a role.
	roleOf := map[string]string{}
	for _, a := range mustAgents(w) {
		roleOf[a.ID] = a.Role
	}

	commitsBy := map[string]int{}
	for _, e := range commits {
		commitsBy[e.Actor]++
		for _, l := range strings.Split(e.Body, "\n") {
			if strings.HasPrefix(l, "role: ") && roleOf[e.Actor] == "" {
				roleOf[e.Actor] = strings.TrimPrefix(l, "role: ")
			}
		}
	}

	// The improvement join, now real: a reviewer files a finding --against
	// <agent-id> (from dacli blame). Count those per agent whose work drew
	// them — the signal for which role produces which class of defect.
	againstBy := map[string]int{}
	findings, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventFinding}})
	for _, e := range findings {
		if e.Against != "" {
			againstBy[e.Against]++
		}
	}
	if ps, _ := store.ListProjects(w); ps != nil {
		for _, p := range ps {
			notes, _ := store.ListNotes(w, p.Slug, model.NoteFinding)
			for _, n := range notes {
				if ag, _ := n.Front.Get("against"); ag != "" {
					againstBy[ag]++
				}
			}
		}
	}

	// Roll up to roles.
	roleCommits := map[string]int{}
	roleAgainst := map[string]int{}
	everyone := map[string]bool{}
	for a := range commitsBy {
		everyone[a] = true
	}
	for a := range againstBy {
		everyone[a] = true
	}
	for a := range everyone {
		r := clikit.OrDash(roleOf[a])
		roleCommits[r] += commitsBy[a]
		roleAgainst[r] += againstBy[a]
	}

	fmt.Fprintln(ctx.Stdout, "by role  (commits · findings-against · defect rate):")
	for _, r := range sortedKeys(roleCommits) {
		fmt.Fprintf(ctx.Stdout, "  %-14s %d commit(s) · %d finding(s)-against%s\n",
			r, roleCommits[r], roleAgainst[r], rate(roleAgainst[r], roleCommits[r]))
	}
	fmt.Fprintln(ctx.Stdout, "by agent:")
	agents := make([]string, 0, len(everyone))
	for a := range everyone {
		agents = append(agents, a)
	}
	sort.Slice(agents, func(i, j int) bool { return commitsBy[agents[i]] > commitsBy[agents[j]] })
	for _, a := range agents {
		fmt.Fprintf(ctx.Stdout, "  %-16s %-12s %d commit(s) · %d finding(s)-against\n",
			a, "("+clikit.OrDash(roleOf[a])+")", commitsBy[a], againstBy[a])
	}
	fmt.Fprintln(ctx.Stdout, "(a high defect rate for a role is where to focus improvement — better prompts, tighter scope, or a heavier model)")
	return nil
}

// rate renders a defect rate only when there is enough to mean anything.
func rate(against, commits int) string {
	if commits == 0 || against == 0 {
		return ""
	}
	return fmt.Sprintf(" · %.1f per commit", float64(against)/float64(commits))
}

func mustAgents(w *workspace.Workspace) []store.AgentInfo {
	a, _ := store.ListAgents(w)
	return a
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m[out[i]] != m[out[j]] {
			return m[out[i]] > m[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}
