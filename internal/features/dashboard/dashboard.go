// Package dashboard serves a self-contained, read-only local web UI over the
// workspace: projects, tasks by status with a burndown, and the live agent
// swarm (task/role/runtime/last activity), read straight from the store and
// the run directory's proc.txt records — the same sources `dacli status` and
// `dacli agents` already read. Nothing here mutates the workspace.
package dashboard

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "dashboard", Brief: "Serve a local web UI: projects, burndown, and the live agent swarm", Run: cmdDashboard},
}

//go:embed static/index.html
var indexHTML []byte

// cmdDashboard binds a localhost listener (an ephemeral port unless --port
// pins one) and serves the dashboard until the process is killed. The page
// itself polls /api/state, so a loop's agents appear live without a restart.
func cmdDashboard(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	port := 0
	if p := f.Get("port"); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			return clikit.Usagef("--port must be a number, got %q", p)
		}
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}
	fmt.Fprintf(ctx.Stdout, "dacli dashboard: http://%s (Ctrl+C to stop)\n", ln.Addr().String())
	return http.Serve(ln, newHandler(w))
}

// newHandler builds the whole server: the embedded page at "/" and the JSON
// snapshot it polls at "/api/state". Factored out so tests can drive it
// through httptest without binding a real port.
func newHandler(w *workspace.Workspace) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(rw, r)
			return
		}
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = rw.Write(indexHTML)
	})
	mux.HandleFunc("/api/state", func(rw http.ResponseWriter, r *http.Request) {
		state, err := buildState(w)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(rw)
		enc.SetIndent("", "  ")
		_ = enc.Encode(state)
	})
	return mux
}

// --- Snapshot assembly ---

type dashboardState struct {
	Generated string        `json:"generated"`
	Projects  []projectView `json:"projects"`
	Agents    []agentView   `json:"agents"`
}

type projectView struct {
	Slug     string         `json:"slug"`
	Title    string         `json:"title"`
	Stage    string         `json:"stage"`
	Total    int            `json:"total"`
	Counts   map[string]int `json:"counts"` // status -> task count
	Burndown burndownView   `json:"burndown"`
}

type burndownView struct {
	DonePoints      float64       `json:"done_points"`
	RemainingPoints float64       `json:"remaining_points"`
	Unestimated     int           `json:"unestimated"`
	PerDay          []burndownDay `json:"per_day"`
}

type burndownDay struct {
	Day    string  `json:"day"`
	Points float64 `json:"points"`
}

type agentView struct {
	RunID        string `json:"run_id"`
	Child        string `json:"child"`
	Task         string `json:"task"`
	Role         string `json:"role"`
	Runtime      string `json:"runtime"`
	PID          int    `json:"pid"`
	Started      string `json:"started"`
	RuntimeSecs  int64  `json:"runtime_secs"`
	LastActivity string `json:"last_activity"`
}

// buildState reads the workspace fresh on every call — the dashboard has no
// cache of its own, so a page poll always reflects the current store and run
// directory (the same honesty rule `dacli agents` follows: liveness is never
// trusted from a stale read).
func buildState(w *workspace.Workspace) (dashboardState, error) {
	st := dashboardState{Generated: time.Now().UTC().Format(time.RFC3339)}

	projects, err := store.ListProjects(w)
	if err != nil {
		return st, err
	}
	for _, p := range projects {
		st.Projects = append(st.Projects, buildProjectView(w, p))
	}

	for _, rec := range liveAgents(w) {
		st.Agents = append(st.Agents, buildAgentView(w, rec))
	}
	return st, nil
}

func buildProjectView(w *workspace.Workspace, p *store.Project) projectView {
	tasks, _ := store.ListTasks(w, p.Slug, "")
	counts := map[string]int{}
	var doneP, remP float64
	var unestimated int
	perDay := map[string]float64{}
	for _, t := range tasks {
		counts[string(t.Status)]++
		tp, ok := t.Estimate()
		if !ok {
			unestimated++
			continue
		}
		if t.Status == model.StatusDone {
			doneP += tp.Expected()
			if day, ok := completionDay(t); ok {
				perDay[day] += tp.Expected()
			}
		} else {
			remP += tp.Expected()
		}
	}
	days := make([]string, 0, len(perDay))
	for d := range perDay {
		days = append(days, d)
	}
	sort.Strings(days)
	perDaySlice := make([]burndownDay, 0, len(days))
	for _, d := range days {
		perDaySlice = append(perDaySlice, burndownDay{Day: d, Points: perDay[d]})
	}
	return projectView{
		Slug: p.Slug, Title: p.Title, Stage: p.Stage,
		Total: len(tasks), Counts: counts,
		Burndown: burndownView{
			DonePoints: doneP, RemainingPoints: remP,
			Unestimated: unestimated, PerDay: perDaySlice,
		},
	}
}

// completionDay mirrors insight.completionDay: the task's Log section records
// "<date> ... completed by <actor>" on close, and that date is the only
// record of when a done task's points actually landed. Duplicated rather than
// imported — feature slices never import each other (arch_test.go).
func completionDay(t *store.Task) (string, bool) {
	s, ok := t.Doc.Section("Log")
	if !ok {
		return "", false
	}
	for _, line := range strings.Split(s.Content, "\n") {
		if strings.Contains(line, "completed by") {
			fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "- "))
			if len(fields) > 0 && len(fields[0]) >= 10 {
				return fields[0][:10], true
			}
		}
	}
	return "", false
}

func buildAgentView(w *workspace.Workspace, rec procmon.Record) agentView {
	last := rec.Started
	if fi, err := os.Stat(filepath.Join(w.RunDir(rec.RunID), "transcript.log")); err == nil {
		last = fi.ModTime()
	}
	return agentView{
		RunID: rec.RunID, Child: rec.Child, Task: rec.Task, Role: rec.Role,
		Runtime: rec.Runtime, PID: rec.PID,
		Started:      rec.Started.UTC().Format(time.RFC3339),
		RuntimeSecs:  int64(time.Since(rec.Started).Seconds()),
		LastActivity: last.UTC().Format(time.RFC3339),
	}
}

// liveAgents mirrors execution.liveAgents: read every run's proc.txt, keep
// the ones whose leader process is still alive (never trust the file alone —
// AliveRecord re-probes the PID/start-time pair), newest first. Duplicated
// for the same no-cross-slice-import reason as completionDay.
func liveAgents(w *workspace.Workspace) []procmon.Record {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	var out []procmon.Record
	for _, n := range names {
		rec, err := procmon.ReadRecord(filepath.Join(w.RunDir(n), "proc.txt"))
		if err != nil {
			continue
		}
		if procmon.AliveRecord(rec) {
			out = append(out, rec)
		}
	}
	return out
}
