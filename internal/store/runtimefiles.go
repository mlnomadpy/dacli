package store

import (
	"fmt"
	"os"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Runtime is a parsed coding-agent CLI adapter. Everything in it is an
// assumption until `runtime doctor` probes the installed binary — including
// the presets dacli ships.
type Runtime struct {
	Name   string
	Binary string
	Mode   string // stdin | arg — how the prompt is delivered
	Flag   string // arg mode: the flag preceding the prompt (e.g. -p)
	Args   []string

	// SandboxRO are the args that put this runtime in a read-only mode. An
	// EMPTY list means the runtime cannot enforce read-only — and per
	// RUNTIMES.md § 8 that is a refusal to spawn ro children, never a silent
	// downgrade.
	SandboxRO []string

	// Env lists variable NAMES passed through from the parent environment.
	// Values never enter the workspace; the workspace is committed to git.
	Env []string

	// ModelFlag is the flag that selects a model tier on this CLI (e.g.
	// --model). Empty means the runtime has no model selection — and then
	// role-level model routing is announced as inoperative, not ignored.
	ModelFlag string

	// Skill delivery (SKILLS.md § 3): where this runtime loads native skills
	// from, and/or which startup context file it reads. Both empty = the
	// inline floor.
	SkillsNativeDir   string
	SkillsContextFile string

	// UsageFormat OPTS this runtime into token-usage capture (F1). Empty is the
	// default and preserves today's behavior byte-for-byte: a plain-text
	// transcript and wall-clock actuals. Set to "stream-json" and dacli asks the
	// child for `--output-format stream-json`, parses the stream into a readable
	// transcript, and records the final `usage` (output tokens, turns, cost) so
	// calibration can measure in tokens instead of a wall-clock proxy.
	UsageFormat string

	Path string
}

// CreateRuntime writes .dacli/runtimes/<name>.md.
func CreateRuntime(w *workspace.Workspace, actor string, rt Runtime, note string) error {
	if rt.Name == "" || rt.Binary == "" {
		return fmt.Errorf("a runtime needs at least --name and --binary")
	}
	if rt.Mode == "" {
		rt.Mode = "stdin"
	}
	path := w.RuntimePath(rt.Name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("runtime %q already exists", rt.Name)
	}

	d := &mdstore.Doc{}
	d.Front.Set("id", "rt-"+rt.Name)
	d.Front.Set("kind", string(model.KindRuntime))
	d.Front.Set("created", now())
	d.Front.Set("created_by", actor)
	d.Front.Set("name", rt.Name)
	d.Front.Set("binary", rt.Binary)
	d.Front.Set("invoke_mode", rt.Mode)
	if rt.Flag != "" {
		d.Front.Set("invoke_flag", rt.Flag)
	}
	setInline := func(k string, v []string) {
		if len(v) > 0 {
			d.Front.Set(k, "["+strings.Join(v, ", ")+"]")
		}
	}
	setInline("invoke_args", rt.Args)
	setInline("sandbox_ro_args", rt.SandboxRO)
	setInline("env_passthrough", rt.Env)
	if rt.ModelFlag != "" {
		d.Front.Set("model_flag", rt.ModelFlag)
	}
	if rt.SkillsNativeDir != "" {
		d.Front.Set("skills_native_dir", rt.SkillsNativeDir)
	}
	if rt.SkillsContextFile != "" {
		d.Front.Set("skills_context_file", rt.SkillsContextFile)
	}
	if rt.UsageFormat != "" {
		d.Front.Set("usage_format", rt.UsageFormat)
	}
	if note == "" {
		note = "Flags here are assumptions until `dacli runtime doctor` verifies them against the installed binary."
	}
	d.Sections = []mdstore.Section{{Level: 1, Title: rt.Name, Content: note + "\n"}}
	return mdstore.WriteFile(path, d)
}

// parseRuntime builds a Runtime from a parsed adapter doc at path. It returns
// ok=false for a malformed adapter (no name or no binary), matching the filter
// LoadRuntimes has always applied.
func parseRuntime(d *mdstore.Doc, path string) (Runtime, bool) {
	rt := Runtime{Path: path}
	rt.Name, _ = d.Front.Get("name")
	rt.Binary, _ = d.Front.Get("binary")
	rt.Mode, _ = d.Front.Get("invoke_mode")
	rt.Flag, _ = d.Front.Get("invoke_flag")
	rt.Args = d.Front.GetList("invoke_args")
	rt.SandboxRO = d.Front.GetList("sandbox_ro_args")
	rt.Env = d.Front.GetList("env_passthrough")
	rt.ModelFlag, _ = d.Front.Get("model_flag")
	rt.SkillsNativeDir, _ = d.Front.Get("skills_native_dir")
	rt.SkillsContextFile, _ = d.Front.Get("skills_context_file")
	rt.UsageFormat, _ = d.Front.Get("usage_format")
	if rt.Mode == "" {
		rt.Mode = "stdin"
	}
	return rt, rt.Name != "" && rt.Binary != ""
}

// LoadRuntimes parses every adapter.
func LoadRuntimes(w *workspace.Workspace) ([]Runtime, error) {
	entries, err := os.ReadDir(w.RuntimesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no runtimes dir yet is not an error
		}
		return nil, err // a real I/O/permission failure must not read as "empty"
	}
	var out []Runtime
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := w.RuntimePath(strings.TrimSuffix(e.Name(), ".md"))
		d, err := mdstore.ReadFile(path)
		if err != nil {
			continue
		}
		if rt, ok := parseRuntime(d, path); ok {
			out = append(out, rt)
		}
	}
	return out, nil
}

// LoadRuntime reads one adapter by name from its exact file, rather than
// scanning the whole directory through LoadRuntimes.
func LoadRuntime(w *workspace.Workspace, name string) (Runtime, error) {
	path := w.RuntimePath(name)
	d, err := mdstore.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Runtime{}, ErrNotFound{Ref: "runtime/" + name}
		}
		return Runtime{}, err
	}
	if rt, ok := parseRuntime(d, path); ok {
		return rt, nil
	}
	return Runtime{}, ErrNotFound{Ref: "runtime/" + name}
}
