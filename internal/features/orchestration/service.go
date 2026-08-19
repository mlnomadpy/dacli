package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

type serviceCheckpoint struct {
	Project    string    `json:"project"`
	Status     string    `json:"status"`
	Reason     string    `json:"reason"`
	Invocation int       `json:"invocation"`
	Failures   int       `json:"consecutive_infrastructure_failures"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func servicePath(w *workspace.Workspace, p OperatingProfile, suffix string) string {
	return filepath.Join(w.Root, workspace.Dir, "profiles", p.Project+"-service."+suffix)
}

func writeServiceCheckpoint(w *workspace.Workspace, p OperatingProfile, st serviceCheckpoint) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeStateFile(servicePath(w, p, "json"), string(b)+"\n")
}

func runService(ctx *clikit.Ctx, w *workspace.Workspace, p OperatingProfile, r runner) error {
	lease := servicePath(w, p, "lease")
	if err := os.MkdirAll(filepath.Dir(lease), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(lease, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return clikit.Refusedf("service lease is held (%s); inspect the checkpoint and lease owner before recovery", lease)
	}
	_, _ = fmt.Fprintf(f, "pid: %d\nstarted: %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	_ = f.Close()
	defer os.Remove(lease)

	failures := 0
	for invocation := 1; invocation <= p.Execution.ServiceInvocations; invocation++ {
		if _, err := os.Stat(filepath.Join(w.Root, p.Recovery.StopFile)); err == nil {
			st := serviceCheckpoint{Project: p.Project, Status: "halt", Reason: "STOP requested; remove the stop file and restart", Invocation: invocation, Failures: failures, UpdatedAt: time.Now().UTC()}
			_ = writeServiceCheckpoint(w, p, st)
			fmt.Fprintf(ctx.Stdout, "service halted at checkpoint: %s\n", st.Reason)
			return nil
		}
		if _, err := os.Stat(lease); err != nil {
			st := serviceCheckpoint{Project: p.Project, Status: "halt", Reason: "lease lost; inspect ownership before restart", Invocation: invocation, Failures: failures, UpdatedAt: time.Now().UTC()}
			_ = writeServiceCheckpoint(w, p, st)
			return clikit.Refusedf("%s", st.Reason)
		}
		_ = os.Chtimes(lease, time.Now(), time.Now())
		st := serviceCheckpoint{Project: p.Project, Status: "running", Reason: "bounded loop invocation", Invocation: invocation, Failures: failures, UpdatedAt: time.Now().UTC()}
		if err := writeServiceCheckpoint(w, p, st); err != nil {
			return fmt.Errorf("persist service checkpoint: %w", err)
		}
		out, runErr := r.run("bounded-loop", profileLoopArgs(p)...)
		fmt.Fprint(ctx.Stdout, out)
		if runErr == nil {
			if loop, stateErr := readLoopState(w, p.Project); stateErr == nil {
				switch {
				case loop.Status == SleepWindow.String():
					st.Status, st.Reason, st.UpdatedAt = "halt", "rolling budget exhausted; resume after the recorded window resets", time.Now().UTC()
					_ = writeServiceCheckpoint(w, p, st)
					return clikit.Refusedf("%s", st.Reason)
				case p.Recovery.UnknownLandingStops && strings.Contains(strings.ToLower(loop.Reason), "unknown landing"):
					st.Status, st.Reason, st.UpdatedAt = "halt", "unknown landing state; reconcile the PR and fetched trunk before restart", time.Now().UTC()
					_ = writeServiceCheckpoint(w, p, st)
					return clikit.Refusedf("%s", st.Reason)
				}
			}
		}
		if runErr != nil {
			failures++
		} else {
			failures = 0
		}
		if failures >= p.Recovery.InfrastructureFailureLimit {
			st.Status, st.Reason, st.Failures, st.UpdatedAt = "halt", "infrastructure circuit breaker opened; fix the repeated failure and restart", failures, time.Now().UTC()
			_ = writeServiceCheckpoint(w, p, st)
			return clikit.Refusedf("%s", st.Reason)
		}
	}
	st := serviceCheckpoint{Project: p.Project, Status: "halt", Reason: "finite service invocation bound reached; restart to continue", Invocation: p.Execution.ServiceInvocations, Failures: failures, UpdatedAt: time.Now().UTC()}
	if err := writeServiceCheckpoint(w, p, st); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "service checkpoint: %s\n", st.Reason)
	return nil
}
