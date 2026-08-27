package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func runtimeWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	return w
}

func TestLegacyRuntimeHarnessInferenceIsExactAndSafe(t *testing.T) {
	w := runtimeWorkspace(t)
	legacy := Runtime{Name: "old-gemini-writer", Binary: "gemini", Mode: "arg", Flag: "-p", ModelFlag: "--model", UsageFormat: "gemini-stream-json"}
	if err := CreateRuntime(w, "a-root", legacy, ""); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRuntime(w, legacy.Name)
	if err != nil || got.Harness != "gemini" {
		t.Fatalf("legacy shipped contract harness = %q, err=%v", got.Harness, err)
	}
	custom := Runtime{Name: "custom", Binary: "gemini", Mode: "stdin"}
	if err := CreateRuntime(w, "a-root", custom, ""); err != nil {
		t.Fatal(err)
	}
	got, err = LoadRuntime(w, custom.Name)
	if err != nil || got.Harness != "custom" {
		t.Fatalf("ambiguous custom adapter harness = %q, err=%v; want safe runtime-name family", got.Harness, err)
	}
}

func TestRuntimeROProbeCacheIsLocalAndInvalidatesOnAdapterChange(t *testing.T) {
	w := runtimeWorkspace(t)
	bin := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(bin, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	rt := Runtime{Name: "fixture", Binary: bin, SandboxRO: []string{"--allowedTools", "Read"}}
	if err := SaveRuntimeROProbe(w, rt, bin, RuntimeROVerified, "test"); err != nil {
		t.Fatal(err)
	}
	if got := HydrateRuntimeROProbe(w, rt, bin); got.ROProbe != RuntimeROVerified {
		t.Fatalf("matching cached probe = %q, want verified", got.ROProbe)
	}
	changed := rt
	changed.SandboxRO = []string{"--allowedTools", "Read,Grep"}
	if got := HydrateRuntimeROProbe(w, changed, bin); got.ROProbe != RuntimeROUnknown {
		t.Fatalf("stale cached probe = %q, want unknown", got.ROProbe)
	}
}

func TestRuntimeROProbeCacheInvalidatesOnSameMetadataBinaryReplacement(t *testing.T) {
	w := runtimeWorkspace(t)
	bin := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(bin, []byte("original"), 0o755); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	rt := Runtime{Name: "fixture", Binary: bin, SandboxRO: []string{"--allowedTools", "Read"}}
	if err := SaveRuntimeROProbe(w, rt, bin, RuntimeROVerified, "test"); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(bin, []byte("replaced"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(bin, fi.ModTime(), fi.ModTime()); err != nil {
		t.Fatal(err)
	}
	replaced, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Size() != fi.Size() || !replaced.ModTime().Equal(fi.ModTime()) {
		t.Fatalf("replacement metadata changed: size %d/%d mtime %v/%v", replaced.Size(), fi.Size(), replaced.ModTime(), fi.ModTime())
	}

	if got := HydrateRuntimeROProbe(w, rt, bin); got.ROProbe != RuntimeROUnknown {
		t.Fatalf("cached probe after byte-different same-metadata replacement = %q, want unknown", got.ROProbe)
	}
}

func TestRuntimeROProbeCacheContainsHandEditedNames(t *testing.T) {
	w := runtimeWorkspace(t)
	bin := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(bin, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	rt := Runtime{Name: "../../escaped", Binary: bin, SandboxRO: []string{"--allowedTools", "Read"}}
	if err := SaveRuntimeROProbe(w, rt, bin, RuntimeROVerified, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(w.Root, workspace.Dir, "escaped.json")); !os.IsNotExist(err) {
		t.Fatalf("malformed runtime name escaped the probe cache: %v", err)
	}
	if got := HydrateRuntimeROProbe(w, rt, bin); got.ROProbe != RuntimeROVerified {
		t.Fatalf("contained probe cache = %q, want verified", got.ROProbe)
	}
}

// TestRuntimeWritable is the grant/runtime coupling check (dacli 250): a
// runtime that pins an --allowedTools allowlist can host an rw child only if
// that allowlist names a write tool. The read-only `cc` shape (allowlist in the
// ro sandbox, no write tool) must read as non-writable so its rw role is caught
// before a run is burned; the write-capable `cc-rw` shape must read as writable;
// and a runtime that pins no allowlist at all must stay writable so plain
// adapters like generic-exec are never falsely refused.
func TestRuntimeWritable(t *testing.T) {
	// The comma-joined form (`"Edit,Write,Read"` as one arg) and the
	// one-token-per-arg form both occur on disk, so both must be recognized.
	for _, tc := range []struct {
		name string
		rt   Runtime
		want bool
	}{
		{"cc: ro allowlist, no write tool", Runtime{SandboxRO: []string{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"}}, false},
		{"cc-rw: write tool one-per-arg", Runtime{Args: []string{"--allowedTools", "Edit", "Write", "Read"}}, true},
		{"write tool comma-joined in one arg", Runtime{Args: []string{"--allowedTools", "Edit,Write,Read"}}, true},
		{"invoke allowlist without write tool", Runtime{Args: []string{"--allowedTools", "Read,Grep"}}, false},
		{"no allowlist anywhere (generic-exec)", Runtime{Args: []string{"-p"}}, true},
		{"bash-only rule is not a write tool", Runtime{Args: []string{"--allowedTools", "Read,Bash(git:*)"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeWritable(tc.rt); got != tc.want {
				t.Errorf("RuntimeWritable(%#v) = %v, want %v", tc.rt, got, tc.want)
			}
		})
	}
}

// TestRuntimeEnforcesRO is the ro half of the same coupling: a declaration is
// not evidence; only a successful local probe can back an ro child.
func TestRuntimeEnforcesRO(t *testing.T) {
	if RuntimeEnforcesRO(Runtime{}) {
		t.Error("a runtime with no SandboxRO must not claim to enforce read-only")
	}
	if RuntimeEnforcesRO(Runtime{SandboxRO: []string{"--allowedTools", "Read"}}) {
		t.Error("a declaration-only runtime must not claim to enforce read-only")
	}
	if !RuntimeEnforcesRO(Runtime{SandboxRO: []string{"--allowedTools", "Read"}, ROProbe: RuntimeROVerified}) {
		t.Error("a runtime with a verified probe must enforce read-only")
	}
	if RuntimeEnforcesRO(Runtime{SandboxRO: []string{"--allowedTools", "Read"}, ROProbe: RuntimeROFailed}) {
		t.Error("a failed probe must not enforce read-only")
	}
}

func TestRuntimeROProbeableRequiresAFullyRecognizedReadOnlyAllowlist(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{"preset shape", []string{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"}, true},
		{"split tool values", []string{"--allowed-tools", "Read", "Grep"}, true},
		{"declaration only vendor mode", []string{"--permission-mode", "plan"}, false},
		{"missing allowlist value", []string{"--allowedTools"}, false},
		{"write tool", []string{"--allowedTools", "Read,Edit"}, false},
		{"write-capable shell rule", []string{"--allowedTools", "Read,Bash(git:*)"}, false},
		{"unknown tool", []string{"--allowedTools", "Read,VendorMagic"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeROProbeable(Runtime{SandboxRO: tc.args}); got != tc.want {
				t.Errorf("RuntimeROProbeable(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

// TestRuntimeAllowsDacli is the dacli-267 coupling check: the spawn preamble
// tells the child to run the dacli binary at os.Executable(), but the runtime's
// --allowedTools allowlist pins an ABSOLUTE-path Bash rule for dacli (cc-rw's
// `Bash(/Users/.../go/bin/dacli:*)`). A child can only run the binary its own
// preamble names when that path equals an allowlisted dacli path. Both on-disk
// allowlist encodings — one-token-per-arg (how the pre-quoting cc-rw.md splits)
// and comma-joined-in-one-arg — must be recognized.
func TestRuntimeAllowsDacli(t *testing.T) {
	const allowed = "/Users/tahabsn/go/bin/dacli"
	// The real cc-rw shape: the allowlist split into one token per arg, with the
	// dacli Bash rule as an absolute path alongside git/go/gofmt rules.
	ccrwSplit := []string{"--allowedTools", "Edit", "Write", "Read", "Grep", "Glob", "LS",
		"Bash(/Users/tahabsn/go/bin/dacli:*)", "Bash(git:*)", "Bash(go:*)", "Bash(gofmt:*)"}
	// Same allowlist as one comma-joined arg (the post-quoting encoding).
	ccrwJoined := []string{"--allowedTools",
		"Edit,Write,Read,Grep,Glob,LS,Bash(/Users/tahabsn/go/bin/dacli:*),Bash(git:*),Bash(go:*),Bash(gofmt:*)"}

	for _, tc := range []struct {
		name          string
		args          []string
		exe           string
		wantOK        bool
		wantAllowlist []string
	}{
		{"cc-rw split: exe matches the allowlisted path", ccrwSplit, allowed, true, []string{allowed}},
		{"cc-rw split: exe is a DIFFERENT dacli (repo build)", ccrwSplit, "/repo/dacli", false, []string{allowed}},
		{"cc-rw split: exe is a worktree dacli", ccrwSplit, "/repo/.dacli/worktrees/x/dacli", false, []string{allowed}},
		{"cc-rw joined: exe matches", ccrwJoined, allowed, true, []string{allowed}},
		{"cc-rw joined: exe mismatched", ccrwJoined, "/tmp/build/dacli", false, []string{allowed}},
		{"bare Bash(dacli) does not permit an absolute exe", []string{"--allowedTools", "Read,Bash(dacli:*)"}, allowed, false, []string{"dacli"}},
		{"no dacli Bash rule: nothing to contradict", []string{"--allowedTools", "Read,Bash(git:*)"}, allowed, true, nil},
		{"no allowlist at all (generic-exec)", []string{"-p"}, allowed, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok, allowlisted := RuntimeAllowsDacli(tc.args, tc.exe)
			if ok != tc.wantOK {
				t.Errorf("RuntimeAllowsDacli(%v, %q) ok = %v, want %v", tc.args, tc.exe, ok, tc.wantOK)
			}
			if !reflect.DeepEqual(allowlisted, tc.wantAllowlist) {
				t.Errorf("allowlisted = %#v, want %#v", allowlisted, tc.wantAllowlist)
			}
		})
	}
}

// TestNamedTools is dacli 272's third preflight class: a role prompt names a
// tool the same way this codebase already reads other signals precisely — a
// backtick code span, not prose. "Write the failing test" is an instruction,
// not a claim that the role needs the Write tool; a code-quoted WebFetch is.
func TestNamedTools(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prompt string
		want   []string
	}{
		{"prose verbs are not tool names", "Write the failing test first, then Read the code.", nil},
		{"one tool named in a code span", "Use `WebFetch` to pull the changelog before summarizing.", []string{"WebFetch"}},
		{"multiple tools, deduped in first-seen order", "Call `Bash` for git, then `Edit` the file, then `Bash` again.", []string{"Bash", "Edit"}},
		{"an unknown backtick token is not a tool", "Run `go test ./...` before committing.", nil},
		{"empty prompt", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NamedTools(tc.prompt); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NamedTools(%q) = %#v, want %#v", tc.prompt, got, tc.want)
			}
		})
	}
}

// TestRuntimeAllowsTool mirrors RuntimeWritable/RuntimeAllowsDacli's shape: a
// runtime that pins no allowlist anywhere makes no restrictive promise, so
// every tool is permitted; one that pins an allowlist permits only what it
// names, matched case-insensitively and whether comma-joined or one-per-arg.
func TestRuntimeAllowsTool(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		tool string
		want bool
	}{
		{"no allowlist pinned: nothing to contradict", []string{"-p"}, "WebFetch", true},
		{"allowlist grants the tool, one-per-arg", []string{"--allowedTools", "Edit", "Write", "WebFetch"}, "WebFetch", true},
		{"allowlist grants the tool, comma-joined", []string{"--allowedTools", "Edit,Write,WebFetch"}, "WebFetch", true},
		{"allowlist omits the tool", []string{"--allowedTools", "Edit,Write,Read"}, "WebFetch", false},
		{"match is case-insensitive", []string{"--allowedTools", "edit,write"}, "Edit", true},
		{"a Bash(...) rule does not grant WebFetch", []string{"--allowedTools", "Read,Bash(git:*)"}, "WebFetch", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RuntimeAllowsTool(tc.args, tc.tool); got != tc.want {
				t.Errorf("RuntimeAllowsTool(%v, %q) = %v, want %v", tc.args, tc.tool, got, tc.want)
			}
		})
	}
}

// TestRuntimeInlineListRoundTripsCommaContainingElements proves a list
// element containing a literal comma -- like the claude-code preset's
// --allowedTools value -- survives CreateRuntime + LoadRuntime as ONE
// element, not silently re-split into several argv tokens.
func TestRuntimeInlineListRoundTripsCommaContainingElements(t *testing.T) {
	w := runtimeWorkspace(t)

	sandboxRO := []string{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"}
	args := []string{"-p", "a, b, c"}
	env := []string{"HOME", "PATH,EXTRA"}

	rt := Runtime{
		Name:      "claude-code",
		Binary:    "claude",
		Mode:      "arg",
		Flag:      "-p",
		Args:      args,
		SandboxRO: sandboxRO,
		Env:       env,
	}
	if err := CreateRuntime(w, "a-root", rt, ""); err != nil {
		t.Fatalf("CreateRuntime: %v", err)
	}

	got, err := LoadRuntime(w, "claude-code")
	if err != nil {
		t.Fatalf("LoadRuntime: %v", err)
	}

	check := func(field string, want, got []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: got %d elements %#v, want %d elements %#v", field, len(got), got, len(want), want)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %#v, want %#v", field, got, want)
		}
	}
	check("SandboxRO", sandboxRO, got.SandboxRO)
	check("Args", args, got.Args)
	check("Env", env, got.Env)
}

// dacli's own outward writes are gated — rw grant at the dispatcher, recorded
// disclosure consent before a public mirror. All of it is bypassed the moment
// a child can shell to `gh` directly, which is how an agent once created a
// repo, set origin, pushed, and merged PRs with nobody approving any of it
// (task 308, issue #382 item 6).
func TestUngatedOutwardGrantNamesTheTool(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"gh in a comma list", []string{"--allowedTools", "Read,Edit,Bash(gh:*)"}, "bash(gh:*)"},
		{"gh as its own arg", []string{"--allowedTools", "Bash(gh:*)"}, "bash(gh:*)"},
		{"unrestricted bash", []string{"--allowedTools", "Read,Bash(*)"}, "bash(*)"},
		{"curl", []string{"--allowedTools", "Bash(curl:*)"}, "bash(curl:*)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry, why := UngatedOutwardGrant(tc.args)
			if entry != tc.want {
				t.Errorf("entry = %q, want %q", entry, tc.want)
			}
			if why == "" {
				t.Error("a detection must say WHY it matters, or nobody acts on it")
			}
		})
	}
}

// git is deliberately allowed: an implementer needs it for its own branch and
// commits, and a push still needs a remote the operator configured. Flagging
// it would make the warning fire on every ordinary implementer runtime, which
// is how a warning stops being read.
func TestUngatedOutwardGrantAllowsTheOrdinaryImplementerToolset(t *testing.T) {
	for _, args := range [][]string{
		{"--allowedTools", "Read,Grep,Glob,LS,Edit,Write,Bash(git:*),Bash(dacli:*)"},
		{"--allowedTools", "Read,Grep,Glob,LS,Bash(dacli:*)"},
		nil,
	} {
		if entry, _ := UngatedOutwardGrant(args); entry != "" {
			t.Errorf("UngatedOutwardGrant(%v) flagged %q; the ordinary toolset must not warn", args, entry)
		}
	}
}

func TestGradleRequiresProviderNeutralCoordinationSocketCapability(t *testing.T) {
	commands := []string{"./gradlew testDebugUnitTest"}
	want := []ExecutionCapability{ExecutionCapabilityLocalCoordinationSocket}
	if got := RequiredExecutionCapabilities(commands); !reflect.DeepEqual(got, want) {
		t.Fatalf("Gradle requirements = %v, want %v", got, want)
	}
	for _, name := range []string{"codex-rw", "claude-rw", "gemini-rw", "copilot-rw", "generic"} {
		rt := Runtime{Name: name}
		if RuntimeHasExecutionCapability(rt, want[0]) {
			t.Fatalf("runtime name %q implicitly granted %s", name, want[0])
		}
		rt.ExecutionCapabilities = want
		if !RuntimeHasExecutionCapability(rt, want[0]) {
			t.Fatalf("runtime %q ignored declared capability", name)
		}
	}
}
