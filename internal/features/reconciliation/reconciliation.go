// Package reconciliation renders the canonical read-only delivery projection.
package reconciliation

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{{Path: "reconcile", Brief: "Project canonical read-only delivery state across local, git, loop, event, run, and GitHub evidence", JSON: true, Usage: "dacli reconcile --project <slug> [--dry-run]", Run: cmdReconcile}}

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
