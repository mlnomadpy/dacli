// Package shortcuts is the memoized-command slice: guarded execution of
// commands somebody already got right.
package shortcuts

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/shortcut"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "shortcut add", Brief: "Define a shortcut", Run: cmdAdd},
	{Path: "run", Brief: "Expand and run a shortcut (--dry-run, --confirm, --list)", Run: cmdRun},
}

func cmdAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || f.Get("command") == "" {
		return clikit.Usagef("usage: dacli shortcut add <name> --command 'tmpl {{p}}' --effect read|write|destructive [--summary s] [--param name=default]... [--role r]... [--why text]")
	}
	if err := store.CreateShortcut(w, id.ID, f.Pos[0], f.Get("summary"), f.Get("command"),
		f.Get("effect"), f.All("param"), f.All("role"), f.Get("why")); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "shortcut %q defined\n", f.Pos[0])
	return nil
}

// cmdRun expands and executes a shortcut. The security boundary is the
// engine's quoting — every parameter is POSIX-quoted — and the effect guard:
// read for anyone, write needs rw, destructive needs rw AND --confirm, so
// "deploy" is never one token away from "test".
func cmdRun(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)

	if f.Bool("list") {
		scs, _ := store.LoadShortcuts(w)
		FillUses(w, scs)
		fmt.Fprint(ctx.Stdout, shortcut.Catalog(scs, id.Role, 0))
		return nil
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli run <name> [--<param> value]... [--dry-run] [--confirm] | dacli run --list")
	}

	sc, err := store.LoadShortcut(w, f.Pos[0])
	if err != nil {
		return err
	}

	// Every non-reserved flag is a parameter. Unknown ones are rejected by
	// the engine — a silently dropped typo runs against the wrong target.
	reserved := map[string]bool{"dry-run": true, "confirm": true, "list": true}
	params := map[string]string{}
	for k, v := range f.Raw() {
		if !reserved[k] {
			params[k] = v[len(v)-1]
		}
	}
	expanded, err := shortcut.Expand(sc, params)
	if err != nil {
		return clikit.Usagef("%v", err)
	}

	if f.Bool("dry-run") {
		// Inspection bypasses the effect gate on purpose: a reviewing agent
		// must be able to see what a shortcut WOULD do.
		fmt.Fprintln(ctx.Stdout, expanded)
		return nil
	}
	if err := shortcut.Guard(sc, id.Role, id.Grant == model.GrantRW, f.Bool("confirm")); err != nil {
		return clikit.Refusedf("%v", err)
	}

	cmd := exec.Command("sh", "-c", expanded)
	cmd.Dir = w.Root
	if sc.Dir != "" && sc.Dir != "." {
		cmd.Dir = filepath.Join(w.Root, sc.Dir)
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

// FillUses derives use counts from run events — never stored, always counted
// (the C3 lesson: an incremented-in-place counter is a contention bug).
func FillUses(w *workspace.Workspace, scs []shortcut.Shortcut) {
	events, _ := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
	counts := map[string]int{}
	for _, e := range events {
		counts[e.About]++
	}
	for i := range scs {
		scs[i].Uses = counts[scs[i].Name]
	}
}
