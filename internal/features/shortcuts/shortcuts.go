// Package shortcuts is the memoized-command slice: guarded execution of
// commands somebody already got right.
package shortcuts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/shortcut"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "shortcut add", Brief: "Define a shortcut", Run: cmdAdd},
	{Path: "shortcut promote", Brief: "Turn a repeated ad-hoc command into a shortcut", Run: cmdPromote},
	{Path: "run", Brief: "Expand and run a shortcut, or an ad-hoc command (--cmd, --dry-run, --confirm, --list)", Run: cmdRun},
}

// adhocPrefix marks a run event's About as an untracked ad-hoc invocation
// rather than a named shortcut. About is a wikilink target in the event
// format, so it must stay a short, safe atom — never the raw command text,
// which can carry anything (quotes, newlines, "]]") — hence the content hash.
const adhocPrefix = "adhoc:"

func adhocKey(cmdStr string) string {
	sum := sha256.Sum256([]byte(cmdStr))
	return adhocPrefix + hex.EncodeToString(sum[:])[:12]
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

// cmdPromote materializes a repeated ad-hoc `dacli run --cmd` invocation into
// a named shortcut. It only reads what --cmd already wrote to the event log —
// there is no separate "candidate" store — so a run event's About (the
// content hash from adhocKey) is both the dedup key and the repetition count.
//
// Promotion requires the SAME literal command to have run at least twice: a
// one-off is not what "repeated" means, and refusing a single run keeps
// `shortcut add` (which writes a command by hand) the path for a first-time,
// deliberately-authored shortcut.
func cmdPromote(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || f.Get("from-event") == "" {
		return clikit.Usagef("usage: dacli shortcut promote <name> --from-event <run-event-id> --effect read|write|destructive [--summary s] [--param name=default]... [--role r]... [--why text]")
	}
	name := f.Pos[0]
	eventID := f.Get("from-event")

	events, err := eventlog.List(w, eventlog.Query{Kinds: []model.EventKind{model.EventRun}})
	if err != nil {
		return err
	}
	var target *eventlog.Event
	for _, e := range events {
		if e.ID == eventID {
			target = e
			break
		}
	}
	if target == nil {
		return store.ErrNotFound{Ref: "run-event/" + eventID}
	}
	if !strings.HasPrefix(target.About, adhocPrefix) {
		return clikit.Refusedf("event %s is a named-shortcut run (%s already a shortcut) — there is nothing to promote", eventID, target.About)
	}

	repeats := 0
	for _, e := range events {
		if e.About == target.About {
			repeats++
		}
	}
	if repeats < 2 {
		return clikit.Refusedf("this ad-hoc command has run once — promotion is for a REPEATED command (docs/SHORTCUTS.md § promotion); run it again first if it really recurs")
	}

	// Body is "<command>\n<status>" (see runAdhoc); the command is the first line.
	cmdStr, _, _ := strings.Cut(target.Body, "\n")

	if err := store.CreateShortcut(w, id.ID, name, f.Get("summary"), cmdStr,
		f.Get("effect"), f.All("param"), f.All("role"), f.Get("why")); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "promoted ad-hoc command (%d runs) → shortcut %q\n", repeats, name)
	return nil
}

// runAdhoc executes and tracks a literal command that has no shortcut file.
// There is no declared effect to gate on — an ad-hoc command is arbitrary
// text an agent chose to run — so the floor is the same one an undeclared
// shortcut effect would refuse at: this never runs read-only, unattended.
func runAdhoc(ctx *clikit.Ctx, w *workspace.Workspace, id *agentid.Identity, f *clikit.Flags, cmdStr string) error {
	if f.Bool("dry-run") {
		fmt.Fprintln(ctx.Stdout, cmdStr)
		return nil
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("ad-hoc commands need an rw grant: there is no declared effect to gate on, so a read-only agent cannot run one")
	}

	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Dir = w.Root
	cmd.Stdout, cmd.Stderr = ctx.Stdout, ctx.Stderr
	runErr := cmd.Run()

	status := "exit 0"
	if runErr != nil {
		status = runErr.Error()
	}
	if _, evErr := eventlog.Append(w, id.ID, model.EventRun, adhocKey(cmdStr), "", cmdStr+"\n"+status); evErr != nil {
		return evErr
	}
	if runErr != nil {
		return fmt.Errorf("%s: %v", cmdStr, runErr)
	}
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
	// --cmd is the ad-hoc path: a literal command that is not (yet) a named
	// shortcut. It is tracked the same way a named run is — an attributed
	// EventRun — which is the substrate `shortcut promote` reads to turn a
	// command that keeps recurring into a real, reviewable shortcut.
	if cmdStr := f.Get("cmd"); cmdStr != "" {
		return runAdhoc(ctx, w, id, f, cmdStr)
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli run <name> [--<param> value]... [--dry-run] [--confirm] | dacli run --cmd '<command>' [--dry-run] | dacli run --list")
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
