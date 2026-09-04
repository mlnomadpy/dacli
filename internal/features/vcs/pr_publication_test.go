package vcs

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
)

func TestCanonicalPRPublicationPushesNamedBranchFromUnrelatedCheckout(t *testing.T) {
	dir, w, task := prIntegrateEnv(t)
	bare := addBareOrigin(t, dir)
	gitAt(t, dir, "push", "-q", "origin", "main")
	if got := gitAt(t, dir, "branch", "--show-current"); got != "main" {
		t.Fatalf("fixture checkout = %s, want unrelated main", got)
	}

	cp, err := publishCanonicalTaskBranch(w, task, "main")
	if err != nil {
		t.Fatal(err)
	}
	remote := strings.TrimSpace(runGitCommand(t, "--git-dir", bare, "rev-parse", "refs/heads/"+BranchFor(task)))
	if remote != cp.LocalOID || cp.RemoteOID != cp.LocalOID || cp.Stage != "pushed" {
		t.Fatalf("checkpoint=%+v remote=%s", cp, remote)
	}
	if got := gitAt(t, dir, "branch", "--show-current"); got != "main" {
		t.Fatalf("publication changed operator checkout to %s", got)
	}

	pushCalls := 0
	originalPush := publishTaskBranch
	publishTaskBranch = func(string, string) (string, error) {
		pushCalls++
		return "", errors.New("restart must not push an already exact ref")
	}
	t.Cleanup(func() { publishTaskBranch = originalPush })
	if _, err := publishCanonicalTaskBranch(w, task, "main"); err != nil {
		t.Fatalf("restart reuse: %v", err)
	}
	if pushCalls != 0 {
		t.Fatalf("restart issued %d duplicate pushes", pushCalls)
	}
}

func TestCanonicalPRPublicationRefusesNoCommitAndDivergentRemote(t *testing.T) {
	t.Run("no task commit", func(t *testing.T) {
		dir, w, task := prIntegrateEnv(t)
		gitAt(t, dir, "branch", "-f", BranchFor(task), "main")
		_, err := publishCanonicalTaskBranch(w, task, "main")
		if err == nil || !strings.Contains(err.Error(), "no commits beyond main") {
			t.Fatalf("no-change publication = %v", err)
		}
	})

	t.Run("remote divergence", func(t *testing.T) {
		dir, w, task := prIntegrateEnv(t)
		addBareOrigin(t, dir)
		gitAt(t, dir, "push", "-q", "origin", "main:refs/heads/"+BranchFor(task))
		_, err := publishCanonicalTaskBranch(w, task, "main")
		if err == nil || !strings.Contains(err.Error(), "divergent branch") || strings.Contains(err.Error(), "git push") {
			t.Fatalf("divergent publication = %v", err)
		}
		if !strings.Contains(err.Error(), "dacli push --task "+task.ID) {
			t.Fatalf("refusal lacks dacli-native remediation: %v", err)
		}
	})
}

func TestCanonicalPRPublicationPreservesPushDiagnostic(t *testing.T) {
	_, w, task := prIntegrateEnv(t)
	originalObserve, originalPush := observeRemoteBranch, publishTaskBranch
	observeRemoteBranch = func(string, string) (string, error) { return "", nil }
	sentinel := errors.New("authentication rejected")
	publishTaskBranch = func(string, string) (string, error) { return "", sentinel }
	t.Cleanup(func() { observeRemoteBranch, publishTaskBranch = originalObserve, originalPush })

	_, err := publishCanonicalTaskBranch(w, task, "main")
	if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "publish canonical branch") {
		t.Fatalf("push diagnostic was not preserved: %v", err)
	}
}

func runGitCommand(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestPRPublicationCheckpointRecordsCreatedPRForRestart(t *testing.T) {
	dir, w, task := prIntegrateEnv(t)
	addBareOrigin(t, dir)
	gitAt(t, dir, "push", "-q", "origin", "main")
	cp, err := publishCanonicalTaskBranch(w, task, "main")
	if err != nil {
		t.Fatal(err)
	}
	const url = "https://github.com/acme/widgets/pull/1022"
	if err := recordPRPublication(w, cp, url); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadPRPublication(w, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Stage != "pr-recorded" || loaded.PRURL != url || loaded.LocalOID != loaded.RemoteOID {
		t.Fatalf("restart checkpoint=%+v", loaded)
	}
}
