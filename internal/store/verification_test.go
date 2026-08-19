package store

import (
	"os"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestRunVerificationCapturesCompleteProvenance(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-b", "evidence-branch"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
		if out, err := gitx.Run(w.Root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(w.Root+"/artifact.txt", []byte("seed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "artifact.txt"}, {"commit", "-m", "seed"}} {
		if out, err := gitx.Run(w.Root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	ev, out, err := RunVerification(w.Root, "a-verifier", "printf artifact")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "artifact" || ev.Command != "printf artifact" || ev.ExitCode != 0 || ev.DurationMS < 0 || ev.ArtifactHash == "" || ev.Verifier != "a-verifier" || ev.Branch != "evidence-branch" || ev.CommitSHA == "" {
		t.Fatalf("incomplete verification provenance: output=%q evidence=%#v", out, ev)
	}
}

func TestRunVerificationKeepsUnknownProvenanceOutsideGit(t *testing.T) {
	dir := t.TempDir()
	ev, out, err := RunVerification(dir, "a-verifier", "printf artifact")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "artifact" || ev.Branch != "" || ev.CommitSHA != "" || ev.ArtifactHash == "" {
		t.Fatalf("non-git verification fabricated provenance: output=%q evidence=%#v", out, ev)
	}
}

func TestVerificationEvidenceRoundTripAndLegacyMigration(t *testing.T) {
	doc := &mdstore.Doc{}
	tk := &Task{Doc: doc}
	want := VerificationEvidence{
		Command:      "go test ./...",
		ExitCode:     0,
		DurationMS:   17,
		ArtifactHash: "sha256:abc123",
		Verifier:     "a-reviewer",
		Branch:       "dacli/432-evidence",
		CommitSHA:    "deadbeef",
	}
	if err := AppendVerificationEvidence(tk, want); err != nil {
		t.Fatal(err)
	}
	got := VerificationEvidenceRecords(tk)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}

	legacy := &Task{Doc: &mdstore.Doc{Sections: []mdstore.Section{{Level: 2, Title: "Log", Content: "- 2026-08-13T10:00:00Z verified by `go test ./...` (exit 0) in branch old at abc123\n"}}}}
	records := VerificationEvidenceRecords(legacy)
	if len(records) != 1 {
		t.Fatalf("legacy records = %#v, want one readable record", records)
	}
	if records[0].Legacy == "" || records[0].Command != "" || records[0].ArtifactHash != "" || records[0].Verifier != "" || records[0].Branch != "" || records[0].CommitSHA != "" || records[0].DurationMS != 0 {
		t.Fatalf("legacy parser fabricated structured fields: %#v", records[0])
	}
}

func TestCommandCriterionRequiresCompleteVerificationEvidence(t *testing.T) {
	tk := &Task{Doc: &mdstore.Doc{Sections: []mdstore.Section{{Level: 2, Title: "Acceptance", Content: "- [ ] `go test ./...` exits zero\n"}}}}
	if !AcceptanceRequiresCommandVerification(tk, 1) {
		t.Fatal("inline command criterion was not recognized as command verification")
	}
	for _, evidence := range []VerificationEvidence{
		{Command: "go test ./...", ExitCode: 0, Verifier: "a-reviewer"},
		{Command: "go test ./...", ExitCode: 0, ArtifactHash: "sha256:abc"},
	} {
		if err := ValidateCommandVerification(evidence); err == nil || !strings.Contains(err.Error(), "missing") {
			t.Fatalf("incomplete evidence %#v accepted: %v", evidence, err)
		}
	}
}
