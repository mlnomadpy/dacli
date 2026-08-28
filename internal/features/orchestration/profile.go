package orchestration

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/prompts"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const startUsage = "dacli start [--project SLUG] [--profile inspect|task|wave|loop|service] [--width N] [--harness FAMILY]... [--hybrid] [--allow-advisory-tokens] [--dry-run] [--configure] [--show] [--json]"

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
	AllowedRuntimes    []string `json:"allowed_runtimes,omitempty"`
	HarnessMode        string   `json:"harness_mode,omitempty"`
	AllowedHarnesses   []string `json:"allowed_harnesses,omitempty"`
	ImplementationRole string   `json:"implementation_role,omitempty"`
	ReviewRole         string   `json:"review_role,omitempty"`
	Selection          string   `json:"selection"`
	ConsequenceUplift  bool     `json:"consequence_uplift"`
	Fallback           string   `json:"fallback"`
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
	PerTaskTokens       int64         `json:"per_task_tokens"`
	PerCycleTokens      int64         `json:"per_cycle_tokens"`
	RollingTokens       int64         `json:"rolling_tokens"`
	RollingWindow       time.Duration `json:"rolling_window"`
	InvocationTime      time.Duration `json:"invocation_time"`
	AllowAdvisoryTokens bool          `json:"allow_advisory_tokens,omitempty"`
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
		Routing:      RoutingPolicy{HarnessMode: "single", Selection: "cheapest-capable", ConsequenceUplift: true, Fallback: "capability-and-cost"},
		Execution:    ExecutionPolicy{Profile: name, TaskLimit: tasks, CyclesPerInvocation: cycles, ServiceInvocations: invocations, IdleBackoff: 30 * time.Minute, LeaseTTL: 2 * time.Minute, Heartbeat: 30 * time.Second},
		Budgets:      BudgetPolicy{PerTaskTokens: 20000, PerCycleTokens: int64(max(1, width)) * 20000, RollingTokens: 240000, RollingWindow: 24 * time.Hour, InvocationTime: 6 * time.Hour},
		Verification: VerificationPolicy{MutationRequired: true, IndependentReviews: 1, ProviderDiversity: true},
		Landing:      LandingPolicy{Mode: "project", ChecksRequired: true, ReviewsRequired: 1, AutoMerge: false},
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

// repositoryProfile fills only repository-derived fields. Operating-mode
// defaults are provider-neutral, but verification cannot be: issue #801 showed
// that hard-coding this repository's Go gates into every new profile creates a
// policy that is both unrunnable and falsely authoritative on Python/Vue.
func repositoryProfile(w *workspace.Workspace, project, name string) (OperatingProfile, error) {
	p, err := defaultProfile(project, name)
	if err != nil || name == "inspect" {
		return p, err
	}
	commands, err := projectVerificationCommands(w, project)
	if err != nil {
		return OperatingProfile{}, err
	}
	p.Verification.Commands = commands
	projectRecord, err := store.LoadProject(w, project)
	if err != nil {
		return OperatingProfile{}, err
	}
	p.Landing.ProtectedBranch = projectRecord.Landing.Base
	return p, nil
}

func projectVerificationCommands(w *workspace.Workspace, project string) ([]string, error) {
	p, err := store.LoadProject(w, project)
	if err != nil {
		return nil, err
	}
	stack := prompts.StackFromProject(p.Doc)
	if stack.Recorded() {
		if stack.Build == "" || stack.Test == "" {
			return nil, clikit.Refusedf("project %s records stack %s but not both verification commands; add `Build with ` and `test with ` commands to the project Constraints, or configure verification.commands in %s before starting", project, stack.Label, profileFile(w, project))
		}
		commands := []string{stack.Build}
		if stack.Test != stack.Build {
			commands = append(commands, stack.Test)
		}
		return commands, nil
	}
	section, _ := p.Doc.Section("Codebase map")
	languages := map[string]bool{}
	inLanguages := false
	for _, line := range strings.Split(section.Content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**") {
			inLanguages = strings.EqualFold(strings.Trim(line, "* :"), "Languages")
			continue
		}
		if !inLanguages {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line == "" {
			continue
		}
		if label, _, ok := strings.Cut(line, " ("); ok {
			languages[strings.ToLower(strings.TrimSpace(label))] = true
		}
	}
	var commands []string
	add := func(values ...string) {
		for _, value := range values {
			if !slices.Contains(commands, value) {
				commands = append(commands, value)
			}
		}
	}
	if languages["go"] {
		add("gofmt -l .", "go vet ./...", "golangci-lint run", "go test ./...")
	}
	if languages["python"] {
		add("python -m pytest")
	}
	if languages["typescript"] || languages["javascript"] || languages["vue"] {
		add("npm test", "npm run build")
	}
	if languages["rust"] {
		add("cargo fmt --check", "cargo clippy --all-targets --all-features -- -D warnings", "cargo test --all-features")
	}
	if len(commands) == 0 {
		return nil, clikit.Refusedf("cannot derive verification for project %s from its adopted codebase map — run `dacli adopt --project %s` to refresh detected languages, then configure explicit verification commands if the stack is still unknown or ambiguous", project, project)
	}
	return commands, nil
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
	if p.Execution.Profile != "inspect" && len(p.Verification.Commands) == 0 {
		return clikit.Refusedf("profile %s has no verification commands; configure commands supported by the adopted codebase map before execution", p.Execution.Profile)
	}
	if p.Execution.Profile != "inspect" {
		switch p.Routing.HarnessMode {
		case "", "single":
			if len(p.Routing.AllowedHarnesses) > 1 {
				return clikit.Refusedf("single-harness profile permits at most one allowed harness")
			}
		case "hybrid":
			if len(p.Routing.AllowedHarnesses) < 2 {
				return clikit.Refusedf("hybrid profile needs at least two explicitly allowed harnesses")
			}
		default:
			return clikit.Refusedf("unknown harness mode %q (want single or hybrid)", p.Routing.HarnessMode)
		}
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
	f, err := clikit.ParseFlags(args, "project", "profile", "width", "harness")
	if err != nil {
		return err
	}
	if err := f.Reject("project", "profile", "width", "harness", "hybrid", "allow-advisory-tokens", "dry-run", "configure", "show"); err != nil {
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
		// A repeated --profile is a field override, not permission to replace
		// every omitted repository-specific field with package defaults. Load the
		// persisted project policy first; only a project with no profile yet gets
		// stack-aware defaults.
		p, err = loadProfile(w, project)
		if errors.Is(err, os.ErrNotExist) {
			p, err = repositoryProfile(w, project, name)
			p.Provenance.Source = "--profile"
		} else if err == nil {
			p.Execution.Profile = name
			p.Provenance.Source = "persisted+--profile"
			if p.Provenance.Overrides == nil {
				p.Provenance.Overrides = map[string]string{}
			}
			p.Provenance.Overrides["profile"] = name
			p.Provenance.ResolvedAt = time.Now().UTC()
		}
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
		if p.Provenance.Overrides == nil {
			p.Provenance.Overrides = map[string]string{}
		}
		p.Provenance.Overrides["width"] = f.Get("width")
	}
	if f.Bool("allow-advisory-tokens") {
		p.Budgets.AllowAdvisoryTokens = true
		if p.Provenance.Overrides == nil {
			p.Provenance.Overrides = map[string]string{}
		}
		p.Provenance.Overrides["allow_advisory_tokens"] = "true"
	}
	if err := resolveProfileHarness(w, &p, f.All("harness"), f.Bool("hybrid")); err != nil {
		return err
	}
	if err := resolveProfileRoles(w, &p); err != nil {
		return err
	}
	if err := validateProfile(p); err != nil {
		return err
	}
	plan, err := buildProfilePlan(w, p)
	if err != nil {
		return err
	}
	noLaunch := f.Bool("dry-run") || f.Bool("configure") || f.Bool("show") || ctx.JSON
	// Inspect is an actual read-only operating mode, not merely a loop that
	// happens to select no writers. Persist it only when the operator explicitly
	// asks to configure it; otherwise a ro reviewer can execute the audit without
	// changing project state. The dispatcher has the matching conditional gate.
	persist := !f.Bool("dry-run") && !f.Bool("show") && (p.Execution.Profile != "inspect" || f.Bool("configure"))
	if persist {
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
	roles = rolesForHarnesses(w, roles, p.Routing.AllowedHarnesses)
	var tokenExclusions []string
	if p.Budgets.PerTaskTokens > 0 && !p.Budgets.AllowAdvisoryTokens {
		eligible := roles[:0]
		for _, role := range roles {
			rt, err := store.LoadRuntime(w, role.Runtime)
			if err != nil || rt.TokenLimitFlag == "" {
				tokenExclusions = append(tokenExclusions, fmt.Sprintf("%s/%s", role.Name, clikit.OrDash(role.Runtime)))
				continue
			}
			eligible = append(eligible, role)
		}
		roles = eligible
	}
	limit := p.Execution.TaskLimit
	if limit > len(ready) {
		limit = len(ready)
	}
	wave := selectClaimCompatibleWave(w.Root, ready, limit)
	for _, t := range wave.Tasks {
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
			pt.RoutingReason = "no capable roster role; execution will refuse rather than bypass policy"
			if len(tokenExclusions) > 0 {
				pt.RoutingReason += "; hard token policy excluded " + strings.Join(tokenExclusions, ", ")
			}
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
	switch p.Execution.Profile {
	case "inspect":
		fmt.Fprintln(w, "  sizing: not required; inspect reads evidence and launches no workers")
	case "task":
		fmt.Fprintln(w, "  sizing: optional for one bounded task; acceptance and capability still gate execution")
	case "wave", "loop", "service":
		fmt.Fprintln(w, "  sizing: estimate tasks to make capacity, critical-path slack, timeout, and spend projections meaningful; unestimated work remains visible as unknown")
	}
	fmt.Fprintf(w, "  scheduling: width=%d wip=%d ordering=%s\n", p.Scheduling.Width, p.Scheduling.WIP, strings.Join(p.Scheduling.Ordering, " → "))
	fmt.Fprintf(w, "  harnesses: mode=%s allowed=%s\n", p.Routing.HarnessMode, strings.Join(p.Routing.AllowedHarnesses, ","))
	fmt.Fprintf(w, "  routing: implementation=%s review=%s selection=%s\n", clikit.OrDash(p.Routing.ImplementationRole), clikit.OrDash(p.Routing.ReviewRole), p.Routing.Selection)
	fmt.Fprintf(w, "  budgets: task=%d cycle=%d rolling=%d/%s invocation=%s advisory-tokens=%t\n", p.Budgets.PerTaskTokens, p.Budgets.PerCycleTokens, p.Budgets.RollingTokens, p.Budgets.RollingWindow, p.Budgets.InvocationTime, p.Budgets.AllowAdvisoryTokens)
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
	plan, err := buildProfilePlan(w, p)
	if err != nil {
		return err
	}
	if err := requireVerificationCapabilities(w, p, plan); err != nil {
		return err
	}
	if p.Execution.Profile == "service" {
		return runService(ctx, w, p, r)
	}
	args := profileLoopArgs(p)
	out, err := r.run(p.Execution.Profile, args...)
	fmt.Fprint(ctx.Stdout, out)
	return err
}

func requireVerificationCapabilities(w *workspace.Workspace, p OperatingProfile, plan ProfilePlan) error {
	for _, capability := range store.RequiredExecutionCapabilities(p.Verification.Commands) {
		for _, task := range plan.Tasks {
			if task.Runtime == "" {
				continue
			}
			rt, err := store.LoadRuntime(w, task.Runtime)
			if err != nil {
				return err
			}
			if !store.RuntimeHasExecutionCapability(rt, capability) {
				return clikit.Refusedf("runtime %s has startup compatibility but does not declare required build capability %s for Gradle verification; no worker was started. Use a runtime whose documented sandbox contract permits local coordination sockets, or run Gradle verification outside the worker sandbox", rt.Name, capability)
			}
		}
	}
	return nil
}

func profileLoopArgs(p OperatingProfile) []string {
	cycles, width := p.Execution.CyclesPerInvocation, p.Scheduling.Width
	args := []string{"loop", "--project", p.Project, "--width", fmt.Sprint(width), "--max-cycles", fmt.Sprint(cycles), "--max-tokens", fmt.Sprint(p.Budgets.PerTaskTokens), "--window-tokens", fmt.Sprint(p.Budgets.RollingTokens), "--token-window", p.Budgets.RollingWindow.String(), "--idle", p.Execution.IdleBackoff.String(), "--stop-file", p.Recovery.StopFile}
	if p.Routing.ImplementationRole != "" {
		// A profile-resolved role is the automatic router's project-specific
		// fallback, not an operator override. Keeping those meanings separate
		// preserves per-task cheapest-capable selection and consequence uplift.
		args = append(args, "--impl-role-fallback", p.Routing.ImplementationRole)
	}
	if p.Routing.ReviewRole != "" {
		args = append(args, "--review-role", p.Routing.ReviewRole)
	}
	for _, harness := range p.Routing.AllowedHarnesses {
		args = append(args, "--harness", harness)
	}
	if p.Routing.HarnessMode == "hybrid" {
		args = append(args, "--hybrid")
	}
	if p.Budgets.AllowAdvisoryTokens {
		args = append(args, "--allow-advisory-tokens")
	}
	switch p.Landing.Mode {
	case "pr":
		args = append(args, "--pr")
	case "local":
		args = append(args, "--no-pr")
	}
	if p.Landing.ProtectedBranch != "" {
		args = append(args, "--into", p.Landing.ProtectedBranch)
	}
	if p.Landing.AutoMerge {
		args = append(args, "--auto-merge")
	} else {
		args = append(args, "--no-auto-merge")
	}
	return args
}

func resolveProfileHarness(w *workspace.Workspace, p *OperatingProfile, requested []string, hybrid bool) error {
	if len(requested) > 0 {
		p.Routing.AllowedHarnesses = uniqueStrings(requested)
		if hybrid {
			if len(p.Routing.AllowedHarnesses) < 2 {
				return clikit.Usagef("--hybrid requires at least two distinct --harness values")
			}
			p.Routing.HarnessMode = "hybrid"
		} else {
			if len(p.Routing.AllowedHarnesses) != 1 {
				return clikit.Usagef("single-harness mode needs exactly one --harness value")
			}
			p.Routing.HarnessMode = "single"
		}
		if p.Provenance.Overrides == nil {
			p.Provenance.Overrides = map[string]string{}
		}
		p.Provenance.Overrides["harnesses"] = strings.Join(p.Routing.AllowedHarnesses, ",")
		p.Provenance.Overrides["harness_mode"] = p.Routing.HarnessMode
	} else if hybrid {
		return clikit.Usagef("--hybrid requires at least two --harness values")
	}
	if len(p.Routing.AllowedHarnesses) == 0 && p.Execution.Profile != "inspect" {
		stack := loopStack(w, p.Project)
		inRoster := func(name string) bool { _, ok := store.LoadRole(w, name); return ok }
		roleName := p.Routing.ImplementationRole
		if roleName == "" {
			roleName = prompts.RoleFor(stack, "fixer", "fixer", inRoster)
		}
		role, ok := store.LoadRole(w, roleName)
		if ok {
			if rt, err := store.LoadRuntime(w, role.Runtime); err == nil {
				p.Routing.HarnessMode = "single"
				p.Routing.AllowedHarnesses = []string{rt.Harness}
			}
		}
	}
	if p.Routing.HarnessMode == "" {
		p.Routing.HarnessMode = "single"
	}
	p.Verification.ProviderDiversity = p.Routing.HarnessMode == "hybrid"
	return nil
}

// resolveProfileRoles makes project-declared stack roles part of the durable
// profile. Issue #798 showed that recomputing defaults inside loop made the
// preview promise Android seats while execution tried nonexistent Go seats.
func resolveProfileRoles(w *workspace.Workspace, p *OperatingProfile) error {
	if p.Execution.Profile == "inspect" {
		return nil
	}
	project, err := store.LoadProject(w, p.Project)
	if err != nil {
		return err
	}
	stack := prompts.StackFromProject(project.Doc)
	if !stack.Recorded() {
		return nil
	}
	roles, err := store.LoadRoles(w)
	if err != nil {
		return err
	}
	resolve := func(current, kind, suggested string) (string, error) {
		if current != "" {
			role, ok := store.LoadRole(w, current)
			if !ok || !strings.EqualFold(role.Kind, kind) {
				return "", clikit.Refusedf("profile declares missing %s role %s; run `dacli role add %s --kind %s` or configure routing.%s_role in %s", kind, current, current, kind, kindName(kind), profileFile(w, p.Project))
			}
			if len(p.Routing.AllowedHarnesses) > 0 && !roleAllowedByHarness(w, role, p.Routing.AllowedHarnesses) {
				return "", clikit.Refusedf("profile %s role %s is outside harness policy %s:%s; configure a compatible role before preview or execution", kind, current, p.Routing.HarnessMode, strings.Join(p.Routing.AllowedHarnesses, ","))
			}
			return current, nil
		}
		var matches []team.Role
		for _, role := range roles {
			if strings.EqualFold(role.Kind, kind) && roleMatchesStack(role, stack) &&
				(len(p.Routing.AllowedHarnesses) == 0 || roleAllowedByHarness(w, role, p.Routing.AllowedHarnesses)) {
				matches = append(matches, role)
			}
		}
		if role, ok := team.CheapestCapableForTitled(matches, kind, 0, nil, suggested, ""); ok {
			return role.Name, nil
		}
		name := stackRoleStem(stack) + "-" + kind
		return "", clikit.Refusedf("project %s declares stack %s but no %s role; run `dacli role add %s --kind %s` or configure routing.%s_role in %s", p.Project, stack.Label, kind, name, kind, kindName(kind), profileFile(w, p.Project))
	}
	p.Routing.ImplementationRole, err = resolve(p.Routing.ImplementationRole, "implementer", "Implement "+stack.Label+" project work")
	if err != nil {
		return err
	}
	// A new profile with no explicit harness derives its single-family boundary
	// from the resolved implementation seat before selecting review. This keeps
	// the preview and the live loop on one harness instead of letting the cheaper
	// reviewer silently cross providers (issue #845).
	if len(p.Routing.AllowedHarnesses) == 0 {
		role, _ := store.LoadRole(w, p.Routing.ImplementationRole)
		runtime, runtimeErr := store.LoadRuntime(w, role.Runtime)
		if runtimeErr != nil || strings.TrimSpace(runtime.Harness) == "" {
			return clikit.Refusedf("resolved implementation role %s has no observable harness; configure its runtime before preview or execution", p.Routing.ImplementationRole)
		}
		p.Routing.HarnessMode = "single"
		p.Routing.AllowedHarnesses = []string{runtime.Harness}
	}
	p.Routing.ReviewRole, err = resolve(p.Routing.ReviewRole, "reviewer", "Review "+stack.Label+" project work")
	p.Verification.ProviderDiversity = p.Routing.HarnessMode == "hybrid"
	return err
}

func stackRoleStem(stack prompts.Stack) string {
	for _, part := range strings.FieldsFunc(strings.ToLower(stack.Label), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if part != "" {
			return part
		}
	}
	return "project"
}

func kindName(kind string) string {
	if kind == "implementer" {
		return "implementation"
	}
	return "review"
}

func roleMatchesStack(role team.Role, stack prompts.Stack) bool {
	declared := strings.ToLower(role.Name + " " + role.Summary + " " + strings.Join(role.Skills, " ") + " " + strings.Join(role.Scope, " "))
	for _, part := range strings.FieldsFunc(strings.ToLower(stack.Label), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if len(part) > 1 && strings.Contains(declared, part) {
			return true
		}
	}
	return false
}

func rolesForHarnesses(w *workspace.Workspace, roles []team.Role, allowed []string) []team.Role {
	if len(allowed) == 0 {
		return roles
	}
	var out []team.Role
	for _, role := range roles {
		rt, err := store.LoadRuntime(w, role.Runtime)
		if err == nil && slices.Contains(allowed, rt.Harness) {
			out = append(out, role)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !slices.Contains(out, value) {
			out = append(out, value)
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
