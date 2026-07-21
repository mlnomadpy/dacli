// Package stagegate is the controlled-steps slice: project templates and
// the stage gates between them.
package stagegate

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/model"
)

var Commands = []clikit.Command{
	{Path: "template list", Brief: "Available project templates and their stated cost", Run: cmdList},
	{Path: "template show", Brief: "Stages, required docs, and gates for a template", Run: cmdShow},
	{Path: "template add", Brief: "Vendor a template into this workspace for editing", Run: cmdVendor},
	{Path: "stage", Brief: "Current stage and gate status for a project", Run: cmdStage},
	{Path: "stage advance", Brief: "Advance if the gate opens; refuses with the unmet list", Run: cmdAdvance},
}

func cmdList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	ts, err := gates.Load(w)
	if err != nil {
		return err
	}
	for _, t := range ts {
		fmt.Fprintf(ctx.Stdout, "%-16s %-10s stages:%d  %s\n  cost: %s\n", t.Name, t.Origin, len(t.Stages), t.Summary, t.Cost)
	}
	return nil
}

func cmdShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli template show <name>")
	}
	t, err := gates.Get(w, f.Pos[0])
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, t.Manifest)
	return nil
}

func cmdVendor(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli template add <name>")
	}
	path, err := gates.Vendor(w, f.Pos[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "vendored to %s — edits there win over the embedded default\n", path)
	return nil
}

func cmdStage(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli stage <project>")
	}
	st, err := gates.Status(w, f.Pos[0])
	if err != nil {
		return err
	}
	if st.Complete {
		if st.Template == "" || st.Template == "solo" {
			fmt.Fprintln(ctx.Stdout, "no template (solo): no gates — attach one with `project add --template` if someone else depends on this work")
		} else {
			fmt.Fprintf(ctx.Stdout, "template %s: complete\n", st.Template)
		}
		return nil
	}
	line := fmt.Sprintf("template %s · stage %s", st.Template, st.Stage)
	if ph, err := gates.PhaseFor(w, f.Pos[0]); err == nil && ph.Gated {
		line += fmt.Sprintf(" · phase %s", ph.Name)
		if len(ph.Allows) > 0 {
			line += " · roles: " + strings.Join(ph.Allows, ", ")
		}
	}
	fmt.Fprintln(ctx.Stdout, line)
	for _, c := range st.Checks {
		mark := "✓"
		if !c.OK {
			mark = "✗"
		}
		line := fmt.Sprintf("%s %s", mark, c.Desc)
		if !c.OK && c.Why != "" {
			line += " — " + c.Why
		}
		fmt.Fprintln(ctx.Stdout, line)
	}
	return nil
}

func cmdAdvance(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli stage advance <project>")
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("advancing a stage rewrites the project file, which needs an rw grant")
	}
	newStage, unmet, err := gates.Advance(w, f.Pos[0])
	if err != nil {
		return err
	}
	if len(unmet) > 0 {
		msg := "gate closed — unmet:"
		for _, c := range unmet {
			msg += "\n  ✗ " + c.Desc
			if c.Why != "" {
				msg += " — " + c.Why
			}
		}
		// A gate that can be argued past gets worked around; the refusal
		// names exactly what is missing.
		msg += "\nfill what is missing — a closed gate is an answer, do not retry"
		return clikit.Refusedf("%s", msg)
	}
	if newStage == "complete" {
		fmt.Fprintln(ctx.Stdout, "template complete — every gate passed")
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "advanced to stage %s (cone narrows: estimates now report tighter)\n", newStage)
	return nil
}
