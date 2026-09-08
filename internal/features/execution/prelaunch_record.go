package execution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// prelaunchRun makes identity minting and run recovery one transaction. The
// process has no PID until the runtime starts, but its child/task/worktree
// ownership must already be durable so any intervening refusal can release it.
type prelaunchRun struct {
	runID    string
	runDir   string
	procPath string
	record   runRecord
	proc     procmon.Record
	pending  bool
}

func beginPrelaunchRun(ctx *clikit.Ctx, w *workspace.Workspace, task *store.Task, child, role, runtime string, timeout int, claims []string) (*prelaunchRun, error) {
	runID := ulid.New()
	ctx.Result = commandresult.Spawn{RunID: runID}
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	run := &prelaunchRun{
		runID: runID, runDir: runDir, procPath: filepath.Join(runDir, "proc.txt"),
		record: openRunRecord(runDir, ctx.Stderr), pending: true,
		proc: procmon.Record{
			RunID: runID, Child: child, Task: task.ID, Role: role, Runtime: runtime,
			Started: time.Now(), Timeout: time.Duration(timeout) * time.Second, Claims: claims,
		},
	}
	if err := procmon.WriteRecord(run.procPath, run.proc); err != nil {
		return nil, fmt.Errorf("record critical prelaunch artifact proc.txt: %w", err)
	}
	return run, nil
}

func (r *prelaunchRun) finalize(retErr *error) {
	if retErr == nil || *retErr == nil || !r.pending {
		return
	}
	reason := clikit.ErrStr(*retErr)
	if err := r.record.critical("outcome.md", fmt.Sprintf("outcome: prelaunch-failed\nexit: %s\nphase: before-runtime-start\n", reason)); err != nil {
		*retErr = errors.Join(*retErr, err)
	}
	if err := procmon.CompleteRecord(r.procPath, r.proc, "prelaunch-failed"); err != nil {
		*retErr = errors.Join(*retErr, fmt.Errorf("finalize critical prelaunch artifact proc.txt: %w", err))
	}
}
