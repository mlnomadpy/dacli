package store

import (
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestReleaseTrainDurableRoundTripAndIdentity(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "train")
	if err != nil {
		t.Fatal(err)
	}
	tx := ReleaseTrain{Schema: ReleaseTrainSchema, Project: "core", Source: "dev", Target: "main", SourceSHA: "source", TargetSHA: "target", Phase: "pr-created", PullRequest: 42}
	if err := WriteReleaseTrain(w, tx); err != nil {
		t.Fatal(err)
	}
	got, err := ReadReleaseTrain(w, "core", "dev", "main")
	if err != nil {
		t.Fatal(err)
	}
	if got.PullRequest != 42 || got.UpdatedAt.IsZero() {
		t.Fatalf("round trip = %+v", got)
	}
	if err := WriteReleaseTrain(w, ReleaseTrain{Schema: ReleaseTrainSchema, Project: "../escape", Source: "dev", Target: "main", SourceSHA: "s", TargetSHA: "t", Phase: "planned"}); err == nil {
		t.Fatal("unsafe identity accepted")
	}
	if err := WriteReleaseTrain(w, ReleaseTrain{Schema: ReleaseTrainSchema, Project: "core", Source: "release/v2", Target: "production/main", SourceSHA: "s", TargetSHA: "t", Phase: "planned"}); err != nil {
		t.Fatalf("valid slash branches rejected: %v", err)
	}
	path, err := releaseTrainPath(w, "core", "dev", "main")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema":"wrong"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadReleaseTrain(w, "core", "dev", "main"); err == nil {
		t.Fatal("corrupt transaction accepted")
	}
}
