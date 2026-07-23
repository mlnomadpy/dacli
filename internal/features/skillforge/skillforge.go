// Package skillforge is the skill-compilation slice: author once, compile
// to whatever each runtime can load (SKILLS.md).
package skillforge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/skills"
	"github.com/mlnomadpy/dacli/internal/store"
)

var Commands = []clikit.Command{
	{Path: "skill add", Brief: "Author a workspace skill", Run: cmdAdd},
	{Path: "skill list", Brief: "Workspace skills with sizes and delivery floors", Run: cmdList},
	{Path: "skill show", Brief: "One skill: version, changelog, body, resources, est. tokens", Run: cmdShow},
	{Path: "skill bump", Brief: "Increment a skill's version (v1→v2) after a change", Run: cmdBump},
	{Path: "skill import", Brief: "Ingest a native skill tree losslessly", Run: cmdImport},
	{Path: "skill fetch", Brief: "Fetch a skill from skills.sh (owner/repo) into the library", Run: cmdFetch},
	{Path: "skill compile", Brief: "Materialize skills for a role on a runtime (--dry-run)", Run: cmdCompile},
	{Path: "skill promote", Brief: "Owner-gated promotion of a lesson into a skill", Run: cmdPromote},
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
	// Versioned from birth: `skill show` reports it and every edit bumps past it.
	d.Front.Set("version", store.DefaultVersion)
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

// cmdPromote is the SKILLS.md § 6 gate: a lesson (a scope:workspace note)
// never auto-promotes to a skill — an explicit act by the workspace owner
// does, one at a time. Any other identity (every spawned agent, however wide
// its grant) is refused: the escalation path this blocks is a hostile file
// poisoning a finding that distills into a lesson that then auto-compiles
// into standing instructions for every future agent, on every runtime.
func cmdPromote(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.ID != agentid.RootID {
		return clikit.Refusedf("skill promote is an explicit act by the workspace owner (%s); a spawned agent proposing its own lesson into standing instructions is exactly the escalation this gate blocks", agentid.RootID)
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli skill promote <lesson-ref> [--name <skill-name>]")
	}
	ref := f.Pos[0]
	lesson, err := store.FindLesson(w, ref)
	if err != nil {
		return err
	}

	name := f.Get("name")
	if name == "" {
		name = store.Slugify(lesson.Title)
	}
	dir := filepath.Join(w.SkillsLibDir(), name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("skill %q already exists", name)
	}

	d := &mdstore.Doc{}
	d.Front.Set("name", name)
	// Versioned from birth, like any skill (cmdAdd).
	d.Front.Set("version", store.DefaultVersion)
	d.Front.Set("description", lesson.Title)
	d.Front.Set("created_by", id.ID)
	d.Front.Set("promoted_from", lesson.ID)
	if lesson.Origin != "" {
		// Compiled output inherits the provenance of its sources (SKILLS.md §
		// 6), so `dacli taint` walking a suspect origin can still reach the
		// standing instructions it ended up compiled into.
		d.Front.Set("origin", lesson.Origin)
	}
	d.Sections = []mdstore.Section{{Level: 1, Title: name, Content: ""}}
	if lesson.Body != "" {
		d.Sections = append(d.Sections, mdstore.Section{Level: 0, Content: lesson.Body + "\n"})
	}
	if err := mdstore.WriteFile(filepath.Join(dir, "skill.md"), d); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "promoted lesson %s (%s) → skill %s at %s\n", lesson.ID, lesson.Project, name, dir)
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
	manifest := skillManifest(s.Dir)
	version := store.FileVersion(manifest)
	fmt.Fprintf(ctx.Stdout, "%s — %s\nversion: %s · floor: %s · ~%d tokens · resources: %s\n",
		s.Name, s.Desc, version, s.MinDelivery, s.EstTokens, strings.Join(s.Resources, ", "))
	if stale, since := store.VersionIsStale(manifest, version); stale {
		if since > 0 {
			fmt.Fprintf(ctx.Stdout, "⚠ changed in %d commit(s) since %s was set — bump with `dacli skill bump %s`\n", since, version, s.Name)
		} else {
			fmt.Fprintf(ctx.Stdout, "⚠ uncommitted edits — bump with `dacli skill bump %s` before committing\n", s.Name)
		}
	}
	changes, seen := store.FileChangelog(manifest, 10)
	fmt.Fprintf(ctx.Stdout, "\nchangelog:\n%s\n", store.FormatChangelog(changes, seen))
	fmt.Fprintf(ctx.Stdout, "\n%s\n", s.Body)
	return nil
}

// skillManifest resolves a skill directory's manifest file (skill.md or the
// native SKILL.md), matching case-insensitively so an imported SKILL.md is
// found while the real on-disk name — the one git tracks — is returned.
func skillManifest(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return filepath.Join(dir, "skill.md")
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), "skill.md") {
			return filepath.Join(dir, e.Name())
		}
	}
	return filepath.Join(dir, "skill.md")
}

func cmdBump(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if id.Grant != model.GrantRW {
		return clikit.Refusedf("bumping a skill version rewrites its file, which needs an rw grant")
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli skill bump <name>")
	}
	s, err := skills.LoadSkill(w, f.Pos[0])
	if err != nil {
		return err
	}
	old, next, err := store.BumpFileVersion(skillManifest(s.Dir))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "skill %s: %s → %s — commit it with `dacli commit`\n", s.Name, old, next)
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

func cmdFetch(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli skill fetch <owner/repo>   (from skills.sh, e.g. mattpocock/skills)")
	}
	imported, err := skills.Fetch(w, f.Pos[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "fetched %d skill(s) from %s: %s\n", len(imported), f.Pos[0], strings.Join(imported, ", "))
	fmt.Fprintln(ctx.Stdout, "add to a role with `dacli role add <name> --skill "+imported[0]+"`, then `dacli skill compile`")
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
