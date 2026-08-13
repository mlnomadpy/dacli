// Package stagegate is the controlled-steps slice: project templates and
// the stage gates between them.
package stagegate

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "template list", Brief: "Available project templates and their stated cost", Usage: "dacli template list", Run: cmdList},
	{Path: "template show", Brief: "Stages, required docs, and gates for a template", Usage: "dacli skill show <name>", Run: cmdShow},
	{Path: "template add", Brief: "Vendor a template into this workspace for editing", Mutates: true, Usage: "dacli template add <name>", Run: cmdVendor},
	{Path: "stage", Brief: "Current stage and gate status for a project", Usage: "dacli stage <project>", Run: cmdStage},
	{Path: "stage advance", Brief: "Record an idempotent gate success, retryable failure, or terminal dead-letter transition", Mutates: true, Usage: "dacli stage advance <project> [--key id] [--retry reason|--terminal reason]", Run: cmdAdvance},
}

func cmdList(ctx *clikit.Ctx, args []string) error {
	// This command takes no flags, so ANY flag is a typo. An empty allowlist
	// rejects every one — without it a mistyped flag was dropped and the
	// command ran as if nothing were wrong.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
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
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
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
	f, err := clikit.ParseFlags(args, "key", "retry", "terminal")
	if err != nil {
		return err
	}
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject("key", "retry", "terminal"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli stage advance <project> [--key id] [--retry reason|--terminal reason]")
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("advancing a stage rewrites the project file, which needs an rw grant")
	}
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	projectID, _ := p.Doc.Front.Get("id")
	stage, _ := p.Doc.Front.Get("template_stage")
	if stage == "" {
		stage = "solo"
	}
	retryReason, terminalReason := f.Get("retry"), f.Get("terminal")
	if retryReason != "" && terminalReason != "" {
		return clikit.Usagef("choose exactly one failure classification: --retry or --terminal")
	}
	outcome := "success"
	if retryReason != "" {
		outcome = "retryable"
	} else if terminalReason != "" {
		outcome = "terminal"
	}
	key := f.Get("key")
	if key == "" {
		key = fmt.Sprintf("stage:%s:%s:%s", f.Pos[0], stage, outcome)
	}
	receipts := filepath.Join(filepath.Dir(p.Path), "stage.transitions")
	if stageTransitionSeen(receipts, key) {
		fmt.Fprintf(ctx.Stdout, "transition %s already applied — no-op\n", key)
		return nil
	}
	if retryReason != "" {
		if err := recordStageTransition(w, id.ID, projectID, filepath.Dir(p.Path), receipts, key, stage, outcome, retryReason, false); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "stage retry recorded: %s\n", retryReason)
		return nil
	}
	if terminalReason != "" {
		if err := recordStageTransition(w, id.ID, projectID, filepath.Dir(p.Path), receipts, key, stage, outcome, terminalReason, true); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "stage dead-lettered: %s\n", terminalReason)
		return nil
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
	if err := recordStageTransition(w, id.ID, projectID, filepath.Dir(p.Path), receipts, key, stage, "success", "", false); err != nil {
		return err
	}
	if newStage == "complete" {
		fmt.Fprintln(ctx.Stdout, "template complete — every gate passed")
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "advanced to stage %s (cone narrows: estimates now report tighter)\n", newStage)
	return nil
}

func stageTransitionPath(dir, key string) string {
	return filepath.Join(dir, fmt.Sprintf("%x.md", sha256.Sum256([]byte(key))))
}

func stageTransitionSeen(dir, key string) bool {
	_, err := os.Stat(stageTransitionPath(dir, key))
	return err == nil
}

func recordStageTransition(w *workspace.Workspace, actor, projectID, projectDir, receipts, key, stage, outcome, reason string, dead bool) error {
	body := fmt.Sprintf("stage transition key=%q outcome=%s stage=%q", key, outcome, stage)
	if reason != "" {
		body += " reason=" + fmt.Sprintf("%q", reason)
	}
	if _, err := eventlog.Append(w, actor, model.EventRun, projectID, "agent", body); err != nil {
		return err
	}
	dir := receipts
	if dead {
		dir = filepath.Join(projectDir, "stage.dead-letter")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(stageTransitionPath(dir, key), []byte(body+"\n"), 0o644); err != nil {
		return err
	}
	if dead {
		if err := os.MkdirAll(receipts, 0o755); err != nil {
			return err
		}
		return os.WriteFile(stageTransitionPath(receipts, key), []byte(body+"\n"), 0o644)
	}
	return nil
}
