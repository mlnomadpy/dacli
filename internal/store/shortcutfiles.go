package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/shortcut"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// CreateShortcut writes .dacli/shortcuts/<name>.md. Params use the inline
// form "name" or "name=default"; body is where the WHY goes — the flag that
// took an hour to find is the part worth keeping.
func CreateShortcut(w *workspace.Workspace, actor, name, summary, command, effect string, params, roles []string, body string) error {
	if effect == "" {
		// A shortcut with no declared effect does not run; refusing at
		// creation is kinder than refusing at invocation.
		return fmt.Errorf("--effect is required (read|write|destructive): defaulting would let a typo downgrade a deploy")
	}
	path := w.ShortcutPath(name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("shortcut %q already exists", name)
	}

	d := &mdstore.Doc{}
	d.Front.Set("id", "sc-"+name)
	d.Front.Set("kind", string(model.KindShortcut))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	d.Front.Set("name", name)
	d.Front.Set("summary", summary)
	d.Front.Set("command", command)
	d.Front.Set("effect", effect)
	if len(params) > 0 {
		d.Front.Set("params", "["+strings.Join(params, ", ")+"]")
	}
	if len(roles) > 0 {
		d.Front.Set("roles", "["+strings.Join(roles, ", ")+"]")
	}
	d.Sections = []mdstore.Section{{Level: 1, Title: name, Content: body + "\n"}}
	return mdstore.WriteFile(path, d)
}

// LoadShortcuts parses every shortcut file into the pure engine's type.
// `uses` stays zero here — it is derived from run events by callers that
// have event access, because L2 must not read upward into L3.
func LoadShortcuts(w *workspace.Workspace) ([]shortcut.Shortcut, error) {
	entries, err := os.ReadDir(w.ShortcutsDir())
	if err != nil {
		return nil, nil
	}
	var out []shortcut.Shortcut
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d, err := mdstore.ReadFile(filepath.Join(w.ShortcutsDir(), e.Name()))
		if err != nil {
			continue
		}
		sc := shortcut.Shortcut{}
		sc.Name, _ = d.Front.Get("name")
		sc.Summary, _ = d.Front.Get("summary")
		sc.Command, _ = d.Front.Get("command")
		if eff, ok := d.Front.Get("effect"); ok {
			sc.Effect = shortcut.Effect(eff)
		}
		sc.Dir, _ = d.Front.Get("dir")
		sc.Roles = d.Front.GetList("roles")
		for _, p := range d.Front.GetList("params") {
			param := shortcut.Param{Name: p}
			if i := strings.Index(p, "="); i >= 0 {
				param.Name, param.Default = p[:i], p[i+1:]
			}
			sc.Params = append(sc.Params, param)
		}
		if sc.Name != "" {
			out = append(out, sc)
		}
	}
	return out, nil
}

// LoadShortcut finds one by name.
func LoadShortcut(w *workspace.Workspace, name string) (shortcut.Shortcut, error) {
	all, err := LoadShortcuts(w)
	if err != nil {
		return shortcut.Shortcut{}, err
	}
	for _, sc := range all {
		if sc.Name == name {
			return sc, nil
		}
	}
	return shortcut.Shortcut{}, ErrNotFound{Ref: "shortcut/" + name}
}
