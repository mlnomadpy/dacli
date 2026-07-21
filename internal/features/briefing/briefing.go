// Package briefing is the context slice — the product. Everything else in
// the tool exists so this command has something to slice.
package briefing

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/ulid"
)

var Commands = []clikit.Command{
	{Path: "context", Brief: "Assemble a scoped context brief for an agent (the main event)", Run: cmdContext},
}

func cmdContext(ctx *clikit.Ctx, args []string) error {
	if ctx.JSON {
		return cmdContextJSON(ctx, args)
	}
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli context <task-ref> [--budget N] [--record]")
	}
	budget, _ := strconv.Atoi(f.Get("budget"))
	b, err := brief.Assemble(w, f.Pos[0], brief.Options{Budget: budget})
	if err != nil {
		return err
	}
	out := b.Render()
	fmt.Fprint(ctx.Stdout, out)

	// --record freezes the brief for replay (PROPOSALS P3): what was this
	// agent actually told. History not captured is history lost.
	if f.Bool("record") {
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

func cmdContextJSON(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli context <task-ref> [--budget N]")
	}
	budget := 0
	fmt.Sscanf(f.Get("budget"), "%d", &budget)
	b, err := brief.Assemble(w, f.Pos[0], brief.Options{Budget: budget})
	if err != nil {
		return err
	}
	type section struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	out := struct {
		TaskID   string    `json:"task_id"`
		Sections []section `json:"sections"`
		Omitted  []string  `json:"omitted"`
	}{TaskID: b.TaskID, Omitted: b.Omitted}
	if out.Omitted == nil {
		out.Omitted = []string{}
	}
	for _, s := range b.Sections {
		out.Sections = append(out.Sections, section{s.Title, s.Content})
	}
	return clikit.EmitJSON(ctx, out)
}
