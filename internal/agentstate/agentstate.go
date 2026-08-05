// Package agentstate derives the honest per-agent activity label — thinking,
// acting, waiting, stalled, blocked, or silent — from a live agent's task
// status and transcript. It is the single source both `dacli agents` and the
// dashboard read, so the two surfaces can never disagree about what
// "stalled" means (dacli 270). Neither `dacli agents` nor the dashboard may
// duplicate this logic; they call Derive.
package agentstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// States a live agent can be reported in. Thinking/Acting/Waiting/Stalled are
// derived purely from the transcript (see Derive); Blocked and Silent add the
// two signals a transcript alone can't carry: an outstanding `dacli ask` on
// the agent's task, and a text runtime that has stayed quiet suspiciously
// long (briefly-buffered output is normal, minutes of it is worth a look).
const (
	Thinking = "thinking"
	Acting   = "acting"
	Waiting  = "waiting"
	Stalled  = "stalled"
	Blocked  = "blocked"
	Silent   = "silent"
)

// StallAfter is how long a live agent's transcript (or, for a text runtime,
// its total silence) may go without a new rendered line before Derive calls
// it stalled/silent rather than thinking/acting/waiting. A stream-json agent
// writes a line every few seconds while it works, so a freeze this long while
// the process is still alive is the honest "possibly hung" signal. It is
// deliberately generous: a single long tool call (a slow test run, a big
// clone) legitimately produces no transcript output while it runs, and from
// the transcript ALONE a wedged agent and one waiting on a long tool are
// indistinguishable — so we wait before crying "hung".
const StallAfter = 120 * time.Second

// Derive reads a live agent's task status and transcript and returns its
// honest activity — never a guess from RAM or CPU (a reasoning agent and a
// wedged one can hold identical memory):
//
//   - blocked  — the agent's task is blocked on an unanswered `dacli ask`
//     (collab.cmdAsk sets model.StatusBlocked). Whatever the transcript is
//     doing, the agent is waiting on a human, not on its own reasoning.
//   - waiting  — nothing rendered yet and the silence is still short: a
//     freshly-spawned agent, or a stream runtime with nothing new yet.
//   - silent   — a text runtime (its child fully-buffers stdout until it
//     exits) has produced nothing for longer than StallAfter. Never called
//     "stalled": that silence can be entirely normal for a slow buffered run,
//     but past StallAfter it deserves the same attention a stall would.
//   - stalled  — the transcript had output and has now frozen for longer than
//     StallAfter while the process is still alive: it WAS moving and has gone
//     quiet ("possibly hung").
//   - acting   — the last rendered line is a [tool: X] marker: the agent is
//     executing a tool.
//   - thinking — the last rendered line is assistant prose: the agent is
//     reasoning.
//
// tasks is an optional pre-built store.TaskIndex (store.BuildTaskIndex) so a
// caller listing many live agents pays for the task tree scan once, not once
// per agent (the same discipline eventlog.Sync and acceptance.go follow). A
// nil index just means "never blocked" — a degraded-but-honest fallback, the
// same rule buildRoles and buildBurn apply to their own read failures.
func Derive(w *workspace.Workspace, rec procmon.Record, tasks *store.TaskIndex) string {
	if isBlocked(tasks, rec) {
		return Blocked
	}
	path := filepath.Join(w.RunDir(rec.RunID), "transcript.log")
	line := lastActivityLine(path)
	fi, statErr := os.Stat(path)
	quiet := statErr == nil && time.Since(fi.ModTime()) > StallAfter
	if line == "" {
		if IsTextRuntime(w, rec.Runtime) {
			if quiet {
				return Silent
			}
			return Waiting
		}
		if quiet {
			return Stalled
		}
		return Waiting
	}
	if quiet {
		return Stalled
	}
	if strings.HasPrefix(line, "[tool:") {
		return Acting
	}
	return Thinking
}

// isBlocked reports whether rec's task is currently blocked on an unanswered
// `dacli ask` (model.StatusBlocked). An agent with no task association, one
// whose task can't be resolved, or a nil index is never blocked by this
// check — it falls through to the transcript-derived states.
func isBlocked(tasks *store.TaskIndex, rec procmon.Record) bool {
	if tasks == nil || rec.Task == "" {
		return false
	}
	t, err := tasks.Find(rec.Task)
	if err != nil {
		return false
	}
	return t.Status == model.StatusBlocked
}

// IsTextRuntime reports whether the named runtime has no usage_format set — a
// text runtime whose child CLI fully-buffers stdout, so transcript.log stays
// empty until the process exits (not "stuck"). An unresolvable name (empty,
// or no such adapter) reports false.
func IsTextRuntime(w *workspace.Workspace, name string) bool {
	if name == "" {
		return false
	}
	rt, err := store.LoadRuntime(w, name)
	return err == nil && rt.UsageFormat == ""
}

// lastActivityLine returns a transcript's most recent human-readable line —
// the agent's current activity. A detached stream-json child writes raw JSON
// events here, so each candidate line is rendered on read (assistant text /
// [tool: X]); events with no human-facing content are skipped. Missing/empty
// file yields "".
func lastActivityLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	end := len(data)
	for end > 0 {
		start := bytes.LastIndexByte(data[:end], '\n')
		raw := bytes.TrimSpace(data[start+1 : end])
		if len(raw) > 0 {
			if text := RenderTranscriptLine(raw); text != "" {
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

// transcriptEvent is the minimal stream-json shape needed to tell thinking
// (assistant text) from acting ([tool: X]).
type transcriptEvent struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
			Name string `json:"name"`
		} `json:"content"`
	} `json:"message"`
}

// RenderTranscriptLine turns one transcript line into its human-readable
// form: assistant text and [tool: X] markers, "" for events with no
// human-facing content (system/result/empty). A line that is not a JSON
// event passes through verbatim so a plain-text runtime's transcript renders
// unchanged.
func RenderTranscriptLine(line []byte) string {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return ""
	}
	if trimmed[0] != '{' {
		return string(trimmed)
	}
	var ev transcriptEvent
	if err := json.Unmarshal(trimmed, &ev); err != nil {
		return string(trimmed)
	}
	if ev.Type != "assistant" {
		return ""
	}
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
	return strings.TrimRight(b.String(), "\n")
}
