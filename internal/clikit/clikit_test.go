package clikit

import (
	"bytes"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/model"
)

// RequireRW is the single capability gate the privileged subcommands share
// (dacli 162). A read-only identity must be refused with exit 3 — never a
// retryable error — and a read-write identity must pass.
func TestRequireRW(t *testing.T) {
	ro := &agentid.Identity{ID: "a-child", Grant: model.GrantRO, Role: "junior"}
	if err := RequireRW(ro, "defining a shortcut"); err == nil {
		t.Fatal("RequireRW(ro) = nil; want refusal")
	} else if ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refused, never retried)", ExitCode(err))
	}

	rw := &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
	if err := RequireRW(rw, "defining a shortcut"); err != nil {
		t.Errorf("RequireRW(rw) = %v; want nil", err)
	}
}

// The run 01KY2K8N4C regression: a runtime adapter's value flag whose value
// itself looks like a flag (--sandbox-ro-arg --allowedTools) must not be
// silently swallowed as a bare boolean.
func TestParseFlagsValueFlagCapturesDashLeadingValue(t *testing.T) {
	f, err := ParseFlags([]string{"--sandbox-ro-arg", "--allowedTools", "--env", "PATH"}, "sandbox-ro-arg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := f.Get("sandbox-ro-arg"); got != "--allowedTools" {
		t.Errorf("sandbox-ro-arg = %q, want --allowedTools", got)
	}
	if got := f.Get("env"); got != "PATH" {
		t.Errorf("env = %q, want PATH", got)
	}
}

func TestParseFlagsValueFlagRepeatable(t *testing.T) {
	f, err := ParseFlags([]string{"--arg", "-p", "--arg", "--model", "x"}, "arg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := f.All("arg")
	want := []string{"-p", "--model"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("arg = %v, want %v", got, want)
	}
	if len(f.Pos) != 1 || f.Pos[0] != "x" {
		t.Errorf("Pos = %v, want [x]", f.Pos)
	}
}

func TestParseFlagsValueFlagMissingValueErrors(t *testing.T) {
	f, err := ParseFlags([]string{"--sandbox-ro-arg"}, "sandbox-ro-arg")
	if err == nil {
		t.Fatal("expected an error for a value-flag with no following value")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", ExitCode(err))
	}
	if f == nil {
		t.Fatal("ParseFlags must still return a non-nil *Flags on error")
	}
}

// The -- terminator: any flag, whitelisted or not, can force a literal
// dash-leading value without the caller pre-declaring it.
func TestParseFlagsDoubleDashTerminatorForcesLiteralValue(t *testing.T) {
	f, err := ParseFlags([]string{"--model-flag", "--", "--model"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := f.Get("model-flag"); got != "--model" {
		t.Errorf("model-flag = %q, want --model", got)
	}
}

// The = form keeps working unchanged.
func TestParseFlagsEqualsFormCapturesDashLeadingValue(t *testing.T) {
	f, err := ParseFlags([]string{"--model-flag=--model"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := f.Get("model-flag"); got != "--model" {
		t.Errorf("model-flag = %q, want --model", got)
	}
}

// Two adjacent bare boolean flags (neither declared as a value flag) must
// keep working — this is the ambiguity a schema-free parser cannot resolve
// on its own, and plenty of real commands rely on it (e.g. --cooperative
// --review).
func TestParseFlagsAdjacentBareBooleansUnaffected(t *testing.T) {
	f, err := ParseFlags([]string{"--cooperative", "--review"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Bool("cooperative") || !f.Bool("review") {
		t.Errorf("cooperative=%v review=%v, want both true", f.Bool("cooperative"), f.Bool("review"))
	}
}

// The dacli 143 regression: a typo'd or unsupported flag must fail loudly
// (exit 2) instead of ParseFlags silently dropping it with no error.
func TestFlagsRejectUnknownFlag(t *testing.T) {
	f, err := ParseFlags([]string{"--project", "core", "--acccept", "y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = f.Reject("project", "accept")
	if err == nil {
		t.Fatal("expected an error for the unknown --acccept flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", ExitCode(err))
	}
	if !bytes.Contains([]byte(err.Error()), []byte("acccept")) {
		t.Errorf("error %q should name the offending flag", err.Error())
	}
}

func TestFlagsRejectKnownSetPasses(t *testing.T) {
	f, err := ParseFlags([]string{"--project", "core", "--accept", "y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := f.Reject("project", "accept", "force"); err != nil {
		t.Errorf("unexpected error for an all-known flag set: %v", err)
	}
}

// A *bytes.Buffer is what every test harness and the MCP executor write to —
// neither is a terminal, so color must stay off regardless of NO_COLOR or
// any other setting. This is the load-bearing property: it is what keeps
// agent-facing and test output byte-identical to before color existed.
func TestNewPaletteOffForNonFileWriter(t *testing.T) {
	var buf bytes.Buffer
	pal := NewPalette(&Ctx{Stdout: &buf})
	if pal.Enabled() {
		t.Fatal("palette should be off for a non-*os.File Stdout")
	}
	if got := pal.Red("x"); got != "x" {
		t.Errorf("Red(%q) = %q, want unchanged (color off)", "x", got)
	}
}

// --json must never carry color, even if Stdout were somehow a terminal —
// machine consumers get plain bytes, no exceptions.
func TestNewPaletteOffForJSON(t *testing.T) {
	var buf bytes.Buffer
	pal := NewPalette(&Ctx{Stdout: &buf, JSON: true})
	if pal.Enabled() {
		t.Fatal("palette should be off in JSON mode")
	}
}

// Paint helpers are no-ops on an empty string — an empty colored field must
// not become two invisible-but-present escape sequences.
func TestPaletteOnLeavesEmptyStringEmpty(t *testing.T) {
	pal := Palette{}
	if got := pal.Bold(""); got != "" {
		t.Errorf("Bold(\"\") = %q, want empty", got)
	}
}
