package store

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestPRPublicationCheckpointRoundTripAndTamperRefusal(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "pr-publication")
	if err != nil {
		t.Fatal(err)
	}
	cp := PRPublication{Schema: PRPublicationSchema, TaskID: "t-1", Branch: "dacli/001-x", Base: "main", LocalOID: strings.Repeat("a", 40), RemoteOID: strings.Repeat("a", 40), Stage: "pushed", UpdatedAt: time.Now().UTC()}
	if err := SavePRPublication(w, cp); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPRPublication(w, cp.TaskID)
	if err != nil || got.LocalOID != cp.LocalOID || got.Stage != "pushed" {
		t.Fatalf("round trip=%+v err=%v", got, err)
	}
	if err := os.WriteFile(PRPublicationPath(w, cp.TaskID), []byte(`{"schema":"pr-publication/v1","task_id":"t-1","branch":"dacli/001-x","base":"main","local_oid":"a","stage":"pr-recorded","updated_at":"2026-01-01T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPRPublication(w, cp.TaskID); err == nil {
		t.Fatal("incomplete pr-recorded checkpoint was accepted")
	}
}
