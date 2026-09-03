package execution

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentstate"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func cmdRunsList(ctx *clikit.Ctx, args []string) error {
	// This command takes no flags, so ANY flag is a typo. An empty allowlist
	// rejects every one — without it a mistyped flag was dropped and the
	// command ran as if nothing were wrong.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		// No runs yet is normal and lists nothing. An unreadable runs directory
		// is a different fact, and reporting both as "no runs" hid the second
		// entirely. This function CAN return an error, unlike its two siblings
		// that cannot (see dacli 337).
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot read the runs directory at %s: %w", w.RunsDir(), err)
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names))) // ULIDs: newest first
	for _, n := range names {
		line := "(no outcome recorded)"
		if raw, err := os.ReadFile(filepath.Join(w.RunDir(n), "outcome.md")); err == nil {
			line = strings.ReplaceAll(strings.TrimSpace(string(raw)), "\n", " · ")
		}
		fmt.Fprintf(ctx.Stdout, "%s  %s\n", clikit.Short(n, 10), line)
	}
	return nil
}

func cmdRunsShow(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli runs show <run-id-prefix>")
	}
	entries, _ := os.ReadDir(w.RunsDir())
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), f.Pos[0]) {
			continue
		}
		for _, name := range []string{"invocation.txt", "outcome.md", store.RootHandoffFile, store.ParentCommitRequestFile, store.ParentCommitReceiptFile, "brief.md", "transcript.log", "diagnostics.txt"} {
			if raw, err := os.ReadFile(filepath.Join(w.RunDir(e.Name()), name)); err == nil {
				fmt.Fprintf(ctx.Stdout, "=== %s ===\n%s\n", name, strings.TrimSpace(string(raw)))
			}
		}
		// First match wins, DELIBERATELY, and this return is a success rather
		// than a verdict — the distinction that makes it unlike the two landing
		// bugs the candidate-loop sweep found (task 363), where an early return
		// of a NEGATIVE result made every later candidate unreachable. Run ids
		// are ULIDs, so a prefix long enough to be typed is effectively unique,
		// and entries are read in sorted order, so the choice is deterministic.
		//
		// It does mean an ambiguous prefix shows one run without saying so,
		// where FindTask refuses an ambiguous task ref outright. That
		// inconsistency is recorded as a finding rather than changed here:
		// tightening it is a behaviour change to a read-only command, not part
		// of the sweep.
		return nil
	}
	return store.ErrNotFound{Ref: "run " + f.Pos[0]}
}

func cmdRunsPrune(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("keep"); err != nil {
		return err
	}
	keep := 20
	if n, err := f.Int("keep", 0); err != nil {
		return err
	} else if n > 0 {
		keep = n
	}
	entries, _ := os.ReadDir(w.RunsDir())
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // oldest first
	// Everything outside the newest `keep` is a prune CANDIDATE — but a
	// candidate whose process is still running is skipped (dacli 208).
	// proc.txt, the transcript and the usage file are the only handles dacli
	// has on a live agent: `agents`, `wait` and `kill` all read them back from
	// disk. RemoveAll on a still-executing run orphans that agent — it keeps
	// burning tokens with nothing able to observe or stop it. Retention is a
	// disk-space policy; it does not get to blind us to a running process.
	// Skips are reported, not silently absorbed, so an operator who expected
	// `--keep 5` and got 7 directories knows exactly which two ran long.
	pruned, skipped := 0, 0
	for _, n := range names[:max(0, len(names)-keep)] {
		if rec, rerr := procmon.ReadRecord(filepath.Join(w.RunDir(n), "proc.txt")); rerr == nil && runStillLive(rec) {
			skipped++
			fmt.Fprintf(ctx.Stdout, "kept %s: %s still live (pid %d, group %d) — pruning it would orphan a running agent\n",
				clikit.Short(n, 10), clikit.OrDash(rec.Child), rec.PID, rec.PGID)
			continue
		}
		if err := os.RemoveAll(w.RunDir(n)); err != nil {
			return err
		}
		pruned++
	}
	fmt.Fprintf(ctx.Stdout, "pruned %d run(s), kept %d\n", pruned, len(names)-pruned)
	if skipped > 0 {
		fmt.Fprintf(ctx.Stdout, "%d live run(s) kept beyond --keep %d; re-run after they finish\n", skipped, keep)
	}
	return nil
}

// lifecycleNow is the single observation clock for every run-liveness reader.
// Keeping it as a seam makes the startup/transcript grace boundary testable
// without asking a loaded scheduler to finish several commands inside a small
// wall-clock margin (issue #896).
var lifecycleNow = time.Now

// cmdAgents lists agents whose process tree is still alive, with the RAM/CPU
// (and GPU where measurable) the whole group is holding right now, plus each
// agent's honest activity state (agentstate.Derive — thinking/acting/waiting/
// stalled/blocked/silent) so RAM and uptime alone never have to answer "is it
// still working?" A run's proc.txt is written at spawn; liveness is probed
// live, so an exited agent simply doesn't appear — the list is
// runaways-included, ghosts-excluded. During the bounded registration window,
// fresh transcript activity is also treated as live by runLifecycleLive.
func cmdAgents(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("max-rss", "max-runtime", "reap", "tail", "project"); err != nil {
		return err
	}
	project := f.Get("project")
	if project != "" && !workspace.SafeSegment(project) {
		return clikit.Usagef("--project requires a valid project slug")
	}
	if ctx.JSON || project != "" {
		if f.Bool("reap") || f.Get("max-rss") != "" || f.Get("max-runtime") != "" || f.Bool("tail") {
			return clikit.Usagef("--project/--json is a read-only progress view and cannot be combined with --reap, resource limits, or --tail")
		}
		return renderAgentProgress(ctx, w, project, time.Now())
	}
	// Listing agents is a read; --reap KILLS whole process trees, which `kill`
	// has always required rw for. The gate is here rather than on the command
	// table because the two live under one path: a read-only agent must keep
	// its view of the swarm while never being able to end a sibling's run.
	if f.Bool("reap") {
		if err := clikit.RequireRW(id, "reaping an agent (--reap)"); err != nil {
			return err
		}
	}
	// Optional budgets: an agent over either limit is a runaway. --reap kills
	// it (whole tree); without --reap it is only flagged, so you can look first.
	maxRSS := parseBytes(f.Get("max-rss"))           // e.g. 2G, 500M; 0 = no limit
	maxRun := parseDurationArg(f.Get("max-runtime")) // e.g. 15m, 900; 0 = no limit
	reap := f.Bool("reap")
	// --tail: under each agent, print the last non-empty transcript line — its
	// current activity. RAM/CPU alone can't tell a reasoning agent from a wedged
	// one; the live tail can (a thinking agent's last line keeps moving).
	tail := f.Bool("tail")
	textRuntime := map[string]bool{} // runtime name -> no usage_format (buffers to exit)
	// One task-tree scan for every live agent's blocked check, not one per
	// agent (store.BuildTaskIndex — the same discipline eventlog.Sync and
	// acceptance.go follow). A failed build degrades to nil: agentstate.Derive
	// then just never reports "blocked", never an error.
	tasks, _ := store.BuildTaskIndex(w)

	live, err := liveAgents(w)
	if err != nil {
		return err
	}
	liveRunIDs := map[string]bool{}
	for _, rec := range live {
		liveRunIDs[rec.RunID] = true
		u := procmon.SampleGroup(rec.PGID)
		age := time.Since(rec.Started).Round(time.Second)
		over := ""
		if maxRSS > 0 && int64(u.RSSKB)*1024 > maxRSS {
			over += fmt.Sprintf(" OVER-RAM(>%s)", humanBytes(maxRSS))
		}
		if maxRun > 0 && age > maxRun {
			over += fmt.Sprintf(" OVER-TIME(>%s)", maxRun)
		}
		// state is the same thinking/acting/waiting/stalled/blocked/silent label
		// the dashboard shows (agentstate.Derive is the single shared source) —
		// RAM/CPU/uptime alone can't tell a reasoning agent from a wedged one.
		// Printed uppercase for the states that want an operator's attention, so
		// a stalled agent stands out from a busy one without needing --tail.
		state := agentstate.Derive(w, rec, tasks)
		// A child that raised the break-glass BLOCKED channel is tagged distinctly
		// and its reason printed below — a run reporting it cannot run dacli must
		// never read as a normal live agent (task 269). The tag is kept OUT of
		// `over` so --reap (for RAM/time runaways) never kills an agent whose only
		// signal is that it asked for help. It composes with agentstate: Derive
		// reads the task's outstanding ask, this reads the run's blocked.txt.
		blocked := readBlocked(w, rec.RunID)
		status := over
		if _, reason := runLifecycleLive(w, rec, lifecycleNow()); reason != "process live" {
			status += " " + strings.ToUpper(strings.ReplaceAll(reason, " ", "-"))
		}
		if blocked != "" {
			status += " BLOCKED"
		}
		handoffRequired := store.RootHandoffRequested(w, rec.RunID)
		if _, err := os.Stat(store.RootHandoffPathForRun(w, rec.RunID)); err == nil {
			handoffRequired = true
		}
		if handoffRequired {
			status += " HANDOFF-REQUIRED"
		}
		// CPUPct is ps's %cpu: cputime/elapsed AVERAGED over each process's whole
		// lifetime, NOT an instantaneous sample. Labelled "CPUavg" so an operator
		// does not read a long-idle agent's high lifetime average as current load.
		fmt.Fprintf(ctx.Stdout, "%s  %-14s %-12s %-10s pid %-7d %2d proc  %8s RAM  %5.0f%% CPUavg  %7s GPU  up %s  [%s]%s\n",
			rec.RunID[:min(10, len(rec.RunID))], clikit.OrDash(rec.Child), clikit.OrDash(rec.Runtime),
			"task "+clikit.OrDash(rec.Task), rec.PID, u.Procs, humanKB(u.RSSKB), u.CPUPct, gpuStr(u.GPUMiB), age, stateLabel(state), status)
		if blocked != "" {
			fmt.Fprintf(ctx.Stdout, "            ⚠ BLOCKED: %s\n", truncateLine(firstLine(blocked), 100))
		}
		if handoffRequired {
			fmt.Fprintln(ctx.Stdout, "            ⚠ HANDOFF-REQUIRED: root must re-observe and consume the structured handoff")
		}
		if tail {
			line := tailLine(w, filepath.Join(w.RunDir(rec.RunID), "transcript.log"), rec.Runtime, textRuntime)
			fmt.Fprintf(ctx.Stdout, "            ↳ %s\n", truncateLine(line, 100))
		}
		if over != "" && reap {
			killOne(ctx, w, rec, 3*time.Second)
		}
	}
	if len(live) == 0 {
		fmt.Fprintln(ctx.Stdout, "no live agents")
	}
	for _, handoff := range pendingRootHandoffs(w) {
		if liveRunIDs[handoff.RunID] {
			continue
		}
		fmt.Fprintf(ctx.Stdout, "%s  %-14s task %-10s  HANDOFF-REQUIRED\n            ↳ %s\n",
			clikit.Short(handoff.RunID, 10), clikit.OrDash(handoff.ChildID), clikit.OrDash(handoff.TaskID), truncateLine(handoff.NextAction, 100))
	}
	return nil
}

type agentProgressView struct {
	Schema     string                `json:"schema"`
	Version    int                   `json:"version"`
	ObservedAt time.Time             `json:"observed_at"`
	Workers    []store.WorkerExplain `json:"workers"`
}

// renderAgentProgress consumes the same store projection as `explain`; it does
// not infer a parallel worker lifecycle from PID/RAM presentation details.
func renderAgentProgress(ctx *clikit.Ctx, w *workspace.Workspace, project string, now time.Time) error {
	projects := []string{project}
	if project == "" {
		projects = nil
		all, err := store.ListProjects(w)
		if err != nil {
			return err
		}
		for _, item := range all {
			projects = append(projects, item.Slug)
		}
	}
	view := agentProgressView{Schema: store.ProgressExplainSchema, Version: 1, ObservedAt: now.UTC(), Workers: []store.WorkerExplain{}}
	for _, slug := range projects {
		projection, err := store.ExplainProject(w, slug, now)
		if err != nil {
			return err
		}
		view.Workers = append(view.Workers, projection.Workers...)
	}
	sort.Slice(view.Workers, func(i, j int) bool { return view.Workers[i].RunID.Value < view.Workers[j].RunID.Value })
	if ctx.JSON {
		return clikit.EmitJSON(ctx, view)
	}
	if len(view.Workers) == 0 {
		fmt.Fprintln(ctx.Stdout, "no recorded workers for project")
		return nil
	}
	for _, worker := range view.Workers {
		fmt.Fprintf(ctx.Stdout, "%s agent=%s task=%s role=%s runtime=%s state=%s (source=%s observed=%s stale=%t)\n  claims: %s\n  next: %s\n",
			clikit.Short(worker.RunID.Value, 10), clikit.OrDash(worker.AgentID.Value), clikit.OrDash(worker.TaskID.Value), clikit.OrDash(worker.Role.Value), clikit.OrDash(worker.Runtime.Value), worker.State.Value,
			worker.State.Source, worker.State.ObservedAt.Format(time.RFC3339), worker.State.Stale, clikit.OrDash(strings.Join(worker.Claims.Value, ", ")), worker.NextAction.Value)
	}
	return nil
}

// stateLabel renders an agentstate.Derive result for the agents list: the
// three "needs a look" states (stalled/silent/blocked) are uppercased so they
// read as distinct from the three healthy ones (thinking/acting/waiting) at a
// glance — the same all-caps-for-attention convention OVER-RAM/OVER-TIME
// already use on this line.
func stateLabel(state string) string {
	switch state {
	case agentstate.Stalled, agentstate.Silent, agentstate.Blocked:
		return strings.ToUpper(state)
	default:
		return state
	}
}

// tailLine resolves what `agents --tail` shows under one agent: the
// transcript's last rendered line, or — when there is none yet — a note that
// tells a text runtime (whose child fully-buffers stdout until it exits) apart
// from a stream-json runtime that simply has nothing new to show.
func tailLine(w *workspace.Workspace, transcriptPath, runtimeName string, cache map[string]bool) string {
	if line := lastTranscriptLine(transcriptPath); line != "" {
		return line
	}
	if isTextRuntime(w, runtimeName, cache) {
		return "(text runtime — output appears at exit)"
	}
	return "(no transcript output yet)"
}

// isTextRuntime reports whether runtime name has no usage_format set — a text
// runtime whose child CLI fully-buffers stdout, so transcript.log stays empty
// until the process exits (not "stuck"). cache memoizes the LoadRuntime lookup
// across the agents list. An unresolvable name (empty, or no such adapter)
// reports false so --tail falls back to the generic no-output message.
func isTextRuntime(w *workspace.Workspace, name string, cache map[string]bool) bool {
	if name == "" {
		return false
	}
	if v, ok := cache[name]; ok {
		return v
	}
	rt, err := store.LoadRuntime(w, name)
	textOnly := err == nil && rt.UsageFormat == ""
	cache[name] = textOnly
	return textOnly
}

// parseBytes reads a size like "2G", "500M", "1024K", or a bare byte count.
func parseBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mult := int64(1)
	switch last := s[len(s)-1]; last {
	case 'G', 'g':
		mult, s = 1<<30, s[:len(s)-1]
	case 'M', 'm':
		mult, s = 1<<20, s[:len(s)-1]
	case 'K', 'k':
		mult, s = 1<<10, s[:len(s)-1]
	}
	n, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return int64(n * float64(mult))
}

func humanBytes(b int64) string { return humanKB(int(b / 1024)) }

// parseDurationArg reads "15m"/"2h"/"90s" or a bare seconds count.
func parseDurationArg(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Second
	}
	return 0
}

// cmdLogs prints, or with -f follows, a run's transcript. A detached child
// streams straight to the transcript file, so -f tails a live agent's output
// the way `tail -f` would — the missing "what is it actually doing" view.
func cmdLogs(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, err := clikit.ParseFlags(args, "tail")
	if err != nil {
		return err
	}
	if err := f.Reject("f", "follow", "tail"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli logs <run-id-prefix|child-id> [-f] [--tail N]")
	}
	ref := f.Pos[0]
	rec, haveRec := readProcByRef(w, ref)
	runID := rec.RunID
	if !haveRec {
		entries, _ := os.ReadDir(w.RunsDir())
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), ref) {
				runID = e.Name()
				break
			}
		}
	}
	if runID == "" {
		return store.ErrNotFound{Ref: "run " + ref}
	}
	path := filepath.Join(w.RunDir(runID), "transcript.log")

	data, _ := os.ReadFile(path)
	if n, err := f.Int("tail", 0); err != nil {
		return err
	} else if n <= 0 && len(f.All("tail")) > 0 {
		return clikit.Usagef("--tail must be a positive integer, got %d", n)
	} else if n > 0 {
		data = lastLines(data, n)
	}
	// Detached stream-json runs write RAW JSON events to the transcript (the tee
	// only runs on the foreground path), so render each line to readable text on
	// read — logs and -f show the same legible output as a text runtime.
	renderTranscriptTo(ctx.Stdout, data)
	var offset int64
	if fi, e := os.Stat(path); e == nil {
		offset = fi.Size()
	}
	if !(f.Bool("f") || f.Bool("follow")) {
		return nil
	}
	// Follow: drain appended bytes until the agent's process is gone (one final
	// drain after it exits), so the tail ends when the work does. Advance the
	// offset only to the last newline so a JSON event line is never split across
	// two renders.
	drain := func(final bool) {
		fi, e := os.Stat(path)
		if e != nil || fi.Size() <= offset {
			return
		}
		chunk := make([]byte, fi.Size()-offset)
		fh, e2 := os.Open(path)
		if e2 != nil {
			return
		}
		n, _ := fh.ReadAt(chunk, offset)
		_ = fh.Close()
		chunk = chunk[:n]
		if !final {
			nl := bytes.LastIndexByte(chunk, '\n')
			if nl < 0 {
				return // no complete line yet; wait for the rest
			}
			chunk = chunk[:nl+1]
		}
		renderTranscriptTo(ctx.Stdout, chunk)
		offset += int64(len(chunk))
	}
	for {
		time.Sleep(700 * time.Millisecond)
		drain(false)
		if !(haveRec && procmon.AliveRecord(rec)) {
			drain(true) // flush any trailing partial line once the work is done
			return nil
		}
	}
}

// renderTranscriptTo writes b to out with each complete line rendered from
// stream-json to readable text (assistant text / [tool: X] markers); a
// plain-text line passes through unchanged and blank lines are dropped. This is
// the read-side counterpart of teeStreamJSON: it makes a detached run's raw
// stream-json transcript as legible as a foreground run's already-teed one.
func renderTranscriptTo(out io.Writer, b []byte) {
	for _, ln := range bytes.Split(b, []byte("\n")) {
		if len(bytes.TrimSpace(ln)) == 0 {
			continue
		}
		text, _ := renderStreamLine(ln)
		if text == "" {
			text, _ = renderCodexLine(ln, streamUsage{})
		}
		if text != "" {
			fmt.Fprintln(out, text)
		}
	}
}

// lastTranscriptLine reads path and returns its most recent readable line — the
// agent's current activity for `dacli agents --tail`. A detached stream-json
// child writes raw JSON events here, so each candidate line is rendered on read
// (assistant text / [tool: X]); events with no human-facing content are skipped.
// Missing/empty file yields "".
func lastTranscriptLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// walk backwards for the last line that renders to non-empty text.
	end := len(data)
	for end > 0 {
		start := bytes.LastIndexByte(data[:end], '\n')
		raw := bytes.TrimSpace(data[start+1 : end])
		if len(raw) > 0 {
			text, _ := renderStreamLine(raw)
			if text == "" {
				text, _ = renderCodexLine(raw, streamUsage{})
			}
			if text != "" {
				// A rendered assistant event may span lines; the current activity
				// is its last line.
				if i := strings.LastIndexByte(text, '\n'); i >= 0 {
					text = text[i+1:]
				}
				return text
			}
		}
		if start < 0 {
			break
		}
		end = start
	}
	return ""
}

// truncateLine shortens s to at most max runes, appending an ellipsis when cut.
func truncateLine(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// lastLines returns the last n newline-delimited lines of b.
func lastLines(b []byte, n int) []byte {
	count := 0
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			count++
			if count > n {
				return b[i+1:]
			}
		}
	}
	return b
}

// splitClaims parses one comma-separated --claim value into cleaned paths.
func splitClaims(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitClaimValues accumulates every repeated --claim occurrence while also
// supporting comma-separated paths within each occurrence. Order is preserved
// so preview output matches the operator's invocation and the durable record.
func splitClaimValues(values []string) []string {
	var out []string
	for _, value := range values {
		out = append(out, splitClaims(value)...)
	}
	return out
}

// cmdKill terminates one agent's whole process tree, or --all of them. The
// group is SIGTERM'd, then SIGKILL'd after a grace window if anything survives
// — so a well-behaved agent exits cleanly and a hung one is still guaranteed
// dead, with no orphaned children left holding resources.
