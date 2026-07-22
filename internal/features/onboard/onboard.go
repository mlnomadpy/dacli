// Package onboard is the adoption slice: drop dacli into an already-started
// repo. It reads the real files, organizes what it finds into a codebase map
// the briefs will carry, and (opt-in) seeds tasks from the TODO markers the
// code already contains — so an agent picking up an existing project starts
// from its actual context, not a blank workspace.
package onboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func agentIdentity(w *workspace.Workspace) (string, error) {
	id, err := agentid.Resolve(w)
	if err != nil {
		return "", err
	}
	return id.ID, nil
}

var Commands = []clikit.Command{
	{Path: "adopt", Brief: "Onboard an existing repo: init, project, codebase map, TODO tasks, --provision-roles", Run: cmdAdopt},
}

// skipDir names directories that are noise for a codebase map.
var skipDir = map[string]bool{
	".git": true, ".dacli": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, ".next": true, "target": true, "__pycache__": true,
	".venv": true, "venv": true, ".idea": true, ".vscode": true,
}

var langByExt = map[string]string{
	".go": "Go", ".ts": "TypeScript", ".tsx": "TypeScript", ".js": "JavaScript",
	".jsx": "JavaScript", ".py": "Python", ".rs": "Rust", ".java": "Java",
	".rb": "Ruby", ".c": "C", ".h": "C", ".cpp": "C++", ".cc": "C++",
	".swift": "Swift", ".kt": "Kotlin", ".php": "PHP", ".tex": "LaTeX",
	".md": "Markdown", ".sh": "Shell", ".sql": "SQL",
}

func cmdAdopt(ctx *clikit.Ctx, args []string) error {
	f, _ := clikit.ParseFlags(args)

	// Init the workspace if this repo has none — adoption is the common
	// first contact, so it should not require a separate init.
	w, err := workspace.Find(ctx.Cwd)
	if err != nil {
		name := f.Get("name")
		if name == "" {
			name = filepath.Base(ctx.Cwd)
		}
		w, err = workspace.Init(ctx.Cwd, name)
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stderr, "initialized workspace %q\n", w.Name)
	}
	id, err := agentIdentity(w)
	if err != nil {
		return err
	}

	slug := f.Get("project")
	if slug == "" {
		slug = store.Slugify(filepath.Base(w.Root))
	}
	goal := f.Get("goal")
	if goal == "" {
		goal = inferGoal(w.Root)
	}
	if _, err := store.LoadProject(w, slug); err != nil {
		if _, err := store.CreateProject(w, id, filepath.Base(w.Root), slug, goal, ""); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "project %s created\n", slug)
	} else {
		fmt.Fprintf(ctx.Stderr, "project %s already exists — refreshing its codebase map\n", slug)
	}

	scan := walk(w.Root)
	mapBody := renderMap(scan)
	// The map is a section on the PROJECT itself, so it rides into every
	// brief the way the goal and glossary do — an agent onboarding sees the
	// real structure without re-walking it. (A ref note would not: briefs
	// deliberately don't pull refs, or every retro would flood them.)
	p, err := store.LoadProject(w, slug)
	if err != nil {
		return err
	}
	p.Doc.SetSection("Codebase map", mapBody)
	if err := store.SaveProject(p); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "codebase map written (%d files, %d languages) — reaches every brief\n",
		scan.files, len(scan.langs))

	if f.Bool("todos") {
		created := 0
		for _, td := range scan.todos {
			if created >= 50 {
				fmt.Fprintf(ctx.Stderr, "capped at 50 TODO tasks (%d more found) — the rest are in the codebase map\n", len(scan.todos)-created)
				break
			}
			title := td.text
			if len(title) > 70 {
				title = title[:67] + "..."
			}
			if _, err := store.CreateTask(w, id, slug, title, store.TaskOpts{
				Context: fmt.Sprintf("%s marker at %s", td.marker, td.loc),
			}); err != nil {
				continue
			}
			created++
		}
		fmt.Fprintf(ctx.Stdout, "seeded %d task(s) from TODO/FIXME markers\n", created)
	} else if len(scan.todos) > 0 {
		fmt.Fprintf(ctx.Stdout, "%d TODO/FIXME markers found — re-run with --todos to seed them as tasks\n", len(scan.todos))
	}

	// --provision-roles: before any implementation work, seed ONE discovery task
	// that hands the codebase map to a role-architect agent, whose job is to
	// decide the MINIMAL role roster this project needs and create those roles.
	// onboard SEEDS the task and PRINTS the spawn line — it never spawns the
	// architect itself (no cross-slice import; the operator triggers it), keeping
	// the phase discipline: no implementation before the team is set.
	if f.Bool("provision-roles") {
		name := p.Title
		if name == "" {
			name = slug
		}
		title := fmt.Sprintf("Provision the team for %s", name)
		body := provisionBrief(name, mapBody)
		t, err := store.CreateTask(w, id, slug, title, store.TaskOpts{
			Priority: "should",
			Context:  body,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.Stdout, "seeded team-provisioning task %03d for a role-architect\n", t.Seq)
		fmt.Fprintf(ctx.Stdout, "next: dacli spawn --task %d --role role-architect\n", t.Seq)
	}

	fmt.Fprintf(ctx.Stdout, "\nadopted. next: `dacli context <task>` to onboard an agent, or `dacli lint` to sharpen the seeded tasks\n")
	return nil
}

// provisionBrief builds the Context body for the team-provisioning task: the
// detected languages and codebase map (reused verbatim from the scan) plus a
// directive telling a role-architect agent the exact primitives to run. It is
// discovery-then-provision — the architect decides the roster, dacli only
// carries the context and names the steps.
//
// Labels are BOLD, not ATX headings: the whole string is the CONTENT of the
// task's "## Context" section, so an inner "##" would be re-parsed as a sibling
// section and split the body (the same trap renderMap documents).
func provisionBrief(project, mapBody string) string {
	var b strings.Builder
	b.WriteString("A role-architect must provision this project's team BEFORE implementation work starts.\n\n")
	b.WriteString("**Codebase map:**\n")
	b.WriteString(mapBody)
	if !strings.HasSuffix(mapBody, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n**Directive:**\n")
	fmt.Fprintf(&b, "Analyze %s's stack and domains (see the languages and codebase map above). ", project)
	b.WriteString("Decide the MINIMAL role roster it needs (e.g. an implementer, a reviewer, a language-specific auditor, a docs writer — justify EACH against the codebase; do not over-staff).\n\n")
	b.WriteString("For each role:\n")
	b.WriteString("1. Pick relevant skills from skills.sh and run `dacli skill fetch <owner/repo>`.\n")
	b.WriteString("2. Create the role with `dacli role add <name> --kind implementer|reviewer|researcher|designer --grant ro|rw --model <tier> --skills <...>`.\n\n")
	b.WriteString("Finish with `dacli note add decision` documenting the roster and why.\n")
	return b.String()
}

type todo struct {
	marker, text, loc string
}

type scanResult struct {
	files int
	langs map[string]int
	dirs  []string
	docs  []string
	todos []todo
}

func walk(root string) scanResult {
	r := scanResult{langs: map[string]int{}}
	topDirs := map[string]bool{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		first := strings.SplitN(rel, string(os.PathSeparator), 2)[0]
		if d.IsDir() {
			if skipDir[d.Name()] || strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			if !strings.Contains(rel, string(os.PathSeparator)) {
				topDirs[rel] = true
			}
			return nil
		}
		_ = first
		r.files++
		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := langByExt[ext]; ok {
			r.langs[lang]++
		}
		if ext == ".md" {
			r.docs = append(r.docs, rel)
		}
		// TODO markers — cheap scan of text-ish files only.
		if isScannable(ext) {
			scanTodos(path, rel, &r)
		}
		return nil
	})
	for d := range topDirs {
		r.dirs = append(r.dirs, d)
	}
	sort.Strings(r.dirs)
	sort.Strings(r.docs)
	return r
}

func isScannable(ext string) bool {
	_, ok := langByExt[ext]
	return ok && ext != ".md"
}

func scanTodos(path, rel string, r *scanResult) {
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) > 512*1024 {
		return
	}
	for i, line := range strings.Split(string(raw), "\n") {
		for _, marker := range []string{"TODO", "FIXME", "HACK", "XXX"} {
			if idx := strings.Index(line, marker); idx >= 0 {
				text := strings.TrimSpace(line[idx+len(marker):])
				text = strings.TrimLeft(text, ":() -")
				if text == "" {
					text = marker + " (no description)"
				}
				r.todos = append(r.todos, todo{marker: marker, text: text, loc: fmt.Sprintf("%s:%d", rel, i+1)})
				break
			}
		}
	}
}

func renderMap(r scanResult) string {
	// Bold labels, NOT sub-headings: the map is the CONTENT of the project's
	// "## Codebase map" section, and any inner ATX heading would be parsed as
	// a sibling section, leaving the map empty in the brief (found in test).
	var b strings.Builder
	b.WriteString("**Languages:**\n")
	type lc struct {
		lang string
		n    int
	}
	var langs []lc
	for l, n := range r.langs {
		langs = append(langs, lc{l, n})
	}
	sort.Slice(langs, func(i, j int) bool { return langs[i].n > langs[j].n })
	for _, l := range langs {
		fmt.Fprintf(&b, "- %s (%d files)\n", l.lang, l.n)
	}
	b.WriteString("\n**Top-level structure:**\n")
	for _, d := range r.dirs {
		fmt.Fprintf(&b, "- %s/\n", d)
	}
	if len(r.docs) > 0 {
		b.WriteString("\n**Existing docs:**\n")
		for _, d := range r.docs {
			if strings.Count(d, string(os.PathSeparator)) <= 2 {
				fmt.Fprintf(&b, "- %s\n", d)
			}
		}
	}
	if len(r.todos) > 0 {
		fmt.Fprintf(&b, "\n**Open markers (%d):**\n", len(r.todos))
		for i, td := range r.todos {
			if i >= 20 {
				fmt.Fprintf(&b, "- ...and %d more\n", len(r.todos)-20)
				break
			}
			fmt.Fprintf(&b, "- %s %s — %s\n", td.marker, td.loc, td.text)
		}
	}
	return b.String()
}

// inferGoal reads the first heading of README as a starting goal, rather than
// a placeholder the lint will flag.
func inferGoal(root string) string {
	for _, name := range []string{"README.md", "readme.md", "README"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				return "Continue " + strings.TrimSpace(line[2:]) + " (adopted; refine this goal)."
			}
		}
	}
	return "Adopted existing repository — set a real goal before spawning agents."
}
