package store

import (
	"fmt"
	"os"
	"path/filepath"
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
			d.Front.SetList(k, v)
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

// writeToolTokens are the --allowedTools entries that let a child MODIFY the
// workspace. A runtime whose allowlist grants none of them cannot write,
// whatever grant a role hands it. Matched as whole tokens, case-insensitive, so
// a Bash rule (`Bash(git:*)`) or a model name can never be mistaken for one.
var writeToolTokens = map[string]bool{
	"edit":         true,
	"write":        true,
	"multiedit":    true,
	"notebookedit": true,
}

// argsPinAllowlist reports whether args carry an --allowedTools flag — the
// marker that this runtime restricts the child's tools to an explicit allowlist
// rather than leaving the vendor's default in place. Only such a runtime makes
// a checkable claim about what it can and cannot do.
func argsPinAllowlist(args []string) bool {
	for _, a := range args {
		if strings.EqualFold(a, "--allowedTools") || strings.EqualFold(a, "--allowed-tools") {
			return true
		}
	}
	return false
}

// allowlistGrantsWrite reports whether the --allowedTools list in args names a
// write tool. Tokens arrive either one-per-arg (`Edit`, `Write`) or comma-joined
// in one arg (`"Edit,Write,Read"`) depending on how the adapter was written, so
// every arg is split on commas before matching.
func allowlistGrantsWrite(args []string) bool {
	for _, a := range args {
		for _, tok := range strings.Split(a, ",") {
			if writeToolTokens[strings.ToLower(strings.TrimSpace(tok))] {
				return true
			}
		}
	}
	return false
}

// RuntimeWritable reports whether an rw-granted child can actually modify the
// workspace on this runtime. sandboxFor adds nothing for a non-ro grant, so the
// tools an rw child runs with are exactly the runtime's invoke args. When those
// args (or the runtime's ro sandbox) pin an --allowedTools allowlist, the child
// can write only if that allowlist names a write tool: junior's `cc` pins
// Read/Grep/Glob/LS + the dacli binary and so, despite its rw grant, can never
// edit a file — the spawn burns a whole run discovering it (dacli 250). A
// runtime that pins NO allowlist anywhere makes no such promise, so it is
// treated as writable — we refuse only a runtime that PROVABLY cannot write,
// leaving generic-exec and other non-allowlist adapters unaffected.
func RuntimeWritable(rt Runtime) bool {
	if !argsPinAllowlist(rt.Args) && !argsPinAllowlist(rt.SandboxRO) {
		return true
	}
	return allowlistGrantsWrite(rt.Args)
}

// dacliBashPrefixes extracts the command prefix of every `Bash(<prefix>:*)`
// allowlist rule in args whose binary basename is base. Tokens arrive either
// one-per-arg or comma-joined (same as allowlistGrantsWrite), so every arg is
// split on commas first. Claude Code's Bash rule is `Bash(<prefix>:<argpattern>)`;
// the command prefix is everything up to the first ':' (a unix binary path never
// contains one), so `Bash(/Users/x/go/bin/dacli:*)` yields `/Users/x/go/bin/dacli`.
func dacliBashPrefixes(args []string, base string) []string {
	var out []string
	for _, a := range args {
		for _, tok := range strings.Split(a, ",") {
			tok = strings.TrimSpace(tok)
			if !strings.HasPrefix(tok, "Bash(") || !strings.HasSuffix(tok, ")") {
				continue
			}
			inner := tok[len("Bash(") : len(tok)-1]
			if i := strings.Index(inner, ":"); i >= 0 {
				inner = inner[:i]
			}
			inner = strings.TrimSpace(inner)
			if inner != "" && filepath.Base(inner) == base {
				out = append(out, inner)
			}
		}
	}
	return out
}

// RuntimeAllowsDacli reports whether an --allowedTools allowlist built from args
// permits a headless child to run the dacli binary at exe — the path the spawn
// preamble hands the child (os.Executable(), rendered as the brief's Exe). Claude
// Code's Bash rule pins an EXACT command prefix, so exe must equal an allowlisted
// dacli path; cc-rw's `Bash(/Users/.../go/bin/dacli:*)` permits the installed
// binary and nothing else. When the allowlist names no dacli binary at all
// (allowlisted is empty) there is nothing to contradict, so it returns
// permitted=true — only an allowlist that PROVABLY pins a different dacli path is
// flagged, leaving non-allowlist and non-Bash-scoped adapters (generic-exec)
// untouched. The allowlisted dacli paths are returned so a caller can report what
// the allowlist actually permits when exe does not match (dacli 267).
func RuntimeAllowsDacli(args []string, exe string) (bool, []string) {
	prefixes := dacliBashPrefixes(args, filepath.Base(exe))
	if len(prefixes) == 0 {
		return true, nil
	}
	for _, p := range prefixes {
		if p == exe {
			return true, prefixes
		}
	}
	return false, prefixes
}

// RuntimeEnforcesRO reports whether the runtime can hold a child to read-only.
// An empty SandboxRO means it cannot, and per RUNTIMES.md § 8 that is a refusal
// to spawn a ro child, never a silent downgrade. It is the single definition
// both the spawn gate (sandboxFor) and `doctor` read, so what is shown and what
// is enforced can never diverge.
func RuntimeEnforcesRO(rt Runtime) bool {
	return len(rt.SandboxRO) > 0
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
