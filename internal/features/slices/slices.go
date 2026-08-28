// Package slices exposes generation-scoped product delivery as typed child
// tasks. It deliberately owns no second status machine (issue #872).
package slices

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "slice add", Brief: "Create an independently landable typed child task", Mutates: true, Usage: "dacli slice add --task <ref> --title <title> --accept <criterion>... [--optional] [--terminal]", Run: cmdSliceAdd},
	{Path: "slice reconcile", Brief: "Freshly bind exact GitHub PR and merge identities to a delivery generation", JSON: true, Mutates: true, Usage: "dacli slice reconcile --task <parent-ref>", Run: cmdSliceReconcile},
	{Path: "task progress", Brief: "Show required delivery-slice progress for one parent task", JSON: true, Usage: "dacli task progress <ref>", Run: cmdTaskProgress},
}

func cmdSliceAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task", "title", "accept", "optional", "terminal"); err != nil {
		return err
	}
	if f.Get("task") == "" || strings.TrimSpace(f.Get("title")) == "" || len(f.All("accept")) == 0 {
		return clikit.Usagef("usage: dacli slice add --task <ref> --title <title> --accept <criterion>... [--optional] [--terminal]")
	}
	if err := clikit.RequireRW(id, "creating a delivery slice"); err != nil {
		return err
	}
	t, created, err := store.CreateDeliverySlice(w, id.ID, f.Get("task"), strings.TrimSpace(f.Get("title")), f.All("accept"), !f.Bool("optional"), f.Bool("terminal"))
	if err != nil {
		return err
	}
	verb := "reused"
	if created {
		verb = "created"
	}
	fmt.Fprintf(ctx.Stdout, "slice %s: %s generation=%d parent_generation=%d branch=%s\n", verb, t.ID, t.DeliveryGeneration(), t.DeliveryParentGeneration(), store.TaskBranch(t))
	return nil
}

func cmdTaskProgress(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject(); err != nil {
		return err
	}
	if len(f.Pos) != 1 {
		return clikit.Usagef("usage: dacli task progress <ref>")
	}
	parent, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	if parent.IsDeliverySlice() {
		return clikit.Usagef("task progress expects a parent task, got delivery slice %s", parent.ID)
	}
	p, err := store.DeliveryProgressFor(w, parent)
	if err != nil {
		return err
	}
	if ctx.JSON {
		return clikit.EmitJSON(ctx, p)
	}
	fmt.Fprintf(ctx.Stdout, "task %s generation %d: required %d/%d ready (close=%t)\n", p.ParentTask, p.ParentGeneration, p.RequiredDone, p.RequiredTotal, p.ReadyToClose)
	for _, s := range p.Slices {
		fmt.Fprintf(ctx.Stdout, "  slice %s g%d %-8s acceptance=%d/%d landed=%t branch=%s pr=%s cleanup=%s\n", s.ID, s.Generation, s.Status, s.AcceptanceDone, s.AcceptanceTotal, s.Landed, s.Branch, clikit.OrDash(s.PRURL, "-"), s.CleanupState)
	}
	return nil
}

func cmdSliceReconcile(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("task"); err != nil {
		return err
	}
	if f.Get("task") == "" {
		return clikit.Usagef("usage: dacli slice reconcile --task <parent-ref>")
	}
	if err := clikit.RequireRW(id, "recording delivery reconciliation"); err != nil {
		return err
	}
	parent, err := store.FindTask(w, f.Get("task"))
	if err != nil {
		return err
	}
	slices, err := store.DeliverySlices(w, parent)
	if err != nil {
		return err
	}
	prs, err := store.ObserveDeliveryPRs(w.Root)
	if err != nil {
		return err
	}
	byBranch := map[string][]store.DeliveryPR{}
	for _, pr := range prs {
		byBranch[pr.HeadRefName] = append(byBranch[pr.HeadRefName], pr)
	}
	for _, slice := range slices {
		matches := byBranch[store.TaskBranch(slice)]
		if len(matches) == 0 {
			continue
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].Number < matches[j].Number })
		if err := store.RecordDeliveryObservation(w, slice, matches[len(matches)-1]); err != nil {
			return err
		}
	}
	p, err := store.DeliveryProgressFor(w, parent)
	if err != nil {
		return err
	}
	if ctx.JSON {
		return clikit.EmitJSON(ctx, p)
	}
	fmt.Fprintf(ctx.Stdout, "reconciled %d delivery slice(s); required %d/%d ready\n", len(p.Slices), p.RequiredDone, p.RequiredTotal)
	return nil
}
