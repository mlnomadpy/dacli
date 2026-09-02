package docs_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

type privacyManifest struct {
	Schema           string                         `json:"schema"`
	DefaultPolicy    string                         `json:"default_policy"`
	RetentionClasses map[string]string              `json:"retention_classes"`
	Events           map[string][]privacyFieldGroup `json:"events"`
}

type privacyFieldGroup struct {
	Fields         []string `json:"fields"`
	Purpose        string   `json:"purpose"`
	Direction      string   `json:"direction"`
	Retention      string   `json:"retention"`
	Visibility     string   `json:"visibility"`
	DefaultEnabled *bool    `json:"default_enabled"`
}

func TestControlPlanePrivacyManifestCoversEveryPayloadFieldExactlyOnce(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(here), ".."))
	contract := filepath.Join(root, "contracts", "controlplane", "v1")
	raw, err := os.ReadFile(filepath.Join(contract, "privacy-fields.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest privacyManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != "control-plane-privacy/v1" || manifest.DefaultPolicy != "deny" {
		t.Fatalf("privacy header = schema %q default %q", manifest.Schema, manifest.DefaultPolicy)
	}

	paths, err := filepath.Glob(filepath.Join(contract, "*.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := map[string]bool{}
	for _, path := range paths {
		if filepath.Base(path) == "envelope.schema.json" {
			continue
		}
		eventType := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(path), ".schema.json"), "-", "_")
		wantEvents[eventType] = true
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read schema %s: %v", path, err)
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("parse schema %s: %v", path, err)
		}
		seen := map[string]int{}
		for _, group := range manifest.Events[eventType] {
			if group.Purpose == "" || group.Direction == "" || group.Visibility == "" || group.DefaultEnabled == nil {
				t.Errorf("%s has incomplete privacy classification: %+v", eventType, group)
			}
			if _, ok := manifest.RetentionClasses[group.Retention]; !ok {
				t.Errorf("%s uses unknown retention class %q", eventType, group.Retention)
			}
			for _, field := range group.Fields {
				seen[field]++
			}
		}
		for field := range schema.Properties {
			if seen[field] != 1 {
				t.Errorf("%s.%s privacy classifications = %d, want exactly one", eventType, field, seen[field])
			}
			delete(seen, field)
		}
		for field := range seen {
			t.Errorf("privacy manifest contains unknown field %s.%s", eventType, field)
		}
	}
	var unknown []string
	for eventType := range manifest.Events {
		if !wantEvents[eventType] {
			unknown = append(unknown, eventType)
		}
	}
	sort.Strings(unknown)
	if len(manifest.Events) != len(wantEvents) || len(unknown) > 0 {
		t.Fatalf("privacy event registry count=%d want=%d unknown=%v", len(manifest.Events), len(wantEvents), unknown)
	}
}

func TestControlPlaneDocsKeepShippedAndFutureBoundariesExplicit(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(here), ".."))
	wants := map[string][]string{
		"docs/decisions/0001-control-plane-boundary.md": {"modular monolith", "Primary MVP persona", "Remote execution", "service is still unshipped"},
		"docs/CONTROL_PLANE_THREAT_MODEL.md":            {"hosted service is not shipped", "Cross-tenant", "exact action hash", "Residual risks"},
		"docs/CONTROL_PLANE_PRIVACY.md":                 {"SaaS service is not shipped", "deny-by-default", "Never valid v1 metadata", "must not rank individual developers"},
	}
	for name, phrases := range wants {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, phrase := range phrases {
			if !strings.Contains(string(raw), phrase) {
				t.Errorf("%s missing boundary %q", name, phrase)
			}
		}
	}
}
