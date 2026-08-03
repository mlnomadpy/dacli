package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Attribution is the loudest fingerprint dacli leaves on a repository: every
// commit carries `Dacli-Agent:` and an author at `@agent.dacli`, which makes a
// corpus of generated repositories trivially clusterable as same-origin. It is
// configurable — but the defaults must be exactly what they always were, or
// every existing workspace silently changes how it signs its history
// (dacli 196).
func TestAttributionDefaultsAreUnchanged(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "x")
	if err != nil {
		t.Fatal(err)
	}
	domain, prefix := w.Attribution()
	if domain != "@agent.dacli" {
		t.Errorf("default domain = %q, want @agent.dacli", domain)
	}
	if prefix != "Dacli" {
		t.Errorf("default trailer prefix = %q, want Dacli", prefix)
	}
}

func TestAttributionOverrideFromConfig(t *testing.T) {
	root := t.TempDir()
	w, err := workspace.Init(root, "x")
	if err != nil {
		t.Fatal(err)
	}
	cfg := w.ConfigPath()
	body, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, append(body, []byte("attribution_domain: example.com\ntrailer_prefix: Acme\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	reopened, err := workspace.Find(filepath.Join(root))
	if err != nil {
		t.Fatal(err)
	}
	domain, prefix := reopened.Attribution()
	if domain != "@example.com" {
		t.Errorf("domain = %q, want @example.com (a bare domain must gain the @)", domain)
	}
	if prefix != "Acme" {
		t.Errorf("trailer prefix = %q, want Acme", prefix)
	}
}

// A domain already carrying "@" must not be double-prefixed.
func TestAttributionDomainNotDoublePrefixed(t *testing.T) {
	w := &workspace.Workspace{AttributionDomain: "@corp.example", TrailerPrefix: "X"}
	if domain, _ := w.Attribution(); domain != "@corp.example" {
		t.Errorf("domain = %q, want @corp.example", domain)
	}
}
