package githubapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPilotManifestIsClosedLeastPrivilegeSet(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "github-app-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Public      bool              `json:"public"`
		Permissions map[string]string `json:"default_permissions"`
		Events      []string          `json:"default_events"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	wantPermissions := map[string]string{"metadata": "read", "issues": "read", "pull_requests": "read", "checks": "write"}
	if manifest.Public || !reflect.DeepEqual(manifest.Permissions, wantPermissions) {
		t.Fatalf("pilot public=%v permissions=%v, want private and exactly %v", manifest.Public, manifest.Permissions, wantPermissions)
	}
	wantEvents := []string{"installation", "installation_repositories", "issues", "issue_comment", "pull_request", "repository"}
	if !reflect.DeepEqual(manifest.Events, wantEvents) {
		t.Fatalf("events = %v, want exactly %v", manifest.Events, wantEvents)
	}
}
