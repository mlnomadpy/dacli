// Package clikit is the shared kernel of the feature-sliced app layer: the
// command type, flag parsing, the exit-code contract, and workspace/identity
// resolution. Every feature slice imports this; no feature slice imports
// another — that isolation is the point of the slicing, and it is enforced
// by a test (internal/cli/arch_test.go).
package clikit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Ctx carries the process-wide state a command needs: where to write, and
// whether the caller wants machine-readable output.
type Ctx struct {
	Stdout io.Writer
	Stderr io.Writer
	Cwd    string
	JSON   bool
}

// Command is one subcommand. Path is the space-separated invocation, e.g.
// "task add". Feature slices export a Commands slice; the app layer
// aggregates them into one table.
type Command struct {
	Path  string
	Brief string
	Run   func(ctx *Ctx, args []string) error
}

// --- The exit-code contract (ARCHITECTURE § 4). The 1/3 distinction is the
// one that matters: retrying a refusal is the loop a supervisor must never
// enter.

type exitErr struct {
	code int
	msg  string
}

func (e exitErr) Error() string { return e.msg }

// Usagef is exit 2: the caller's mistake.
func Usagef(format string, a ...any) error { return exitErr{2, fmt.Sprintf(format, a...)} }

// Refusedf is exit 3: policy said no. An answer, never retried.
func Refusedf(format string, a ...any) error { return exitErr{3, fmt.Sprintf(format, a...)} }

// ExitCode maps an error onto the contract.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee exitErr
	if errors.As(err, &ee) {
		return ee.code
	}
	var nf store.ErrNotFound
	if errors.As(err, &nf) {
		return 4
	}
	if errors.Is(err, workspace.ErrNotFound) {
		return 4
	}
	return 1
}

// Planned returns an honest stub: what the command is waiting on and where
// the design lives. "not implemented — see DESIGN.md" told nobody anything.
func Planned(what, doc string) func(*Ctx, []string) error {
	return func(ctx *Ctx, args []string) error {
		return fmt.Errorf("not built yet: %s. The design is in %s — implementation lands with that subsystem", what, doc)
	}
}

// --- Flags: --key value, --key=value, repeatable keys, positionals.
//
// A parser with no per-flag schema fundamentally cannot tell "--key --other"
// (bool key, then a separate bool flag other) from "--key --other" (key's
// value happens to start with --) — Go's own flag package has the same gap
// and resolves it the same way. Two escapes make a dash-leading value
// unambiguous without requiring a schema:
//   - the = form: --key=--value
//   - the -- terminator: --key -- --value (the literal "--" token forces
//     the token after it to be taken as key's value verbatim)
//
// A caller that knows some of its own flags are never boolean (they always
// take a value, e.g. runtime add's --arg/--sandbox-ro-arg/--model-flag) can
// name them via valueFlags so the space form works directly for those keys
// without either escape — see cmdRuntimeAdd. Silently defaulting such a key
// to "true" is exactly the corruption filed against run 01KY2K8N4C.

type Flags struct {
	Pos  []string
	vals map[string][]string
}

func ParseFlags(args []string, valueFlags ...string) (*Flags, error) {
	valueOnly := make(map[string]bool, len(valueFlags))
	for _, k := range valueFlags {
		valueOnly[k] = true
	}
	f := &Flags{vals: map[string][]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			f.Pos = append(f.Pos, a)
			continue
		}
		key := a[2:]
		if eq := strings.Index(key, "="); eq >= 0 {
			f.vals[key[:eq]] = append(f.vals[key[:eq]], key[eq+1:])
			continue
		}
		if i+2 < len(args) && args[i+1] == "--" {
			i += 2
			f.vals[key] = append(f.vals[key], args[i])
			continue
		}
		if valueOnly[key] {
			if i+1 >= len(args) {
				return f, Usagef("--%s requires a value (use --%s=VALUE or --%s -- VALUE)", key, key, key)
			}
			i++
			f.vals[key] = append(f.vals[key], args[i])
			continue
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			f.vals[key] = append(f.vals[key], "true") // bare flag
			continue
		}
		i++
		f.vals[key] = append(f.vals[key], args[i])
	}
	return f, nil
}

func (f *Flags) Get(k string) string {
	if v := f.vals[k]; len(v) > 0 {
		return v[len(v)-1]
	}
	return ""
}
func (f *Flags) All(k string) []string { return f.vals[k] }
func (f *Flags) Bool(k string) bool    { return f.Get(k) == "true" }

// Raw exposes every parsed flag, for commands (like `run`) that forward
// unknown flags as parameters.
func (f *Flags) Raw() map[string][]string { return f.vals }

// --- Shared plumbing ---

// OpenWorkspace resolves the workspace from cwd and the acting identity from
// the environment.
func OpenWorkspace(ctx *Ctx) (*workspace.Workspace, *agentid.Identity, error) {
	w, err := workspace.Find(ctx.Cwd)
	if err != nil {
		return nil, nil, err
	}
	id, err := agentid.Resolve(w)
	if err != nil {
		return nil, nil, err
	}
	return w, id, nil
}

// EmitJSON writes v as indented JSON to stdout.
func EmitJSON(ctx *Ctx, v any) error {
	enc := json.NewEncoder(ctx.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// OrDash returns s, or a default, or "-".
func OrDash(s string, def ...string) string {
	if s != "" {
		return s
	}
	if len(def) > 0 {
		return def[0]
	}
	return "-"
}

// ErrStr renders an error for run records; nil is "0" (a clean exit).
func ErrStr(err error) string {
	if err == nil {
		return "0"
	}
	return err.Error()
}
