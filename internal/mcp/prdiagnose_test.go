package mcp

import (
	"reflect"
	"testing"
)

func TestDiagnosePRBuildsTypedJSONCommand(t *testing.T) {
	tl, ok := toolByName("diagnose_pr")
	if !ok {
		t.Fatal("diagnose_pr tool missing")
	}
	got, jsonMode, err := tl.build(map[string]any{"task": "858"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"pr", "diagnose", "--task", "858"}
	if !reflect.DeepEqual(got, want) || !jsonMode {
		t.Fatalf("argv=%#v json=%v, want %#v true", got, jsonMode, want)
	}
}
