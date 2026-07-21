// v0.1 command implementations: the dogfood wedge from ARCHITECTURE § 3.
// Everything here is thin — resolve workspace and identity, call into
// internal/, format the result. Logic lives below this layer.
package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// --- Exit-code contract (ARCHITECTURE § 4). The 1/3 distinction is the one
// that matters: retrying a refusal is the loop a supervisor must never enter.

type exitErr struct {
	code int
	msg  string
}

func (e exitErr) Error() string { return e.msg }

func usagef(format string, a ...any) error   { return exitErr{2, fmt.Sprintf(format, a...)} }
func refusedf(format string, a ...any) error { return exitErr{3, fmt.Sprintf(format, a...)} }

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee exitErr
	if errors.As(err, &ee) {
		return ee.code
	}
	var nf store.ErrNotFound
	if errors.As(err, &nf) {
		return 4
	}
	if errors.Is(err, workspace.ErrNotFound) {
		return 4
	}
	return 1
}

// --- Tiny flag parser: --key value, --key=value, repeatable keys,
// positionals. Deliberately minimal; the flag surfaces are part of the format
// contract (REVIEW G13) and should stay small enough to list.

type flags struct {
	pos  []string
	vals map[string][]string
}

func parseFlags(args []string) (*flags, error) {
	f := &flags{vals: map[string][]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			f.pos = append(f.pos, a)
			continue
		}
		key := a[2:]
		if eq := strings.Index(key, "="); eq >= 0 {
			f.vals[key[:eq]] = append(f.vals[key[:eq]], key[eq+1:])
			continue
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			f.vals[key] = append(f.vals[key], "true") // bare flag
			continue
		}
		i++
		f.vals[key] = append(f.vals[key], args[i])
	}
	return f, nil
}

func (f *flags) get(k string) string {
	if v := f.vals[k]; len(v) > 0 {
		return v[len(v)-1]
	}
	return ""
}
func (f *flags) all(k string) []string { return f.vals[k] }
func (f *flags) bool(k string) bool    { return f.get(k) == "true" }

// --- Shared plumbing ---

func openWorkspace(ctx *Ctx) (*workspace.Workspace, *agentid.Identity, error) {
	w, err := workspace.Find(ctx.Cwd)
	if err != nil {
		return nil, nil, err
	}
	id, err := agentid.Resolve(w)
	if err != nil {
		return nil, nil, err
	}
	return w, id, nil
}

// --- Commands ---

func cmdInit(ctx *Ctx, args []string) error {
	f, _ := parseFlags(args)
	name := f.get("name")
	if name == "" {
		name = filepath.Base(ctx.Cwd)
	}
	w, err := workspace.Init(ctx.Cwd, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "initialized workspace %q (%s) at %s\n", w.Name, w.ID, filepath.Join(w.Root, workspace.Dir))
	return nil
}

func cmdWhoami(ctx *Ctx, args []string) error {
	_, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.Role != "" {
		fmt.Fprintf(ctx.Stdout, "%s (grant: %s, role: %s)\n", id.ID, id.Grant, id.Role)
	} else {
		fmt.Fprintf(ctx.Stdout, "%s (grant: %s)\n", id.ID, id.Grant)
	}
	return nil
}

func cmdProjectAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli project add <title> [--slug s] [--goal g] [--stage definition|elicitation|approach|design]")
	}
	title := strings.Join(f.pos, " ")
	p, err := store.CreateProject(w, id.ID, title, f.get("slug"), f.get("goal"), f.get("stage"))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "project %s created (stage: %s)\n", p.Slug, p.Stage)
	return nil
}

func cmdProjectList(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	ps, err := store.ListProjects(w)
	if err != nil {
		return err
	}
	for _, p := range ps {
		fmt.Fprintf(ctx.Stdout, "%-16s %-12s %s\n", p.Slug, p.Stage, p.Title)
	}
	return nil
}

func cmdTaskAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 || f.get("project") == "" {
		return usagef("usage: dacli task add <title> --project <slug> [--priority must|should|could|wont] [--estimate o,m,p] [--accept criterion]... [--so-that why]")
	}
	title := strings.Join(f.pos, " ")

	// Ambiguity lint on the title, before the task exists. Titles get the
	// strict pass because a vague title becomes three different deliverables.
	if finds := spm.Scan(title, spm.Options{}); len(finds) > 0 {
		fmt.Fprintf(ctx.Stderr, "warning: ambiguous title —\n")
		for _, fd := range finds {
			fmt.Fprintf(ctx.Stderr, "  %s\n", fd)
		}
	}

	t, err := store.CreateTask(w, id.ID, f.get("project"), title, store.TaskOpts{
		Priority:  f.get("priority"),
		Estimate:  f.get("estimate"),
		Accept:    f.all("accept"),
		SoThat:    f.get("so-that"),
		Context:   f.get("context"),
		DependsOn: f.all("depends-on"),
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "task %03d-%s created (%s)\n", t.Seq, t.Slug, t.ID)
	if len(t.Acceptance()) == 0 {
		fmt.Fprintf(ctx.Stderr, "warning: no acceptance criteria — an agent given this task cannot know when to stop (add --accept)\n")
	}
	return nil
}

func cmdTaskList(ctx *Ctx, args []string) error {
	if ctx.JSON {
		return cmdTaskListJSON(ctx, args)
	}
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	ts, err := store.ListTasks(w, f.get("project"), model.Status(f.get("status")))
	if err != nil {
		return err
	}
	for _, t := range ts {
		boxes := t.Acceptance()
		done := 0
		for _, b := range boxes {
			if b.Done {
				done++
			}
		}
		fmt.Fprintf(ctx.Stdout, "%-10s %03d-%-28s %-8s %-7s [%d/%d] %s\n",
			t.Project, t.Seq, t.Slug, t.Status, t.Priority(), done, len(boxes), t.Title)
	}
	return nil
}

func cmdTaskShow(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli task show <ref>")
	}
	t, err := store.FindTask(w, f.pos[0])
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, mdstore.Render(t.Doc))
	return nil
}

func cmdTaskClaim(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli task claim <ref>")
	}
	t, err := store.FindTask(w, f.pos[0])
	if err != nil {
		return err
	}

	if owner := t.Owner(); owner != "" && owner != id.ID && t.Status == model.StatusActive {
		return refusedf("task is owned by %s and active; ask them or wait for release", owner)
	}
	if !id.CanMutate(t.Owner()) {
		// A read-only agent claims via an event, not a rewrite.
		if _, err := eventlog.Append(w, id.ID, model.EventClaim, t.ID, "", ""); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "claim recorded as event (read-only grant); the owner applies it on sync\n")
		return nil
	}

	t.Doc.Front.Set("owner", id.ID)
	store.AppendLog(t, "claimed by "+id.ID)
	if err := store.SaveTask(t); err != nil {
		return err
	}
	if t.Status == model.StatusOpen {
		if err := store.MoveTask(w, t, model.StatusActive); err != nil {
			return err
		}
	}
	fmt.Fprintf(ctx.Stdout, "claimed %03d-%s\n", t.Seq, t.Slug)
	return nil
}

func cmdTaskCheck(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli task check <ref> [--n N | --all]")
	}
	t, err := store.FindTask(w, f.pos[0])
	if err != nil {
		return err
	}
	if !id.CanMutate(t.Owner()) {
		return refusedf("only the owner (%s) checks acceptance boxes; report a finding instead", t.Owner())
	}
	sec, ok := t.Doc.Section("Acceptance")
	if !ok {
		return fmt.Errorf("task has no acceptance section")
	}
	boxes := mdstore.Checkboxes(sec.Content)
	if f.bool("all") {
		for i := range boxes {
			boxes[i].Done = true
		}
	} else {
		n, err := strconv.Atoi(f.get("n"))
		if err != nil || n < 1 || n > len(boxes) {
			return usagef("--n must be 1..%d", len(boxes))
		}
		boxes[n-1].Done = true
	}
	t.Doc.SetSection("Acceptance", mdstore.RenderCheckboxes(boxes))
	return store.SaveTask(t)
}

func cmdTaskDone(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli task done <ref>")
	}
	t, err := store.FindTask(w, f.pos[0])
	if err != nil {
		return err
	}
	if !id.CanMutate(t.Owner()) {
		if _, err := eventlog.Append(w, id.ID, model.EventProposeStatus, t.ID, "", "propose: done"); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "done proposed as event (read-only grant)\n")
		return nil
	}

	// done VERIFIES, not just records: unchecked acceptance is a refusal
	// naming the criterion — "no" is an answer, not a failure.
	var unmet []string
	for _, box := range t.Acceptance() {
		if !box.Done {
			unmet = append(unmet, box.Text)
		}
	}
	if len(unmet) > 0 {
		return refusedf("acceptance unmet:\n  - %s\nfix, `task check`, or `ask` if the criterion is wrong — do not retry", strings.Join(unmet, "\n  - "))
	}

	store.AppendLog(t, "completed by "+id.ID) // the actuals stamp (capture field)
	if err := store.SaveTask(t); err != nil {
		return err
	}
	if err := store.MoveTask(w, t, model.StatusDone); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "done: %03d-%s\n", t.Seq, t.Slug)
	return nil
}

func cmdNoteAdd(ctx *Ctx, args []string) error {
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) < 2 {
		return usagef("usage: dacli note add <decision|finding|metric|ref> <title> --project <slug> [--about ref] [--body text] [--rejected text --because text] [--severity major|moderate|minor] [--scope project|workspace]")
	}
	kind := model.NoteKind(f.pos[0])
	title := strings.Join(f.pos[1:], " ")
	project := f.get("project")
	if project == "" {
		return usagef("--project is required")
	}

	// A read-only agent's finding is an event; the owner promotes it on sync.
	if id.Grant != model.GrantRW && kind == model.NoteFinding {
		origin := f.get("origin")
		if _, err := eventlog.Append(w, id.ID, model.EventFinding, f.get("about"), origin, title+"\n\n"+f.get("body")); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "finding recorded as event — visible to every reader immediately\n")
		return nil
	}

	path, err := store.CreateNote(w, id.ID, project, kind, title, store.NoteOpts{
		About:    f.get("about"),
		Rejected: f.get("rejected"),
		Because:  f.get("because"),
		Severity: f.get("severity"),
		Scope:    f.get("scope"),
		Body:     f.get("body"),
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "note written: %s\n", path)
	return nil
}

func cmdContext(ctx *Ctx, args []string) error {
	if ctx.JSON {
		return cmdContextJSON(ctx, args)
	}
	w, id, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	if len(f.pos) == 0 {
		return usagef("usage: dacli context <task-ref> [--budget N] [--record]")
	}
	budget, _ := strconv.Atoi(f.get("budget"))
	b, err := brief.Assemble(w, f.pos[0], brief.Options{Budget: budget})
	if err != nil {
		return err
	}
	out := b.Render()
	fmt.Fprint(ctx.Stdout, out)

	// --record freezes the brief for replay (PROPOSALS P3): what was this
	// agent actually told. History not captured is history lost.
	if f.bool("record") {
		run := w.RunDir(ulid.New())
		if err := os.MkdirAll(run, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(run, "brief.md"), []byte(out), 0o644); err != nil {
			return err
		}
		meta := fmt.Sprintf("task: %s\nactor: %s\nbudget: %d\n", b.TaskID, id.ID, budget)
		if err := os.WriteFile(filepath.Join(run, "invocation.txt"), []byte(meta), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stderr, "recorded: %s\n", run)
	}
	return nil
}

func cmdStatus(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	ps, err := store.ListProjects(w)
	if err != nil {
		return err
	}
	for _, p := range ps {
		counts := map[model.Status]int{}
		ts, _ := store.ListTasks(w, p.Slug, "")
		for _, t := range ts {
			counts[t.Status]++
		}
		fmt.Fprintf(ctx.Stdout, "%-16s open:%d active:%d blocked:%d done:%d  %s\n",
			p.Slug, counts[model.StatusOpen], counts[model.StatusActive],
			counts[model.StatusBlocked], counts[model.StatusDone], p.Title)
	}
	pending, _ := eventlog.List(w, eventlog.Query{Pending: true})
	if len(pending) > 0 {
		fmt.Fprintf(ctx.Stdout, "pending events: %d (run `dacli sync` as the owner to materialize)\n", len(pending))
	}
	return nil
}

func cmdEventsTail(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	limit, _ := strconv.Atoi(f.get("limit"))
	if limit == 0 {
		limit = 20
	}
	events, err := eventlog.List(w, eventlog.Query{Limit: limit})
	if err != nil {
		return err
	}
	for _, e := range events {
		body := e.Body
		if len(body) > 60 {
			body = body[:57] + "..."
		}
		fmt.Fprintf(ctx.Stdout, "%s %-16s %-10s %-30s %s\n", e.ID[:10], e.Kind, e.Actor, e.About, strings.ReplaceAll(body, "\n", " "))
	}
	return nil
}

// cmdLint applies the asymmetric scope policy from SPM.md: titles and
// acceptance at moderate-and-above, bodies at major only — the places where
// ambiguity becomes a wrong deliverable get the strict pass.
func cmdLint(ctx *Ctx, args []string) error {
	w, _, err := openWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := parseFlags(args)
	var tasks []*store.Task
	if len(f.pos) > 0 {
		t, err := store.FindTask(w, f.pos[0])
		if err != nil {
			return err
		}
		tasks = []*store.Task{t}
	} else {
		tasks, err = store.ListTasks(w, f.get("project"), "")
		if err != nil {
			return err
		}
	}

	total := 0
	for _, t := range tasks {
		report := func(where string, finds []spm.Finding) {
			for _, fd := range finds {
				total++
				fmt.Fprintf(ctx.Stdout, "%03d-%s %s: %s\n", t.Seq, t.Slug, where, fd)
			}
		}
		report("title", spm.Scan(t.Title, spm.Options{}))
		for _, box := range t.Acceptance() {
			report("acceptance", spm.Scan(box.Text, spm.Options{}))
		}
		for _, s := range t.Doc.Sections {
			if s.Level > 1 && !strings.EqualFold(s.Title, "Acceptance") && !strings.EqualFold(s.Title, "Log") {
				report("body", spm.Scan(s.Content, spm.Options{MinSeverity: spm.SevMajor}))
			}
		}
		if t.Status != model.StatusDone && len(t.Acceptance()) == 0 {
			total++
			fmt.Fprintf(ctx.Stdout, "%03d-%s INVEST: no acceptance criteria — the agent cannot know when to stop\n", t.Seq, t.Slug)
		}
	}
	if total == 0 {
		fmt.Fprintln(ctx.Stdout, "clean")
	}
	return nil
}
