// Package dashboard serves a self-contained, read-only local web UI over the
// workspace: projects, tasks by status with a burndown, the live agent swarm
// (task/role/runtime/last activity), and the pending (unsynced) event count —
// read straight from the store, the run directory's proc.txt records, and the
// event log, the same sources `dacli status` and `dacli agents` already read.
// Nothing here mutates the workspace.
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
	"github.com/mlnomadpy/dacli/internal/eventlog"
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

// newHandler builds the whole server: the embedded page at "/", the combined
// JSON snapshot the legacy page polls at "/api/state", and the four typed
// per-surface endpoints the Vue SPA reads (/api/overview, /api/projects,
// /api/tasks, /api/agents). Every JSON handler reads the workspace fresh on
// each request — no cache — so a poll always reflects the live store and event
// log (the same honesty rule buildState follows). Factored out so tests can
// drive it through httptest without binding a real port.
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
	// Legacy combined snapshot — the self-contained static/index.html polls
	// this. Preserved verbatim so the existing dashboard keeps working until
	// the SPA (which reads the typed endpoints below) replaces it.
	mux.HandleFunc("/api/state", func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, func() (any, error) { return buildState(w) })
	})
	// Typed per-surface endpoints for the SPA. Each is an envelope carrying its
	// own `generated` stamp so a surface can be polled independently and still
	// reason about freshness. Their payloads reuse the same view builders as
	// /api/state, so the two contracts can never drift.
	mux.HandleFunc("/api/overview", func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, func() (any, error) { return buildOverview(w) })
	})
	mux.HandleFunc("/api/projects", func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, func() (any, error) { return buildProjects(w) })
	})
	mux.HandleFunc("/api/tasks", func(rw http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("project")
		writeJSON(rw, func() (any, error) { return buildTasks(w, project) })
	})
	mux.HandleFunc("/api/agents", func(rw http.ResponseWriter, r *http.Request) {
		writeJSON(rw, func() (any, error) { return buildAgents(w) })
	})
	return mux
}

// writeJSON runs build (a fresh workspace read), then encodes the result as
// indented JSON with the dashboard's standard content type. A build error
// becomes a 500 with the error text, mirroring the original /api/state handler.
func writeJSON(rw http.ResponseWriter, build func() (any, error)) {
	v, err := build()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(rw)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// --- Snapshot assembly ---

type dashboardState struct {
	Generated     string        `json:"generated"`
	Projects      []projectView `json:"projects"`
	Agents        []agentView   `json:"agents"`
	PendingEvents int           `json:"pending_events"` // unsynced child events (eventlog), as `dacli status` reports
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

	pending, _ := eventlog.List(w, eventlog.Query{Pending: true})
	st.PendingEvents = len(pending)
	return st, nil
}

// --- Typed per-surface endpoints (SPA contract) ---
//
// Each endpoint returns an envelope whose `generated` field is an RFC3339 UTC
// stamp of when the snapshot was built, exactly like dashboardState.Generated,
// so a client polling one surface can still show freshness. Payloads reuse the
// buildProjectView / buildAgentView builders that /api/state uses, so the typed
// endpoints and the combined snapshot can never disagree about a shared field.

// overviewResponse is GET /api/overview: the workspace-health signals the
// Overview surface reads at a glance — project/task totals, aggregate task
// counts across every project, the unsynced-event count, and how many agents
// are live right now. It carries no per-project or per-task detail; those live
// on /api/projects and /api/tasks.
type overviewResponse struct {
	Generated     string         `json:"generated"`
	ProjectCount  int            `json:"project_count"`
	TaskCount     int            `json:"task_count"`     // total tasks across all projects
	Counts        map[string]int `json:"counts"`         // status -> task count, summed across projects
	PendingEvents int            `json:"pending_events"` // unsynced child events, as `dacli status` reports
	LiveAgents    int            `json:"live_agents"`    // count of liveness-probed running agents
}

// projectsResponse is GET /api/projects: the full projectView list (slug,
// title, stage, per-status counts, and burndown) — identical to the `projects`
// array inside /api/state.
type projectsResponse struct {
	Generated string        `json:"generated"`
	Projects  []projectView `json:"projects"`
}

// tasksResponse is GET /api/tasks: individual task rows, the per-task detail
// /api/state deliberately omits (it carries only per-status counts). This is
// the "future snapshot [that] adds a task list" DESIGN.md §7.2 anticipated, so
// the board can render real task identities instead of magnitude-only chips.
// An optional ?project=<slug> query filters to one project; absent, every
// project's tasks are returned, sorted by project then sequence (ListTasks'
// order).
type tasksResponse struct {
	Generated string     `json:"generated"`
	Tasks     []taskView `json:"tasks"`
}

// taskView is one task row. Points is the PERT expected value of the task's
// three-point estimate; when the task has no valid estimate, Estimated is false
// and Points is 0 (it contributes to counts/totals but not to burndown points,
// the same rule buildProjectView applies).
type taskView struct {
	ID        string  `json:"id"`
	Project   string  `json:"project"`
	Seq       int     `json:"seq"`
	Slug      string  `json:"slug"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	Priority  string  `json:"priority"`
	Owner     string  `json:"owner"`
	Points    float64 `json:"points"`    // PERT expected; 0 when unestimated
	Estimated bool    `json:"estimated"` // whether a valid three-point estimate exists
}

// agentsResponse is GET /api/agents: the live agent swarm, identical to the
// `agents` array inside /api/state — newest-first and already liveness-filtered
// (never trust proc.txt alone; AliveRecord re-probes each PID).
type agentsResponse struct {
	Generated string      `json:"generated"`
	Agents    []agentView `json:"agents"`
}

func nowStamp() string { return time.Now().UTC().Format(time.RFC3339) }

func buildOverview(w *workspace.Workspace) (overviewResponse, error) {
	resp := overviewResponse{Generated: nowStamp(), Counts: map[string]int{}}
	projects, err := store.ListProjects(w)
	if err != nil {
		return resp, err
	}
	resp.ProjectCount = len(projects)
	for _, p := range projects {
		tasks, _ := store.ListTasks(w, p.Slug, "")
		resp.TaskCount += len(tasks)
		for _, t := range tasks {
			resp.Counts[string(t.Status)]++
		}
	}
	pending, _ := eventlog.List(w, eventlog.Query{Pending: true})
	resp.PendingEvents = len(pending)
	resp.LiveAgents = len(liveAgents(w))
	return resp, nil
}

func buildProjects(w *workspace.Workspace) (projectsResponse, error) {
	resp := projectsResponse{Generated: nowStamp()}
	projects, err := store.ListProjects(w)
	if err != nil {
		return resp, err
	}
	for _, p := range projects {
		resp.Projects = append(resp.Projects, buildProjectView(w, p))
	}
	return resp, nil
}

// buildTasks lists task rows, optionally filtered to one project. An empty
// project filter yields every project's tasks (ListTasks with project == ""),
// so a single request can drive the whole board.
func buildTasks(w *workspace.Workspace, project string) (tasksResponse, error) {
	resp := tasksResponse{Generated: nowStamp()}
	tasks, err := store.ListTasks(w, project, "")
	if err != nil {
		return resp, err
	}
	for _, t := range tasks {
		tv := taskView{
			ID: t.ID, Project: t.Project, Seq: t.Seq, Slug: t.Slug,
			Title: t.Title, Status: string(t.Status),
			Priority: t.Priority(), Owner: t.Owner(),
		}
		if tp, ok := t.Estimate(); ok {
			tv.Points = tp.Expected()
			tv.Estimated = true
		}
		resp.Tasks = append(resp.Tasks, tv)
	}
	return resp, nil
}

func buildAgents(w *workspace.Workspace) (agentsResponse, error) {
	resp := agentsResponse{Generated: nowStamp()}
	for _, rec := range liveAgents(w) {
		resp.Agents = append(resp.Agents, buildAgentView(w, rec))
	}
	return resp, nil
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
