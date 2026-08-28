package mcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/prompts"
)

const toolSchemaVersion = 1

// ToolCapability is the read-only registry view used by the generated CLI
// capability manifest. It is derived from tools below, never maintained as a
// second MCP catalog.
type ToolCapability struct {
	Name          string
	SchemaVersion int
}

func ToolCapabilities() []ToolCapability {
	out := make([]ToolCapability, 0, len(tools))
	for _, t := range tools {
		out = append(out, ToolCapability{Name: t.name, SchemaVersion: toolSchemaVersion})
	}
	return out
}

func ToolSchemaVersion() int { return toolSchemaVersion }

// tool is one Tier-1 entry: a typed schema over a CLI command. Both tiers
// build argv for the same dispatch table, which is the no-drift property —
// the tiering replaced the original one-tool-per-command promise because a
// 50-schema catalog is the per-agent tax this design refuses elsewhere.
type tool struct {
	name   string
	desc   string
	schema map[string]any
	build  func(args map[string]any) (argv []string, jsonMode bool, err error)
}

func toolByName(name string) (tool, bool) {
	for _, t := range tools {
		if t.name == name {
			return t, true
		}
	}
	return tool{}, false
}

// --- schema helpers: hand-rolled JSON Schema fragments ---

func obj(required []string, props map[string]any) map[string]any {
	props["schema_version"] = map[string]any{
		"type":        "integer",
		"const":       toolSchemaVersion,
		"description": "tool input schema version",
	}
	required = append([]string{"schema_version"}, required...)
	s := map[string]any{"type": "object", "properties": props, "required": required}
	return s
}
func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any { return map[string]any{"type": "integer", "description": desc} }
func boolp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func strs(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

// --- argument helpers ---

func s(args map[string]any, k string) string {
	if v, ok := args[k].(string); ok {
		return v
	}
	return ""
}

// i coerces an MCP argument to an int. JSON numbers arrive as float64, but a
// caller building the map in Go can hand over an int, and returning 0 for that
// is a silently wrong answer rather than a refusal.
func i(args map[string]any, k string) int {
	switch v := args[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

// b coerces an MCP argument to a bool, INCLUDING the string spellings.
//
// It used to accept only a real bool, which made it inconsistent with i()
// — that one has always accepted "3" — and the inconsistency was not
// cosmetic. `b(a, "dry_run")` gates the preview path: a client sending
// {"dry_run": "true"}, which several MCP clients do because they stringify
// scalars, silently got FALSE and a real mutation instead of a rehearsal.
// That is the `--dry-run 001` incident one layer up, and the same reasoning
// applies — a safety flag the caller wrote must not read as unset.
//
// Anything genuinely uninterpretable still yields false. That window is
// narrower than it was, and closing it entirely means refusing rather than
// coercing, which changes every call site — filed rather than half-done here.
func b(args map[string]any, k string) bool {
	switch v := args[k].(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && parsed
	}
	return false
}
func list(args map[string]any, k string) []string {
	raw, _ := args[k].([]any)
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if sv, ok := r.(string); ok {
			out = append(out, sv)
		}
	}
	return out
}

func need(args map[string]any, keys ...string) error {
	for _, k := range keys {
		if s(args, k) == "" {
			return fmt.Errorf("missing required argument %q", k)
		}
	}
	return nil
}

// The Tier-1 surface: the verbs an agent uses between claim and done, plus
// the `cli` escape hatch for the admin tail. Descriptions carry the workflow
// — for the primary audience, they ARE the documentation.
var tools = []tool{
	{
		name: "get_context",
		desc: prompts.MCPDesc("get_context"),
		schema: obj([]string{"ref"}, map[string]any{
			"ref":    str("task reference: ULID, NNN, or slug"),
			"budget": num("approximate token ceiling; sections trim bottom-up, announced"),
			"record": boolp("freeze this brief under .dacli/runs/ for replay"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "ref"); err != nil {
				return nil, false, err
			}
			argv := []string{"context", s(a, "ref")}
			if n := i(a, "budget"); n > 0 {
				argv = append(argv, "--budget", strconv.Itoa(n))
			}
			if b(a, "record") {
				argv = append(argv, "--record")
			}
			return argv, false, nil
		},
	},
	{
		name:   "whoami",
		desc:   prompts.MCPDesc("whoami"),
		schema: obj(nil, map[string]any{}),
		build:  func(a map[string]any) ([]string, bool, error) { return []string{"whoami"}, false, nil },
	},
	{
		name:   "status",
		desc:   prompts.MCPDesc("status"),
		schema: obj(nil, map[string]any{}),
		build:  func(a map[string]any) ([]string, bool, error) { return []string{"status"}, false, nil },
	},
	{
		name: "add_task",
		desc: prompts.MCPDesc("add_task"),
		schema: obj([]string{"project", "title"}, map[string]any{
			"project":    str("project slug"),
			"title":      str("specific, unambiguous task title"),
			"priority":   str("must | should | could | wont"),
			"estimate":   str("three-point 'optimistic,probable,pessimistic', e.g. '2,5,14'"),
			"accept":     strs("acceptance criteria — binary, checkable"),
			"so_that":    str("the value clause: why this task matters"),
			"depends_on": strs("task refs, optionally typed: '001' or '001:SS' (SS = may overlap)"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "project", "title"); err != nil {
				return nil, false, err
			}
			argv := []string{"task", "add", s(a, "title"), "--project", s(a, "project")}
			for _, f := range []string{"priority", "estimate"} {
				if v := s(a, f); v != "" {
					argv = append(argv, "--"+f, v)
				}
			}
			if v := s(a, "so_that"); v != "" {
				argv = append(argv, "--so-that", v)
			}
			for _, acc := range list(a, "accept") {
				argv = append(argv, "--accept", acc)
			}
			for _, d := range list(a, "depends_on") {
				argv = append(argv, "--depends-on", d)
			}
			return argv, false, nil
		},
	},
	{
		name: "list_tasks",
		desc: prompts.MCPDesc("list_tasks"),
		schema: obj(nil, map[string]any{
			"project": str("project slug filter"),
			"status":  str("status filter"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			argv := []string{"task", "list"}
			if v := s(a, "project"); v != "" {
				argv = append(argv, "--project", v)
			}
			if v := s(a, "status"); v != "" {
				argv = append(argv, "--status", v)
			}
			return argv, true, nil
		},
	},
	{
		name:   "claim_task",
		desc:   prompts.MCPDesc("claim_task"),
		schema: obj([]string{"ref"}, map[string]any{"ref": str("task reference")}),
		build:  refCmd("task", "claim"),
	},
	{
		name: "check_task",
		desc: prompts.MCPDesc("check_task"),
		schema: obj([]string{"ref"}, map[string]any{
			"ref": str("task reference"),
			"n":   num("1-based box number; omit with all=true"),
			"all": boolp("check every box"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "ref"); err != nil {
				return nil, false, err
			}
			argv := []string{"task", "check", s(a, "ref")}
			if b(a, "all") {
				argv = append(argv, "--all")
			} else if n := i(a, "n"); n > 0 {
				argv = append(argv, "--n", strconv.Itoa(n))
			}
			return argv, false, nil
		},
	},
	{
		name:   "finish_task",
		desc:   prompts.MCPDesc("finish_task"),
		schema: obj([]string{"ref"}, map[string]any{"ref": str("task reference")}),
		build:  refCmd("task", "done"),
	},
	{
		name: "block_task",
		desc: prompts.MCPDesc("block_task"),
		schema: obj([]string{"ref"}, map[string]any{
			"ref": str("task reference"),
			"by":  str("blocking task/object ref"),
			"why": str("one-line reason"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "ref"); err != nil {
				return nil, false, err
			}
			argv := []string{"task", "block", s(a, "ref")}
			if v := s(a, "by"); v != "" {
				argv = append(argv, "--by", v)
			}
			if v := s(a, "why"); v != "" {
				argv = append(argv, "--why", v)
			}
			return argv, false, nil
		},
	},
	{
		name: "add_note",
		desc: prompts.MCPDesc("add_note"),
		schema: obj([]string{"kind", "title", "project"}, map[string]any{
			"kind":     str("decision | finding | metric | ref"),
			"title":    str("one-line summary"),
			"project":  str("project slug"),
			"about":    str("task/object this attaches to"),
			"body":     str("the content"),
			"severity": str("findings: major | moderate | minor"),
			"rejected": str("decisions: what was rejected (required)"),
			"because":  str("decisions: why the rejection holds"),
			"scope":    str("project | workspace — workspace lessons reach other projects"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "kind", "title", "project"); err != nil {
				return nil, false, err
			}
			argv := []string{"note", "add", s(a, "kind"), s(a, "title"), "--project", s(a, "project")}
			for _, f := range []string{"about", "body", "severity", "rejected", "because", "scope"} {
				if v := s(a, f); v != "" {
					argv = append(argv, "--"+f, v)
				}
			}
			return argv, false, nil
		},
	},
	{
		name: "ask",
		desc: prompts.MCPDesc("ask"),
		schema: obj([]string{"question", "about"}, map[string]any{
			"question": str("the specific question"),
			"about":    str("the task this blocks"),
			"need":     str("path or object the question concerns"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "question", "about"); err != nil {
				return nil, false, err
			}
			argv := []string{"ask", s(a, "question"), "--about", s(a, "about")}
			if v := s(a, "need"); v != "" {
				argv = append(argv, "--need", v)
			}
			return argv, false, nil
		},
	},
	{
		name: "answer",
		desc: prompts.MCPDesc("answer"),
		schema: obj([]string{"question_id", "answer"}, map[string]any{
			"question_id": str("the question event id (prefix ok)"),
			"answer":      str("the answer"),
			"as":          str("decision | finding (default finding)"),
			"rejected":    str("decisions: what was rejected"),
			"because":     str("decisions: why"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "question_id", "answer"); err != nil {
				return nil, false, err
			}
			argv := []string{"answer", s(a, "question_id"), s(a, "answer")}
			for _, f := range []string{"as", "rejected", "because"} {
				if v := s(a, f); v != "" {
					argv = append(argv, "--"+f, v)
				}
			}
			return argv, false, nil
		},
	},
	{
		name: "run_shortcut",
		desc: prompts.MCPDesc("run_shortcut"),
		schema: obj([]string{"name"}, map[string]any{
			"name":    str("shortcut name (see the brief's Shortcuts section)"),
			"params":  map[string]any{"type": "object", "description": "parameter values; every value is shell-quoted", "additionalProperties": map[string]any{"type": "string"}},
			"dry_run": boolp("print the expansion instead of executing"),
			"confirm": boolp("required for destructive shortcuts"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "name"); err != nil {
				return nil, false, err
			}
			argv := []string{"run", s(a, "name")}
			if params, ok := a["params"].(map[string]any); ok {
				for k, v := range params {
					argv = append(argv, "--"+k, fmt.Sprint(v))
				}
			}
			if b(a, "dry_run") {
				argv = append(argv, "--dry-run")
			}
			if b(a, "confirm") {
				argv = append(argv, "--confirm")
			}
			return argv, false, nil
		},
	},
	{
		name:   "queue_next",
		desc:   prompts.MCPDesc("queue_next"),
		schema: obj([]string{"queue"}, map[string]any{"queue": str("queue slug")}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "queue"); err != nil {
				return nil, false, err
			}
			return []string{"queue", "next", s(a, "queue")}, false, nil
		},
	},
	{
		name: "queue_advance",
		desc: prompts.MCPDesc("queue_advance"),
		schema: obj([]string{"queue"}, map[string]any{
			"queue":       str("queue slug"),
			"fail_reason": str("halts the queue instead of advancing"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "queue"); err != nil {
				return nil, false, err
			}
			argv := []string{"queue", "advance", s(a, "queue")}
			if v := s(a, "fail_reason"); v != "" {
				argv = append(argv, "--fail", v)
			}
			return argv, false, nil
		},
	},
	{
		name: "cleanup_repository",
		desc: prompts.MCPDesc("cleanup_repository"),
		schema: obj([]string{"project"}, map[string]any{
			"project":    str("project slug"),
			"dry_run":    boolp("produce the immutable cleanup plan without writing"),
			"apply_safe": str("exact plan id returned by a prior dry-run"),
			"restore":    str("cleanup plan id whose audit recorded the quarantine move"),
			"artifact":   str("artifact identity from that cleanup audit"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "project"); err != nil {
				return nil, false, err
			}
			apply, restore := s(a, "apply_safe"), s(a, "restore")
			dry, artifact := b(a, "dry_run"), s(a, "artifact")
			modes := 0
			if dry {
				modes++
			}
			if apply != "" {
				modes++
			}
			if restore != "" {
				modes++
			}
			if modes != 1 {
				return nil, false, fmt.Errorf("choose exactly one of dry_run, apply_safe, or restore")
			}
			if (restore != "") != (artifact != "") {
				return nil, false, fmt.Errorf("artifact is required exactly when restore is set")
			}
			argv := []string{"cleanup", "--project", s(a, "project")}
			switch {
			case dry:
				argv = append(argv, "--dry-run")
			case apply != "":
				argv = append(argv, "--apply-safe", apply)
			default:
				argv = append(argv, "--restore", restore, "--artifact", artifact)
			}
			return argv, true, nil
		},
	},
	{
		name: "reconcile_event_journal",
		desc: prompts.MCPDesc("reconcile_event_journal"),
		schema: obj([]string{"project"}, map[string]any{
			"project":         str("project slug"),
			"archive_classes": strs("evidence classes eligible for recoverable archival: complete-journal and/or complete-mailbox"),
			"dry_run":         boolp("classify and report count/byte impact without writing"),
			"apply_safe":      str("exact immutable plan id returned by dry_run"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "project"); err != nil {
				return nil, false, err
			}
			apply, dry := s(a, "apply_safe"), b(a, "dry_run")
			if dry == (apply != "") {
				return nil, false, fmt.Errorf("choose exactly one of dry_run or apply_safe")
			}
			argv := []string{"events", "reconcile", "--project", s(a, "project")}
			for _, class := range list(a, "archive_classes") {
				argv = append(argv, "--archive-class", class)
			}
			if dry {
				argv = append(argv, "--dry-run")
			} else {
				argv = append(argv, "--apply-safe", apply)
			}
			return argv, true, nil
		},
	},
	{
		name: "diagnose_pr",
		desc: prompts.MCPDesc("diagnose_pr"),
		schema: obj([]string{"task"}, map[string]any{
			"task": str("task reference whose canonical head and pull request should be diagnosed"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "task"); err != nil {
				return nil, false, err
			}
			return []string{"pr", "diagnose", "--task", s(a, "task")}, true, nil
		},
	},
	{
		name: "release_train",
		desc: prompts.MCPDesc("release_train"),
		schema: obj([]string{"project", "source", "target"}, map[string]any{
			"project":          str("project with the exact configured GitHub repository"),
			"source":           str("exact integration branch to promote"),
			"target":           str("exact protected target branch"),
			"dry_run":          boolp("observe exact SHAs and render notes without mutation"),
			"apply":            boolp("create or resume the durable canonical promotion PR"),
			"required_checks":  strs("required GitHub check names"),
			"required_reviews": num("minimum approving reviews"),
			"merge":            boolp("request merge; also needs recorded project release_merge_authority"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "project", "source", "target"); err != nil {
				return nil, false, err
			}
			if b(a, "dry_run") == b(a, "apply") {
				return nil, false, fmt.Errorf("choose exactly one of dry_run or apply")
			}
			argv := []string{"release", "train", "--project", s(a, "project"), "--source", s(a, "source"), "--target", s(a, "target")}
			if b(a, "dry_run") {
				argv = append(argv, "--dry-run")
			} else {
				argv = append(argv, "--apply")
			}
			for _, check := range list(a, "required_checks") {
				argv = append(argv, "--required-check", check)
			}
			if n := i(a, "required_reviews"); n > 0 {
				argv = append(argv, "--required-reviews", strconv.Itoa(n))
			}
			if b(a, "merge") {
				argv = append(argv, "--merge")
			}
			return argv, true, nil
		},
	},
	{
		name: "github_projection",
		desc: prompts.MCPDesc("github_projection"),
		schema: obj([]string{"project"}, map[string]any{
			"project":          str("linked project whose outbound GitHub policy should be inspected"),
			"include_internal": boolp("request internal evidence; the result shows whether separate recorded authority permits it"),
			"terminal":         boolp("model an explicitly terminal delivery that may close its mapped issue"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			if err := need(a, "project"); err != nil {
				return nil, false, err
			}
			argv := []string{"github", "projection", s(a, "project")}
			if b(a, "include_internal") {
				argv = append(argv, "--include-internal")
			}
			if b(a, "terminal") {
				argv = append(argv, "--terminal")
			}
			return argv, true, nil
		},
	},
	{
		name: "cli",
		desc: prompts.MCPDesc("cli"),
		schema: obj([]string{"argv"}, map[string]any{
			"argv": strs("command tokens, e.g. [\"risk\",\"add\",\"title\",\"--project\",\"p\",...]"),
			"json": boolp("request --json output where supported"),
		}),
		build: func(a map[string]any) ([]string, bool, error) {
			argv := list(a, "argv")
			if len(argv) == 0 {
				return nil, false, fmt.Errorf("argv must be a non-empty string array")
			}
			return argv, b(a, "json"), nil
		},
	},
}

// refCmd builds "<verb> <sub> <ref>" tools.
func refCmd(verb, sub string) func(map[string]any) ([]string, bool, error) {
	return func(a map[string]any) ([]string, bool, error) {
		if err := need(a, "ref"); err != nil {
			return nil, false, err
		}
		return []string{verb, sub, s(a, "ref")}, false, nil
	}
}

// validateArgs refuses an argument whose value cannot mean what the tool's
// schema says it is.
//
// The coercers below (i, b, list) are total: they return the zero value for
// anything they cannot read. That is the right shape for a helper and the
// wrong shape for a boundary — {"dry_run": "yes please"} coerced to false, and
// the caller who asked for a rehearsal got a real mutation. This is the check
// that turns those into a refusal naming the argument (dacli 361).
//
// Only DECLARED properties are checked, and only for interpretability. An
// argument the schema does not mention is left alone: MCP clients add fields,
// and refusing them here would be a different decision from the one this task
// asked for.
func validateArgs(t tool, args map[string]any) error {
	props, _ := t.schema["properties"].(map[string]any)
	if props == nil {
		return nil
	}
	rawVersion, present := args["schema_version"]
	if !present || rawVersion == nil {
		return fmt.Errorf("missing required argument %q", "schema_version")
	}
	if present {
		if ok, _ := interpretable("integer", rawVersion); !ok {
			return fmt.Errorf("argument %q must be integer, got %T (%v)", "schema_version", rawVersion, rawVersion)
		}
		if got := i(args, "schema_version"); got != toolSchemaVersion {
			return fmt.Errorf("unsupported schema_version %d for tool %q (supported: %d)", got, t.name, toolSchemaVersion)
		}
	}
	for key, raw := range args {
		spec, _ := props[key].(map[string]any)
		if spec == nil {
			continue // undeclared: not this check's business
		}
		want, _ := spec["type"].(string)
		if raw == nil {
			continue // absent-by-null is the same as absent
		}
		if ok, hint := interpretable(want, raw); !ok {
			return fmt.Errorf("argument %q must be %s, got %T (%v)%s — dacli refuses rather than reading it as the zero value, which would silently change what you asked for",
				key, want, raw, raw, hint)
		}
	}
	return nil
}

// interpretable reports whether raw can be read as the declared type, using
// exactly the rules the coercers use — so validation and coercion can never
// disagree about what is acceptable.
func interpretable(want string, raw any) (bool, string) {
	switch want {
	case "boolean":
		switch v := raw.(type) {
		case bool:
			return true, ""
		case string:
			if _, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
				return true, ""
			}
			return false, ` — accepted spellings are true/false, 1/0`
		}
		return false, ` — accepted spellings are true/false, 1/0`
	case "integer", "number":
		switch v := raw.(type) {
		case float64, int, int64:
			return true, ""
		case string:
			if _, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				return true, ""
			}
			return false, ""
		}
		return false, ""
	case "array":
		_, ok := raw.([]any)
		return ok, ""
	case "string":
		_, ok := raw.(string)
		return ok, ""
	}
	return true, "" // a type this validator does not model is not its to refuse
}
