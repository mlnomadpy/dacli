// Package collab is the cooperative-loop slice: the event log's human/agent
// exchange — sync, tailing, questions, answers, and escalation out of the
// tree.
package collab

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "sync", Brief: "Apply pending child events to objects you own", Run: cmdSync},
	{Path: "events tail", Brief: "Follow the append-only write log", Run: cmdEventsTail},
	{Path: "ask", Brief: "Ask a blocking question; the asking task blocks until answered", Run: cmdAsk},
	{Path: "answer", Brief: "Answer a question; the answer becomes a durable note", Run: cmdAnswer},
	{Path: "threads", Brief: "Questions and their answers, open first", Run: cmdThreads},
	{Path: "escalate", Brief: "Escalate out of the tree to a human (--github files an issue)", Run: cmdEscalate},
}

func cmdSync(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	res, err := eventlog.Sync(w, id.ID, id.CanMutate)
	if err != nil {
		return err
	}
	for _, n := range res.Notes {
		fmt.Fprintf(ctx.Stdout, "applied %s\n", n)
	}
	fmt.Fprintf(ctx.Stdout, "sync: %d applied, %d left pending\n", res.Applied, res.Skipped)
	return nil
}

func cmdEventsTail(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	limit := 20
	fmt.Sscanf(f.Get("limit"), "%d", &limit)
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

func cmdAsk(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || f.Get("about") == "" {
		return clikit.Usagef("usage: dacli ask \"question\" --about <task-ref> [--need path]")
	}
	t, err := store.FindTask(w, f.Get("about"))
	if err != nil {
		return err
	}
	question := strings.Join(f.Pos, " ")
	if need := f.Get("need"); need != "" {
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

func cmdAnswer(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) < 2 {
		return clikit.Usagef("usage: dacli answer <question-id-prefix> <answer...> [--as decision|finding] [--rejected text --because text]")
	}
	prefix, answer := f.Pos[0], strings.Join(f.Pos[1:], " ")

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
	kind := model.NoteKind(clikit.OrDash(f.Get("as"), string(model.NoteFinding)))
	title := "Answer: " + strings.SplitN(q.Body, "\n", 2)[0]
	if _, err := store.CreateNote(w, id.ID, t.Project, kind, title, store.NoteOpts{
		About:    q.About,
		Body:     fmt.Sprintf("Q (%s): %s\n\nA: %s", q.Actor, q.Body, answer),
		Rejected: f.Get("rejected"),
		Because:  f.Get("because"),
	}); err != nil {
		return err
	}
	// The answer event points at the QUESTION it resolves, not the task. A task
	// can carry several questions answered by different agents; keying the
	// answer to the task would collapse them to one attribution. `about` is the
	// question id so `dacli threads` names the right answerer per question.
	if _, err := eventlog.Append(w, id.ID, model.EventAnswer, q.ID, "", answer); err != nil {
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

func cmdThreads(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	// One kind-filtered walk of the event tree covers both sides of the thread;
	// List already parsed each event's `applied` flag, so no question file is
	// re-read to learn whether it was answered.
	events, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventHelp, model.EventAnswer}})
	if err != nil {
		return err
	}
	var questions []*eventlog.Event
	// Keyed by QUESTION id (EventAnswer.About), not task id, so two questions
	// on one task answered by different agents each attribute correctly.
	answered := map[string]string{}
	for _, e := range events {
		switch e.Kind {
		case model.EventHelp:
			questions = append(questions, e)
		case model.EventAnswer:
			if _, seen := answered[e.About]; !seen {
				answered[e.About] = e.Actor
			}
		}
	}
	for _, q := range questions {
		status := "OPEN"
		if q.Applied {
			status = "answered by " + clikit.OrDash(answered[q.ID])
		}
		firstLine := strings.SplitN(q.Body, "\n", 2)[0]
		fmt.Fprintf(ctx.Stdout, "%s [%s] %s asks about %s: %s\n", q.ID[:10], status, q.Actor, q.About, firstLine)
	}
	if len(questions) == 0 {
		fmt.Fprintln(ctx.Stdout, "no questions asked yet")
	}
	return nil
}

// cmdEscalate is the terminal hop: nothing in the tree owns this, so it
// leaves the tree.
func cmdEscalate(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli escalate \"summary\" [--about task] [--github]")
	}
	summary := strings.Join(f.Pos, " ")
	about := f.Get("about")
	if about != "" {
		if t, err := store.FindTask(w, about); err == nil {
			about = t.ID
		}
	}
	ev, err := eventlog.Append(w, id.ID, model.EventHelp, about, "", "[escalation to human]\n"+summary)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "escalated %s — no role in the tree owns this; a human does now\n", ev.ID[:10])

	if f.Bool("github") {
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("--github needs the gh CLI on PATH")
		}
		body := fmt.Sprintf("Escalated from dacli workspace %q by %s.\n\n%s\n\nAnswer with: `dacli answer %s \"...\"`", w.Name, id.ID, summary, ev.ID[:10])
		// gh is network- and auth-bound; a deadline keeps a hung request (no
		// network, an interactive auth prompt) from blocking the caller — and,
		// under `dacli mcp serve`, the entire stdio loop. The escalation event
		// above already stands regardless of whether this mirror succeeds.
		gctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		out, gherr := exec.CommandContext(gctx, "gh", "issue", "create", "--title", "[dacli] "+summary, "--body", body).Output()
		if gctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gh issue create timed out (the escalation event %s still stands)", ev.ID[:10])
		}
		if gherr != nil {
			return fmt.Errorf("gh issue create failed: %v (the escalation event %s still stands)", gherr, ev.ID[:10])
		}
		fmt.Fprintf(ctx.Stdout, "issue: %s", string(out))
	}
	return nil
}
