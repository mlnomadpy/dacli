package mcp

import (
	"reflect"
	"strings"
	"testing"
)

func TestCleanupRepositoryBuildsAuditedArtifactRestore(t *testing.T) {
	tl, ok := toolByName("cleanup_repository")
	if !ok {
		t.Fatal("cleanup_repository tool missing")
	}
	want := []string{"cleanup", "--project", "p", "--restore", "plan-1", "--artifact", "run:r1:tmp"}
	got, mutates, err := tl.build(map[string]any{
		"project":  "p",
		"restore":  "plan-1",
		"artifact": "run:r1:tmp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !mutates {
		t.Fatal("artifact restore must retain the mutating capability gate")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restore argv = %#v, want %#v", got, want)
	}
}

func TestCleanupRepositoryRestoreFailsClosedOnIncompleteOrMixedModes(t *testing.T) {
	tl, ok := toolByName("cleanup_repository")
	if !ok {
		t.Fatal("cleanup_repository tool missing")
	}
	cases := []map[string]any{
		{"project": "p", "restore": "plan-1"},
		{"project": "p", "artifact": "run:r1:tmp", "dry_run": true},
		{"project": "p", "restore": "plan-1", "artifact": "run:r1:tmp", "dry_run": true},
		{"project": "p", "restore": "plan-1", "artifact": "run:r1:tmp", "apply_safe": "plan-2"},
	}
	for _, args := range cases {
		if _, _, err := tl.build(args); err == nil || (!strings.Contains(err.Error(), "exactly") && !strings.Contains(err.Error(), "required")) {
			t.Fatalf("args %#v returned non-actionable error %v", args, err)
		}
	}
}
