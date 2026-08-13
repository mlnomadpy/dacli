package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStableToolSchemasMatchGoldenFixtures(t *testing.T) {
	for _, tl := range tools {
		t.Run(tl.name, func(t *testing.T) {
			path := filepath.Join("testdata", tl.name+".json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden schema %s: %v", path, err)
			}
			var golden map[string]any
			if err := json.Unmarshal(raw, &golden); err != nil {
				t.Fatalf("parse golden schema %s: %v", path, err)
			}
			// Normalize Go-specific int and []string values to the same shapes
			// that clients and fixtures observe over JSON.
			currentJSON, err := json.Marshal(tl.schema)
			if err != nil {
				t.Fatal(err)
			}
			var current map[string]any
			if err := json.Unmarshal(currentJSON, &current); err != nil {
				t.Fatal(err)
			}
			if err := compatibleSchema("$", golden, current); err != nil {
				t.Error(err)
			}
		})
	}
}

// compatibleSchema enforces the additive-only promise recursively:
// every published field and type remains, while new fields are ignored.
func compatibleSchema(path string, stable, current any) error {
	switch want := stable.(type) {
	case map[string]any:
		got, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("%s changed type from object to %T", path, current)
		}
		for key, value := range want {
			actual, exists := got[key]
			if !exists {
				return fmt.Errorf("%s.%s was removed or renamed", path, key)
			}
			if err := compatibleSchema(path+"."+key, value, actual); err != nil {
				return err
			}
		}
	case []any:
		got, ok := current.([]any)
		if !ok {
			return fmt.Errorf("%s changed type from array to %T", path, current)
		}
		for _, required := range want {
			if !containsJSONValue(got, required) {
				return fmt.Errorf("%s no longer contains stable value %v", path, required)
			}
		}
	default:
		if want != current {
			return fmt.Errorf("%s changed from %v to %v", path, want, current)
		}
	}
	return nil
}

func containsJSONValue(values []any, want any) bool {
	for _, got := range values {
		if got == want {
			return true
		}
	}
	return false
}

func TestCompatibilityRuleAllowsOnlyAdditiveChanges(t *testing.T) {
	stable := map[string]any{
		"type":     "object",
		"required": []any{"schema_version", "ref"},
		"properties": map[string]any{
			"schema_version": map[string]any{"type": "integer", "const": float64(1)},
			"ref":            map[string]any{"type": "string"},
		},
	}
	additive := map[string]any{
		"type":     "object",
		"required": []any{"schema_version", "ref", "optional_later"},
		"properties": map[string]any{
			"schema_version": map[string]any{"type": "integer", "const": float64(1), "description": "added"},
			"ref":            map[string]any{"type": "string", "description": "added"},
			"optional_later": map[string]any{"type": "boolean"},
		},
		"description": "added",
	}
	if err := compatibleSchema("$", stable, additive); err != nil {
		t.Fatalf("additive schema change was rejected: %v", err)
	}

	for _, tc := range []struct {
		name    string
		current map[string]any
	}{
		{"removed", map[string]any{"type": "object", "required": []any{"schema_version", "ref"}, "properties": map[string]any{"schema_version": map[string]any{"type": "integer", "const": float64(1)}}}},
		{"renamed", map[string]any{"type": "object", "required": []any{"schema_version", "task_ref"}, "properties": map[string]any{"schema_version": map[string]any{"type": "integer", "const": float64(1)}, "task_ref": map[string]any{"type": "string"}}}},
		{"retyped", map[string]any{"type": "object", "required": []any{"schema_version", "ref"}, "properties": map[string]any{"schema_version": map[string]any{"type": "integer", "const": float64(1)}, "ref": map[string]any{"type": "integer"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := compatibleSchema("$", stable, tc.current); err == nil {
				t.Fatalf("%s stable field was accepted", tc.name)
			}
		})
	}
}

func TestSchemaVersionIsValidatedBeforeExecution(t *testing.T) {
	mutating, ok := toolByName("finish_task")
	if !ok {
		t.Fatal("finish_task missing from stable tool table")
	}
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"malformed", map[string]any{"schema_version": "one", "ref": "001"}},
		{"unsupported", map[string]any{"schema_version": float64(2), "ref": "001"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			executed := false
			res := call(mutating, tc.args, func([]string, bool) (string, string, int) {
				executed = true
				return "", "", 0
			})
			if !res.IsError {
				t.Fatalf("version error was accepted: %+v", res)
			}
			if executed {
				t.Fatal("mutating executor ran before schema-version validation")
			}
		})
	}
}
