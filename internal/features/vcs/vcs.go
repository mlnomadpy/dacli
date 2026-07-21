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
	"fmt"
	"os/exec"
	"sort"
	"strings"

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

func git(w *workspace.Workspace, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = w.Root
	out, err := cmd.CombinedOutput()
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

	// Never commit to the default branch — the same rule the git_workflow
	// prompt gives agents, enforced here so it cannot be skipped.
	// branch --show-current names an unborn branch too, where rev-parse
	// --abbrev-ref returns a useless "HEAD".
	branch, _ := git(w, "branch", "--show-current")
	if branch == "main" || branch == "master" {
		return clikit.Refusedf("refusing to commit on %s — branch first (git checkout -b dacli/<task>-<slug>)", branch)
	}

	if !f.Bool("no-add") {
		if _, err := git(w, "add", "-A"); err != nil {
			return fmt.Errorf("git add: %w", err)
		}
	}
	if staged, _ := git(w, "diff", "--cached", "--name-only"); staged == "" {
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

	out, err := git(w,
		"-c", "user.name="+name, "-c", "user.email="+email,
		"commit", "--author", fmt.Sprintf("%s <%s>", name, email),
		"-m", msg+trailers)
	if err != nil {
		return fmt.Errorf("git commit failed: %s", out)
	}
	sha, _ := git(w, "rev-parse", "--short", "HEAD")

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
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli blame <file>")
	}
	out, err := git(w, "blame", "--line-porcelain", f.Pos[0])
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
	byRole := map[string]int{}
	byAgent := map[string]int{}
	roleOf := map[string]string{}
	for _, e := range commits {
		byAgent[e.Actor]++
		role := "-"
		for _, l := range strings.Split(e.Body, "\n") {
			if strings.HasPrefix(l, "role: ") {
				role = strings.TrimPrefix(l, "role: ")
			}
		}
		byRole[role]++
		roleOf[e.Actor] = role
	}

	// The improvement join: how many findings each agent's work drew. A
	// reviewer files findings; the agent whose commit they concern is named
	// in the finding body (git blame → dacli blame → the responsible id).
	// Absent that link we count findings authored ABOUT each agent's tasks
	// as a rough signal — refined once review names the agent explicitly.
	fmt.Fprintln(ctx.Stdout, "by role:")
	roles := sortedKeys(byRole)
	for _, r := range roles {
		fmt.Fprintf(ctx.Stdout, "  %-14s %d commit(s)\n", r, byRole[r])
	}
	fmt.Fprintln(ctx.Stdout, "by agent:")
	for _, a := range sortedKeys(byAgent) {
		fmt.Fprintf(ctx.Stdout, "  %-16s %-12s %d commit(s)\n", a, "("+clikit.OrDash(roleOf[a])+")", byAgent[a])
	}
	fmt.Fprintln(ctx.Stdout, "(reviewers: `dacli blame <file>` to name the agent behind a defect, then file a finding against that role)")
	return nil
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
