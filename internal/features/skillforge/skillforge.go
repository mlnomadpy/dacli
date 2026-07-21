// Package skillforge is the skill-compilation slice: author once, compile
// to whatever each runtime can load (SKILLS.md).
package skillforge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/skills"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "skill add", Brief: "Author a workspace skill", Run: cmdAdd},
	{Path: "skill list", Brief: "Workspace skills with sizes and delivery floors", Run: cmdList},
	{Path: "skill show", Brief: "One skill: body, resources, est. tokens", Run: cmdShow},
	{Path: "skill import", Brief: "Ingest a native skill tree losslessly", Run: cmdImport},
	{Path: "skill compile", Brief: "Materialize skills for a role on a runtime (--dry-run)", Run: cmdCompile},
	{Path: "skill promote", Brief: "Owner-gated promotion of a lesson into a skill", Run: clikit.Planned("lessons landing as promotable objects — the P1 store exists; the gate does not", "docs/SKILLS.md § 6")},
}

func cmdAdd(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 || f.Get("desc") == "" {
		return clikit.Usagef("usage: dacli skill add <name> --desc \"trigger description\" [--body text] [--min-delivery native|context|inline]")
	}
	name := f.Pos[0]
	dir := filepath.Join(w.SkillsLibDir(), name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("skill %q already exists", name)
	}

	d := &mdstore.Doc{}
	d.Front.Set("name", name)
	d.Front.Set("description", f.Get("desc"))
	if md := f.Get("min-delivery"); md != "" {
		d.Front.Set("min_delivery", md)
	}
	d.Front.Set("created_by", id.ID)
	d.Sections = []mdstore.Section{{Level: 1, Title: name, Content: ""}}
	if body := f.Get("body"); body != "" {
		d.Sections = append(d.Sections, mdstore.Section{Level: 0, Content: body + "\n"})
	}
	if err := mdstore.WriteFile(filepath.Join(dir, "skill.md"), d); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "skill %s created at %s\n", name, dir)
	return nil
}

func cmdList(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	list, _ := skills.LoadSkills(w)
	for _, s := range list {
		fmt.Fprintf(ctx.Stdout, "%-24s ~%4d tok  floor:%-8s res:%d scripts:%d  %s\n",
			s.Name, s.EstTokens, s.MinDelivery, len(s.Resources), len(s.Scripts), s.Desc)
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
		return clikit.Usagef("usage: dacli skill show <name>")
	}
	s, err := skills.LoadSkill(w, f.Pos[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "%s — %s\nfloor: %s · ~%d tokens · resources: %s\n\n%s\n",
		s.Name, s.Desc, s.MinDelivery, s.EstTokens, strings.Join(s.Resources, ", "), s.Body)
	return nil
}

func cmdImport(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli skill import <dir>   (e.g. ~/.claude/skills)")
	}
	imported, err := skills.Import(w, f.Pos[0])
	if err != nil {
		return err
	}
	if len(imported) == 0 {
		return fmt.Errorf("no skill directories found under %s", f.Pos[0])
	}
	fmt.Fprintf(ctx.Stdout, "imported %d skill(s) losslessly: %s\n", len(imported), strings.Join(imported, ", "))
	return nil
}

func cmdCompile(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	roleName, rtName := f.Get("role"), f.Get("runtime")
	if rtName == "" {
		return clikit.Usagef("usage: dacli skill compile --runtime <rt> [--role <r>] [--dry-run]")
	}
	rt, err := store.LoadRuntime(w, rtName)
	if err != nil {
		return err
	}

	// The role names the skills; no role means the whole library.
	var list []skills.Skill
	if roleName != "" {
		role, ok := store.LoadRole(w, roleName)
		if !ok {
			return store.ErrNotFound{Ref: "role " + roleName}
		}
		for _, name := range role.Skills {
			s, err := skills.LoadSkill(w, name)
			if err != nil {
				return fmt.Errorf("role %s names skill %q, which is not in the library", roleName, name)
			}
			list = append(list, s)
		}
	} else {
		list, _ = skills.LoadSkills(w)
		roleName = "_all"
	}
	if len(list) == 0 {
		return fmt.Errorf("nothing to compile: no skills selected")
	}

	items := skills.Plan(list, rt)
	tax := 0
	for _, it := range items {
		line := fmt.Sprintf("%-24s → %-8s", it.Skill.Name, it.Mode)
		if it.Mode == skills.Context || it.Mode == skills.Inline {
			// The acceptance criterion, and the economics made visible: an
			// always-loaded skill costs its FULL BODY on every single turn.
			line += fmt.Sprintf("  per-turn tax ~%d tokens (full body, every turn)", it.Skill.EstTokens)
			tax += it.Skill.EstTokens
		}
		if it.Reason != "" {
			line += "  [" + it.Reason + "]"
		}
		fmt.Fprintln(ctx.Stdout, line)
	}
	if tax > 0 {
		fmt.Fprintf(ctx.Stdout, "total per-turn tax on %s: ~%d tokens — progressive disclosure is gone on this target\n", rt.Name, tax)
		if tax > 4000 {
			fmt.Fprintf(ctx.Stderr, "warning: heavy always-on payload; trim the role's skill list or raise min_delivery floors\n")
		}
	}

	if f.Bool("dry-run") {
		fmt.Fprintln(ctx.Stdout, "(dry-run: nothing written)")
		return nil
	}
	outDir, _, err := skills.Compile(w, roleName, rt, items)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "compiled to %s (regenerable projection — delete freely)\n", outDir)
	return nil
}
