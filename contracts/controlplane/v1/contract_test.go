package controlplane_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/cloudsync"
)

type schema struct {
	ID                   string            `json:"$id"`
	Required             []string          `json:"required"`
	Properties           map[string]schema `json:"properties"`
	AdditionalProperties any               `json:"additionalProperties"`
	Enum                 []string          `json:"enum"`
}

var payloadSchemaNames = []string{
	"agent-state", "approval", "budget-state", "device-registration", "event-summary",
	"gate-evidence", "installation", "policy-bundle", "project-registration", "repository",
	"role-bundle", "run-summary", "sync-cursor", "task-proposal",
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
	for _, typ := range payloadSchemaNames {
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

func TestEnvelopeSchemaAndRuntimeValidatorUseOneEventRegistry(t *testing.T) {
	envelope := readSchema(t, "envelope.schema.json")
	want := cloudsync.PayloadTypes()
	got := append([]string(nil), envelope.Properties["event_type"].Enum...)
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("envelope event types = %v, runtime validator = %v", got, want)
	}
	var schemas []string
	for _, name := range payloadSchemaNames {
		schemas = append(schemas, strings.ReplaceAll(name, "-", "_"))
	}
	sort.Strings(schemas)
	if !reflect.DeepEqual(schemas, want) {
		t.Fatalf("payload schemas = %v, runtime validator = %v", schemas, want)
	}
}

func TestValidFixturesCoverSchemaRequiredFields(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "payloads", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]map[string]any
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	for eventType, payload := range fixtures {
		name := strings.ReplaceAll(eventType, "_", "-")
		s := readSchema(t, name+".schema.json")
		for _, field := range s.Required {
			if _, ok := payload[field]; !ok {
				t.Errorf("%s valid fixture misses required field %q", eventType, field)
			}
		}
		for field := range payload {
			if _, ok := s.Properties[field]; !ok {
				t.Errorf("%s valid fixture contains non-schema field %q", eventType, field)
			}
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if err := cloudsync.ValidatePayload(eventType, encoded); err != nil {
			t.Errorf("%s schema fixture fails runtime validator: %v", eventType, err)
		}
	}
}

func TestPayloadAllowlistExcludesSensitiveFields(t *testing.T) {
	for _, name := range payloadSchemaNames {
		s := readSchema(t, name+".schema.json")
		for _, forbidden := range []string{"source_code", "prompt", "transcript", "command_output", "environment", "environment_values", "secret", "secrets"} {
			if _, ok := s.Properties[forbidden]; ok {
				t.Errorf("%s allowlists forbidden field %q", name, forbidden)
			}
		}
	}
}

func TestPayloadSchemasDeclareBounds(t *testing.T) {
	for _, name := range payloadSchemaNames {
		raw, err := os.ReadFile(name + ".schema.json")
		if err != nil {
			t.Fatal(err)
		}
		var document any
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatal(err)
		}
		assertBounds(t, name, document)
	}
}

func assertBounds(t *testing.T, path string, value any) {
	t.Helper()
	switch node := value.(type) {
	case map[string]any:
		if node["type"] == "string" {
			if _, ok := node["maxLength"]; !ok {
				t.Errorf("%s string has no maxLength", path)
			}
		}
		if node["type"] == "array" {
			if _, ok := node["maxItems"]; !ok {
				t.Errorf("%s array has no maxItems", path)
			}
		}
		if node["type"] == "integer" {
			if _, ok := node["maximum"]; !ok {
				t.Errorf("%s integer has no maximum", path)
			}
		}
		for key, child := range node {
			assertBounds(t, path+"."+key, child)
		}
	case []any:
		for i, child := range node {
			assertBounds(t, path+"["+strconv.Itoa(i)+"]", child)
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
