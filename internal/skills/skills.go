// Package skills implements SKILLS.md: one canonical skill format (the
// richest native target's shape, verbatim), compiled outward to whatever
// delivery mechanism a runtime actually has. Import is lossless — byte
// copies, never rewrites — because the anti-fifteenth-standard mitigation is
// that a dacli skill IS a valid native skill.
package skills

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Delivery is the fidelity ladder.
type Delivery string

const (
	Native  Delivery = "native"  // lazy-loaded skill dir: near-free until triggered
	Context Delivery = "context" // startup file section: the FULL BODY, EVERY TURN
	Inline  Delivery = "inline"  // prepended to the brief: the floor, always works
	Omitted Delivery = "omitted" // min_delivery unmet: announced, never silent
)

func rank(d Delivery) int {
	switch d {
	case Native:
		return 3
	case Context:
		return 2
	case Inline:
		return 1
	}
	return 0
}

// Skill is one canonical skill directory.
type Skill struct {
	Name        string
	Desc        string
	MinDelivery Delivery
	Dir         string
	Body        string
	Resources   []string // non-main files, verbatim payload
	Scripts     []string // executable-ish resources: cannot ride a context file
	EstTokens   int      // chars/4 of what a context/inline target would carry
}

// mainFile finds the skill's manifest: skill.md or the native SKILL.md —
// import never renames, so the reader accepts both. Matching walks the
// directory listing case-insensitively rather than os.Stat'ing candidate
// names: on macOS's case-insensitive filesystem, Stat("skill.md") matches
// SKILL.md and returns the WRONG canonical name, which then fails every
// equality check downstream. Found by the lossless-import test.
func mainFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), "skill.md") {
			return e.Name()
		}
	}
	return ""
}

// LoadSkills reads every skill in the workspace library.
func LoadSkills(w *workspace.Workspace) ([]Skill, error) {
	entries, err := os.ReadDir(w.SkillsLibDir())
	if err != nil {
		return nil, nil
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if s, err := load(filepath.Join(w.SkillsLibDir(), e.Name()), e.Name()); err == nil {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// LoadSkill finds one by name.
func LoadSkill(w *workspace.Workspace, name string) (Skill, error) {
	s, err := load(filepath.Join(w.SkillsLibDir(), name), name)
	if err != nil {
		return Skill{}, store.ErrNotFound{Ref: "skill " + name}
	}
	return s, nil
}

func load(dir, fallbackName string) (Skill, error) {
	main := mainFile(dir)
	if main == "" {
		return Skill{}, fmt.Errorf("no skill.md in %s", dir)
	}
	d, err := mdstore.ReadFile(filepath.Join(dir, main))
	if err != nil {
		return Skill{}, err
	}
	s := Skill{Dir: dir, MinDelivery: Inline}
	s.Name, _ = d.Front.Get("name")
	if s.Name == "" {
		s.Name = fallbackName
	}
	s.Desc, _ = d.Front.GetText("description") // real skills write `description: |`
	if md, ok := d.Front.Get("min_delivery"); ok {
		s.MinDelivery = Delivery(md)
	}
	var body strings.Builder
	for _, sec := range d.Sections {
		if sec.Level == 1 {
			continue
		}
		if sec.Title != "" {
			body.WriteString("## " + sec.Title + "\n")
		}
		body.WriteString(sec.Content)
	}
	s.Body = strings.TrimSpace(body.String())
	s.EstTokens = (len(s.Desc) + len(s.Body)) / 4

	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if f.IsDir() || f.Name() == main {
			continue
		}
		s.Resources = append(s.Resources, f.Name())
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if info, err := f.Info(); err == nil && (info.Mode()&0o111 != 0 || ext == ".sh" || ext == ".py") {
			s.Scripts = append(s.Scripts, f.Name())
		}
	}
	return s, nil
}

// Fetch pulls a skill from skills.sh, which hosts skills as GitHub
// `owner/repo` repositories in the same native format Import already ingests.
// We git-clone the repo to a temp dir and Import it — no bespoke API client,
// no fabricated endpoints; the registry's own convention (owner/repo) is the
// whole contract. A repo may hold one skill at its root or several in
// subdirectories; Import handles both.
func Fetch(w *workspace.Workspace, ownerRepo string) (imported []string, err error) {
	if !strings.Contains(ownerRepo, "/") || strings.Count(ownerRepo, "/") != 1 {
		return nil, fmt.Errorf("skills.sh skills are owner/repo (e.g. mattpocock/skills); got %q", ownerRepo)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git not on PATH — needed to fetch from skills.sh")
	}
	tmp, err := os.MkdirTemp("", "dacli-skillfetch-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	url := "https://github.com/" + ownerRepo + ".git"
	cmd := exec.Command("git", "clone", "--depth", "1", "-q", url, tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone %s failed: %s", url, strings.TrimSpace(string(out)))
	}

	// A skill at the repo root (SKILL.md present) imports as one; otherwise
	// the repo is a collection and Import walks its subdirectories.
	if mainFile(tmp) != "" {
		name := ownerRepoLeaf(ownerRepo)
		dst := filepath.Join(w.SkillsLibDir(), name)
		if _, err := os.Stat(dst); err == nil {
			return nil, fmt.Errorf("skill %q already in the library", name)
		}
		if err := copyTree(tmp, dst); err != nil {
			return nil, err
		}
		return []string{name}, nil
	}
	return Import(w, tmp)
}

func ownerRepoLeaf(ownerRepo string) string {
	parts := strings.Split(ownerRepo, "/")
	return parts[len(parts)-1]
}

// Import copies every skill directory under src into the workspace library,
// VERBATIM — no renames, no rewrites, no dropped files. Lossless is the
// whole contract: the copy must remain a valid native skill.
func Import(w *workspace.Workspace, src string) (imported []string, err error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || mainFile(filepath.Join(src, e.Name())) == "" {
			continue
		}
		dst := filepath.Join(w.SkillsLibDir(), e.Name())
		if _, err := os.Stat(dst); err == nil {
			return imported, fmt.Errorf("skill %q already exists in the library", e.Name())
		}
		if err := copyTree(filepath.Join(src, e.Name()), dst); err != nil {
			return imported, err
		}
		imported = append(imported, e.Name())
	}
	return imported, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		info, _ := d.Info()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

// PlanItem is one skill's compilation decision for a target runtime.
type PlanItem struct {
	Skill  Skill
	Mode   Delivery
	Reason string // for Omitted and script deferrals
}

// Plan decides delivery per skill for a runtime: the best mode the runtime
// supports, floored by each skill's min_delivery — below the floor is
// OMITTED AND ANNOUNCED, because a silently absent skill is a role lying
// about its own competence.
func Plan(list []Skill, rt store.Runtime) []PlanItem {
	best := Inline
	if rt.SkillsContextFile != "" {
		best = Context
	}
	if rt.SkillsNativeDir != "" {
		best = Native
	}
	items := make([]PlanItem, 0, len(list))
	for _, s := range list {
		it := PlanItem{Skill: s, Mode: best}
		if rank(best) < rank(s.MinDelivery) {
			it.Mode = Omitted
			it.Reason = fmt.Sprintf("min_delivery %s, but %s only offers %s", s.MinDelivery, rt.Name, best)
		} else if it.Mode != Native && len(s.Scripts) > 0 {
			it.Reason = fmt.Sprintf("%d script(s) cannot ride a %s target; script→shortcut compilation is deferred (SKILLS.md § 3)", len(s.Scripts), it.Mode)
		}
		items = append(items, it)
	}
	return items
}

// Compile materializes a plan into .dacli/build/skills/<runtime>/<role>/ —
// a REGENERABLE PROJECTION: the whole role dir is deleted and rebuilt, same
// doctrine as the GitHub mirror. Returns the output dir and the total
// per-turn token tax of the always-loaded modes.
func Compile(w *workspace.Workspace, role string, rt store.Runtime, items []PlanItem) (outDir string, turnTax int, err error) {
	outDir = w.BuildSkillsDir(rt.Name, role)
	if err := os.RemoveAll(outDir); err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", 0, err
	}

	var contextBuf, inlineBuf strings.Builder
	for _, it := range items {
		switch it.Mode {
		case Native:
			if err := copyTree(it.Skill.Dir, filepath.Join(outDir, filepath.Base(it.Skill.Dir))); err != nil {
				return "", 0, err
			}
		case Context, Inline:
			buf := &contextBuf
			if it.Mode == Inline {
				buf = &inlineBuf
			}
			fmt.Fprintf(buf, "<!-- dacli:skill:%s begin -->\n## Skill: %s\n%s\n\n%s\n<!-- dacli:skill:%s end -->\n\n",
				it.Skill.Name, it.Skill.Name, it.Skill.Desc, it.Skill.Body, it.Skill.Name)
			turnTax += it.Skill.EstTokens
		}
	}
	if contextBuf.Len() > 0 {
		name := rt.SkillsContextFile
		if name == "" {
			name = "AGENTS.md"
		}
		if err := os.WriteFile(filepath.Join(outDir, filepath.Base(name)), []byte(contextBuf.String()), 0o644); err != nil {
			return "", 0, err
		}
	}
	if inlineBuf.Len() > 0 {
		if err := os.WriteFile(filepath.Join(outDir, "inline.md"), []byte(inlineBuf.String()), 0o644); err != nil {
			return "", 0, err
		}
	}
	return outDir, turnTax, nil
}
