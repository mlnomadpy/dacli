// Package knowledge is the durable-output slice: notes, retros, and the
// prompt registry's audit surface.
package knowledge

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "note add", Brief: "Record a decision, finding, metric, or reference", Run: cmdNoteAdd},
	{Path: "retro", Brief: "Harvest a task/project: went well, didn't, improve", Run: cmdRetro},
	{Path: "prompt list", Brief: "The prompt registry; overrides marked", Run: cmdPromptList},
	{Path: "prompt show", Brief: "One prompt's resolved template", Run: cmdPromptShow},
}

func cmdNoteAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) < 2 {
		return clikit.Usagef("usage: dacli note add <decision|finding|metric|ref> <title> --project <slug> [--about ref] [--body text] [--rejected text --because text] [--severity major|moderate|minor] [--scope project|workspace]")
	}
	kind := model.NoteKind(f.Pos[0])
	title := strings.Join(f.Pos[1:], " ")
	project := f.Get("project")
	if project == "" {
		return clikit.Usagef("--project is required")
	}

	// A read-only agent's finding is an event; the owner promotes it on sync.
	if id.Grant != model.GrantRW && kind == model.NoteFinding {
		// Resolve the ref NOW, at the write site: an event about "001" never
		// matches a brief filtering on the ULID id. The live mock-child demo
		// caught this — unit tests had pre-resolved ids and sailed past it.
		about := f.Get("about")
		if about != "" {
			if t, err := store.FindTask(w, about); err == nil {
				about = t.ID
			}
		}
		if _, err := eventlog.Append(w, id.ID, model.EventFinding, about, f.Get("origin"), title+"\n\n"+f.Get("body")); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "finding recorded as event — visible to every reader immediately\n")
		return nil
	}

	path, err := store.CreateNote(w, id.ID, project, kind, title, store.NoteOpts{
		About:    f.Get("about"),
		Rejected: f.Get("rejected"),
		Because:  f.Get("because"),
		Severity: f.Get("severity"),
		Scope:    f.Get("scope"),
		Body:     f.Get("body"),
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "note written: %s\n", path)
	return nil
}

// cmdRetro harvests a completed task or project into a durable note — in the
// went-well / didn't / improve order because the order is the technique.
func cmdRetro(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || (len(f.All("well"))+len(f.All("bad"))+len(f.All("improve"))) == 0 {
		return clikit.Usagef("usage: dacli retro <task-or-project-ref> --well x [--well ...] --bad y --improve z")
	}
	ref := f.Pos[0]
	project := ref
	about := ref
	if t, err := store.FindTask(w, ref); err == nil {
		project = t.Project
		about = t.ID
	} else if _, err := store.LoadProject(w, ref); err != nil {
		return store.ErrNotFound{Ref: ref}
	}

	bullets := func(items []string) string {
		var b strings.Builder
		for _, it := range items {
			b.WriteString("- " + it + "\n")
		}
		return b.String()
	}
	body := "## Went well\n" + bullets(f.All("well")) +
		"\n## Didn't go well\n" + bullets(f.All("bad")) +
		"\n## Improve next time\n" + bullets(f.All("improve"))

	path, err := store.CreateNote(w, id.ID, project, model.NoteRef, "Retro: "+ref, store.NoteOpts{
		About: about,
		Body:  body,
		Scope: f.Get("scope"), // --scope workspace makes it a cross-project lesson
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "retro recorded: %s\n", path)
	return nil
}

func cmdPromptList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	for _, name := range prompts.Names() {
		_, overridden, _ := prompts.Resolve(w.PromptsDir(), name)
		mark := "embedded"
		if overridden {
			mark = "OVERRIDDEN in .dacli/prompts/"
		}
		fmt.Fprintf(ctx.Stdout, "%-24s %s\n", name, mark)
	}
	return nil
}

func cmdPromptShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli prompt show <name>")
	}
	content, overridden, err := prompts.Resolve(w.PromptsDir(), f.Pos[0])
	if err != nil {
		return store.ErrNotFound{Ref: "prompt " + f.Pos[0]}
	}
	if overridden {
		fmt.Fprintf(ctx.Stderr, "(workspace override)\n")
	}
	fmt.Fprint(ctx.Stdout, content)
	return nil
}
