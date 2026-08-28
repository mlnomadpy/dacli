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
	{Path: "reconcile", Brief: "Project canonical read-only delivery state across local, git, loop, event, run, and GitHub evidence", JSON: true, Usage: "dacli reconcile --project <slug> [--dry-run]", Run: cmdReconcile},
	{Path: "explain", Brief: "Explain task rank, blockers, role eligibility, workers, landing, and next actions with sourced freshness", JSON: true, Usage: "dacli explain --project <slug>", Run: cmdExplain},
}

func cmdExplain(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("project"); err != nil {
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
	p, err := store.ExplainProject(w, project, time.Now())
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
		fmt.Fprintf(ctx.Stdout, "%s [%s] rank=%d slack=%s · %s\n", task.ID.Value, task.Status.Value, task.Rank.Value, slack, task.Title.Value)
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
		fmt.Fprintf(ctx.Stdout, "  next: %s\n", task.NextAction.Value)
	}
	for _, worker := range p.Workers {
		fmt.Fprintf(ctx.Stdout, "worker %s agent=%s task=%s role=%s runtime=%s state=%s stale=%t\n  next: %s\n", worker.RunID.Value, worker.AgentID.Value, worker.TaskID.Value, worker.Role.Value, worker.Runtime.Value, worker.State.Value, worker.State.Stale, worker.NextAction.Value)
	}
	return err
}

func cmdReconcile(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("project", "dry-run"); err != nil {
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
		if f.Bool("dry-run") {
			fmt.Fprintln(ctx.Stdout, "dry-run: read-only projection; nothing was written")
		}
	}
	if observeErr != nil {
		return fmt.Errorf("delivery state is not reconciled: %w", observeErr)
	}
	return nil
}
