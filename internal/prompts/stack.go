package prompts

// Stack awareness (dacli 192).
//
// dacli builds products in any language, but the default path was Go-only: the
// git-discipline prompt ordered EVERY read-write child to `gofmt -w` its `.go`
// files, and the loop handed every project a `go-auditor`. Both were injected
// regardless of what the project is written in — confirmed live on a Python
// recipe app, which was reviewed by go-auditor and told to gofmt files it does
// not have.
//
// The fix consumes what `dacli new` already records. That command resolves a
// stack profile and writes it into two project sections, in prose a human reads
// and a machine can parse back:
//
//	## Constraints
//	Stack: Python. Build with `python -m build`; test with `pytest`. …
//
//	## Architecture
//	**Stack:** Python — scaffold with `…`, build with `…`, test with `…`.
//
// So this file parses, it does not detect. No manifest sniffing, no second
// source of truth: if the project document says nothing, the answer is "no
// recorded stack" and every caller falls back to today's behavior verbatim.
// That fallback is the whole backwards-compatibility contract — every project
// that predates `dacli new` has no Stack: line, and dacli's own workspace is
// one of them.

import (
	"regexp"
	"strings"

	"github.com/mlnomadpy/dacli/internal/mdstore"
)

// Stack is a project's recorded toolchain: the label a prompt names and the
// commands an agent actually runs. A zero Stack means "nothing recorded" and
// every field-conditional in a template must treat it as today's default, not
// as an empty stack.
type Stack struct {
	// Key is the lower-cased stack identifier (`go`, `python`, …), derived
	// from Label. Empty when no stack is recorded.
	Key string
	// Label is the human name as written in the project doc ("Python").
	Label string
	// Build, Test are the commands the project's Success criteria bind to.
	Build, Test string
	// Format is the formatter invocation. `dacli new` does not record one
	// (it records scaffold/build/test), so for a known Key it comes from
	// formatCmds below; a project doc may state its own with a
	// "format with `…`" clause, which always wins.
	Format string
}

// Recorded reports whether the project document actually named a stack. The
// negative case is load-bearing: it is what keeps a pre-192 workspace on the
// exact prompt and role defaults it had before this change.
func (s Stack) Recorded() bool { return s.Label != "" }

// IsGo reports whether the recorded stack is Go — used to decide whether the
// Go-specific advice a template used to hardcode is still the right advice.
func (s Stack) IsGo() bool { return s.Key == "go" }

// formatCmds is the formatter per stack key. It is a small table rather than a
// third section in the project doc because a formatter is a property of the
// toolchain, not of the product — and because inventing a project section that
// `dacli new` does not write would put two sources of truth in the repo. A key
// missing here yields an empty Format, and a template that says nothing about
// formatting is strictly better than one that names the wrong tool.
var formatCmds = map[string]string{
	"go":         "gofmt -w",
	"python":     "ruff format",
	"typescript": "npx prettier --write",
	"rust":       "cargo fmt",
}

// stackSections are the project sections that can carry the record, in
// precedence order. Constraints is first because it is the section the brief
// assembler already carries into every brief, which is exactly why `dacli new`
// puts the build/test pair there.
var stackSections = []string{"Constraints", "Architecture"}

// labelRe finds the recorded stack name in either of the two shapes `dacli new`
// writes: a bare "Stack: Go." line in Constraints, or a bolded
// "**Stack:** Go — …" lead-in in Architecture. The name runs until the first
// sentence-ending period, em dash, or comma, so "Stack: Go. Build with…" yields
// "Go" and not the rest of the paragraph.
var labelRe = regexp.MustCompile(`(?i)\*{0,2}Stack:?\*{0,2}[:\s]\s*([^.\n—,;]+)`)

// cmdRe extracts a `backticked` command introduced by a "<verb> with" clause,
// which is the one phrasing every seeded body uses ("Build with `go build ./...`",
// "scaffold with `cargo init`"). Anchoring on the verb rather than on position
// means a planner agent may rewrite the prose around it without breaking the
// parse, as long as the clause survives.
func cmdRe(verb string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b` + verb + `\s+with\s+` + "`" + `([^` + "`" + `]+)` + "`")
}

var (
	buildRe  = cmdRe("build")
	testRe   = cmdRe("test")
	formatRe = cmdRe("format")
)

// StackFromProject reads a project document's recorded stack. It never errors:
// an unparseable or absent record is indistinguishable from "no stack was ever
// recorded", and both must land on the same conservative fallback.
func StackFromProject(doc *mdstore.Doc) Stack {
	if doc == nil {
		return Stack{}
	}
	var body strings.Builder
	for _, name := range stackSections {
		if s, ok := doc.Section(name); ok {
			body.WriteString(s.Content)
			body.WriteString("\n")
		}
	}
	return ParseStack(body.String())
}

// ParseStack pulls a Stack out of raw project prose. Exported separately from
// StackFromProject so the parse is testable against a literal string and so a
// caller holding text rather than a Doc need not fabricate one.
func ParseStack(body string) Stack {
	var s Stack
	if m := labelRe.FindStringSubmatch(body); m != nil {
		s.Label = strings.TrimSpace(m[1])
		s.Key = strings.ToLower(s.Label)
	}
	if s.Label == "" {
		// No label means no record. Returning a zero Stack even when a stray
		// "build with `make`" appears keeps Recorded() honest: half a record
		// is not a stack, and a template branching on it would be guessing.
		return Stack{}
	}
	if m := buildRe.FindStringSubmatch(body); m != nil {
		s.Build = strings.TrimSpace(m[1])
	}
	if m := testRe.FindStringSubmatch(body); m != nil {
		s.Test = strings.TrimSpace(m[1])
	}
	if m := formatRe.FindStringSubmatch(body); m != nil {
		s.Format = strings.TrimSpace(m[1])
	} else {
		s.Format = formatCmds[s.Key]
	}
	return s
}

// RoleFor picks the roster role for a phase (`auditor`, `fixer`) given the
// recorded stack, falling back to def — today's hardcoded default — whenever
// the stack is unrecorded, is Go, or names no role the workspace actually has.
//
// exists is the roster lookup, injected so this stays a pure decision the
// prompts package can hold without importing store. Choosing from the roster
// rather than inventing a name is the point: spawning `--role python-auditor`
// into a workspace with no such role would fail the spawn, which is a worse
// outcome than reviewing Python with the generic default.
func RoleFor(s Stack, phase, def string, exists func(string) bool) string {
	if !s.Recorded() || s.IsGo() || exists == nil {
		return def
	}
	for _, cand := range []string{s.Key + "-" + phase, phase} {
		if cand != def && exists(cand) {
			return cand
		}
	}
	return def
}
