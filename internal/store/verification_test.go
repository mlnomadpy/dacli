package store

import (
	"os"
	"path/filepath"
	"reflect"
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
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", "seed"}} {
		if out, err := gitx.Run(w.Root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	ev, out, err := RunAcceptanceVerification(w.Root, "a-verifier", "printf artifact")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "artifact" || ev.Command != "printf artifact" || ev.ExitCode != 0 || ev.DurationMS < 0 || ev.ArtifactHash == "" || ev.Verifier != "a-verifier" || ev.Branch != "evidence-branch" || ev.CommitSHA == "" || ev.TreeSHA == "" || !ev.Clean || len(ev.Argv) != 3 || ev.WorkingDirectory != w.Root || len(ev.RuntimeVersions) == 0 || len(ev.ToolVersions) == 0 {
		t.Fatalf("incomplete verification provenance: output=%q evidence=%#v", out, ev)
	}
	if err := ValidateFinalTreeVerification(ev, ev.CommitSHA, ev.TreeSHA); err != nil {
		t.Fatalf("fresh immutable evidence rejected: %v", err)
	}
}

func TestRunVerificationKeepsUnknownProvenanceOutsideGit(t *testing.T) {
	dir := t.TempDir()
	ev, out, err := RunAdvisoryVerification(dir, "a-verifier", "printf artifact")
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
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
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

func TestAcceptanceVerificationRefusesDirtyParentTreeBeforeRunning(t *testing.T) {
	dir := verificationRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "command-ran")
	ev, _, err := RunAcceptanceVerification(dir, "reviewer", "touch command-ran")
	if err == nil || !IsVerificationPolicyError(err) || !strings.Contains(err.Error(), "working tree is dirty") {
		t.Fatalf("dirty tree result = %#v, %v; want policy refusal naming dirty tree", ev, err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("verification command ran despite dirty preflight: %v", statErr)
	}
	if ev.Clean {
		t.Fatalf("dirty parent-SHA evidence asserted clean: %#v", ev)
	}
}

func TestAcceptanceVerificationRejectsTreeMutationDuringCommand(t *testing.T) {
	dir := verificationRepo(t)
	ev, _, err := RunAcceptanceVerification(dir, "reviewer", "printf mutation >> source.txt")
	if err == nil || !IsVerificationPolicyError(err) {
		t.Fatalf("mutating verification result = %#v, %v; want policy refusal", ev, err)
	}
	for _, want := range []string{"changed during execution", "before commit", "after commit", "clean=true", "clean=false"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("mutation refusal %q missing %q", err, want)
		}
	}
	if err := ValidateFinalTreeVerification(ev, ev.CommitSHA, ev.TreeSHA); err != nil {
		t.Fatalf("captured before-state should itself be structurally complete: %v", err)
	}
}

func TestFinalTreeGateRejectsHistoricalAndStaleEvidence(t *testing.T) {
	legacy := VerificationEvidence{Legacy: "- verified by old text"}
	if err := ValidateFinalTreeVerification(legacy, "reviewed", "tree"); err == nil || !strings.Contains(err.Error(), "not acceptance-grade") {
		t.Fatalf("legacy gate = %v, want explicit non-acceptance-grade rejection", err)
	}
	stale := VerificationEvidence{CommitSHA: "old", TreeSHA: "old-tree", Clean: true}
	if err := ValidateFinalTreeVerification(stale, "reviewed", "reviewed-tree"); err == nil || !strings.Contains(err.Error(), "rerun verification on the reviewed head") {
		t.Fatalf("stale gate = %v, want actionable reviewed-head rejection", err)
	}
}

func TestExternalVerificationCannotTreatSkippedAsGreen(t *testing.T) {
	var ev VerificationEvidence
	err := AttachExternalVerification(&ev, ExternalVerificationEvidence{Provider: "github", WorkflowRunID: "123", State: "skipped", Conclusion: "success", SkipReason: "not triggered"})
	if err == nil || !strings.Contains(err.Error(), "cannot carry conclusion") || len(ev.External) != 0 {
		t.Fatalf("skipped green attachment = %#v, %v; want rejection without append", ev.External, err)
	}
	if err := AttachExternalVerification(&ev, ExternalVerificationEvidence{Provider: "github", WorkflowRunID: "123", CheckRunID: "456", State: "observed", Conclusion: "success"}); err != nil {
		t.Fatalf("observed GitHub IDs rejected: %v", err)
	}
	if err := AttachExternalVerification(&ev, ExternalVerificationEvidence{Provider: "github", State: "unobservable", SkipReason: "API unavailable"}); err != nil {
		t.Fatalf("explicitly unobservable check rejected: %v", err)
	}
}

func verificationRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}} {
		if out, err := gitx.Run(dir, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("seed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "source.txt"}, {"commit", "-m", "seed"}} {
		if out, err := gitx.Run(dir, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return dir
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
