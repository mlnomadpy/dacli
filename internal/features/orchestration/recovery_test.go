package orchestration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func stubDeliveryPRs(t *testing.T, fn func(string) ([]store.DeliveryPR, error)) {
	t.Helper()
	original := store.ObserveDeliveryPRs
	store.ObserveDeliveryPRs = fn
	t.Cleanup(func() { store.ObserveDeliveryPRs = original })
}

func pendingRecoveryDriver(t *testing.T) (*driver, *store.Task) {
	t.Helper()
	w := loopEnv(t)
	commitTo(t, w.Root, "seed.txt")
	task, err := store.CreateTask(w, "a-root", "p", "Recover durable landing", store.TaskOpts{Accept: []string{"observable result"}})
	if err != nil {
		t.Fatal(err)
	}
	d := newDriver(w, &fakeRunner{}, &Governor{})
	d.pendingAccept = []pendingAccept{{Seq: task.Seq, Branch: taskBranch(task), Generation: task.Generation(), GenerationSet: true}}
	d.pendingLand = []string{taskBranch(task)}
	d.lastTrunkMarker, d.lastTrunkKnown = 7, true
	return d, task
}

func TestPreCycleRecoveryFailsClosedOnUnknownGitHubWithoutResettingGovernor(t *testing.T) {
	d, task := pendingRecoveryDriver(t)
	d.gov.Restore(governorState{Cycle: 4, WindowStart: time.Unix(900000, 0), WindowSpent: 321, ZeroStreak: 2})
	stubDeliveryPRs(t, func(string) ([]store.DeliveryPR, error) {
		return nil, fmt.Errorf("connection timed out")
	})

	cp, err := d.reconcileBeforeCycle()
	if err != nil {
		t.Fatal(err)
	}
	if cp == nil || cp.Schema != "" || cp.HaltClass != "transient-infrastructure-failure" || !cp.Retryable {
		t.Fatalf("unknown GitHub recovery = %+v", cp)
	}
	if !hasRecoveryRef(cp.AffectedRefs, "task", task.ID) && !hasRecoveryRef(cp.AffectedRefs, "project", "p") {
		t.Fatalf("checkpoint lacks exact affected ref: %+v", cp.AffectedRefs)
	}
	state := d.gov.State()
	if state.Cycle != 4 || state.WindowSpent != 321 || state.ZeroStreak != 2 {
		t.Fatalf("reconciliation reset governor counters: %+v", state)
	}
}

func TestPreCycleRecoveryAllowsUnmeasurableLocalGitWhenNothingExternalIsPending(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "non-git")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, "a-root", "P", "p", "goal", ""); err != nil {
		t.Fatal(err)
	}
	d := newDriver(w, &fakeRunner{}, &Governor{})
	cp, err := d.reconcileBeforeCycle()
	if err != nil || cp != nil {
		t.Fatalf("local git is advisory with no durable external landing: checkpoint=%+v err=%v", cp, err)
	}
}

func TestPreCycleRecoveryDistinguishesExternalBlockersAndResumesOnMerge(t *testing.T) {
	pendingPR := func(number int, checkName, status, conclusion string) store.DeliveryPR {
		p := store.DeliveryPR{Number: number, DeliveryConfidence: "OPEN", URL: fmt.Sprintf("https://example.test/%d", number)}
		p.StatusCheckRollup = append(p.StatusCheckRollup, struct {
			DeliveryConfidence string `json:"state"`
			Conclusion         string `json:"conclusion"`
			Name               string `json:"name"`
		}{DeliveryConfidence: status, Conclusion: conclusion, Name: checkName})
		return p
	}
	for _, tc := range []struct {
		name string
		prs  []store.DeliveryPR
		want string
	}{
		{name: "missing canonical PR", want: "missing_canonical_pr"},
		{name: "closed unmerged", prs: []store.DeliveryPR{{Number: 41, DeliveryConfidence: "CLOSED", URL: "https://example.test/41"}}, want: "closed_unmerged"},
		{name: "account restriction", prs: []store.DeliveryPR{pendingPR(42, "billing restriction", "COMPLETED", "FAILURE")}, want: "billing_restriction"},
		{name: "pending CI", prs: []store.DeliveryPR{pendingPR(43, "linux", "IN_PROGRESS", "")}, want: "ci_pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, task := pendingRecoveryDriver(t)
			for i := range tc.prs {
				tc.prs[i].HeadRefName = taskBranch(task)
				tc.prs[i].HeadRefOid = "head"
			}
			stubDeliveryPRs(t, func(string) ([]store.DeliveryPR, error) { return tc.prs, nil })
			cp, err := d.reconcileBeforeCycle()
			if err != nil || cp == nil || cp.HaltClass != "external-blocker" {
				t.Fatalf("checkpoint=%+v err=%v", cp, err)
			}
			seen := false
			for _, observation := range cp.Observed {
				seen = seen || observation.State == tc.want
			}
			if !seen {
				t.Fatalf("diagnosis %q absent: %+v", tc.want, cp.Observed)
			}
			if tc.prs != nil && !hasRecoveryRef(cp.AffectedRefs, "pull_request", fmt.Sprintf("#%d", tc.prs[0].Number)) {
				t.Fatalf("exact PR ref absent: %+v", cp.AffectedRefs)
			}
		})
	}

	d, task := pendingRecoveryDriver(t)
	d.gov.Restore(governorState{Cycle: 8, WindowSpent: 654, ZeroStreak: 3})
	if err := writeLoopRecovery(d.w, loopRecoveryCheckpoint{Project: "p", Cycle: 8, Checkpoint: "pre-cycle-reconciliation", HaltClass: "external-blocker", AffectedRefs: []recoveryRef{{Kind: "pull_request", ID: "#44"}}, NextAction: "wait", Reason: "pending", ObservedAt: time.Unix(1, 0)}); err != nil {
		t.Fatal(err)
	}
	stubDeliveryPRs(t, func(string) ([]store.DeliveryPR, error) {
		return []store.DeliveryPR{{Number: 44, DeliveryConfidence: "MERGED", URL: "https://example.test/44", HeadRefName: taskBranch(task), HeadRefOid: "head"}}, nil
	})
	cp, err := d.reconcileBeforeCycle()
	if err != nil || cp != nil {
		t.Fatalf("observed merge should resume existing reconciliation, checkpoint=%+v err=%v", cp, err)
	}
	state := d.gov.State()
	if state.Cycle != 8 || state.WindowSpent != 654 || state.ZeroStreak != 3 {
		t.Fatalf("external recovery cleared counters: %+v", state)
	}
	if !strings.Contains(d.recovery, "observed prior external-blocker resolved") || !strings.Contains(d.recovery, "without resetting") {
		t.Fatalf("resolved external condition was not recognized: %q", d.recovery)
	}
}

func TestLoopStatusJSONReportsTypedCheckpointAndPreservedCounters(t *testing.T) {
	w := loopEnv(t)
	writeLoopState(w, loopState{Project: "p", Cycle: 9, TrunkMarker: 17, WindowTokens: 800, Backlog: 2, Status: "halt", Reason: "waiting", UpdatedAt: time.Unix(100, 0)})
	cp := loopRecoveryCheckpoint{Project: "p", Cycle: 9, Checkpoint: "pre-cycle-reconciliation", HaltClass: "external-blocker", AffectedRefs: []recoveryRef{{Kind: "pull_request", ID: "#51"}}, TrunkMarker: 17, TrunkKnown: true, Retryable: true, NextAction: "wait for check linux", Reason: "pending CI", ObservedAt: time.Unix(100, 0)}
	if err := writeLoopRecovery(w, cp); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdLoopStatus(ctx, []string{"--project", "p"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("status was not JSON: %v\n%s", err, out)
	}
	for key, want := range map[string]any{"schema": loopRecoverySchema, "checkpoint": "pre-cycle-reconciliation", "halt_class": "external-blocker", "window_tokens": float64(800)} {
		if got[key] != want {
			t.Errorf("%s=%v, want %v", key, got[key], want)
		}
	}
}

func TestReadLoopRecoveryRejectsUnknownOrTornCheckpoint(t *testing.T) {
	w := loopEnv(t)
	path := loopRecoveryFile(w, "p")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema":"loop-recovery/v999","version":999}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readLoopRecovery(w, "p"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unknown schema did not fail closed: %v", err)
	}
}

func hasRecoveryRef(refs []recoveryRef, kind, id string) bool {
	for _, ref := range refs {
		if ref.Kind == kind && ref.ID == id {
			return true
		}
	}
	return false
}
