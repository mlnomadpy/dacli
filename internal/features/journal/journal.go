// Package journal provides read-only event-journal classification and an
// explicitly reviewed append-only reconciliation/archival path (issue #878).
package journal

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{{Path: "events reconcile", Brief: "Plan or apply append-only journal reconciliation and recoverable archival", JSON: true, Mutates: true, Usage: "dacli events reconcile --project <slug> [--archive-class complete-journal] (--dry-run | --apply-safe <plan-id>)", Run: cmdReconcile}}

func cmdReconcile(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args, "project", "archive-class", "apply-safe")
	if err != nil {
		return err
	}
	if err := f.Reject("project", "archive-class", "apply-safe", "dry-run"); err != nil {
		return err
	}
	project, applyID := f.Get("project"), f.Get("apply-safe")
	if project == "" || !workspace.SafeSegment(project) {
		return clikit.Usagef("--project requires a valid project slug")
	}
	if f.Bool("dry-run") == (applyID != "") {
		return clikit.Usagef("choose exactly one of --dry-run or --apply-safe <plan-id>")
	}
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	for _, values := range f.All("archive-class") {
		for _, class := range strings.Split(values, ",") {
			class = strings.TrimSpace(class)
			if class != "complete-journal" && class != "complete-mailbox" {
				return clikit.Usagef("--archive-class must be complete-journal or complete-mailbox, got %q", class)
			}
		}
	}
	plan, err := eventlog.PlanJournal(w, project, f.All("archive-class"), time.Now())
	if err != nil {
		return err
	}
	if f.Bool("dry-run") {
		return renderPlan(ctx, plan)
	}
	if id.ID != agentid.RootID {
		return clikit.Refusedf("only the root owner may apply repository-wide event journal reconciliation")
	}
	if applyID != plan.ID {
		return clikit.Refusedf("event journal plan changed or is unknown (requested %s, current %s); review a new --dry-run", applyID, plan.ID)
	}
	snapshot, err := eventlog.ApplyJournalPlan(w, id.ID, project, f.All("archive-class"), applyID, time.Now())
	if err != nil {
		return clikit.Refusedf("event journal reconciliation refused: %v", err)
	}
	if ctx.JSON {
		return json.NewEncoder(ctx.Stdout).Encode(snapshot)
	}
	fmt.Fprintf(ctx.Stdout, "applied event journal plan %s; snapshot contains %d classified record(s)\n", plan.ID, len(snapshot.Items))
	return nil
}

func renderPlan(ctx *clikit.Ctx, plan eventlog.JournalPlan) error {
	if ctx.JSON {
		enc := json.NewEncoder(ctx.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(plan)
	}
	fmt.Fprintf(ctx.Stdout, "event journal plan %s · schema %s · project %s · archive policy %s\n", plan.ID, plan.Schema, plan.Project, strings.Join(plan.ArchiveClasses, ","))
	for _, item := range plan.Items {
		fmt.Fprintf(ctx.Stdout, "%s %s kind=%s target=%s action=%s bytes=%d hash=%s\n  reason: %s\n", item.Classification, item.ID, item.Kind, item.About, item.Action, item.Bytes, item.Hash, item.Reason)
		if item.ManualAction != "" {
			fmt.Fprintf(ctx.Stdout, "  manual: %s\n", item.ManualAction)
		}
	}
	fmt.Fprintf(ctx.Stdout, "impact: dismiss=%d archive=%d archive_bytes=%d total_bytes=%d\n", plan.DismissCount, plan.ArchiveCount, plan.ArchiveBytes, plan.TotalBytes)
	fmt.Fprintf(ctx.Stdout, "dry-run: nothing was written; apply this immutable plan with --apply-safe %s\n", plan.ID)
	return nil
}
