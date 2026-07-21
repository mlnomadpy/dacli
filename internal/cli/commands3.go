// Third slice: real agent identity (spawn/tree), guarded shortcut execution,
// and the ask/answer help-request loop.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/shortcut"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func cmdAgentSpawn(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	grant := model.Grant(f.get("grant"))
	roleName := f.get("role")

	// A role file supplies defaults and limits. Its grant is a ceiling
	// REQUEST — attenuation against the parent still wins in Spawn.
	var roleSkills, roleShortcuts []string
	if role, ok := store.LoadRole(w, roleName); ok {
		if grant == "" && role.Grant != "" {
			grant = model.Grant(role.Grant)
		}
		roleSkills, roleShortcuts = role.Skills, role.Shortcuts
		if role.WIP > 0 {
			if active := store.ActiveInRole(w, roleName); active >= role.WIP {
				// Burning Across made preventable rather than detectable:
				// the refusal happens BEFORE the thirty-first child exists.
				return refusedf("role %s is at its WIP limit (%d/%d) — `dacli agent retire` one, or raise wip in the role file",
					roleName, active, role.WIP)
			}
		}
	}

	childID, token, err := agentid.Spawn(w, id, roleName, grant)
	if err != nil {
		if err == agentid.ErrAttenuation {
			return refusedf("%v: your grant is %s", err, id.Grant)
		}
		return err
	}
	if len(roleSkills) > 0 {
		fmt.Fprintf(ctx.Stderr, "role skills to load for the child: %s\n", strings.Join(roleSkills, ", "))
	}
	if len(roleShortcuts) > 0 {
		fmt.Fprintf(ctx.Stderr, "role toolkit: %s\n", strings.Join(roleShortcuts, ", "))
	}
	// The token goes to stdout ALONE so `TOKEN=$(dacli agent spawn ...)`
	// captures exactly it; everything human-facing goes to stderr. It is
	// shown once and never stored — a lost token means a new agent.
	fmt.Fprintln(ctx.Stdout, token)
	fmt.Fprintf(ctx.Stderr, "spawned %s (role: %s, grant: %s)\ntoken shown once above — pass it to the child as %s\n",
		childID, orDash(f.get("role")), orDash(string(grant), "ro"), agentid.EnvVar)
	return nil
}

func orDash(s string, def ...string) string {
	if s != "" {
		return s
	}
	if len(def) > 0 {
		return def[0]
	}
	return "-"
}

func cmdAgentTree(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(w.AgentsDir())
	if err != nil {
		return err
	}

	type agent struct {
		id, parent, grant, role string
	}
	byParent := map[string][]agent{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d, err := mdstore.ReadFile(w.AgentPath(strings.TrimSuffix(e.Name(), ".md")))
		if err != nil {
			continue
		}
		a := agent{}
		a.id, _ = d.Front.Get("id")
		a.grant, _ = d.Front.Get("grant")
		a.role, _ = d.Front.Get("role")
		if p, ok := d.Front.Get("parent"); ok {
			a.parent = strings.TrimSuffix(strings.TrimPrefix(p, "[["), "]]")
		}
		byParent[a.parent] = append(byParent[a.parent], a)
	}

	// Write attribution: how many events each agent has actually produced.
	writes := map[string]int{}
	events, _ := eventlog.List(w, eventlog.Query{})
	for _, e := range events {
		writes[e.Actor]++
	}

	var render func(a agent, depth int)
	render = func(a agent, depth int) {
		role := a.role
		if role != "" {
			role = " · " + role
		}
		fmt.Fprintf(ctx.Stdout, "%s%s (%s%s) — %d events\n", strings.Repeat("  ", depth), a.id, a.grant, role, writes[a.id])
		kids := byParent[a.id]
		sort.Slice(kids, func(i, j int) bool { return kids[i].id < kids[j].id })
		for _, k := range kids {
			render(k, depth+1)
		}
	}
	for _, root := range byParent[""] {
		render(root, 0)
	}
	return nil
}

func cmdShortcutAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 || f.get("command") == "" {
		return usagef("usage: dacli shortcut add <name> --command 'tmpl {{p}}' --effect read|write|destructive [--summary s] [--param name=default]... [--role r]... [--why text]")
	}
	if err := store.CreateShortcut(w, id.ID, f.pos[0], f.get("summary"), f.get("command"),
		f.get("effect"), f.all("param"), f.all("role"), f.get("why")); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "shortcut %q defined\n", f.pos[0])
	return nil
}

// cmdRun expands and executes a shortcut. The security boundary is the
// engine's quoting — every parameter is POSIX-quoted — and the effect guard:
// read for anyone, write needs rw, destructive needs rw AND --confirm, so
// "deploy" is never one token away from "test".
func cmdRun(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)

	if f.bool("list") {
		scs, _ := store.LoadShortcuts(w)
		fillUses(w, scs)
		fmt.Fprint(ctx.Stdout, shortcut.Catalog(scs, id.Role, 0))
		return nil
	}
	if len(f.pos) == 0 {
		return usagef("usage: dacli run <name> [--<param> value]... [--dry-run] [--confirm] | dacli run --list")
	}

	sc, err := store.LoadShortcut(w, f.pos[0])
	if err != nil {
		return err
	}

	// Every non-reserved flag is a parameter. Unknown ones are rejected by
	// the engine — a silently dropped typo runs against the wrong target.
	reserved := map[string]bool{"dry-run": true, "confirm": true, "list": true}
	params := map[string]string{}
	for k, v := range f.vals {
		if !reserved[k] {
			params[k] = v[len(v)-1]
		}
	}
	expanded, err := shortcut.Expand(sc, params)
	if err != nil {
		return usagef("%v", err)
	}

	if f.bool("dry-run") {
		// Inspection bypasses the effect gate on purpose: a reviewing agent
		// must be able to see what a shortcut WOULD do.
		fmt.Fprintln(ctx.Stdout, expanded)
		return nil
	}
	if err := shortcut.Guard(sc, id.Role, id.Grant == model.GrantRW, f.bool("confirm")); err != nil {
		return refusedf("%v", err)
	}

	cmd := exec.Command("sh", "-c", expanded)
	cmd.Dir = w.Root
	if sc.Dir != "" && sc.Dir != "." {
		cmd.Dir = w.Root + string(os.PathSeparator) + sc.Dir
	}
	cmd.Stdout, cmd.Stderr = ctx.Stdout, ctx.Stderr
	runErr := cmd.Run()

	// Every invocation is an attributed run event — the substrate for uses
	// counts, shortcut promotion, and the calibration loop.
	status := "exit 0"
	if runErr != nil {
		status = runErr.Error()
	}
	if _, evErr := eventlog.Append(w, id.ID, model.EventRun, sc.Name, "", expanded+"\n"+status); evErr != nil {
		return evErr
	}
	if runErr != nil {
		return fmt.Errorf("%s: %v", sc.Name, runErr)
	}
	return nil
}

// fillUses derives use counts from run events — never stored, always counted
// (the C3 lesson: an incremented-in-place counter is a contention bug).
func fillUses(w *workspace.Workspace, scs []shortcut.Shortcut) {
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
	counts := map[string]int{}
	for _, e := range events {
		counts[e.About]++
	}
	for i := range scs {
		scs[i].Uses = counts[scs[i].Name]
	}
}

func cmdAsk(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 || f.get("about") == "" {
		return usagef("usage: dacli ask \"question\" --about <task-ref> [--need path]")
	}
	t, err := store.FindTask(w, f.get("about"))
	if err != nil {
		return err
	}
	question := strings.Join(f.pos, " ")
	if need := f.get("need"); need != "" {
		question += "\n\nneed: " + need
	}

	ev, err := eventlog.Append(w, id.ID, model.EventHelp, t.ID, "", question)
	if err != nil {
		return err
	}
	// The asking task blocks — a question you can proceed without was a
	// comment, not an ask.
	if id.CanMutate(t.Owner()) && t.Status != model.StatusBlocked {
		store.AppendLog(t, fmt.Sprintf("blocked on question %s", ev.ID[:10]))
		if err := store.SaveTask(t); err != nil {
			return err
		}
		if err := store.MoveTask(w, t, model.StatusBlocked); err != nil {
			return err
		}
	}
	fmt.Fprintf(ctx.Stdout, "asked %s — task %03d-%s blocked until answered\n", ev.ID[:10], t.Seq, t.Slug)
	return nil
}

func cmdAnswer(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) < 2 {
		return usagef("usage: dacli answer <question-id-prefix> <answer...> [--as decision|finding] [--rejected text --because text]")
	}
	prefix, answer := f.pos[0], strings.Join(f.pos[1:], " ")

	pending, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp}, Pending: true})
	if err != nil {
		return err
	}
	var q *eventlog.Event
	for _, e := range pending {
		if strings.HasPrefix(e.ID, prefix) {
			q = e
			break
		}
	}
	if q == nil {
		return store.ErrNotFound{Ref: "open question " + prefix}
	}
	t, err := store.FindTask(w, q.About)
	if err != nil {
		return err
	}

	// The question is transient; the answer is permanent. It lands as a
	// durable note that enters every future brief in scope.
	kind := model.NoteKind(orDash(f.get("as"), string(model.NoteFinding)))
	title := "Answer: " + strings.SplitN(q.Body, "\n", 2)[0]
	if _, err := store.CreateNote(w, id.ID, t.Project, kind, title, store.NoteOpts{
		About:    q.About,
		Body:     fmt.Sprintf("Q (%s): %s\n\nA: %s", q.Actor, q.Body, answer),
		Rejected: f.get("rejected"),
		Because:  f.get("because"),
	}); err != nil {
		return err
	}
	if _, err := eventlog.Append(w, id.ID, model.EventAnswer, t.ID, "", answer); err != nil {
		return err
	}
	if err := eventlog.MarkApplied(q.Path); err != nil {
		return err
	}
	// Unblock, if we can; otherwise the owner's sync will see the answer.
	if id.CanMutate(t.Owner()) && t.Status == model.StatusBlocked {
		store.AppendLog(t, fmt.Sprintf("question %s answered by %s", q.ID[:10], id.ID))
		if err := store.SaveTask(t); err != nil {
			return err
		}
		if err := store.MoveTask(w, t, model.StatusActive); err != nil {
			return err
		}
	}
	fmt.Fprintf(ctx.Stdout, "answered %s — recorded as a %s note on %s\n", q.ID[:10], kind, t.Project)
	return nil
}
