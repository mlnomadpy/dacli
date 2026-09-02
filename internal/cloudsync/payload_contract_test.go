package cloudsync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validContractPayloads(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	path := filepath.Join("..", "..", "contracts", "controlplane", "v1", "testdata", "payloads", "valid.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payloads map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payloads); err != nil {
		t.Fatal(err)
	}
	return payloads
}

func TestNewContractPayloadFixturesMatchRuntimeValidator(t *testing.T) {
	payloads := validContractPayloads(t)
	want := []string{"agent_state", "budget_state", "device_registration", "event_summary", "gate_evidence", "project_registration", "role_bundle"}
	for _, eventType := range want {
		raw, ok := payloads[eventType]
		if !ok {
			t.Fatalf("missing valid fixture for %s", eventType)
		}
		if err := ValidatePayload(eventType, raw); err != nil {
			t.Errorf("valid %s: %v", eventType, err)
		}
	}
	if len(payloads) != len(want) {
		t.Fatalf("valid fixture registry has %d entries, want %d", len(payloads), len(want))
	}
}

func TestNewContractPayloadsFailClosedOnPrivacyAndBounds(t *testing.T) {
	for eventType, raw := range validContractPayloads(t) {
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		payload["prompt"] = "do not upload this"
		unsafe, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidatePayload(eventType, unsafe); !errors.Is(err, ErrInvalidPayload) {
			t.Errorf("%s unknown privacy field error = %v", eventType, err)
		}
	}

	tests := []struct {
		name      string
		eventType string
		mutate    func(map[string]any)
	}{
		{"device identifier bound", "device_registration", func(v map[string]any) { v["device_id"] = strings.Repeat("d", 129) }},
		{"device key size", "device_registration", func(v map[string]any) { v["public_key"] = "c2hvcnQ=" }},
		{"project state", "project_registration", func(v map[string]any) { v["state"] = "migrating" }},
		{"project nullable repository", "project_registration", func(v map[string]any) { v["repository_id"] = 7 }},
		{"role semver", "role_bundle", func(v map[string]any) { v["role_version"] = "latest" }},
		{"role capabilities bound", "role_bundle", func(v map[string]any) {
			items := make([]any, 65)
			for i := range items {
				items[i] = "capability-" + strings.Repeat("x", i%3)
			}
			v["capabilities"] = items
		}},
		{"event object type", "event_summary", func(v map[string]any) { v["object_type"] = "developer" }},
		{"budget integer maximum", "budget_state", func(v map[string]any) { v["limit"] = json.Number("9007199254740992") }},
		{"budget period ordering", "budget_state", func(v map[string]any) { v["period_end"] = v["period_start"] }},
		{"agent state", "agent_state", func(v map[string]any) { v["state"] = "unknown" }},
		{"gate commit", "gate_evidence", func(v map[string]any) { v["commit_id"] = "main" }},
	}
	fixtures := validContractPayloads(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var payload map[string]any
			decoder := json.NewDecoder(strings.NewReader(string(fixtures[test.eventType])))
			decoder.UseNumber()
			if err := decoder.Decode(&payload); err != nil {
				t.Fatal(err)
			}
			test.mutate(payload)
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidatePayload(test.eventType, raw); !errors.Is(err, ErrInvalidPayload) {
				t.Fatalf("error = %v, want ErrInvalidPayload", err)
			}
		})
	}
}

func TestProjectRegistrationAllowsExplicitRepositoryAbsence(t *testing.T) {
	var payload map[string]any
	raw := validContractPayloads(t)["project_registration"]
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	payload["repository_id"] = nil
	raw, _ = json.Marshal(payload)
	if err := ValidatePayload("project_registration", raw); err != nil {
		t.Fatalf("repository-less project registration: %v", err)
	}
}
