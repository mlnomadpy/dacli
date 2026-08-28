// Package cleanup exposes content-addressed repository cleanup plans. The
// classifier and mutations live in store so worktree prune and cleanup share
// entity-level policy rather than feature slices importing one another.
package cleanup

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{{
	Path: "cleanup", Brief: "Plan or apply content-addressed, recoverable repository cleanup", JSON: true, Mutates: true,
	Usage: "dacli cleanup --project <slug> (--dry-run | --apply-safe <plan-id>)", Run: cmdCleanup,
}}

func cmdCleanup(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("project", "dry-run", "apply-safe"); err != nil {
		return err
	}
	project := f.Get("project")
	applyID := f.Get("apply-safe")
	if project == "" || !workspace.SafeSegment(project) {
		return clikit.Usagef("--project requires a valid project slug")
	}
	if f.Bool("dry-run") == (applyID != "") {
		return clikit.Usagef("choose exactly one of --dry-run or --apply-safe <plan-id>")
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	plan, err := store.PlanRepositoryCleanup(w, project, time.Now(), ctx.Cwd)
	if err != nil {
		return err
	}
	if f.Bool("dry-run") {
		return renderPlan(ctx, plan)
	}
	if applyID != plan.ID {
		return clikit.Refusedf("cleanup plan changed or is unknown (requested %s, current %s); review a new --dry-run", applyID, plan.ID)
	}
	audit, err := store.ApplyRepositoryCleanup(w, project, applyID, time.Now(), ctx.Cwd)
	if err != nil {
		return clikit.Refusedf("safe cleanup refused: %v", err)
	}
	if ctx.JSON {
		return json.NewEncoder(ctx.Stdout).Encode(audit)
	}
	fmt.Fprintf(ctx.Stdout, "applied cleanup plan %s; removed %d item(s)\n", plan.ID, len(audit.Removed))
	for _, item := range audit.Removed {
		fmt.Fprintf(ctx.Stdout, "removed worktree %s and branch %s at %s\n", item.Worktree, item.Branch, item.Commit)
		for _, recovery := range item.Recovery {
			fmt.Fprintf(ctx.Stdout, "  recover: %s\n", recovery)
		}
	}
	return nil
}

func renderPlan(ctx *clikit.Ctx, plan store.CleanupPlan) error {
	if ctx.JSON {
		enc := json.NewEncoder(ctx.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(plan)
	}
	fmt.Fprintf(ctx.Stdout, "repository cleanup plan %s · schema %s · project %s · base %s@%s · current %s\n", plan.ID, plan.Schema, plan.Project, plan.Base, plan.BaseCommit, plan.CurrentBranch)
	for _, item := range plan.Items {
		verdict := "preserve"
		if item.Eligible {
			verdict = "eligible"
		}
		fmt.Fprintf(ctx.Stdout, "%s %s branch=%s task=%s status=%s owner=%s pr=%s commit=%s\n", verdict, item.Worktree, item.Branch, item.Task, item.TaskStatus, item.Owner, item.PRState, item.Commit)
		for _, pr := range item.PRHistory {
			fmt.Fprintf(ctx.Stdout, "  pr: #%d %s\n", pr.Number, pr.State)
		}
		for _, run := range item.Runs {
			fmt.Fprintf(ctx.Stdout, "  run: %s agent=%s state=%s claims=%v\n", run.ID, run.Agent, run.State, run.Claims)
		}
		for _, reason := range item.Reasons {
			fmt.Fprintf(ctx.Stdout, "  reason: %s\n", reason)
		}
		for _, operation := range item.Operations {
			fmt.Fprintf(ctx.Stdout, "  operation: %s\n", operation)
		}
		for _, recovery := range item.Recovery {
			fmt.Fprintf(ctx.Stdout, "  recovery: %s\n", recovery)
		}
	}
	for _, artifact := range plan.Artifacts {
		fmt.Fprintf(ctx.Stdout, "artifact %s run=%s task=%s class=%s pruneable=%t\n  reason: %s\n", artifact.Path, artifact.RunID, artifact.Task, artifact.Classification, artifact.Pruneable, artifact.Reason)
	}
	fmt.Fprintf(ctx.Stdout, "dry-run: nothing was written; apply this immutable plan with --apply-safe %s\n", plan.ID)
	return nil
}
