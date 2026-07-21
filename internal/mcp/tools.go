package mcp

import (
	"fmt"
	"strconv"

	"github.com/mlnomadpy/dacli/internal/prompts"
)

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
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
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
func i(args map[string]any, k string) int {
	switch v := args[k].(type) {
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}
func b(args map[string]any, k string) bool {
	v, _ := args[k].(bool)
	return v
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
