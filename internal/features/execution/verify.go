// The verification panel (RUNTIMES.md § 10): the strongest argument for
// supporting many runtimes. A finding confirmed by two different-vendor
// models is meaningfully stronger evidence than three samples of one model,
// because the failure modes are uncorrelated.
package execution

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/ulid"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func init() {
	Commands = append(Commands, clikit.Command{
		Path: "verify", Brief: "Adversarial panel: one refuter per runtime; tally derived from the log", Run: cmdVerify,
	})
}

func cmdVerify(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	taskRef := f.Get("task")
	panel := strings.Split(f.Get("panel"), ",")
	if taskRef == "" || f.Get("panel") == "" {
		return clikit.Usagef("usage: dacli verify --task <ref> --panel rt1,rt2[,rt3] [--claim text] [--require N] [--grant ro|rw] [--budget N] [--timeout sec] [--cooperative]")
	}
	t, err := store.FindTask(w, taskRef)
	if err != nil {
		return err
	}

	claim := f.Get("claim")
	if claim == "" {
		claim = latestFinding(w, t)
	}
	if claim == "" {
		return clikit.Usagef("nothing to verify: task %03d has no findings — pass --claim", t.Seq)
	}

	// Majority by default: 2-of-3, 2-of-2, 1-of-1.
	require := len(panel)/2 + 1
	if n, _ := strconv.Atoi(f.Get("require")); n > 0 {
		require = n
	}
	timeout := 300
	if n, _ := strconv.Atoi(f.Get("timeout")); n > 0 {
		timeout = n
	}
	budget, _ := strconv.Atoi(f.Get("budget"))
	grant := model.Grant(clikit.OrDash(f.Get("grant"), string(model.GrantRO)))

	// The diversity warning: a panel drawn from one runtime looks like
	// verification and is really repetition.
	unique := map[string]bool{}
	for _, p := range panel {
		unique[strings.TrimSpace(p)] = true
	}
	if len(unique) < 2 {
		fmt.Fprintln(ctx.Stderr, "warning: single-runtime panel — a single point of failure wearing several hats; prefer different vendors")
	}

	type seat struct {
		runtime, childID, verdict, why string
	}
	seats := make([]seat, 0, len(panel))

	for _, rtName := range panel {
		rtName = strings.TrimSpace(rtName)
		rt, err := store.LoadRuntime(w, rtName)
		if err != nil {
			return err
		}
		if _, err := exec.LookPath(rt.Binary); err != nil {
			return fmt.Errorf("runtime %s: binary %q not on PATH", rt.Name, rt.Binary)
		}
		sandboxArgs, err := sandboxFor(ctx, rt, grant, f.Bool("cooperative"))
		if err != nil {
			return err
		}
		childID, token, err := agentid.Spawn(w, id, "verifier", grant)
		if err != nil {
			return err
		}

		b, err := brief.Assemble(w, taskRef, brief.Options{Budget: budget})
		if err != nil {
			return err
		}
		preamble, err := protocolPreamble(w, childID, grant, t)
		if err != nil {
			return err
		}
		refute, err := refutePrompt(w, t, claim)
		if err != nil {
			return err
		}
		prompt := b.Render() + preamble + "\n" + refute

		runID := ulid.New()
		runDir := w.RunDir(runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return err
		}
		_ = os.WriteFile(filepath.Join(runDir, "brief.md"), []byte(prompt), 0o644)
		_ = os.WriteFile(filepath.Join(runDir, "invocation.txt"),
			[]byte(fmt.Sprintf("run: %s\nverify_panel_seat: %s\ntask: %s\nchild: %s\nclaim: %s\n", runID, rt.Name, t.ID, childID, claim)), 0o644)

		fmt.Fprintf(ctx.Stderr, "panel seat %s: %s\n", rt.Name, childID)
		onStart := func(pid, pgid int) {
			_ = procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), procmon.Record{
				RunID: runID, Child: childID, Task: t.ID, Runtime: rt.Name,
				PID: pid, PGID: pgid, Started: time.Now(),
			})
		}
		out, elapsed, _, runErr := execRuntime(w.Root, rt, prompt, token, sandboxArgs, timeout, onStart)
		_ = os.WriteFile(filepath.Join(runDir, "transcript.log"), out, 0o644)

		// The verdict is DERIVED from the log — same rule as shortcut uses:
		// the tally is recomputed from events, never stored as an integer
		// nobody can audit.
		verdict, why := verdictFor(w, childID)
		_ = os.WriteFile(filepath.Join(runDir, "outcome.md"),
			[]byte(fmt.Sprintf("outcome: %s\nelapsed: %s\nexit: %s\n", verdict, elapsed, clikit.ErrStr(runErr))), 0o644)
		seats = append(seats, seat{rt.Name, childID, verdict, why})
	}

	confirmed := 0
	for _, s := range seats {
		if s.verdict == "confirmed" {
			confirmed++
		}
		line := fmt.Sprintf("%-14s %-12s %s", s.runtime, s.childID, s.verdict)
		if s.why != "" {
			line += " — " + s.why
		}
		fmt.Fprintln(ctx.Stdout, line)
	}
	fmt.Fprintf(ctx.Stdout, "confirmed %d/%d (required %d)\n", confirmed, len(seats), require)
	if confirmed < require {
		// A killed claim is a RESULT, reported operationally (exit 1): the
		// verification worked, the claim did not survive it.
		return fmt.Errorf("claim KILLED by the panel — treat it as unestablished until re-derived")
	}
	fmt.Fprintln(ctx.Stdout, "claim SURVIVES the panel")
	return nil
}

// latestFinding returns the newest finding about the task, from events or
// notes, as the default claim under test.
func latestFinding(w *workspace.Workspace, t *store.Task) string {
	events, _ := eventlog.List(w, eventlog.Query{About: t.ID, Kinds: []model.EventKind{model.EventFinding}, Limit: 1})
	if len(events) > 0 {
		return strings.SplitN(events[0].Body, "\n", 2)[0]
	}
	notes, _ := store.ListNotes(w, t.Project, model.NoteFinding)
	for i := len(notes) - 1; i >= 0; i-- {
		if about, _ := notes[i].Front.Get("about"); strings.Contains(about, t.ID) || strings.Contains(about, fmt.Sprintf("%03d", t.Seq)) {
			for _, s := range notes[i].Sections {
				if s.Level == 1 {
					return s.Title
				}
			}
		}
	}
	return ""
}

// verdictFor reads a panelist's verdict from its finding events. No verdict
// is counted as no confirmation — a silent verifier confirms nothing.
func verdictFor(w *workspace.Workspace, childID string) (verdict, why string) {
	events, _ := eventlog.List(w, eventlog.Query{Actor: childID, Kinds: []model.EventKind{model.EventFinding}})
	for _, e := range events {
		first := strings.ToLower(strings.SplitN(e.Body, "\n", 2)[0])
		if i := strings.Index(first, "verdict: confirmed"); i >= 0 {
			return "confirmed", strings.TrimSpace(strings.TrimPrefix(first[i:], "verdict: confirmed —"))
		}
		if i := strings.Index(first, "verdict: refuted"); i >= 0 {
			return "refuted", strings.TrimSpace(strings.TrimPrefix(first[i:], "verdict: refuted —"))
		}
	}
	return "no-verdict", "panelist reported nothing — counts as unconfirmed"
}

func refutePrompt(w *workspace.Workspace, t *store.Task, claim string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = "dacli"
	}
	return prompts.Render(w.PromptsDir(), "verify_refute", map[string]any{
		"Claim":   claim,
		"Exe":     exe,
		"Project": t.Project,
		"Ref":     fmt.Sprintf("%03d", t.Seq),
	})
}
