package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

// The property, stated once: a name SafeSegment accepts must not be able to
// leave the directory it is joined to.
//
// This is the guard between a user-supplied ref — a project slug, a role name,
// a skill name — and the filesystem. Testing it against a list of known-bad
// strings only ever proves the list; the fuzzer looks for the string nobody
// thought of, and the assertion is the invariant rather than the enumeration
// (dacli 360).
func FuzzSafeSegmentNeverEscapes(f *testing.F) {
	for _, seed := range []string{
		"ok", "with-dash", "under_score", "MiXeD", "123", "",
		".", "..", "../etc", "a/b", "/abs", "C:\\win", "a\\b",
		"...", "....", "a..b", "..a", "a..", "\x00null", "nul\x00",
		"ünïcode", "\u2044slash", "\uFF0Ffullwidth", " space ", "\ttab",
		"a/../../b", "./x", "~", "~/x", "-dashlead", "%2e%2e",
	} {
		f.Add(seed)
	}
	root := "/workspace/root"
	f.Fuzz(func(t *testing.T, name string) {
		if !SafeSegment(name) {
			return // rejected: the guard makes no promise about what it refused
		}

		// ACCEPTED. Joining it must stay inside root, and must stay exactly
		// one level down — a segment is by definition a single component.
		joined := filepath.Join(root, name)
		clean := filepath.Clean(joined)

		if !strings.HasPrefix(clean, root+string(filepath.Separator)) {
			t.Fatalf("SafeSegment(%q) accepted a name that escapes: %q", name, clean)
		}
		if rel, err := filepath.Rel(root, clean); err != nil || strings.Contains(rel, "..") {
			t.Fatalf("SafeSegment(%q) accepted a name that traverses: rel=%q err=%v", name, rel, err)
		}
		// One component: no separator survived the join beyond the root's own.
		if strings.Contains(strings.TrimPrefix(clean, root+string(filepath.Separator)), string(filepath.Separator)) {
			t.Fatalf("SafeSegment(%q) accepted a multi-component name: %q", name, clean)
		}
		// A NUL byte cannot reach a syscall: most filesystems reject it, and a
		// name that fails at the syscall is a name the guard should have
		// refused rather than passed on.
		if strings.ContainsRune(name, 0) {
			t.Fatalf("SafeSegment(%q) accepted a name containing NUL", name)
		}
	})
}

// SafeRelPath permits separators — a gate predicate legitimately names
// internal/api/server.go — so its property is weaker but the escape rule is
// identical: whatever it accepts must stay under the root.
func FuzzSafeRelPathNeverEscapes(f *testing.F) {
	for _, seed := range []string{
		"a/b.go", "internal/api/server.go", "x", "", "..", "../x", "/abs",
		"a/../b", "a/../../b", "./a", "a/./b", "a//b", "C:/x", "a\\..\\b",
		"a/..", "..//..//x", "\x00", "ünï/code",
	} {
		f.Add(seed)
	}
	root := "/workspace/root"
	f.Fuzz(func(t *testing.T, p string) {
		if !SafeRelPath(p) {
			return
		}
		clean := filepath.Clean(filepath.Join(root, p))
		if clean != root && !strings.HasPrefix(clean, root+string(filepath.Separator)) {
			t.Fatalf("SafeRelPath(%q) accepted a path that escapes: %q", p, clean)
		}
		if rel, err := filepath.Rel(root, clean); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			t.Fatalf("SafeRelPath(%q) accepted a traversal: rel=%q err=%v", p, rel, err)
		}
		if strings.ContainsRune(p, 0) {
			t.Fatalf("SafeRelPath(%q) accepted a path containing NUL", p)
		}
	})
}
