package orchestration

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const startUsage = "dacli start [--project SLUG] [--profile inspect|task|wave|loop|service] [--width N] [--dry-run] [--configure] [--show] [--json]"

var profileInput io.Reader = os.Stdin

type OperatingProfile struct {
	Version      int                `json:"version"`
	Project      string             `json:"project"`
	Scheduling   SchedulingPolicy   `json:"scheduling"`
	Routing      RoutingPolicy      `json:"routing"`
	Execution    ExecutionPolicy    `json:"execution"`
	Budgets      BudgetPolicy       `json:"budgets"`
	Verification VerificationPolicy `json:"verification"`
	Landing      LandingPolicy      `json:"landing"`
	Release      ReleasePolicy      `json:"release"`
	Recovery     RecoveryPolicy     `json:"recovery"`
	Provenance   PolicyProvenance   `json:"provenance"`
}

type SchedulingPolicy struct {
	Priorities []string `json:"priorities"`
	Ordering   []string `json:"ordering"`
	Width      int      `json:"width"`
	WIP        int      `json:"wip"`
}
type RoutingPolicy struct {
	AllowedRuntimes   []string `json:"allowed_runtimes,omitempty"`
	Selection         string   `json:"selection"`
	ConsequenceUplift bool     `json:"consequence_uplift"`
	Fallback          string   `json:"fallback"`
}
type ExecutionPolicy struct {
	Profile             string        `json:"profile"`
	TaskLimit           int           `json:"task_limit"`
	CyclesPerInvocation int           `json:"cycles_per_invocation"`
	ServiceInvocations  int           `json:"service_invocations"`
	IdleBackoff         time.Duration `json:"idle_backoff"`
	LeaseTTL            time.Duration `json:"lease_ttl"`
	Heartbeat           time.Duration `json:"heartbeat"`
}
type BudgetPolicy struct {
	PerTaskTokens  int64         `json:"per_task_tokens"`
	PerCycleTokens int64         `json:"per_cycle_tokens"`
	RollingTokens  int64         `json:"rolling_tokens"`
	RollingWindow  time.Duration `json:"rolling_window"`
	InvocationTime time.Duration `json:"invocation_time"`
}
type VerificationPolicy struct {
	MutationRequired   bool     `json:"mutation_required"`
	Commands           []string `json:"commands"`
	IndependentReviews int      `json:"independent_reviews"`
	ProviderDiversity  bool     `json:"provider_diversity"`
}
type LandingPolicy struct {
	Mode            string `json:"mode"`
	ChecksRequired  bool   `json:"checks_required"`
	ReviewsRequired int    `json:"reviews_required"`
	AutoMerge       bool   `json:"auto_merge"`
	ProtectedBranch string `json:"protected_branch,omitempty"`
}
type ReleasePolicy struct {
	Enabled              bool     `json:"enabled"`
	PublicationAuthority bool     `json:"publication_authority"`
	Channel              string   `json:"channel,omitempty"`
	Cadence              string   `json:"cadence,omitempty"`
	Version              string   `json:"version,omitempty"`
	Gates                []string `json:"gates,omitempty"`
}
type RecoveryPolicy struct {
	Journal                    string `json:"journal"`
	StopFile                   string `json:"stop_file"`
	InfrastructureFailureLimit int    `json:"infrastructure_failure_limit"`
	DeadLetterThreshold        int    `json:"dead_letter_threshold"`
	UnknownLandingStops        bool   `json:"unknown_landing_stops"`
}
type PolicyProvenance struct {
	Source     string            `json:"source"`
	Overrides  map[string]string `json:"overrides,omitempty"`
	ResolvedAt time.Time         `json:"resolved_at"`
}

type ProfilePlan struct {
	Policy OperatingProfile `json:"policy"`
	Tasks  []PlannedTask    `json:"tasks"`
}
type PlannedTask struct {
	Ref           string   `json:"ref"`
	Title         string   `json:"title"`
	Priority      string   `json:"priority,omitempty"`
	Slack         *float64 `json:"slack,omitempty"`
	Claims        []string `json:"claims,omitempty"`
	Role          string   `json:"role,omitempty"`
	Runtime       string   `json:"runtime,omitempty"`
	Model         string   `json:"model,omitempty"`
	RoutingReason string   `json:"routing_reason"`
}

func defaultProfile(project, name string) (OperatingProfile, error) {
	width, tasks, cycles, invocations := 1, 1, 1, 1
	switch name {
	case "inspect":
		tasks, cycles = 0, 0
	case "task":
	case "wave":
		width, tasks = 3, 3
	case "loop":
		width, tasks, cycles = 2, 2, 3
	case "service":
		width, tasks, cycles, invocations = 2, 2, 3, 12
	default:
		return OperatingProfile{}, clikit.Usagef("unknown profile %q (want inspect, task, wave, loop, or service)", name)
	}
	p := OperatingProfile{
		Version: 1, Project: project,
		Scheduling:   SchedulingPolicy{Priorities: []string{"must", "should", "could"}, Ordering: []string{"dependency", "critical-path-slack", "priority", "sequence"}, Width: width, WIP: width},
		Routing:      RoutingPolicy{Selection: "cheapest-capable", ConsequenceUplift: true, Fallback: "capability-and-cost"},
		Execution:    ExecutionPolicy{Profile: name, TaskLimit: tasks, CyclesPerInvocation: cycles, ServiceInvocations: invocations, IdleBackoff: 30 * time.Minute, LeaseTTL: 2 * time.Minute, Heartbeat: 30 * time.Second},
		Budgets:      BudgetPolicy{PerTaskTokens: 20000, PerCycleTokens: int64(max(1, width)) * 20000, RollingTokens: 240000, RollingWindow: 24 * time.Hour, InvocationTime: 6 * time.Hour},
		Verification: VerificationPolicy{MutationRequired: true, Commands: []string{"gofmt -l .", "go vet ./...", "golangci-lint run", "go test ./..."}, IndependentReviews: 1, ProviderDiversity: true},
		Landing:      LandingPolicy{Mode: "project", ChecksRequired: true, ReviewsRequired: 1, AutoMerge: true},
		Release:      ReleasePolicy{Enabled: false, PublicationAuthority: false},
		Recovery:     RecoveryPolicy{Journal: filepath.ToSlash(filepath.Join(workspace.Dir, "profiles", project+"-service.json")), StopFile: filepath.ToSlash(filepath.Join(workspace.Dir, "STOP")), InfrastructureFailureLimit: 3, DeadLetterThreshold: 3, UnknownLandingStops: true},
		Provenance:   PolicyProvenance{Source: "defaults", ResolvedAt: time.Now().UTC()},
	}
	if name == "inspect" {
		p.Budgets = BudgetPolicy{}
		p.Verification = VerificationPolicy{}
		p.Landing = LandingPolicy{Mode: "none"}
	}
	return p, nil
}

func profileFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "profiles", project+".json")
}

func saveProfile(w *workspace.Workspace, p OperatingProfile) error {
	if _, err := store.LoadProject(w, p.Project); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	path := profileFile(w, p.Project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeStateFile(path, string(b)+"\n")
}

func loadProfile(w *workspace.Workspace, project string) (OperatingProfile, error) {
	b, err := os.ReadFile(profileFile(w, project))
	if err != nil {
		return OperatingProfile{}, err
	}
	var p OperatingProfile
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("invalid operating profile %s: %w", profileFile(w, project), err)
	}
	if p.Version != 1 || p.Project != project {
		return p, clikit.Refusedf("operating profile identity/version mismatch for project %s; inspect %s", project, profileFile(w, project))
	}
	return p, validateProfile(p)
}

func validateProfile(p OperatingProfile) error {
	if _, err := defaultProfile(p.Project, p.Execution.Profile); err != nil {
		return err
	}
	if p.Execution.Profile != "inspect" && (p.Execution.TaskLimit <= 0 || p.Execution.CyclesPerInvocation <= 0 || p.Budgets.RollingTokens <= 0 || p.Budgets.RollingWindow <= 0) {
		return clikit.Refusedf("profile %s has an unbounded task, cycle, or rolling-token policy; configure finite positive bounds", p.Execution.Profile)
	}
	if p.Execution.Profile == "service" && (p.Execution.ServiceInvocations <= 0 || p.Execution.LeaseTTL <= 0 || p.Execution.Heartbeat <= 0) {
		return clikit.Refusedf("service profile needs finite invocation, lease, and heartbeat bounds")
	}
	if p.Release.Enabled && (!p.Release.PublicationAuthority || p.Release.Channel == "" || p.Release.Cadence == "" || p.Release.Version == "" || len(p.Release.Gates) == 0) {
		return clikit.Refusedf("release publication needs separate explicit authority, channel, cadence, version, and gates")
	}
	return nil
}

func resolveProject(w *workspace.Workspace, requested string) (string, error) {
	if requested != "" {
		_, err := store.LoadProject(w, requested)
		return requested, err
	}
	ps, err := store.ListProjects(w)
	if err != nil {
		return "", err
	}
	if len(ps) != 1 {
		return "", clikit.Usagef("--project is required when the workspace has %d projects", len(ps))
	}
	return ps[0].Slug, nil
}

func cmdStart(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, err := clikit.ParseFlags(args, "project", "profile", "width")
	if err != nil {
		return err
	}
	if err := f.Reject("project", "profile", "width", "dry-run", "configure", "show"); err != nil {
		return err
	}
	project, err := resolveProject(w, f.Get("project"))
	if err != nil {
		return err
	}
	name := f.Get("profile")
	var p OperatingProfile
	if name == "" && (f.Bool("show") || ctx.JSON) {
		p, err = loadProfile(w, project)
		if errors.Is(err, os.ErrNotExist) {
			p, err = defaultProfile(project, "task")
			p.Provenance.Source = "migration-default"
		}
	} else {
		if name == "" {
			fmt.Fprint(ctx.Stdout, "Select profile [inspect/task/wave/loop/service]: ")
			s, readErr := bufio.NewReader(profileInput).ReadString('\n')
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				return readErr
			}
			name = strings.TrimSpace(s)
			if name == "" {
				return clikit.Usagef("a profile selection is required")
			}
		}
		p, err = defaultProfile(project, name)
		p.Provenance.Source = "--profile"
	}
	if err != nil {
		return err
	}
	if width, widthErr := f.Int("width", p.Scheduling.Width); widthErr != nil {
		return widthErr
	} else if f.Get("width") != "" {
		if width <= 0 {
			return clikit.Usagef("--width must be positive")
		}
		p.Scheduling.Width, p.Scheduling.WIP = width, width
		p.Execution.TaskLimit = width
		p.Budgets.PerCycleTokens = int64(width) * p.Budgets.PerTaskTokens
		p.Provenance.Overrides = map[string]string{"width": f.Get("width")}
	}
	if err := validateProfile(p); err != nil {
		return err
	}
	plan, err := buildProfilePlan(w, p)
	if err != nil {
		return err
	}
	noLaunch := f.Bool("dry-run") || f.Bool("configure") || f.Bool("show") || ctx.JSON
	if !f.Bool("dry-run") && !f.Bool("show") {
		if err := clikit.RequireRW(id, "persist operating profile"); err != nil {
			return err
		}
		if err := saveProfile(w, p); err != nil {
			return fmt.Errorf("persist operating profile: %w", err)
		}
	}
	if ctx.JSON {
		return json.NewEncoder(ctx.Stdout).Encode(plan)
	}
	printProfilePlan(ctx.Stdout, plan)
	if noLaunch {
		return nil
	}
	return executeProfile(ctx, w, p)
}

func buildProfilePlan(w *workspace.Workspace, p OperatingProfile) (ProfilePlan, error) {
	plan := ProfilePlan{Policy: p}
	if p.Execution.Profile == "inspect" {
		return plan, nil
	}
	ready, err := readyTasks(w, p.Project)
	if err != nil {
		return plan, err
	}
	rankByPriority(w, p.Project, ready)
	slack, haveSlack := criticalPathSlack(w, p.Project)
	roles, _ := store.LoadRoles(w)
	limit := p.Execution.TaskLimit
	if limit > len(ready) {
		limit = len(ready)
	}
	for _, t := range ready[:limit] {
		pt := PlannedTask{Ref: t.ID, Title: t.Title, Priority: t.Priority(), Claims: store.ClaimHints(w.Root, t), RoutingReason: "cheapest capable tier for estimated complexity"}
		if haveSlack {
			s := slack[t.ID]
			pt.Slack = &s
		}
		te := 0.0
		if est, ok := t.Estimate(); ok {
			te = est.Expected()
		}
		role, ok := team.CheapestCapableForTitled(roles, "implementer", te, pt.Claims, t.Title, "")
		if ok {
			if highConsequence(t.Title, pt.Claims) {
				if up, uplifted := upliftRole(roles, role, te, pt.Claims); uplifted {
					role = up
					pt.RoutingReason = "consequence uplift: security, persistence, or high ambiguity"
				}
			}
			pt.Role, pt.Runtime, pt.Model = role.Name, role.Runtime, role.ModelID()
		} else {
			pt.RoutingReason = "no capable roster role; existing strategy fallback will decide or refuse"
		}
		plan.Tasks = append(plan.Tasks, pt)
	}
	return plan, nil
}

func highConsequence(title string, paths []string) bool {
	s := strings.ToLower(title + " " + strings.Join(paths, " "))
	for _, word := range []string{"security", "auth", "permission", "migration", "persist", "database", "lease", "recovery", "ambiguous"} {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}

func upliftRole(roles []team.Role, current team.Role, te float64, paths []string) (team.Role, bool) {
	sort.SliceStable(roles, func(i, j int) bool {
		return team.ModelTier(roles[i].Profile.CostTier) < team.ModelTier(roles[j].Profile.CostTier)
	})
	cur := team.ModelTier(current.Profile.CostTier)
	for _, r := range roles {
		if r.Kind == current.Kind && team.ModelTier(r.Profile.CostTier) > cur && (r.TaskCapacity() <= 0 || r.TaskCapacity() >= te) {
			good := true
			for _, path := range paths {
				if !r.InScope(path) {
					good = false
				}
			}
			if good {
				return r, true
			}
		}
	}
	return current, false
}

func printProfilePlan(w io.Writer, plan ProfilePlan) {
	p := plan.Policy
	fmt.Fprintf(w, "OperatingProfile %s (project %s, source %s)\n", p.Execution.Profile, p.Project, p.Provenance.Source)
	fmt.Fprintf(w, "  scheduling: width=%d wip=%d ordering=%s\n", p.Scheduling.Width, p.Scheduling.WIP, strings.Join(p.Scheduling.Ordering, " → "))
	fmt.Fprintf(w, "  budgets: task=%d cycle=%d rolling=%d/%s invocation=%s\n", p.Budgets.PerTaskTokens, p.Budgets.PerCycleTokens, p.Budgets.RollingTokens, p.Budgets.RollingWindow, p.Budgets.InvocationTime)
	fmt.Fprintf(w, "  verification: mutation=%t commands=%s reviews=%d diverse=%t\n", p.Verification.MutationRequired, strings.Join(p.Verification.Commands, "; "), p.Verification.IndependentReviews, p.Verification.ProviderDiversity)
	fmt.Fprintf(w, "  landing: mode=%s checks=%t reviews=%d auto-merge=%t\n", p.Landing.Mode, p.Landing.ChecksRequired, p.Landing.ReviewsRequired, p.Landing.AutoMerge)
	fmt.Fprintf(w, "  release: enabled=%t publication-authority=%t\n", p.Release.Enabled, p.Release.PublicationAuthority)
	fmt.Fprintf(w, "  recovery: journal=%s stop=%s breaker=%d unknown-landing-stops=%t\n", p.Recovery.Journal, p.Recovery.StopFile, p.Recovery.InfrastructureFailureLimit, p.Recovery.UnknownLandingStops)
	for _, t := range plan.Tasks {
		slack := "unknown"
		if t.Slack != nil {
			slack = fmt.Sprintf("%.1f", *t.Slack)
		}
		fmt.Fprintf(w, "  task %s priority=%s slack=%s claims=%s route=%s/%s/%s (%s)\n", t.Ref, clikit.OrDash(t.Priority), slack, clikit.OrDash(strings.Join(t.Claims, ",")), clikit.OrDash(t.Role), clikit.OrDash(t.Runtime), clikit.OrDash(t.Model), t.RoutingReason)
	}
}

func executeProfile(ctx *clikit.Ctx, w *workspace.Workspace, p OperatingProfile) error {
	r := execRunner{cwd: ctx.Cwd}
	if p.Execution.Profile == "inspect" {
		for _, argv := range [][]string{{"status"}, {"doctor"}, {"task", "list", "--project", p.Project, "--status", "open"}} {
			out, err := r.run("inspect", argv...)
			fmt.Fprint(ctx.Stdout, out)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if p.Execution.Profile == "service" {
		return runService(ctx, w, p, r)
	}
	args := profileLoopArgs(p)
	out, err := r.run(p.Execution.Profile, args...)
	fmt.Fprint(ctx.Stdout, out)
	return err
}

func profileLoopArgs(p OperatingProfile) []string {
	cycles, width := p.Execution.CyclesPerInvocation, p.Scheduling.Width
	args := []string{"loop", "--project", p.Project, "--width", fmt.Sprint(width), "--max-cycles", fmt.Sprint(cycles), "--max-tokens", fmt.Sprint(p.Budgets.PerTaskTokens), "--window-tokens", fmt.Sprint(p.Budgets.RollingTokens), "--token-window", p.Budgets.RollingWindow.String(), "--idle", p.Execution.IdleBackoff.String(), "--stop-file", p.Recovery.StopFile}
	return args
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
