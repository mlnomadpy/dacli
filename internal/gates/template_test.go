package gates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Load must return every embedded template, each carrying its parsed stages,
// cone/phase/allow lines, and predicate bullets — this is the only place
// `parse` is exercised end to end.
func TestLoadReturnsEveryEmbeddedTemplate(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := Load(w)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Template{}
	for _, tmpl := range ts {
		byName[tmpl.Name] = tmpl
	}
	for _, want := range []string{"product", "research-paper", "solo", "standard", "tdd"} {
		if _, ok := byName[want]; !ok {
			t.Errorf("Load did not return the embedded template %q", want)
		}
	}

	product := byName["product"]
	if product.Origin != "embedded" {
		t.Errorf("product.Origin = %q, want embedded", product.Origin)
	}
	if len(product.Stages) == 0 {
		t.Fatal("product template parsed with zero stages")
	}
	discovery := product.Stages[0]
	if discovery.Name != "discovery" {
		t.Errorf("first stage name = %q, want discovery", discovery.Name)
	}
	if discovery.Cone != "definition" {
		t.Errorf("discovery.Cone = %q, want definition", discovery.Cone)
	}
	if discovery.Phase != "discovery" {
		t.Errorf("discovery.Phase = %q, want discovery", discovery.Phase)
	}
	if !discovery.AllowsKind("researcher") || discovery.AllowsKind("implementer") {
		t.Errorf("discovery.Allow = %v did not gate role kinds correctly", discovery.Allow)
	}
	if len(discovery.Predicates) == 0 {
		t.Fatal("discovery stage parsed with no predicate bullets")
	}
	found := false
	for _, pr := range discovery.Predicates {
		if pr.Kind == "project_sections" && pr.Arg == "Goal | Out of scope" {
			found = true
		}
	}
	if !found {
		t.Errorf("discovery predicates = %+v, missing the project_sections bullet", discovery.Predicates)
	}

	// solo has zero stages: no gates, most work should not pay for process.
	solo := byName["solo"]
	if len(solo.Stages) != 0 {
		t.Errorf("solo template has %d stages, want 0", len(solo.Stages))
	}
}

// A workspace-vendored template of the same name wins over the embedded
// default — the nearest-wins rule the rest of dacli follows.
func TestLoadWorkspaceTemplateOverridesEmbedded(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(w.TemplatesDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "---\nname: solo\nsummary: overridden\ncost: \"free\"\n---\n# solo\n\ncustom body\n"
	if err := os.WriteFile(filepath.Join(w.TemplatesDir(), "solo.md"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpl, err := Get(w, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Origin != "workspace" {
		t.Errorf("Origin = %q, want workspace (vendored copy must win)", tmpl.Origin)
	}
	if tmpl.Summary != "overridden" {
		t.Errorf("Summary = %q, want the vendored override", tmpl.Summary)
	}

	// Load must not double-list it under both origins.
	ts, err := Load(w)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, tp := range ts {
		if tp.Name == "solo" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("solo appears %d times in Load's result, want exactly 1", n)
	}

	// A wholly new workspace template (no embedded twin) is also returned.
	fresh := "---\nname: freshly-vendored\nsummary: net new\ncost: \"free\"\n---\n# freshly-vendored\n"
	if err := os.WriteFile(filepath.Join(w.TemplatesDir(), "freshly-vendored.md"), []byte(fresh), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(w, "freshly-vendored"); err != nil {
		t.Errorf("a workspace-only template must be discoverable via Get: %v", err)
	}
}

// Get must name-not-found rather than silently returning a zero value.
func TestGetUnknownTemplateReturnsErrNotFound(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Get(w, "does-not-exist")
	if err == nil {
		t.Fatal("Get on an unknown template name must return an error")
	}
	var nf store.ErrNotFound
	if !errors.As(err, &nf) {
		t.Errorf("Get's error = %v, want a store.ErrNotFound", err)
	}
}

// Vendor copies an embedded template's raw bytes into the workspace, refuses
// a name with no embedded twin, and refuses to clobber an already-vendored copy.
func TestVendor(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}

	path, err := Vendor(w, "standard")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "name: standard") {
		t.Errorf("vendored file does not look like the standard template: %q", string(raw)[:40])
	}

	// Vendoring the same template twice must refuse rather than overwrite.
	if _, err := Vendor(w, "standard"); err == nil {
		t.Fatal("vendoring an already-vendored template must refuse")
	}

	// Vendoring a name with no embedded template must refuse too.
	if _, err := Vendor(w, "not-a-real-template"); err == nil {
		t.Fatal("vendoring an unknown embedded template name must refuse")
	}
}
