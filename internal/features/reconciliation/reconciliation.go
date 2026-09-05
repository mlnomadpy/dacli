// Package reconciliation renders the canonical read-only delivery projection.
package reconciliation

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "reconcile", Brief: "Observe canonical delivery state or apply one immutable delegated safe-repair plan", JSON: true, Mutates: true, Usage: "dacli reconcile --project <slug> [--dry-run | --apply-safe <plan-id>]", Run: cmdReconcile},
	{Path: "explain", Brief: "Explain one task (or a project) including rank, workers, landing, and exact next actions with sourced freshness", JSON: true, Usage: "dacli explain [<task>] [--project <slug>]", Run: cmdExplain},
	{Path: "task status", Brief: "Task-scoped task, run, claim, review, landing, and next-action facts", JSON: true, Usage: "dacli task status <task> [--project <slug>]", Run: cmdTaskStatus},
}

func cmdTaskStatus(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("project"); err != nil {
		return err
	}
	if len(f.Pos) != 1 {
		return clikit.Usagef("usage: dacli task status <task> [--project <slug>]")
	}
	forward := []string{f.Pos[0]}
	if project := f.Get("project"); project != "" {
		forward = append(forward, "--project", project)
	}
	return cmdExplain(ctx, forward)
}

func cmdExplain(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("project"); err != nil {
		return err
	}
	if len(f.Pos) > 1 {
		return clikit.Usagef("usage: dacli explain [<task>] [--project <slug>]")
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	project := f.Get("project")
	taskRef := ""
	if len(f.Pos) == 1 {
		taskRef = f.Pos[0]
		task, findErr := store.FindTask(w, taskRef)
		if findErr != nil {
			return findErr
		}
		if project != "" && project != task.Project {
			return clikit.Usagef("task %s belongs to project %s, not %s", task.ID, task.Project, project)
		}
		project = task.Project
	}
	if project == "" || !workspace.SafeSegment(project) {
		return clikit.Usagef("provide a task reference or --project with a valid project slug")
	}
	if _, err := store.LoadProject(w, project); err != nil {
		return err
	}
	p, err := store.ExplainProject(w, project, time.Now())
	if taskRef != "" {
		task, _ := store.FindTask(w, taskRef)
		p = filterTaskExplain(p, task.ID)
	}
	if ctx.JSON {
		if emitErr := clikit.EmitJSON(ctx, p); emitErr != nil {
			return emitErr
		}
		return err
	}
	cache := p.CacheState
	if p.Stale {
		cache += " STALE"
	}
	fmt.Fprintf(ctx.Stdout, "progress explain %s · schema %s · observed %s · %s\n", project, p.Schema, p.ObservedAt.Format(time.RFC3339), cache)
	if p.Warning != "" {
		fmt.Fprintf(ctx.Stdout, "warning: %s\n", p.Warning)
	}
	for _, task := range p.Tasks {
		slack := "unknown"
		if task.Slack.Value != nil {
			slack = fmt.Sprintf("%.1f", *task.Slack.Value)
		}
		fmt.Fprintf(ctx.Stdout, "%s [%s] completion=%s rank=%d slack=%s · %s\n", task.ID.Value, task.Status.Value, clikit.OrDash(task.Completion.Value), task.Rank.Value, slack, task.Title.Value)
		if task.Parent.Value != "" || task.Aggregate.Value.Kind == store.TaskKindAggregate {
			fmt.Fprintf(ctx.Stdout, "  hierarchy: parent=%s aggregate=%s %d/%d ready=%t\n", clikit.OrDash(task.Parent.Value), task.Aggregate.Value.Kind, task.Aggregate.Value.RequiredDone, task.Aggregate.Value.Required, task.Aggregate.Value.ReadyToClose)
		}
		if len(task.Claims.Value) > 0 {
			fmt.Fprintf(ctx.Stdout, "  claims: %s\n", strings.Join(task.Claims.Value, ", "))
		}
		for _, blocker := range task.Blockers.Value {
			fmt.Fprintf(ctx.Stdout, "  blocker: %s\n", blocker)
		}
		for _, candidate := range task.RoleRouting.Value.Candidates {
			verdict := "eligible"
			if !candidate.Eligible {
				verdict = "rejected: " + strings.Join(candidate.Exclusions, "; ")
			}
			fmt.Fprintf(ctx.Stdout, "  role %s (%s/%s): %s\n", candidate.Role, clikit.OrDash(candidate.Runtime), clikit.OrDash(candidate.Model), verdict)
		}
		fmt.Fprintf(ctx.Stdout, "  landing: %s (%s; source=%s observed=%s stale=%t)\n", task.Landing.Value.Classification, task.Landing.Value.Confidence, task.Landing.Source, task.Landing.ObservedAt.Format(time.RFC3339), task.Landing.Stale)
		fmt.Fprintf(ctx.Stdout, "  review: %s (run=%s findings=%d; source=%s observed=%s stale=%t)\n", task.Review.Value.State, clikit.OrDash(task.Review.Value.RunID), len(task.Review.Value.FindingIDs), task.Review.Source, task.Review.ObservedAt.Format(time.RFC3339), task.Review.Stale)
		fmt.Fprintf(ctx.Stdout, "  next: %s\n", task.NextAction.Value)
	}
	for _, worker := range p.Workers {
		activity := "unavailable"
		if worker.LastDurableActivity.Value != nil {
			activity = worker.LastDurableActivity.Value.Format(time.RFC3339)
		}
		fmt.Fprintf(ctx.Stdout, "worker %s agent=%s task=%s role=%s runtime=%s state=%s phase=%s command=%s elapsed=%s stale=%t\n", worker.RunID.Value, worker.AgentID.Value, worker.TaskID.Value, worker.Role.Value, worker.Runtime.Value, worker.State.Value, worker.Phase.Value, worker.CurrentCommandCategory.Value, time.Duration(worker.ElapsedMS.Value)*time.Millisecond, worker.State.Stale)
		fmt.Fprintf(ctx.Stdout, "  activity: %s · changed=%s · commit=%s · usage_available=%t\n", activity, strings.Join(worker.ChangedPaths.Value, ","), clikit.OrDash(worker.LastCommit.Value), worker.Usage.Value.Available)
		fmt.Fprintf(ctx.Stdout, "  PR/checks: %s · next transition: %s · operator: %s\n  next: %s\n", worker.PullRequestChecks.Value.Classification, worker.NextTransition.Value, worker.RequiredOperatorAction.Value, worker.NextAction.Value)
	}
	return err
}

func filterTaskExplain(p store.ProgressExplain, taskID string) store.ProgressExplain {
	tasks := p.Tasks[:0]
	for _, task := range p.Tasks {
		if task.ID.Value == taskID {
			tasks = append(tasks, task)
		}
	}
	workers := p.Workers[:0]
	for _, worker := range p.Workers {
		if worker.TaskID.Value == taskID {
			workers = append(workers, worker)
		}
	}
	p.Tasks, p.Workers = tasks, workers
	return p
}

func cmdReconcile(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("project", "dry-run", "apply-safe"); err != nil {
		return err
	}
	project := f.Get("project")
	if project == "" || !workspace.SafeSegment(project) {
		return clikit.Usagef("--project requires a valid project slug")
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if _, err := store.LoadProject(w, project); err != nil {
		return err
	}
	if applyID := f.Get("apply-safe"); applyID != "" {
		_, id, openErr := clikit.OpenWorkspace(ctx)
		if openErr != nil {
			return openErr
		}
		audit, applyErr := applyRepairPlan(w, id.ID, project, applyID, time.Now())
		if applyErr != nil {
			return clikit.Refusedf("safe reconciliation repair refused: %v", applyErr)
		}
		if ctx.JSON {
			return clikit.EmitJSON(ctx, audit)
		}
		fmt.Fprintf(ctx.Stdout, "applied reconciliation repair %s\n", audit.PlanID)
		for _, operation := range audit.Operations {
			fmt.Fprintf(ctx.Stdout, "  %s: %s — %s\n", operation.ID, operation.State, operation.Detail)
		}
		return nil
	}
	if f.Bool("dry-run") {
		plan, planErr := planRepairs(w, project, time.Now())
		if ctx.JSON {
			if emitErr := clikit.EmitJSON(ctx, plan); emitErr != nil {
				return emitErr
			}
		} else {
			fmt.Fprintf(ctx.Stdout, "reconciliation repair plan %s · schema %s · project %s\n", plan.ID, plan.Schema, plan.Project)
			for _, operation := range plan.Operations {
				fmt.Fprintf(ctx.Stdout, "  %s [%s] findings=%s\n    next: %s\n", operation.ID, operation.Mode, strings.Join(operation.FindingIDs, ","), operation.NextAction)
			}
			fmt.Fprintf(ctx.Stdout, "dry-run: nothing was written; apply only this exact plan with --apply-safe %s\n", plan.ID)
		}
		return planErr
	}
	p, observeErr := store.ReconcileDelivery(w, project, time.Now())
	if ctx.JSON {
		enc := json.NewEncoder(ctx.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(p); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(ctx.Stdout, "delivery reconciliation %s · schema %s · observed %s\n", project, p.Schema, p.ObservedAt.Format(time.RFC3339))
		if len(p.Findings) == 0 {
			fmt.Fprintln(ctx.Stdout, "reconciled: no inconsistencies observed")
		}
		for _, finding := range p.Findings {
			fmt.Fprintf(ctx.Stdout, "%s %-26s %s [%s/%s]\n  %s\n  next: %s\n", finding.ObjectID, finding.Classification, finding.Detail, finding.Severity, finding.Confidence, finding.Source, finding.NextAction)
		}
	}
	if observeErr != nil {
		return fmt.Errorf("delivery state is not reconciled: %w", observeErr)
	}
	return nil
}
