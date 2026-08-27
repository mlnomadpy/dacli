package orchestration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

func TestLoopDryRunNamesEffectiveLandingContract(t *testing.T) {
	w := noRemoteRepo(t)
	p, err := store.CreateProject(w, "a-root", "P", "p", "g", "", model.LandingPolicy{Mode: model.LandingLocal, Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdLoop(ctx, []string{"--project", "p", "--dry-run", "--max-cycles", "1"}); err != nil {
		t.Fatal(err)
	}
	line := out.String()
	for _, want := range []string{"mode=local", "base=main", "override=false", "PR action=local merge", "required gates=task acceptance"} {
		if !strings.Contains(line, want) {
			t.Fatalf("dry-run omitted %q:\n%s", want, line)
		}
	}
}

func TestLoopRefusesConfiguredPRWithoutRemoteAndLeavesTaskOpen(t *testing.T) {
	w := noRemoteRepo(t)
	p, err := store.CreateProject(w, "a-root", "P", "p", "g", "", model.LandingPolicy{Mode: model.LandingPR, Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(w, "a-root", "p", "Work", store.TaskOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	err = cmdLoop(ctx, []string{"--project", "p", "--dry-run", "--max-cycles", "1"})
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "no `origin`") {
		t.Fatalf("PR policy without remote = %v (exit %d), want recoverable refusal", err, clikit.ExitCode(err))
	}
	got, findErr := store.FindTask(w, task.ID)
	if findErr != nil || got.Status != model.StatusOpen {
		t.Fatalf("refusal changed task: %+v err=%v", got, findErr)
	}
}

func TestLoopResolvesAndForwardsProjectLandingPolicy(t *testing.T) {
	w := noRemoteRepo(t)
	p, err := store.CreateProject(w, "a-root", "P", "p", "g", "", model.LandingPolicy{Mode: model.LandingPR, Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	f, _ := clikit.ParseFlags(nil)
	policy, explicit, err := resolveLoopLanding(w, "p", f, cycleJournal{})
	if err != nil {
		t.Fatal(err)
	}
	if policy != (model.LandingPolicy{Mode: model.LandingPR, Base: "main"}) || explicit {
		t.Fatalf("effective policy = %+v override=%t", policy, explicit)
	}
	d := &driver{cfg: loopCfg{landing: policy, landingExplicit: true}, trunkBranch: "main"}
	args := d.shipArgs("--project", "p")
	if !contains(args, "--landing-mode") || !contains(args, "pr") || !contains(args, "--landing-base") {
		t.Fatalf("ship boundary did not receive effective policy: %v", args)
	}
}

func TestLoopRestartKeepsResolvedLandingPolicyUnlessExplicitlyOverridden(t *testing.T) {
	w := noRemoteRepo(t)
	p, err := store.CreateProject(w, "a-root", "P", "p", "g", "", model.LandingPolicy{Mode: model.LandingLocal, Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	checkpoint := cycleJournal{Landing: model.LandingPolicy{Mode: model.LandingPR, Base: "release"}, LandingExplicit: true}
	f, _ := clikit.ParseFlags(nil)
	got, explicit, err := resolveLoopLanding(w, "p", f, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if got != checkpoint.Landing || !explicit {
		t.Fatalf("restart changed checkpoint policy: %+v override=%t", got, explicit)
	}
	off, _ := clikit.ParseFlags([]string{"--no-pr", "--into", "main"})
	got, explicit, err = resolveLoopLanding(w, "p", off, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != model.LandingLocal || got.Base != "main" || !explicit {
		t.Fatalf("explicit recovery override lost: %+v override=%t", got, explicit)
	}
}

func TestBareLoopRefusesJournalContradictingPersistedProfile(t *testing.T) {
	w := noRemoteRepo(t)
	project, err := store.CreateProject(w, "a-root", "P", "p", "g", "", model.LandingPolicy{Mode: model.LandingPR, Base: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	profile, err := defaultProfile("p", "loop")
	if err != nil {
		t.Fatal(err)
	}
	profile.Verification.Commands = []string{"python -m pytest"}
	profile.Landing = LandingPolicy{Mode: "project", ProtectedBranch: "dev", AutoMerge: false}
	if err := saveProfile(w, profile); err != nil {
		t.Fatal(err)
	}
	journal := cycleJournal{Landing: model.LandingPolicy{Mode: model.LandingPR, Base: "main"}}
	f, _ := clikit.ParseFlags(nil)
	_, _, err = resolveLoopLanding(w, "p", f, journal)
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "stale loop journal") || !strings.Contains(err.Error(), "--into") {
		t.Fatalf("contradictory journal = %v (exit %d), want actionable refusal", err, clikit.ExitCode(err))
	}

	override, _ := clikit.ParseFlags([]string{"--pr", "--into", "dev"})
	got, explicit, err := resolveLoopLanding(w, "p", override, journal)
	if err != nil || !explicit || got != (model.LandingPolicy{Mode: model.LandingPR, Base: "dev"}) {
		t.Fatalf("explicit recovery override = %+v explicit=%t err=%v", got, explicit, err)
	}
}

func TestCycleJournalPersistsLandingAcrossEveryRemoteCheckpoint(t *testing.T) {
	w := journalWS(t)
	for _, checkpoint := range []string{"pushed", "pr-created", "checks-pending", "merged"} {
		t.Run(checkpoint, func(t *testing.T) {
			want := cycleJournal{PendingAccept: []pendingAccept{{Seq: 8, Branch: "dacli/008-work"}}, PendingLand: []string{"dacli/008-work"}, Landing: model.LandingPolicy{Mode: model.LandingPR, Base: "main"}, LandingExplicit: true}
			mustWriteCycleJournal(t, w, checkpoint, want)
			got, warnings := readCycleJournal(w, checkpoint)
			if len(warnings) != 0 || got.Landing != want.Landing || !got.LandingExplicit || got.PendingAccept[0] != want.PendingAccept[0] {
				t.Fatalf("restart at %s = %+v warnings=%v", checkpoint, got, warnings)
			}
		})
	}
}
