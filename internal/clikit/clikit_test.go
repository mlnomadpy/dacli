package clikit

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestDescribeErrorExposesWrappedExternalDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	cmd := exec.Command("/bin/sh", "-c", "printf 'auth failed decisively\\n' >&2; exit 7")
	cmd.Dir = t.TempDir()
	_, commandErr := commandresult.Run(cmd, commandresult.RunOptions{Operation: "gh auth", WorkspaceRoot: cmd.Dir})
	details := DescribeError(fmt.Errorf("sync failed: %w", commandErr))
	if details.ExitCode != 1 || details.Diagnostic == nil {
		t.Fatalf("details = %#v, want generic CLI exit plus typed diagnostic", details)
	}
	if details.Diagnostic.ExitCode == nil || *details.Diagnostic.ExitCode != 7 || details.Diagnostic.StderrTail != "auth failed decisively" {
		t.Fatalf("wrapped diagnostic was collapsed: %#v", details.Diagnostic)
	}
}

func TestDescribeErrorExposesStructuredReviewValidation(t *testing.T) {
	expected := store.IndependentReviewResult{Schema: store.ReviewResultSchema, ReviewerID: "a-reviewer", ReviewerRole: "reviewer", Runtime: "codex", Model: "gpt", Grant: "ro", CommitSHA: "expected-commit", TreeSHA: "expected-tree"}
	actual := expected
	actual.Model = "other"
	err := store.NewReviewValidationError(expected, actual, []string{"model"}, errors.New("mismatch"))
	details := DescribeError(fmt.Errorf("review failed: %w", err))
	if details.ReviewValidation == nil || details.ReviewValidation.Schema != store.ReviewValidationSchema || details.ReviewValidation.Expected.Model != "gpt" || details.ReviewValidation.Actual.Model != "other" {
		t.Fatalf("review validation details = %+v", details)
	}
}

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

// dacli 338: ExitCode uses errors.As to find the exitErr inside a wrapped
// chain. A refusal that picks up context on its way up the call stack — the
// normal way this codebase's %w-wrapping accumulates context (errorlint now
// enforces it) — must still resolve to exit 3, never fall through to the
// generic exit 1 a caller would retry.
func TestExitCodeSurvivesWrapping(t *testing.T) {
	refusal := Refusedf("policy said no")
	cases := []struct {
		name string
		err  error
	}{
		{"unwrapped", refusal},
		{"wrapped once", fmt.Errorf("running the task: %w", refusal)},
		{"wrapped twice", fmt.Errorf("outer: %w", fmt.Errorf("running the task: %w", refusal))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != 3 {
				t.Errorf("ExitCode(%v) = %d, want 3 (refused, never retried)", tc.err, got)
			}
		})
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

// FirstLine is the single one-line-summary contract that replaced three
// diverged copies (dacli 242). The load-bearing case is the leading-newline
// body: the orchestration copy did no trimming, so a refusal reason that
// began with '\n' was logged BLANK — the exact defect this task exists to fix.
func TestFirstLine(t *testing.T) {
	cases := []struct{ in, want string }{
		{"one\ntwo\nthree", "one"},
		{"  solo  ", "solo"},
		{"\nreal reason here\nmore", "real reason here"}, // regression: was "" before
		{"   \n  refused: policy\n", "refused: policy"},  // leading blank + whitespace
		{"trailing spaces   \nnext", "trailing spaces"},
		{"", ""},
		{"   \n   \n", ""},
	}
	for _, c := range cases {
		if got := FirstLine(c.in); got != c.want {
			t.Errorf("FirstLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The dacli 243 regression: an integer flag whose value fails to parse must
// fail loudly (exit 2) instead of the discarded parse error letting the
// default silently stand. `spawn --timeout 30s` must refuse "30s", not run
// unbounded on the 300s default.
func TestFlagsInt(t *testing.T) {
	// Absent flag returns the caller's default.
	f, _ := ParseFlags([]string{"--other", "x"})
	if n, err := f.Int("timeout", 300); err != nil || n != 300 {
		t.Errorf("Int(absent) = %d, %v; want 300, nil", n, err)
	}

	// A valid value parses.
	f, _ = ParseFlags([]string{"--timeout", "45"})
	if n, err := f.Int("timeout", 300); err != nil || n != 45 {
		t.Errorf("Int(\"45\") = %d, %v; want 45, nil", n, err)
	}

	// Garbage is a usage error (exit 2), and the returned int stays the
	// default rather than a silently-zeroed value.
	f, _ = ParseFlags([]string{"--timeout", "30s"})
	n, err := f.Int("timeout", 300)
	if err == nil {
		t.Fatal("Int(\"30s\") = nil error; want a usage refusal")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", ExitCode(err))
	}
	if n != 300 {
		t.Errorf("Int(\"30s\") value = %d, want the default 300 (never a silent 0)", n)
	}
	if !bytes.Contains([]byte(err.Error()), []byte("timeout")) {
		t.Errorf("error %q should name the offending flag", err.Error())
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

// A budget flag that does not parse must REFUSE, never fall back to the
// default. The helpers these readers replaced returned the default on error,
// so `--window-tokens garbage` became 0 — which the ceiling check reads as
// "unlimited" — and an operator who explicitly asked to be capped ran with no
// cap at all. `50k` is the other half: Sscanf stopped at the 'k', reported no
// error, and silently produced a 50-token ceiling.
func TestNumericFlagsRefuseGarbageInsteadOfDefaulting(t *testing.T) {
	for _, tc := range []struct{ flag, val string }{
		{"window-tokens", "garbage"},
		{"window-tokens", "50k"},
		{"width", "two"},
	} {
		f, err := ParseFlags([]string{"--" + tc.flag, tc.val})
		if err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		var perr error
		if tc.flag == "width" {
			_, perr = f.Int(tc.flag, 2)
		} else {
			_, perr = f.Int64(tc.flag, 0)
		}
		if perr == nil {
			t.Errorf("--%s %q was accepted; a bound the caller asked for must not silently become the default", tc.flag, tc.val)
			continue
		}
		if ExitCode(perr) != 2 {
			t.Errorf("--%s %q: exit %d, want 2 (usage)", tc.flag, tc.val, ExitCode(perr))
		}
	}

	// Absent flags still yield the default, with no error.
	f, _ := ParseFlags(nil)
	if n, err := f.Int64("window-tokens", 7); err != nil || n != 7 {
		t.Errorf("absent flag = %d, %v; want the default and no error", n, err)
	}
}

// Duration accepts both spellings already present in the CLI (Go duration and
// bare seconds) and refuses anything else — `--idle 5min` is not a Go unit and
// used to become the 30m default in silence.
func TestDurationAcceptsBothFormsAndRefusesGarbage(t *testing.T) {
	f, _ := ParseFlags([]string{"--idle", "30m", "--timeout", "90", "--budget-window", "5min"})
	if d, err := f.Duration("idle", time.Hour); err != nil || d != 30*time.Minute {
		t.Errorf("--idle 30m = %v, %v; want 30m", d, err)
	}
	if d, err := f.Duration("timeout", time.Hour); err != nil || d != 90*time.Second {
		t.Errorf("--timeout 90 = %v, %v; want 90s (bare integer means seconds)", d, err)
	}
	d, err := f.Duration("budget-window", 24*time.Hour)
	if err == nil {
		t.Fatalf("--budget-window 5min = %v with no error; an unparseable duration must be refused", d)
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit %d, want 2 (usage)", ExitCode(err))
	}
}

// A bool flag followed by a positional must not eat it. `github push p
// --dry-run 001 002` used to parse dry-run="001", which Bool read as FALSE —
// so the preview flag silently turned itself off and the command wrote to the
// real remote, while task 001 also vanished from the push window.
func TestBoolFlagDoesNotSwallowAFollowingPositional(t *testing.T) {
	f, err := ParseFlags([]string{"p", "--dry-run", "001", "002"})
	if err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if !f.Bool("dry-run") {
		t.Error("--dry-run followed by a positional must still read as SET — anything else turns a rehearsal into a real write")
	}
	want := []string{"p", "001", "002"}
	if len(f.Pos) != len(want) {
		t.Fatalf("positionals = %v, want %v (the swallowed token must stay in the window)", f.Pos, want)
	}
	for i := range want {
		if f.Pos[i] != want[i] {
			t.Errorf("positionals = %v, want %v", f.Pos, want)
			break
		}
	}

	// The explicit off-switch still works, so a scripted --dry-run=false is
	// not silently forced on.
	off, _ := ParseFlags([]string{"--dry-run=false"})
	if off.Bool("dry-run") {
		t.Error("--dry-run=false must read as unset")
	}
	// And a value-taking flag is unaffected.
	v, _ := ParseFlags([]string{"--project", "core"})
	if v.Get("project") != "core" {
		t.Errorf("value flag = %q, want core", v.Get("project"))
	}
}

// One concept, one canonical name, old spellings still accepted. Before this,
// "budget" meant the BRIEF's size in one command and a time period in another
// (--budget-window), so an agent could not predict a flag it had not used from
// the ones it had (task 292).
func TestAliasPrefersCanonicalAndAcceptsOldSpelling(t *testing.T) {
	// The old spelling still works.
	old, _ := ParseFlags([]string{"--budget", "4000"})
	got, err := old.IntAliased(0, "brief-tokens", "budget")
	if err != nil || got != 4000 {
		t.Errorf("old spelling = %d, %v; want 4000 and no error", got, err)
	}

	// The canonical name works.
	nw, _ := ParseFlags([]string{"--brief-tokens", "4000"})
	if got, err := nw.IntAliased(0, "brief-tokens", "budget"); err != nil || got != 4000 {
		t.Errorf("canonical = %d, %v; want 4000", got, err)
	}

	// Canonical wins when both are passed, so the rename is never ambiguous.
	both, _ := ParseFlags([]string{"--budget", "1", "--brief-tokens", "2"})
	if got, _ := both.IntAliased(0, "brief-tokens", "budget"); got != 2 {
		t.Errorf("with both spellings = %d, want 2 (canonical wins)", got)
	}

	// Absent means the default, and garbage is still a usage error under an
	// alias — the rename must not create a hole in the refusing readers.
	none, _ := ParseFlags(nil)
	if got, err := none.IntAliased(7, "brief-tokens", "budget"); err != nil || got != 7 {
		t.Errorf("absent = %d, %v; want the default", got, err)
	}
	bad, _ := ParseFlags([]string{"--budget", "lots"})
	if _, err := bad.IntAliased(0, "brief-tokens", "budget"); err == nil {
		t.Error("garbage under an alias must still be a usage error")
	}
}
