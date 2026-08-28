package execution

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

// newCtx returns a Ctx whose streams are inspectable.
func newCtx(cwd string) (*clikit.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd}, &out, &errb
}

// --max-rss takes an operator-friendly size. A misparse here silently changes
// which agents get reaped, so every suffix (and the bare-byte and garbage
// cases) is pinned.
func TestParseBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"2G", 2 << 30},
		{"2g", 2 << 30},
		{"500M", 500 << 20},
		{"500m", 500 << 20},
		{"1024K", 1024 << 10},
		{"1024k", 1024 << 10},
		{"4096", 4096},
		{" 1G ", 1 << 30},
		{"1.5G", 1610612736}, // fractional sizes are honoured, not truncated to 1G
		{"garbage", 0},
	}
	for _, tc := range cases {
		if got := parseBytes(tc.in); got != tc.want {
			t.Errorf("parseBytes(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// --max-runtime accepts a Go duration or a bare seconds count. "15" must mean
// 15 SECONDS, not 15 nanoseconds — the raw time.Duration conversion would reap
// every agent instantly.
func TestParseDurationArg(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"15m", 15 * time.Minute},
		{"2h", 2 * time.Hour},
		{"90s", 90 * time.Second},
		{"900", 900 * time.Second},
		{"15", 15 * time.Second},
		{"not-a-duration", 0},
	}
	for _, tc := range cases {
		if got := parseDurationArg(tc.in); got != tc.want {
			t.Errorf("parseDurationArg(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestHumanKBAndBytes(t *testing.T) {
	cases := []struct {
		kb   int
		want string
	}{
		{0, "0MiB"},
		{2048, "2MiB"},
		{1024 * 1024, "1.0GiB"},
		{3 * 1024 * 1024, "3.0GiB"},
	}
	for _, tc := range cases {
		if got := humanKB(tc.kb); got != tc.want {
			t.Errorf("humanKB(%d) = %q, want %q", tc.kb, got, tc.want)
		}
	}
	if got := humanBytes(2 << 30); got != "2.0GiB" {
		t.Errorf("humanBytes(2GiB) = %q, want 2.0GiB", got)
	}
}

// GPU memory that cannot be measured (no nvidia-smi) must read n/a, never 0 —
// a fabricated zero would tell an operator the agent holds no GPU when the
// truth is that we cannot see.
func TestGPUStrReportsUnmeasurableAsNA(t *testing.T) {
	if got := gpuStr(-1); got != "n/a" {
		t.Errorf("gpuStr(-1) = %q, want n/a (never a fabricated 0)", got)
	}
	if got := gpuStr(0); got != "0MiB" {
		t.Errorf("gpuStr(0) = %q, want 0MiB (a real measured zero)", got)
	}
	if got := gpuStr(4096); got != "4096MiB" {
		t.Errorf("gpuStr(4096) = %q", got)
	}
}

// truncateLine counts RUNES, not bytes: cutting a multi-byte transcript line
// mid-rune would emit replacement garbage into `agents --tail`.
func TestTruncateLineIsRuneSafe(t *testing.T) {
	if got := truncateLine("short", 10); got != "short" {
		t.Errorf("under the cap must pass through unchanged, got %q", got)
	}
	if got := truncateLine("abcdef", 3); got != "abc…" {
		t.Errorf("truncateLine(abcdef,3) = %q, want abc…", got)
	}
	multi := "ⵟⵟⵟⵟⵟ"
	got := truncateLine(multi, 3)
	if want := "ⵟⵟⵟ…"; got != want {
		t.Errorf("truncateLine on multi-byte runes = %q, want %q", got, want)
	}
	if strings.ContainsRune(got, '�') {
		t.Errorf("truncation split a rune: %q", got)
	}
	// Exactly at the cap: no ellipsis.
	if got := truncateLine("abc", 3); got != "abc" {
		t.Errorf("at the cap = %q, want abc (no ellipsis)", got)
	}
}

func TestLastLines(t *testing.T) {
	body := []byte("one\ntwo\nthree\nfour\n")
	if got := string(lastLines(body, 2)); got != "three\nfour\n" {
		t.Errorf("lastLines(2) = %q", got)
	}
	// Asking for more lines than exist returns everything, not an empty slice.
	if got := string(lastLines(body, 99)); got != string(body) {
		t.Errorf("lastLines(99) = %q, want the whole buffer", got)
	}
	if got := string(lastLines([]byte("no trailing newline"), 1)); got != "no trailing newline" {
		t.Errorf("lastLines on an unterminated line = %q", got)
	}
}

// --claim is the disjointness declaration that keeps parallel agents merge-
// clean. Empty entries must be dropped, not turned into a claim on "" — which
// PathsOverlap would treat as a prefix of EVERY path and deadlock all spawns.
func TestSplitClaims(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"  ", nil},
		{",,", nil},
		{"internal/store", []string{"internal/store"}},
		{"a, b ,c", []string{"a", "b", "c"}},
		{"a,,b,", []string{"a", "b"}},
	}
	for _, tc := range cases {
		if got := splitClaims(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("splitClaims(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

func TestPercentile(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5}
	cases := []struct {
		p    float64
		want float64
	}{
		{0, 1},
		{50, 3},
		{100, 5},
		{25, 2},
		{10, 1.4},
	}
	for _, tc := range cases {
		if got := percentile(xs, tc.p); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("percentile(%v, %g) = %g, want %g", xs, tc.p, got, tc.want)
		}
	}
	if got := percentile(nil, 50); got != 0 {
		t.Errorf("percentile of an empty sample = %g, want 0", got)
	}
	if got := percentile([]float64{7}, 90); got != 7 {
		t.Errorf("percentile of a single sample = %g, want 7", got)
	}
	// The input slice must not be reordered under the caller: printAdvisory
	// reuses the same slice for p10, median and p90.
	unsorted := []float64{5, 1, 3}
	_ = percentile(unsorted, 50)
	if !reflect.DeepEqual(unsorted, []float64{5, 1, 3}) {
		t.Errorf("percentile mutated its input: %v", unsorted)
	}
}

// The § 8 rule: a read-only child needs a runtime that can ENFORCE read-only.
// Labelling an unrestricted process "ro" would be a lie, so it is refused
// (exit 3) unless --cooperative says so out loud.
func TestSandboxFor(t *testing.T) {
	enforcing := store.Runtime{Name: "claude-code", SandboxRO: []string{"--allowedTools", "Read"}, ROProbe: store.RuntimeROVerified}
	bare := store.Runtime{Name: "generic-exec"}
	// writeCap pins an allowlist that names a write tool; readOnly (junior's cc
	// shape) pins one that does not — its only allowlist is the ro sandbox.
	writeCap := store.Runtime{Name: "cc-rw", Args: []string{"--allowedTools", "Edit,Write,Read"}}
	readOnly := store.Runtime{Name: "cc", SandboxRO: []string{"--allowedTools", "Read,Grep"}}

	cases := []struct {
		name        string
		rt          store.Runtime
		grant       model.Grant
		cooperative bool
		wantArgs    []string
		wantExit    int
		wantWarn    bool
	}{
		{"rw needs no sandbox", bare, model.GrantRW, false, nil, 0, false},
		{"rw on a write-capable runtime is allowed", writeCap, model.GrantRW, false, nil, 0, false},
		{"rw on a runtime with no write tool is REFUSED", readOnly, model.GrantRW, false, nil, 3, false},
		{"rw on a no-write runtime with --cooperative is allowed", readOnly, model.GrantRW, true, nil, 0, false},
		{"ro on an enforcing runtime", enforcing, model.GrantRO, false, enforcing.SandboxRO, 0, false},
		{"ro on a declaration-only runtime is REFUSED", store.Runtime{Name: "declared", SandboxRO: []string{"--allowedTools", "Read"}, ROProbe: store.RuntimeROUnknown}, model.GrantRO, false, nil, 3, false},
		{"ro on a failed probe is REFUSED", store.Runtime{Name: "failed", SandboxRO: []string{"--allowedTools", "Read"}, ROProbe: store.RuntimeROFailed}, model.GrantRO, false, nil, 3, false},
		{"ro on a bare runtime is REFUSED", bare, model.GrantRO, false, nil, 3, false},
		{"ro on a bare runtime with --cooperative warns loudly", bare, model.GrantRO, true, nil, 0, true},
		{"ro on a declaration-only runtime with --cooperative applies declared args", store.Runtime{Name: "declared", SandboxRO: []string{"--allowedTools", "Read"}, ROProbe: store.RuntimeROUnknown}, model.GrantRO, true, []string{"--allowedTools", "Read"}, 0, true},
		{"ro on a failed runtime with --cooperative omits rejected args", store.Runtime{Name: "failed", SandboxRO: []string{"--allowedTools", "Read"}, ROProbe: store.RuntimeROFailed}, model.GrantRO, true, nil, 0, true},
		{"--cooperative does not suppress a real sandbox", enforcing, model.GrantRO, true, enforcing.SandboxRO, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, errb := newCtx(t.TempDir())
			got, err := sandboxFor(ctx, tc.rt, tc.grant, tc.cooperative)
			if code := clikit.ExitCode(err); code != tc.wantExit {
				t.Fatalf("exit %d, want %d (err %v)", code, tc.wantExit, err)
			}
			if tc.wantExit == 0 && !reflect.DeepEqual(got, tc.wantArgs) {
				t.Errorf("sandbox args = %#v, want %#v", got, tc.wantArgs)
			}
			if warned := strings.Contains(errb.String(), "COOPERATIVE"); warned != tc.wantWarn {
				t.Errorf("cooperative warning printed = %v, want %v (stderr: %q)", warned, tc.wantWarn, errb.String())
			}
		})
	}
}

// dacli 267: the spawn preamble tells the child to run os.Executable(), but a
// runtime allowlist pins an absolute-path Bash rule for dacli (cc-rw's real
// shape). When this dacli's path is not the allowlisted one, a headless child
// cannot run the binary its preamble names, so spawn WARNS — naming both the
// allowlisted path and the actual exe so the operator knows what to fix.
func TestExeAllowlistWarning(t *testing.T) {
	const allowed = "/Users/tahabsn/go/bin/dacli"
	// The real cc-rw invoke_args: an absolute-path dacli Bash rule in the rw
	// allowlist, so sandbox is empty (rw adds no ro sandbox) and the check runs
	// against invoke_args alone.
	ccrw := store.Runtime{Name: "cc-rw", Args: []string{"--allowedTools", "Edit", "Write", "Read",
		"Bash(/Users/tahabsn/go/bin/dacli:*)", "Bash(git:*)"}}

	t.Run("mismatched exe warns and names both paths", func(t *testing.T) {
		msg, ok := exeAllowlistWarning(ccrw, nil, "/repo/dacli")
		if !ok {
			t.Fatalf("expected a warning for a mismatched exe, got none")
		}
		if !strings.Contains(msg, allowed) || !strings.Contains(msg, "/repo/dacli") {
			t.Errorf("warning must name both the allowlisted path and the actual exe, got %q", msg)
		}
	})
	t.Run("matching exe is silent", func(t *testing.T) {
		if msg, ok := exeAllowlistWarning(ccrw, nil, allowed); ok {
			t.Errorf("no warning expected when exe matches the allowlist, got %q", msg)
		}
	})
	t.Run("ro sandbox allowlist is checked too", func(t *testing.T) {
		// cc's shape: the dacli Bash rule lives in the ro sandbox, applied for an
		// ro grant. A mismatched exe must still be caught against sandbox args.
		cc := store.Runtime{Name: "cc"}
		sandbox := []string{"--allowedTools", "Read,Grep,Bash(/Users/tahabsn/go/bin/dacli:*)"}
		if _, ok := exeAllowlistWarning(cc, sandbox, "/repo/dacli"); !ok {
			t.Errorf("expected a warning for a mismatched exe against the ro sandbox allowlist")
		}
	})
}

// A runtime with no model_flag makes role-level cost routing inoperative. That
// must be ANNOUNCED, never silently ignored — a reviewer role routed to opus
// that quietly ran on the default model would corrupt every cost calibration.
func TestModelArgs(t *testing.T) {
	withFlag := store.Runtime{Name: "claude-code", ModelFlag: "--model"}
	without := store.Runtime{Name: "generic-exec"}

	if got := modelArgs(mustCtx(t), withFlag, ""); got != nil {
		t.Errorf("no model requested should add no args, got %#v", got)
	}
	if got := modelArgs(mustCtx(t), withFlag, "opus"); !reflect.DeepEqual(got, []string{"--model", "opus"}) {
		t.Errorf("modelArgs = %#v, want [--model opus]", got)
	}
	ctx, _, errb := newCtx(t.TempDir())
	if got := modelArgs(ctx, without, "opus"); got != nil {
		t.Errorf("a runtime with no model_flag must add no args, got %#v", got)
	}
	if !strings.Contains(errb.String(), "declares no model_flag") {
		t.Errorf("silent model-routing loss; stderr was %q", errb.String())
	}
}

func mustCtx(t *testing.T) *clikit.Ctx {
	t.Helper()
	ctx, _, _ := newCtx(t.TempDir())
	return ctx
}

// seniorityGate is the mechanical seniority cap: a junior role cannot take the
// hard migration, and — the part that is easy to get wrong — an UNESTIMATED
// task is refused too, because a capped role takes only work whose size
// somebody stated.
func TestSeniorityGate(t *testing.T) {
	w := newExecWS(t)
	junior := team.Role{Name: "junior", MaxPoints: 3}
	profiled := team.Role{Name: "profiled", Profile: team.ModelProfile{MaxTaskPoints: 3}}
	uncapped := team.Role{Name: "senior"}

	unestimated := mustTask(t, w, "unsized task", store.TaskOpts{})
	small := mustTask(t, w, "small task", store.TaskOpts{Estimate: "1,2,3"})   // Te 2.0
	large := mustTask(t, w, "large task", store.TaskOpts{Estimate: "5,10,20"}) // Te 10.8

	cases := []struct {
		name     string
		role     team.Role
		task     *store.Task
		wantExit int
		wantMsg  string
	}{
		{"uncapped role takes anything", uncapped, large, 0, ""},
		{"uncapped role takes unestimated work", uncapped, unestimated, 0, ""},
		{"capped role takes work under its cap", junior, small, 0, ""},
		{"capped role REFUSES unestimated work", junior, unestimated, 3, "takes only estimated tasks"},
		{"capped role REFUSES work over its cap", junior, large, 3, "above role junior's cap"},
		{"profile capacity REFUSES work over its cap", profiled, large, 3, "above role profiled's cap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := seniorityGate(tc.role, tc.task)
			if code := clikit.ExitCode(err); code != tc.wantExit {
				t.Fatalf("exit %d, want %d (err %v)", code, tc.wantExit, err)
			}
			if tc.wantMsg != "" && !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("refusal %q does not name why (want %q)", err, tc.wantMsg)
			}
		})
	}
}

// phaseGate answers "don't start implementation while still in discovery".
// Roles with no kind opt out; ungated (solo/untemplated) projects never gate.
func TestPhaseGate(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "phase-gated task", store.TaskOpts{})

	setPhase(t, w, "discovery", []string{"researcher", "planner"})

	cases := []struct {
		name     string
		role     team.Role
		wantExit int
	}{
		{"a kindless role opts out of gating", team.Role{Name: "any"}, 0},
		{"an allowed kind passes", team.Role{Name: "r", Kind: "researcher"}, 0},
		{"a second allowed kind passes", team.Role{Name: "p", Kind: "planner"}, 0},
		{"a disallowed kind is REFUSED", team.Role{Name: "i", Kind: "implementer"}, 3},
		{"another disallowed kind is REFUSED", team.Role{Name: "rv", Kind: "reviewer"}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := phaseGate(w, task, tc.role)
			if code := clikit.ExitCode(err); code != tc.wantExit {
				t.Fatalf("exit %d, want %d (err %v)", code, tc.wantExit, err)
			}
			if tc.wantExit == 3 && !strings.Contains(err.Error(), "stage advance") {
				t.Errorf("a closed phase gate must name the remedy; got %q", err)
			}
		})
	}

	// An UNGATED project (no phase recorded) never refuses, whatever the kind.
	clearPhase(t, w)
	for _, kind := range []string{"researcher", "planner", "designer", "implementer", "reviewer"} {
		if err := phaseGate(w, task, team.Role{Name: "r", Kind: kind}); err != nil {
			t.Errorf("ungated project refused kind %q: %v", kind, err)
		}
	}
}
