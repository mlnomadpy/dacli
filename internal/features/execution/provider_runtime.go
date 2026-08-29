package execution

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/store"
)

// runtimeLauncher is the package-local provider process port. Command handlers
// resolve policy and records; this component owns process launch and stream
// lifecycle. Keeping the wrapper named execRuntime preserves the tested seam.
type runtimeLauncher interface {
	Launch(dir, transcriptPath string, rt store.Runtime, prompt, token string, extraArgs []string, timeoutSec int, detach bool, onStart func(pid, pgid int)) (time.Duration, bool, error)
}

type localRuntimeLauncher struct{}

func (localRuntimeLauncher) Launch(dir, transcriptPath string, rt store.Runtime, prompt, token string, extraArgs []string, timeoutSec int, detach bool, onStart func(pid, pgid int)) (time.Duration, bool, error) {
	return launchRuntimeProcess(dir, transcriptPath, rt, prompt, token, extraArgs, timeoutSec, detach, onStart)
}

var providerRuntime runtimeLauncher = localRuntimeLauncher{}

func execRuntime(dir, transcriptPath string, rt store.Runtime, prompt, token string, extraArgs []string, timeoutSec int, detach bool, onStart func(pid, pgid int)) (time.Duration, bool, error) {
	return providerRuntime.Launch(dir, transcriptPath, rt, prompt, token, extraArgs, timeoutSec, detach, onStart)
}

func launchRuntimeProcess(dir, transcriptPath string, rt store.Runtime, prompt, token string, extraArgs []string, timeoutSec int, detach bool, onStart func(pid, pgid int)) (elapsed time.Duration, timedOut bool, err error) {
	argv := append([]string{}, rt.GlobalArgs...)
	argv = append(argv, rt.Args...)
	argv = append(argv, extraArgs...)
	// F1: opt-in usage capture. Only when the adapter sets usage_format do we
	// ask the child to emit a machine-readable event stream; an empty
	// UsageFormat leaves argv (and thus a text runtime) exactly as it was. The
	// claude CLI requires --verbose alongside stream-json under --print.
	streamJSON := rt.UsageFormat == "stream-json" || rt.UsageFormat == "codex-jsonl" || rt.UsageFormat == "gemini-stream-json" || rt.UsageFormat == "copilot-json"
	switch rt.UsageFormat {
	case "stream-json":
		argv = append(argv, "--output-format", "stream-json", "--verbose")
	case "gemini-stream-json":
		argv = append(argv, "--output-format", "stream-json")
	case "copilot-json":
		argv = append(argv, "--output-format", "json")
	}
	if rt.Mode == "arg" {
		if rt.Flag != "" {
			argv = append(argv, rt.Flag)
		}
		argv = append(argv, prompt)
	}
	// The denylist is enforced HERE, at the point of use, not only in
	// `runtime add`. Gating just the writer protected one door into a file:
	// a runtime .md hand-edited by an rw agent, written by an older dacli,
	// restored from git, or copied in had its env_passthrough honored
	// verbatim — handing a child the operator's API keys. This is the read
	// every spawn actually makes, so it is the only place the rule cannot be
	// walked around.
	env := []string{agentid.EnvVar + "=" + token}
	for _, name := range rt.Env {
		if bad := deniedEnvPassthrough([]string{name}); bad != "" {
			return 0, false, clikit.Refusedf("runtime %s declares env_passthrough %s, which carries a credential — remove it from %s; a child must never inherit the operator's keys",
				rt.Name, bad, rt.Name)
		}
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	var sink *os.File
	if transcriptPath != "" {
		sink, err = os.Create(transcriptPath)
		if err != nil {
			return 0, false, fmt.Errorf("create transcript %q: %w", transcriptPath, err)
		}
	}
	start := time.Now()
	runtimePath, err := exec.LookPath(rt.Binary)
	if err != nil {
		missing := exec.Command(rt.Binary)
		missing.Dir = dir
		return 0, false, commandresult.NewExternalError(missing, commandresult.RunOptions{
			Operation: "runtime " + rt.Name + " launch", WorkspaceRoot: dir,
		}, nil, nil, err, false)
	}
	exe, err := os.Executable()
	if err != nil {
		return 0, false, fmt.Errorf("resolve dacli guardian: %w", err)
	}
	guardianArgv := []string{"__run-guardian"}
	if transcriptPath != "" {
		// Detached launches outlive this process, so their runtime exit status
		// cannot be returned through execRuntime. The guardian persists it beside
		// the transcript for `dacli wait` to classify (issue #550).
		guardianArgv = append(guardianArgv, "--exit-file", filepath.Join(filepath.Dir(transcriptPath), "runtime-exit.txt"))
	}
	guardianArgv = append(guardianArgv, runtimePath)
	guardianArgv = append(guardianArgv, argv...)

	if detach {
		// Detached: no CommandContext (its deadline would fire on the parent's
		// exit and kill the child). New process group so the tree stays killable
		// and survives this process as its own group; Release() hands it off.
		cmd := exec.Command(exe, guardianArgv...)
		cmd.Dir = dir
		cmd.Env = env
		setNewProcessGroup(cmd)
		if sink != nil {
			cmd.Stdout, cmd.Stderr = sink, sink
		}
		if rt.Mode == "stdin" {
			// A non-*os.File Stdin (e.g. strings.Reader) makes os/exec spawn a
			// parent-side goroutine to copy prompt→pipe, drained only by Wait().
			// Detach calls Release() and returns WITHOUT Wait(), so the parent
			// exits and that goroutine dies mid-copy — a prompt larger than the
			// ~64KB pipe buffer (briefs routinely are) is truncated or lost. Back
			// the child's stdin with a real *os.File instead: its fd is inherited
			// directly at exec, so the child reads the whole prompt with no parent
			// involvement. The unlinked temp file's inode survives via the child's
			// open fd until the child finishes reading.
			tf, terr := os.CreateTemp("", "dacli-stdin-*")
			if terr != nil {
				return 0, false, fmt.Errorf("detached stdin prompt: %w", terr)
			}
			defer func() { _ = tf.Close(); _ = os.Remove(tf.Name()) }()
			if _, werr := tf.WriteString(prompt); werr != nil {
				return 0, false, fmt.Errorf("detached stdin prompt: %w", werr)
			}
			if _, serr := tf.Seek(0, io.SeekStart); serr != nil {
				return 0, false, fmt.Errorf("detached stdin prompt: %w", serr)
			}
			cmd.Stdin = tf
		}
		serr := cmd.Start()
		if sink != nil {
			_ = sink.Close() // the child kept its own dup of the fd
		}
		if serr != nil {
			return 0, false, commandresult.NewExternalError(cmd, commandresult.RunOptions{
				Operation: "runtime guardian start", WorkspaceRoot: dir,
			}, nil, nil, serr, false)
		}
		if onStart != nil {
			onStart(cmd.Process.Pid, cmd.Process.Pid)
		}
		// Reap the child in the background instead of Release()ing it (dacli
		// 217). Release drops our handle without ever waiting, so the child
		// becomes a ZOMBIE the moment it exits — harmless under `dacli spawn`
		// (that parent exits immediately and init reaps the child), but a
		// long-lived parent — `dacli mcp serve`, or any in-process driver —
		// keeps the corpse in the process table for its whole lifetime. A
		// zombie answers signal-0, so procmon would report the finished agent
		// live forever: phantom rows in `dacli agents`, `dacli wait` blocking
		// to its timeout, KillTree escalating to SIGKILL against a corpse, and
		// the PID pinned so no other agent can be recorded under it.
		//
		// The wait runs in a goroutine so detach stays non-blocking and the
		// child still outlives us: if this process exits first the goroutine
		// simply dies and init inherits the child, exactly as before. Liveness
		// is zombie-aware too (procmon.Alive) — belt and braces, because a
		// child of some OTHER long-lived parent is not ours to reap.
		go func() { _, _ = cmd.Process.Wait() }()
		return 0, false, nil
	}

	interruptCtx, stopInterrupt := signal.NotifyContext(context.Background(), interruptSignals()...)
	defer stopInterrupt()
	cctx, cancel := context.WithTimeout(interruptCtx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, exe, guardianArgv...)
	cmd.Dir = dir
	cmd.Env = env
	// New process group: the child becomes group leader (pgid == its pid), and
	// every subprocess it forks inherits the group unless it detaches.
	setNewProcessGroup(cmd)
	// On timeout/cancel, kill the whole GROUP. The default CommandContext
	// cancel kills only the leader — which would orphan the children the agent
	// spawned, exactly the runaway leak we are preventing.
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = killProcessGroup(cmd.Process.Pid)
		}
		return nil
	}
	// Bound how long Wait blocks on output a grandchild may still hold open
	// after the group was killed, so a hung tree can't wedge dacli.
	cmd.WaitDelay = 5 * time.Second
	providerCmd := exec.Command(runtimePath)
	providerCmd.Dir = dir
	var stdoutDiagnostic, stderrDiagnostic runtimeDiagnosticTail
	wrapRuntimeFailure := func(cause error, timeout bool) error {
		if cause == nil {
			return nil
		}
		return commandresult.NewExternalError(providerCmd, commandresult.RunOptions{
			Operation: "runtime " + rt.Name + " launch", WorkspaceRoot: dir,
		}, stdoutDiagnostic.Bytes(), stderrDiagnostic.Bytes(), cause, timeout)
	}

	// stream-json capture: read the child's stdout through a pipe, tee a
	// human-readable rendering into the transcript (so logs -f / --tail keep
	// working) and remember the final usage event. Text runtimes keep the raw
	// stdout+stderr → sink wiring exactly as before.
	var streamPipe io.ReadCloser
	if streamJSON && sink != nil {
		streamPipe, _ = cmd.StdoutPipe()
		cmd.Stderr = io.MultiWriter(sink, &stderrDiagnostic)
		defer func() { _ = sink.Close() }()
	} else if sink != nil {
		cmd.Stdout = io.MultiWriter(sink, &stdoutDiagnostic)
		cmd.Stderr = io.MultiWriter(sink, &stderrDiagnostic)
		defer func() { _ = sink.Close() }()
	} else {
		cmd.Stdout, cmd.Stderr = &stdoutDiagnostic, &stderrDiagnostic
	}
	if rt.Mode == "stdin" {
		cmd.Stdin = strings.NewReader(prompt)
	}
	if serr := cmd.Start(); serr != nil {
		return time.Since(start).Round(time.Millisecond), false, commandresult.NewExternalError(cmd, commandresult.RunOptions{
			Operation: "runtime guardian start", WorkspaceRoot: dir,
		}, stdoutDiagnostic.Bytes(), stderrDiagnostic.Bytes(), serr, false)
	}
	if onStart != nil {
		onStart(cmd.Process.Pid, cmd.Process.Pid) // pgid == leader pid under Setpgid
	}
	if streamPipe != nil {
		// Must drain the pipe fully before Wait (os/exec closes it on exit).
		u := teeStructuredJSON(io.TeeReader(streamPipe, &stdoutDiagnostic), sink, rt.UsageFormat)
		err = cmd.Wait()
		if u.found {
			writeUsage(filepath.Dir(transcriptPath), u)
		} else if u.scanErr != nil {
			// The stream ended before the result event: usage was lost. Make that
			// visible in the transcript instead of falling back to the wall-clock
			// proxy as if this were a plain text runtime.
			fmt.Fprintf(sink, "[dacli: usage capture incomplete — %v]\n", u.scanErr)
		}
		timedOut = cctx.Err() == context.DeadlineExceeded
		return time.Since(start).Round(time.Millisecond), timedOut, wrapRuntimeFailure(err, timedOut)
	}
	err = cmd.Wait()
	timedOut = cctx.Err() == context.DeadlineExceeded
	return time.Since(start).Round(time.Millisecond), timedOut, wrapRuntimeFailure(err, timedOut)
}

// runtimeDiagnosticTail bounds custom streaming captures before they reach the
// shared commandresult redaction/classification policy. Runtime transcripts can
// be arbitrarily large; diagnostics retain only their actionable end.
type runtimeDiagnosticTail struct{ b []byte }

func (t *runtimeDiagnosticTail) Write(p []byte) (int, error) {
	const limit = 8 << 10
	written := len(p)
	if len(p) >= limit {
		t.b = append(t.b[:0], p[len(p)-limit:]...)
		return written, nil
	}
	if overflow := len(t.b) + len(p) - limit; overflow > 0 {
		copy(t.b, t.b[overflow:])
		t.b = t.b[:len(t.b)-overflow]
	}
	t.b = append(t.b, p...)
	return written, nil
}

func (t *runtimeDiagnosticTail) Bytes() []byte { return append([]byte(nil), t.b...) }

// streamUsage is the final `result` event's accounting from a stream-json run.
type streamUsage struct {
	InputTokens  int
	OutputTokens int
	NumTurns     int
	CostUSD      float64
	SessionID    string
	FinalMessage string
	ExitOutcome  string
	found        bool
	// scanErr is a non-EOF read error (or over-long line) that ended the stream
	// BEFORE the terminating `result` event was seen. The result event carries
	// the ONLY usage numbers and arrives last, so an error mid-stream silently
	// loses token capture; callers surface scanErr instead of mistaking it for a
	// clean text-runtime EOF.
	scanErr error
}

type codexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Item     struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Name string `json:"name"`
	} `json:"item"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func renderCodexLine(line []byte, prior streamUsage) (string, streamUsage) {
	var ev codexEvent
	if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
		return string(bytes.TrimSpace(line)), prior
	}
	switch ev.Type {
	case "thread.started":
		prior.SessionID = ev.ThreadID
	case "item.completed":
		if ev.Item.Type == "agent_message" {
			prior.FinalMessage = strings.TrimSpace(ev.Item.Text)
			return prior.FinalMessage, prior
		}
		if ev.Item.Type != "" {
			return "[item: " + ev.Item.Type + "]", prior
		}
	case "turn.completed":
		prior.InputTokens, prior.OutputTokens, prior.ExitOutcome, prior.found = ev.Usage.InputTokens, ev.Usage.OutputTokens, "completed", true
	case "turn.failed":
		prior.ExitOutcome, prior.found = "failed", true
	}
	return "", prior
}

func teeStructuredJSON(r io.Reader, out io.Writer, format string) streamUsage {
	if format == "gemini-stream-json" {
		return teeGeminiStreamJSON(r, out)
	}
	if format != "codex-jsonl" {
		return teeStreamJSON(r, out)
	}
	var u streamUsage
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var text string
			text, u = renderCodexLine(line, u)
			if text != "" {
				fmt.Fprintln(out, text)
			}
		}
		if err != nil {
			if err != io.EOF {
				u.scanErr = err
			}
			break
		}
	}
	return u
}

type geminiEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	Stats     struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"stats"`
}

func teeGeminiStreamJSON(r io.Reader, out io.Writer) streamUsage {
	var u streamUsage
	var final strings.Builder
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var ev geminiEvent
			if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
				fmt.Fprintln(out, string(bytes.TrimSpace(line)))
			} else {
				switch ev.Type {
				case "init":
					u.SessionID = ev.SessionID
				case "message":
					if ev.Role == "assistant" {
						fmt.Fprint(out, ev.Content)
						final.WriteString(ev.Content)
					}
				case "result":
					u.InputTokens, u.OutputTokens = ev.Stats.InputTokens, ev.Stats.OutputTokens
					u.FinalMessage = strings.TrimSpace(final.String())
					u.ExitOutcome = map[bool]string{true: "completed", false: "failed"}[ev.Status == "success"]
					u.found = true
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				u.scanErr = err
			}
			break
		}
	}
	return u
}

// streamEvent is the subset of a `claude --output-format stream-json` event we
// read: assistant content (for the readable rendering) and the result event's
// usage accounting.
type streamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	IsError   bool   `json:"is_error"`
	Message   struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			Name string `json:"name"`
		} `json:"content"`
	} `json:"message"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	NumTurns int     `json:"num_turns"`
	CostUSD  float64 `json:"total_cost_usd"`
}

// renderStreamLine turns one stream-json line into its human-readable rendering
// (assistant text and [tool: X] markers) and, when it is the terminating
// `result` event, its usage. A line that is not a JSON event is returned
// verbatim so nothing the child emits is ever dropped. text is "" for events
// with no human-facing content (system/result/empty), letting callers skip
// them. This is the single shared decoder for both the live tee and the
// render-on-read transcript readers, so foreground and detached runs render
// identically.
func renderStreamLine(line []byte) (text string, usage streamUsage) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return "", streamUsage{}
	}
	if trimmed[0] != '{' { // fast path: not an event object — pass through
		return string(trimmed), streamUsage{}
	}
	var ev streamEvent
	if err := json.Unmarshal(trimmed, &ev); err != nil {
		return string(trimmed), streamUsage{} // not an event — verbatim
	}
	switch ev.Type {
	case "assistant":
		var b strings.Builder
		for _, c := range ev.Message.Content {
			switch c.Type {
			case "text":
				if s := strings.TrimSpace(c.Text); s != "" {
					b.WriteString(s)
					b.WriteByte('\n')
				}
			case "tool_use":
				fmt.Fprintf(&b, "[tool: %s]\n", c.Name)
			}
		}
		return strings.TrimRight(b.String(), "\n"), streamUsage{}
	case "result":
		return "", streamUsage{
			InputTokens:  ev.Usage.InputTokens,
			OutputTokens: ev.Usage.OutputTokens,
			NumTurns:     ev.NumTurns,
			CostUSD:      ev.CostUSD,
			SessionID:    ev.SessionID,
			FinalMessage: strings.TrimSpace(ev.Result),
			ExitOutcome:  map[bool]string{false: "completed", true: "failed"}[ev.IsError],
			found:        true,
		}
	}
	return "", streamUsage{}
}

// teeStreamJSON reads a stream-json event stream from r, writes a human-readable
// rendering to out so the transcript stays as legible as a text runtime's, and
// returns the usage carried by the terminating `result` event. It uses a
// bufio.Reader (not a Scanner) so a single very large event line cannot exceed a
// buffer cap and abort the stream before the result event — the failure that
// silently lost usage. Any non-EOF read error is reported in the returned
// streamUsage.scanErr rather than swallowed.
func teeStreamJSON(r io.Reader, out io.Writer) streamUsage {
	var u streamUsage
	br := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := br.ReadBytes('\n') // no length cap: over-long lines grow, never truncate
		if len(bytes.TrimSpace(line)) > 0 {
			text, usage := renderStreamLine(line)
			if text != "" {
				fmt.Fprintln(out, text)
			}
			if usage.found {
				u = usage
			}
		}
		if err != nil {
			if err != io.EOF {
				u.scanErr = err
			}
			break
		}
	}
	return u
}

// writeUsage records the captured token accounting into the run record so
// calibration can read it back (store.CalibrationSamples). Best-effort: a
// missing usage.txt just means calibration falls back to the wall-clock proxy.
func writeUsage(runDir string, u streamUsage) {
	record := openRunRecord(runDir, nil)
	body := fmt.Sprintf("output_tokens: %d\ninput_tokens: %d\nnum_turns: %d\ncost_usd: %.6f\n",
		u.OutputTokens, u.InputTokens, u.NumTurns, u.CostUSD)
	record.bestEffort("usage.txt", body)
	if u.SessionID != "" || u.FinalMessage != "" || u.ExitOutcome != "" {
		result := fmt.Sprintf("session_id: %s\nexit_outcome: %s\nfinal_message: %s\n", u.SessionID, u.ExitOutcome, u.FinalMessage)
		record.bestEffort("result.txt", result)
	}
}

// promptSuffix assembles everything appended after the brief: the reporting
// protocol, git discipline for writers, review discipline for reviewers.
// All of it lives in the prompt registry, none of it in Fprintf chains.
