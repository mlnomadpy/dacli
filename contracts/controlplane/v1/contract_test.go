package controlplane_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type schema struct {
	ID                   string            `json:"$id"`
	Required             []string          `json:"required"`
	Properties           map[string]schema `json:"properties"`
	AdditionalProperties any               `json:"additionalProperties"`
}

func readSchema(t *testing.T, name string) schema {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var got schema
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return got
}

func TestVersionedSchemasAndEnvelope(t *testing.T) {
	types := []string{"installation", "repository", "task-proposal", "run-summary", "approval", "policy-bundle", "sync-cursor"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			s := readSchema(t, typ+".schema.json")
			if !strings.Contains(s.ID, "/controlplane/v1/") {
				t.Fatalf("$id %q is not versioned at controlplane/v1", s.ID)
			}
			if s.AdditionalProperties != false {
				t.Fatal("payload schema must be closed by default")
			}
		})
	}

	envelope := readSchema(t, "envelope.schema.json")
	want := []string{"schema_version", "tenant_id", "project_id", "event_id", "producer_sequence", "occurred_at", "idempotency_key", "producer_version", "integrity"}
	for _, field := range want {
		if _, ok := envelope.Properties[field]; !ok {
			t.Errorf("envelope missing property %q", field)
		}
		if !contains(envelope.Required, field) {
			t.Errorf("envelope does not require %q", field)
		}
	}
}

func TestPayloadAllowlistExcludesSensitiveFields(t *testing.T) {
	for _, name := range []string{"installation", "repository", "task-proposal", "run-summary", "approval", "policy-bundle", "sync-cursor"} {
		s := readSchema(t, name+".schema.json")
		for _, forbidden := range []string{"source_code", "prompt", "transcript", "command_output", "environment", "environment_values", "secret", "secrets"} {
			if _, ok := s.Properties[forbidden]; ok {
				t.Errorf("%s allowlists forbidden field %q", name, forbidden)
			}
		}
	}
}

type fixture struct {
	Name    string `json:"name"`
	Outcome string `json:"outcome"`
	State   struct {
		LastSequence    int      `json:"last_sequence"`
		SeenEventIDs    []string `json:"seen_event_ids"`
		SeenIdempotency []string `json:"seen_idempotency_keys"`
		MinimumVersion  int      `json:"minimum_schema_version"`
		MaximumVersion  int      `json:"maximum_schema_version"`
		ReplayFloor     int      `json:"replay_floor"`
	} `json:"state"`
	Event struct {
		EventID          string `json:"event_id"`
		IdempotencyKey   string `json:"idempotency_key"`
		ProducerSequence int    `json:"producer_sequence"`
		SchemaVersion    int    `json:"schema_version"`
		SignatureValid   bool   `json:"signature_valid"`
	} `json:"event"`
}

func TestGoldenOutcomes(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 6 {
		t.Fatalf("got %d fixtures, want six", len(paths))
	}
	var outcomes []string
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var f fixture
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if got := classify(f); got != f.Outcome {
			t.Errorf("%s: got %q, want %q", f.Name, got, f.Outcome)
		}
		outcomes = append(outcomes, f.Outcome)
	}
	sort.Strings(outcomes)
	want := []string{"accept-delayed", "accept-reordered", "ignore-duplicate", "refuse-incompatible", "refuse-replay", "refuse-tampered"}
	if !reflect.DeepEqual(outcomes, want) {
		t.Fatalf("outcomes = %v, want distinct %v", outcomes, want)
	}
}

func classify(f fixture) string {
	if !f.Event.SignatureValid {
		return "refuse-tampered"
	}
	if f.Event.SchemaVersion < f.State.MinimumVersion || f.Event.SchemaVersion > f.State.MaximumVersion {
		return "refuse-incompatible"
	}
	if contains(f.State.SeenEventIDs, f.Event.EventID) || contains(f.State.SeenIdempotency, f.Event.IdempotencyKey) {
		return "ignore-duplicate"
	}
	if f.Event.ProducerSequence < f.State.ReplayFloor {
		return "refuse-replay"
	}
	if f.Event.ProducerSequence <= f.State.LastSequence {
		return "accept-reordered"
	}
	return "accept-delayed"
}

func contains[T comparable](values []T, want T) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
