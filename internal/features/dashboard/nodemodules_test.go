package dashboard

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Task 183: `go build/test/vet ./...` walks the whole module tree and descends
// into node_modules, so a stray *.go under the installed JS dependencies is
// compiled and can break the build. node_modules is gitignored, so the fix is
// an npm `postinstall` that drops a lone go.mod into it — a nested-module
// boundary the parent module's `./...` walk skips. These tests guard both the
// wiring (the postinstall hook + its script) and the mechanism it relies on.

// TestNodeModulesExclusionWired fails before the fix: it asserts ui/package.json
// registers the postinstall hook and that the script it names exists and writes
// the go.mod marker into node_modules.
func TestNodeModulesExclusionWired(t *testing.T) {
	const scriptRel = "scripts/exclude-node-modules-from-go.mjs"

	raw, err := os.ReadFile(filepath.Join("ui", "package.json"))
	if err != nil {
		t.Fatalf("read ui/package.json: %v", err)
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("parse ui/package.json: %v", err)
	}
	post := pkg.Scripts["postinstall"]
	if !strings.Contains(post, scriptRel) {
		t.Fatalf("ui/package.json postinstall = %q, want it to run %q", post, scriptRel)
	}

	script, err := os.ReadFile(filepath.Join("ui", filepath.FromSlash(scriptRel)))
	if err != nil {
		t.Fatalf("read %s: %v", scriptRel, err)
	}
	for _, want := range []string{"node_modules", "go.mod"} {
		if !strings.Contains(string(script), want) {
			t.Errorf("%s does not mention %q; it must write a go.mod into node_modules", scriptRel, want)
		}
	}
}

// TestNestedGoModExcludesNodeModules is the mechanism guard: it proves that a
// lone go.mod inside node_modules removes the subtree from `go list ./...`,
// which is exactly what the postinstall marker relies on. Without the marker a
// stray package under node_modules IS listed (the hazard); with it, it is not.
func TestNestedGoModExcludesNodeModules(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/root\n\ngo 1.22\n")
	write("main.go", "package main\n\nfunc main() {}\n")
	// A vendored JS package that happens to ship a Go file, the way real npm
	// deps sometimes do — this is what breaks `go build ./...`.
	write("node_modules/stray/stray.go", "package stray\n")

	list := func() string {
		cmd := exec.Command("go", "list", "./...")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go list ./...: %v\n%s", err, out)
		}
		return string(out)
	}

	if got := list(); !strings.Contains(got, "node_modules/stray") {
		t.Fatalf("control failed: expected the stray node_modules package to be listed without the marker, got:\n%s", got)
	}

	// The marker the postinstall script writes.
	write("node_modules/go.mod", "module example.test/node_modules-not-go\n\ngo 1.22\n")

	if got := list(); strings.Contains(got, "node_modules") {
		t.Fatalf("marker failed to exclude node_modules from `go list ./...`:\n%s", got)
	}
}
