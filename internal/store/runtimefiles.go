package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Runtime is a parsed coding-agent CLI adapter. Everything in it is an
// assumption until `runtime doctor` probes the installed binary — including
// the presets dacli ships.
type Runtime struct {
	Name       string
	Binary     string
	Mode       string   // stdin | arg — how the prompt is delivered
	Flag       string   // arg mode: the flag preceding the prompt (e.g. -p)
	GlobalArgs []string // flags that must precede a runtime subcommand
	Args       []string

	// SandboxRO are the args that put this runtime in a read-only mode. An
	// EMPTY list means the runtime cannot enforce read-only — and per
	// RUNTIMES.md § 8 that is a refusal to spawn ro children, never a silent
	// downgrade.
	SandboxRO []string

	// ROProbe is the local installed binary's verified sandbox state. It is
	// hydrated from .dacli/build (gitignored) rather than the adapter document:
	// a declaration committed by one machine is not evidence about another.
	ROProbe RuntimeROProbe

	// Env lists variable NAMES passed through from the parent environment.
	// Values never enter the workspace; the workspace is committed to git.
	Env []string

	// ModelFlag is the flag that selects a model tier on this CLI (e.g.
	// --model). Empty means the runtime has no model selection — and then
	// role-level model routing is announced as inoperative, not ignored.
	ModelFlag string

	// TokenLimitFlag transmits a hard per-run token ceiling to this CLI. Empty
	// means the runtime cannot enforce --max-tokens; calibrated history remains
	// useful for planning but is never represented as a provider-side cap.
	TokenLimitFlag string

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

	// BehavioralPreflight selects the versioned adapter-owned launch handshake.
	// It is independent of UsageFormat: safe execution must not disappear when
	// an operator changes how provider output is parsed (issue #763).
	BehavioralPreflight           string
	BehavioralPreflightProvenance CapabilityProvenance

	// Context declares the provider-specific discovery contract using a
	// provider-neutral vocabulary. The map key is a ContextClass and the value
	// says whether that class is isolated, enumerable, deliberately allowed, or
	// outside dacli's control. Empty is intentionally not treated as isolated.
	Context map[ContextClass]ContextCapability

	Path string
}

// ContextClass is the external state capable of changing an agent's prompt or
// tools. Keep this vocabulary provider-neutral: adapters map vendor discovery
// rules onto it instead of teaching spawn five incompatible flag languages.
type ContextClass string

const (
	ContextUserConfig       ContextClass = "user-config"
	ContextRepoInstructions ContextClass = "repository-instructions"
	ContextGlobalSkills     ContextClass = "global-skills"
	ContextPlugins          ContextClass = "plugins-extensions"
	ContextMCP              ContextClass = "mcp-servers"
	ContextEnvironment      ContextClass = "environment-config"
)

var ContextClasses = []ContextClass{ContextUserConfig, ContextRepoInstructions, ContextGlobalSkills, ContextPlugins, ContextMCP, ContextEnvironment}

type ContextCapability string

const (
	ContextIsolated    ContextCapability = "isolated"
	ContextEnumerated  ContextCapability = "enumerated"
	ContextAllowed     ContextCapability = "allowed"
	ContextUnsupported ContextCapability = "unsupported"
)

func FormatContextContract(m map[ContextClass]ContextCapability) []string {
	var out []string
	for _, class := range ContextClasses {
		if capability := m[class]; capability != "" {
			out = append(out, string(class)+"="+string(capability))
		}
	}
	return out
}

func ParseContextContract(lines []string) map[ContextClass]ContextCapability {
	out := map[ContextClass]ContextCapability{}
	for _, line := range lines {
		class, capability, ok := strings.Cut(line, "=")
		if ok {
			out[ContextClass(strings.TrimSpace(class))] = ContextCapability(strings.TrimSpace(capability))
		}
	}
	return out
}

// ContextSource records names and paths only. Values from environment or
// config files are never read into a run record.
type ContextSource struct {
	Class ContextClass
	Path  string
}

// DiscoverContextSources enumerates conventional provider roots under the
// child's effective HOME/config roots. It deliberately accepts env as input so
// tests and runtime doctor use fixture homes and never inspect an operator's
// real global configuration.
func DiscoverContextSources(rt Runtime, workdir string, env map[string]string) []ContextSource {
	home := env["HOME"]
	config := env["XDG_CONFIG_HOME"]
	if config == "" && home != "" {
		config = filepath.Join(home, ".config")
	}
	base := strings.TrimSuffix(filepath.Base(rt.Binary), ".exe")
	var candidates []ContextSource
	add := func(class ContextClass, paths ...string) {
		for _, path := range paths {
			if path != "" {
				candidates = append(candidates, ContextSource{Class: class, Path: path})
			}
		}
	}
	switch base {
	case "codex":
		root := env["CODEX_HOME"]
		if root == "" && home != "" {
			root = filepath.Join(home, ".codex")
		}
		add(ContextUserConfig, filepath.Join(root, "config.toml"))
		add(ContextGlobalSkills, filepath.Join(root, "skills"), filepath.Join(home, ".agents", "skills"))
		add(ContextMCP, filepath.Join(root, "config.toml"))
	case "claude":
		add(ContextUserConfig, filepath.Join(home, ".claude.json"), filepath.Join(home, ".claude", "settings.json"))
		add(ContextGlobalSkills, filepath.Join(home, ".claude", "skills"))
		add(ContextPlugins, filepath.Join(home, ".claude", "plugins"))
		add(ContextMCP, filepath.Join(home, ".claude.json"))
	case "gemini":
		add(ContextUserConfig, filepath.Join(home, ".gemini", "settings.json"))
		add(ContextGlobalSkills, filepath.Join(home, ".gemini", "skills"))
		add(ContextPlugins, filepath.Join(home, ".gemini", "extensions"))
		add(ContextMCP, filepath.Join(home, ".gemini", "settings.json"))
	case "copilot":
		add(ContextUserConfig, filepath.Join(config, "github-copilot"))
		add(ContextPlugins, filepath.Join(config, "github-copilot", "extensions"))
		add(ContextMCP, filepath.Join(config, "github-copilot", "mcp.json"))
	}
	add(ContextRepoInstructions, filepath.Join(workdir, "AGENTS.md"), filepath.Join(workdir, "CLAUDE.md"), filepath.Join(workdir, ".github", "copilot-instructions.md"), filepath.Join(workdir, "GEMINI.md"))
	for _, name := range rt.Env {
		if _, ok := env[name]; ok && name != "HOME" && name != "PATH" && name != "USER" && name != "LOGNAME" && name != "TMPDIR" {
			add(ContextEnvironment, "$"+name)
		}
	}
	seen := map[string]bool{}
	var out []ContextSource
	for _, source := range candidates {
		if strings.HasPrefix(source.Path, "$") {
			// Environment-derived provenance records the variable NAME only.
		} else if info, err := os.Stat(source.Path); err != nil || (info.IsDir() && dirEmpty(source.Path)) {
			continue
		}
		key := string(source.Class) + "\x00" + source.Path
		if !seen[key] {
			seen[key] = true
			out = append(out, source)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Class == out[j].Class {
			return out[i].Path < out[j].Path
		}
		return out[i].Class < out[j].Class
	})
	return out
}

func dirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) == 0
}

func ContextSummary(rt Runtime) string {
	parts := FormatContextContract(rt.Context)
	if len(parts) == 0 {
		return "context provenance undeclared"
	}
	return "context " + strings.Join(parts, ",")
}

func ValidateContextContract(rt Runtime) error {
	for _, class := range ContextClasses {
		capability := rt.Context[class]
		switch capability {
		case ContextIsolated, ContextEnumerated, ContextAllowed, ContextUnsupported:
		default:
			return fmt.Errorf("runtime %s context class %s has invalid or missing capability %q", rt.Name, class, capability)
		}
	}
	return nil
}

// RuntimeROProbe keeps declaration and observation separate. Unknown is the
// zero value on purpose: an adapter that has not been checked on this install
// must never accidentally become eligible for an ro spawn.
type RuntimeROProbe string

const (
	RuntimeROUnknown  RuntimeROProbe = "unknown"
	RuntimeROVerified RuntimeROProbe = "verified"
	RuntimeROFailed   RuntimeROProbe = "failed"
)

type runtimeProbeCache struct {
	Fingerprint string                            `json:"fingerprint"`
	ReadOnly    RuntimeROProbe                    `json:"read_only"`
	Detail      string                            `json:"detail,omitempty"`
	Launch      map[string]RuntimeLaunchPreflight `json:"launch,omitempty"`
}

// LaunchLayer is the provider-neutral vocabulary scheduling may act on.
// Provider adapters classify their own output into these values; spawn never
// parses vendor prose (issue #746).
type LaunchLayer string

const (
	LaunchAuthentication LaunchLayer = "authentication"
	LaunchSandbox        LaunchLayer = "sandbox"
	LaunchStartup        LaunchLayer = "startup"
	LaunchQuota          LaunchLayer = "quota"
	LaunchTransport      LaunchLayer = "transport"
)

type LaunchState string

const (
	LaunchCompatible   LaunchState = "compatible"
	LaunchIncompatible LaunchState = "incompatible"
	LaunchTransient    LaunchState = "transient"
	LaunchUnsupported  LaunchState = "unsupported"
)

type CapabilityProvenance string

const (
	ProvenanceDeclared CapabilityProvenance = "declared"
	ProvenanceInferred CapabilityProvenance = "inferred"
	ProvenanceProbed   CapabilityProvenance = "probed"
)

const (
	BehavioralPreflightCodexExecJSONV1 = "codex-exec-json-v1"
	// V2 treats the first valid Codex lifecycle readiness event as the end of
	// the transport probe. V1 waited for a complete model turn, so its fixed
	// deadline measured inference latency rather than startup compatibility.
	BehavioralPreflightCodexExecJSONV2 = "codex-exec-json-v2"
	// ClaudePrintV1 reaches Claude Code's stream-json initialization event. It
	// proves the installed CLI can authenticate before dacli spends a governed
	// run; the adapter owns its invocation and failure prose classification.
	BehavioralPreflightClaudePrintV1 = "claude-print-v1"
)

// RuntimeLaunchPreflight is deliberately safe to print and persist: it holds
// classifications and timestamps, never argv, environment values, or the
// handshake prompt.
type RuntimeLaunchPreflight struct {
	State            LaunchState          `json:"state"`
	Layer            LaunchLayer          `json:"layer,omitempty"`
	Detail           string               `json:"detail,omitempty"`
	Provenance       CapabilityProvenance `json:"provenance"`
	CommandTimestamp time.Time            `json:"command_timestamp"`
}

// CreateRuntime writes .dacli/runtimes/<name>.md.
func CreateRuntime(w *workspace.Workspace, actor string, rt Runtime, note string) error {
	if rt.Name == "" || rt.Binary == "" {
		return fmt.Errorf("a runtime needs at least --name and --binary")
	}
	if rt.Mode == "" {
		rt.Mode = "stdin"
	}
	// Same containment guard its siblings carry: the name becomes a filename,
	// and a runtime file describes what a spawned child is allowed to do, so
	// one written outside .dacli is both an escape and a policy document in a
	// place nothing audits.
	if !workspace.SafeSegment(rt.Name) {
		return fmt.Errorf("invalid runtime name %q: must be a single path segment without '/' or '..'", rt.Name)
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
	setInline("global_args", rt.GlobalArgs)
	setInline("sandbox_ro_args", rt.SandboxRO)
	setInline("env_passthrough", rt.Env)
	if rt.ModelFlag != "" {
		d.Front.Set("model_flag", rt.ModelFlag)
	}
	if rt.TokenLimitFlag != "" {
		d.Front.Set("token_limit_flag", rt.TokenLimitFlag)
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
	if rt.BehavioralPreflight != "" {
		d.Front.Set("behavioral_preflight", rt.BehavioralPreflight)
	}
	if len(rt.Context) > 0 {
		d.Front.SetList("context_provenance", FormatContextContract(rt.Context))
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
	rt := Runtime{Path: path, ROProbe: RuntimeROUnknown}
	rt.Name, _ = d.Front.Get("name")
	rt.Binary, _ = d.Front.Get("binary")
	rt.Mode, _ = d.Front.Get("invoke_mode")
	rt.Flag, _ = d.Front.Get("invoke_flag")
	rt.Args = d.Front.GetList("invoke_args")
	rt.GlobalArgs = d.Front.GetList("global_args")
	rt.SandboxRO = d.Front.GetList("sandbox_ro_args")
	rt.Env = d.Front.GetList("env_passthrough")
	rt.ModelFlag, _ = d.Front.Get("model_flag")
	rt.TokenLimitFlag, _ = d.Front.Get("token_limit_flag")
	rt.SkillsNativeDir, _ = d.Front.Get("skills_native_dir")
	rt.SkillsContextFile, _ = d.Front.Get("skills_context_file")
	rt.UsageFormat, _ = d.Front.Get("usage_format")
	if rt.Mode == "" {
		rt.Mode = "stdin"
	}
	rt.BehavioralPreflight, _ = d.Front.Get("behavioral_preflight")
	if rt.BehavioralPreflight != "" {
		rt.BehavioralPreflightProvenance = ProvenanceDeclared
	} else if legacyCodexBehavioralPreflight(rt) {
		// Mature workspaces predate the capability field. Infer only the two
		// exact shipped exec contracts; a binary or runtime name alone is not
		// authority to execute provider-specific probing (issue #763).
		rt.BehavioralPreflight = BehavioralPreflightCodexExecJSONV2
		rt.BehavioralPreflightProvenance = ProvenanceInferred
	} else if legacyClaudeBehavioralPreflight(rt) {
		// The shipped Claude presets also predate the capability field. Infer
		// only their exact -p/stream-json transport contract; a binary or
		// runtime name by itself is not authority for provider-specific probing.
		rt.BehavioralPreflight = BehavioralPreflightClaudePrintV1
		rt.BehavioralPreflightProvenance = ProvenanceInferred
	}
	rt.Context = ParseContextContract(d.Front.GetList("context_provenance"))
	return rt, rt.Name != "" && rt.Binary != ""
}

func legacyClaudeBehavioralPreflight(rt Runtime) bool {
	if filepath.Base(rt.Binary) != "claude" || rt.Mode != "arg" || rt.Flag != "-p" || rt.UsageFormat != "stream-json" || len(rt.GlobalArgs) != 0 {
		return false
	}
	if rt.ModelFlag != "" && rt.ModelFlag != "--model" {
		return false
	}
	return validClaudeToolArgs(rt.Args) && validClaudeToolArgs(rt.SandboxRO)
}

func validClaudeToolArgs(args []string) bool {
	if len(args) == 0 {
		return true
	}
	if len(args) < 2 || args[0] != "--allowedTools" {
		return false
	}
	for _, tool := range args[1:] {
		if tool == "" || strings.HasPrefix(tool, "--") {
			return false
		}
	}
	return true
}

func legacyCodexBehavioralPreflight(rt Runtime) bool {
	if filepath.Base(rt.Binary) != "codex" || rt.Mode != "stdin" || rt.Flag != "" || (len(rt.SandboxRO) > 0 && !equalStrings(rt.SandboxRO, []string{"--sandbox", "read-only"})) {
		return false
	}
	// Bootstrap adapters written before GlobalArgs existed persisted the same
	// supported Codex contract entirely in invoke_args, with optional context
	// isolation and the two shared worktree directories. Parse that narrow
	// grammar instead of requiring the newer storage layout. Unknown flags fail
	// closed, so a provider-shaped custom adapter is never inferred by name.
	args := append(append([]string{}, rt.GlobalArgs...), rt.Args...)
	seen := map[string]bool{}
	disableAllowed := map[string]bool{"plugins": true, "plugin_sharing": true, "remote_plugin": true}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "exec":
			seen["exec"] = true
		case "--json", "--ephemeral", "--ignore-user-config":
			seen[args[i]] = true
		case "--ask-for-approval":
			i++
			if i >= len(args) || args[i] != "never" {
				return false
			}
			seen["approval"] = true
		case "--color":
			i++
			if i >= len(args) || args[i] != "never" {
				return false
			}
			seen["color"] = true
		case "--sandbox":
			i++
			if i >= len(args) || (args[i] != "read-only" && args[i] != "workspace-write") {
				return false
			}
			seen["sandbox"] = true
		case "--disable":
			i++
			if i >= len(args) || !disableAllowed[args[i]] {
				return false
			}
		case "--add-dir":
			i++
			if i >= len(args) || strings.TrimSpace(args[i]) == "" {
				return false
			}
		default:
			return false
		}
	}
	if !seen["sandbox"] && equalStrings(rt.SandboxRO, []string{"--sandbox", "read-only"}) {
		seen["sandbox"] = true
	}
	return seen["exec"] && seen["--json"] && seen["--ephemeral"] && seen["approval"] && seen["color"] && seen["sandbox"]
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
	if hasArgPair(rt.Args, "--sandbox", "read-only") {
		return false
	}
	if !argsPinAllowlist(rt.Args) && !argsPinAllowlist(rt.SandboxRO) {
		return true
	}
	return allowlistGrantsWrite(rt.Args)
}

func hasArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
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

// knownToolNames is the fixed vocabulary dacli 272's preflight recognizes when
// a role prompt names a tool. Matched against a Claude-Code-shaped
// --allowedTools allowlist, so this list is that surface, not an open one.
var knownToolNames = []string{
	"Bash", "Read", "Write", "Edit", "MultiEdit", "NotebookEdit", "NotebookRead",
	"Grep", "Glob", "LS", "WebFetch", "WebSearch", "Task", "TodoWrite", "ExitPlanMode",
}

// namedToolPattern matches a single backtick-quoted word, e.g. a role prompt
// containing WebFetch wrapped in backticks.
var namedToolPattern = regexp.MustCompile("`([A-Za-z]+)`")

// NamedTools extracts every tool the role prompt names in a backtick code
// span, in first-seen order with duplicates dropped. Prose verbs ("Write the
// failing test", "Read the surrounding code") do not count — only an exact,
// code-quoted match against knownToolNames does. This is the same precision
// discipline dacliBashPrefixes uses for the binary-allowlist check: a plain
// word can never be mistaken for a tool the runtime must actually grant.
func NamedTools(prompt string) []string {
	known := make(map[string]bool, len(knownToolNames))
	for _, n := range knownToolNames {
		known[n] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, m := range namedToolPattern.FindAllStringSubmatch(prompt, -1) {
		tok := m[1]
		if known[tok] && !seen[tok] {
			seen[tok] = true
			out = append(out, tok)
		}
	}
	return out
}

// allowlistedToolNames is allowlistGrantsWrite generalized from the write
// subset to the full set: every non-Bash token an --allowedTools list names,
// lower-cased for case-insensitive lookup. A Bash(<prefix>:*) rule scopes one
// shell command, not a tool, so it is excluded here exactly as
// dacliBashPrefixes excludes everything else from ITS extraction.
func allowlistedToolNames(args []string) map[string]bool {
	out := map[string]bool{}
	for _, a := range args {
		for _, tok := range strings.Split(a, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" || strings.HasPrefix(tok, "Bash(") {
				continue
			}
			out[strings.ToLower(tok)] = true
		}
	}
	return out
}

// RuntimeAllowsTool reports whether an --allowedTools allowlist built from
// args permits tool. Mirrors RuntimeWritable's convention: a runtime that
// pins no allowlist anywhere makes no restrictive promise, so every tool is
// permitted; one that pins an allowlist permits only what it names.
func RuntimeAllowsTool(args []string, tool string) bool {
	if !argsPinAllowlist(args) {
		return true
	}
	return allowlistedToolNames(args)[strings.ToLower(tool)]
}

// RuntimeEnforcesRO reports whether this installed runtime has passed the
// local sandbox probe. A declared argument list is only an assumption and is
// never enough by itself (dacli 365).
func RuntimeEnforcesRO(rt Runtime) bool {
	return rt.ROProbe == RuntimeROVerified
}

var readOnlyToolTokens = map[string]bool{
	"read":         true,
	"grep":         true,
	"glob":         true,
	"ls":           true,
	"notebookread": true,
	"webfetch":     true,
	"websearch":    true,
}

// recognizedReadOnlyAllowlist accepts only the allowlist shape whose semantics
// dacli can inspect completely. In particular Bash(git:*) is not read-only
// merely because it is not named Edit or Write; the only Bash carve-out is the
// dacli reporting channel, whose commands remain grant-gated by the child token.
func recognizedReadOnlyAllowlist(args []string) bool {
	if len(args) < 2 || (!strings.EqualFold(args[0], "--allowedTools") && !strings.EqualFold(args[0], "--allowed-tools")) {
		return false
	}
	seen := false
	for _, arg := range args[1:] {
		for _, raw := range strings.Split(arg, ",") {
			tok := strings.TrimSpace(raw)
			lower := strings.ToLower(tok)
			switch {
			case readOnlyToolTokens[lower]:
				seen = true
			case strings.HasPrefix(lower, "bash(") && strings.HasSuffix(lower, ")"):
				inner := tok[len("Bash(") : len(tok)-1]
				if i := strings.Index(inner, ":"); i >= 0 {
					inner = inner[:i]
				}
				if filepath.Base(strings.TrimSpace(inner)) != "dacli" {
					return false
				}
				seen = true
			default:
				return false
			}
		}
	}
	return seen
}

// RuntimeROProbeable says whether the declaration itself has semantics dacli
// can check without running a paid prompt. Unknown vendor-specific flags and
// partially understood allowlists stay declaration-only rather than being
// blessed by a successful --help.
func RuntimeROProbeable(rt Runtime) bool {
	return recognizedReadOnlyAllowlist(rt.SandboxRO) || hasArgPair(rt.SandboxRO, "--sandbox", "read-only") ||
		hasArgPair(rt.SandboxRO, "--approval-mode", "plan") ||
		(hasArgPair(rt.SandboxRO, "--deny-tool", "write") && hasArgPair(rt.SandboxRO, "--deny-tool", "shell"))
}

func runtimeProbePath(w *workspace.Workspace, name string) string {
	// Runtime documents can be hand-edited, so do not turn their frontmatter
	// name back into a path without the same containment guarantee CreateRuntime
	// applies. Hashing only malformed legacy names keeps even those caches under
	// runtime-probes while preserving readable filenames for valid adapters.
	cacheName := name
	if !workspace.SafeSegment(cacheName) {
		sum := sha256.Sum256([]byte(cacheName))
		cacheName = "invalid-" + hex.EncodeToString(sum[:])
	}
	return filepath.Join(w.Root, workspace.Dir, "build", "runtime-probes", cacheName+".json")
}

func runtimeProbeFingerprint(rt Runtime, binaryPath string) (string, error) {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00", rt.Binary, binaryPath, strings.Join(rt.GlobalArgs, "\x00"), strings.Join(rt.SandboxRO, "\x00"))
	b, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", err
	}
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HydrateRuntimeROProbe applies a cached verdict only when it describes this
// exact adapter declaration and installed binary. Missing, corrupt, or stale
// cache data safely leaves the state unknown.
func HydrateRuntimeROProbe(w *workspace.Workspace, rt Runtime, binaryPath string) Runtime {
	rt.ROProbe = RuntimeROUnknown
	if len(rt.SandboxRO) == 0 {
		rt.ROProbe = RuntimeROFailed
		return rt
	}
	b, err := os.ReadFile(runtimeProbePath(w, rt.Name))
	if err != nil {
		return rt
	}
	var c runtimeProbeCache
	fingerprint, err := runtimeProbeFingerprint(rt, binaryPath)
	if err != nil || json.Unmarshal(b, &c) != nil || c.Fingerprint != fingerprint {
		return rt
	}
	if c.ReadOnly == RuntimeROVerified || c.ReadOnly == RuntimeROFailed {
		rt.ROProbe = c.ReadOnly
	}
	return rt
}

// SaveRuntimeROProbe records a local probe outside the committed adapter.
func SaveRuntimeROProbe(w *workspace.Workspace, rt Runtime, binaryPath string, state RuntimeROProbe, detail string) error {
	fingerprint, err := runtimeProbeFingerprint(rt, binaryPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(runtimeProbePath(w, rt.Name))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	cache := runtimeProbeCache{Fingerprint: fingerprint, ReadOnly: state, Detail: detail}
	if old, readErr := os.ReadFile(runtimeProbePath(w, rt.Name)); readErr == nil {
		var existing runtimeProbeCache
		if json.Unmarshal(old, &existing) == nil && existing.Fingerprint == fingerprint {
			cache.Launch = existing.Launch
		}
	}
	b, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return mdstore.WriteBytes(runtimeProbePath(w, rt.Name), append(b, '\n'), 0o600)
}

func runtimeLaunchFingerprint(rt Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool) (string, error) {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%t\x00",
		rt.Binary, binaryPath, rt.Mode, rt.Flag, strings.Join(rt.GlobalArgs, "\x00"), strings.Join(rt.Args, "\x00"), strings.Join(rt.SandboxRO, "\x00"), strings.Join(rt.Env, "\x00"), rt.BehavioralPreflight, allowUserConfig)
	_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00", grant, selectedModel, rt.ModelFlag)
	b, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", err
	}
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// LoadFreshRuntimeLaunchPreflight returns evidence only for the exact binary,
// adapter declaration, grant, model, and user-configuration policy. A missing,
// future-dated, or expired timestamp cannot authorize launch.
func LoadFreshRuntimeLaunchPreflight(w *workspace.Workspace, rt Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool, now time.Time, maxAge time.Duration) (RuntimeLaunchPreflight, bool) {
	var zero RuntimeLaunchPreflight
	b, err := os.ReadFile(runtimeProbePath(w, rt.Name))
	if err != nil {
		return zero, false
	}
	var cache runtimeProbeCache
	key, err := runtimeLaunchFingerprint(rt, binaryPath, grant, selectedModel, allowUserConfig)
	if err != nil || json.Unmarshal(b, &cache) != nil || cache.Launch == nil {
		return zero, false
	}
	result, ok := cache.Launch[key]
	age := now.Sub(result.CommandTimestamp)
	if !ok || result.CommandTimestamp.IsZero() || age < 0 || age > maxAge {
		return zero, false
	}
	return result, true
}

// SaveRuntimeLaunchPreflight preserves the independent read-only probe in the
// same per-install cache while adding exact launch evidence.
func SaveRuntimeLaunchPreflight(w *workspace.Workspace, rt Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool, result RuntimeLaunchPreflight) error {
	fingerprint, err := runtimeProbeFingerprint(rt, binaryPath)
	if err != nil {
		return err
	}
	cache := runtimeProbeCache{Fingerprint: fingerprint, ReadOnly: RuntimeROUnknown, Launch: map[string]RuntimeLaunchPreflight{}}
	if b, readErr := os.ReadFile(runtimeProbePath(w, rt.Name)); readErr == nil {
		var existing runtimeProbeCache
		if json.Unmarshal(b, &existing) == nil && existing.Fingerprint == fingerprint {
			cache = existing
			if cache.Launch == nil {
				cache.Launch = map[string]RuntimeLaunchPreflight{}
			}
		}
	}
	key, err := runtimeLaunchFingerprint(rt, binaryPath, grant, selectedModel, allowUserConfig)
	if err != nil {
		return err
	}
	cache.Launch[key] = result
	dir := filepath.Dir(runtimeProbePath(w, rt.Name))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	b, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return mdstore.WriteBytes(runtimeProbePath(w, rt.Name), append(b, '\n'), 0o600)
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

// ungatedOutwardTools names allowlist entries that let a child perform an
// OUTWARD write — one that leaves this machine — without passing any of
// dacli's consent controls.
//
// The controls exist and work: `github push`, `github project`, `catalog
// --publish-wiki` and the rest are rw-gated at the dispatcher, and the
// disclosure gate records per-project consent before a public mirror. All of
// it is bypassed the moment a child can shell to `gh` directly. That is not
// hypothetical — an autonomous agent created a private GitHub repo, set it as
// origin, pushed, and opened and merged PRs, none of which the operator
// approved and none of which dacli could see (task 308, issue #382 item 6).
//
// `git` is deliberately NOT here: an implementer needs it for its own branch
// and commits, and a push still needs a remote the operator configured.
var ungatedOutwardTools = map[string]string{
	"bash(gh:*)":   "`gh` can create repositories, set remotes, and open and merge PRs — outside every consent gate dacli has",
	"bash(curl:*)": "`curl` can post this workspace's contents anywhere",
	"bash(*)":      "an unrestricted Bash allowlist grants every outward tool on the machine",
}

// UngatedOutwardGrant returns the first allowlist entry in args that hands a
// child ungated outward reach, with the reason, or "" when none does.
//
// It reports rather than refuses at this layer: a runtime that deliberately
// grants `gh` is a legitimate configuration for an operator who wants it, and
// the failure mode worth preventing is the SILENT one — nobody deciding,
// because nobody was told.
func UngatedOutwardGrant(args []string) (entry, why string) {
	for _, a := range args {
		for _, tok := range strings.Split(a, ",") {
			t := strings.ToLower(strings.TrimSpace(tok))
			if reason, bad := ungatedOutwardTools[t]; bad {
				return t, reason
			}
		}
	}
	return "", ""
}
