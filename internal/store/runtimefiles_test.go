package store

import (
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

// TestRuntimeEnforcesRO is the ro half of the same coupling: only a runtime
// with a declared read-only sandbox can hold a child to read-only.
func TestRuntimeEnforcesRO(t *testing.T) {
	if RuntimeEnforcesRO(Runtime{}) {
		t.Error("a runtime with no SandboxRO must not claim to enforce read-only")
	}
	if !RuntimeEnforcesRO(Runtime{SandboxRO: []string{"--allowedTools", "Read"}}) {
		t.Error("a runtime with a SandboxRO arg set must enforce read-only")
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
