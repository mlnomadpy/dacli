package acceptance

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func evidenceEnv(t *testing.T) (*clikit.Ctx, *workspace.Workspace, *agentid.Identity) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "Core", "core", "", ""); err != nil {
		t.Fatal(err)
	}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	return ctx, w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW, Role: "root"}
}

// taskLog returns the task's "## Log" section text.
func taskLog(t *testing.T, tk *store.Task) string {
	t.Helper()
	s, ok := tk.Doc.Section("Log")
	if !ok {
		return ""
	}
	return s.Content
}

func mkTask(t *testing.T, w *workspace.Workspace, title string) *store.Task {
	t.Helper()
	tk, err := store.CreateTask(w, agentid.RootID, "core", title, store.TaskOpts{Accept: []string{"it works"}})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

// A close must record WHAT certified it. Before this, an unverified close was
// indistinguishable from a verified one in the record — which is what made
// every `done` label an unverified assertion (dacli 184).
func TestCloseRecordsVerificationEvidence(t *testing.T) {
	ctx, w, root := evidenceEnv(t)

	unverified := mkTask(t, w, "closed with no check")
	if err := acceptOne(ctx, w, root, unverified, "", false, false); err != nil {
		t.Fatal(err)
	}
	got, err := store.FindTask(w, unverified.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if body := taskLog(t, got); !strings.Contains(body, "WITHOUT verification") {
		t.Errorf("an unverified close must say so in the task log; got:\n%s", body)
	}

	verified := mkTask(t, w, "closed with a real check")
	if err := acceptOne(ctx, w, root, verified, "true", false, false); err != nil {
		t.Fatal(err)
	}
	got2, err := store.FindTask(w, verified.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if body := taskLog(t, got2); !strings.Contains(body, "verified by") {
		t.Errorf("a verified close must record the command; got:\n%s", body)
	}
}

// --require-verify makes an unverified close impossible rather than merely
// visible: the strict mode for generating repos whose record is the product.
func TestRequireVerifyRefusesUnverifiedClose(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk := mkTask(t, w, "must not close unverified")

	err := acceptOne(ctx, w, root, tk, "", true, false)
	if err == nil {
		t.Fatal("acceptOne with requireVerify and no command must refuse")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refused by policy)", clikit.ExitCode(err))
	}
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Error("a refused close must leave the task open")
	}
}

// The implementer marking its own work complete is the failure the role split
// exists to prevent, and nothing enforced it: the claimant owned the task and
// accepted it (dacli 188).
func TestRequireIndependentBlocksSelfCertification(t *testing.T) {
	ctx, w, _ := evidenceEnv(t)
	tk := mkTask(t, w, "self certified")

	// The worker claims the task, then tries to certify its own work.
	worker := &agentid.Identity{ID: "a-worker", Grant: model.GrantRW, Role: "fixer"}
	store.AppendLog(tk, "claimed by "+worker.ID)
	tk.Doc.Front.Set("owner", worker.ID)
	if err := store.SaveTask(tk); err != nil {
		t.Fatal(err)
	}

	err := acceptOne(ctx, w, worker, tk, "true", false, true)
	if err == nil {
		t.Fatal("the claimant must not be able to certify its own task under --require-independent")
	}
	if clikit.ExitCode(err) != 3 {
		t.Errorf("exit code = %d, want 3 (refused by policy)", clikit.ExitCode(err))
	}

	// A DIFFERENT agent certifying the same task is fine.
	reviewer := &agentid.Identity{ID: "a-reviewer", Grant: model.GrantRW, Role: "reviewer"}
	if err := acceptOne(ctx, w, reviewer, tk, "true", false, true); err != nil {
		t.Fatalf("an independent certifier must be allowed: %v", err)
	}
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusDone {
		t.Errorf("status = %s, want done after independent certification", got.Status)
	}
}

// A failing verification must never close the task.
func TestFailedVerificationLeavesTaskOpen(t *testing.T) {
	ctx, w, root := evidenceEnv(t)
	tk := mkTask(t, w, "verification fails")

	if err := acceptOne(ctx, w, root, tk, "exit 1", false, false); err == nil {
		t.Fatal("a non-zero verify command must refuse the close")
	}
	got, err := store.FindTask(w, tk.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == model.StatusDone {
		t.Error("a task whose verification failed must stay open")
	}
}
