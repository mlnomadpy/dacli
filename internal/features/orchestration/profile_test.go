package orchestration

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestOperatingProfileGoldenDefaultsAreFiniteAndReleaseIsOff(t *testing.T) {
	raw, err := os.ReadFile("testdata/operating-profile-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden OperatingProfile
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	if err := validateProfile(golden); err != nil {
		t.Fatalf("golden profile invalid: %v", err)
	}
	for _, name := range []string{"task", "wave", "loop", "service"} {
		p, err := defaultProfile("p", name)
		if err != nil {
			t.Fatal(err)
		}
		if p.Execution.TaskLimit <= 0 || p.Execution.CyclesPerInvocation <= 0 || p.Budgets.RollingTokens <= 0 || p.Budgets.RollingWindow <= 0 {
			t.Fatalf("%s has an unbounded default: %+v", name, p)
		}
		if p.Release.Enabled || p.Release.PublicationAuthority {
			t.Fatalf("%s silently acquired release authority: %+v", name, p.Release)
		}
	}
	service, _ := defaultProfile("p", "service")
	if service.Execution.ServiceInvocations <= 1 || service.Execution.CyclesPerInvocation <= 0 {
		t.Fatalf("service must repeat bounded loops: %+v", service.Execution)
	}
}

func TestStartHeadlessPersistsAndJSONShowsWithoutLaunching(t *testing.T) {
	w := loopEnv(t)
	setProjectCodebaseMap(t, w, "Go")
	if _, err := store.CreateTask(w, "a-root", "p", "Persist secure recovery policy", store.TaskOpts{Accept: []string{"a"}, Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdStart(ctx, []string{"--project", "p", "--profile", "wave", "--width", "2", "--configure"}); err != nil {
		t.Fatal(err)
	}
	p, err := loadProfile(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if p.Execution.Profile != "wave" || p.Scheduling.Width != 2 || p.Provenance.Overrides["width"] != "2" {
		t.Fatalf("persisted policy lost resolution/provenance: %+v", p)
	}

	out := &bytes.Buffer{}
	ctx = &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdStart(ctx, []string{"--project", "p", "--show"}); err != nil {
		t.Fatal(err)
	}
	var plan ProfilePlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatalf("JSON output: %v\n%s", err, out.String())
	}
	if plan.Policy.Execution.Profile != "wave" || len(plan.Tasks) != 1 {
		t.Fatalf("shown plan = %+v", plan)
	}
	if plan.Tasks[0].Ref == "" || plan.Tasks[0].Title == "" || plan.Tasks[0].RoutingReason == "" || plan.Policy.Release.Enabled {
		t.Fatalf("preview omitted routing reason or enabled release: %+v", plan)
	}
}

func TestStartInteractiveSelectsProfile(t *testing.T) {
	w := loopEnv(t)
	old := profileInput
	profileInput = strings.NewReader("inspect\n")
	t.Cleanup(func() { profileInput = old })
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdStart(ctx, []string{"--project", "p", "--configure"}); err != nil {
		t.Fatal(err)
	}
	p, err := loadProfile(w, "p")
	if err != nil || p.Execution.Profile != "inspect" {
		t.Fatalf("interactive profile = %+v, %v", p, err)
	}
}

func TestInspectDoesNotPersistUnlessConfigured(t *testing.T) {
	w := loopEnv(t)
	p, _ := defaultProfile("p", "inspect")
	if err := saveProfile(w, p); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(profileFile(w, "p")); err != nil {
		t.Fatal(err)
	}
	// The persistence predicate is exercised through a JSON inspection, which
	// does not shell out to the test binary but follows the same inspect path.
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdStart(ctx, []string{"--project", "p", "--profile", "inspect"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(profileFile(w, "p")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inspect changed project policy: %v", err)
	}
}

func TestStartDryRunReportsPolicyAndDoesNotPersist(t *testing.T) {
	w := loopEnv(t)
	setProjectCodebaseMap(t, w, "Go")
	out := &bytes.Buffer{}
	ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdStart(ctx, []string{"--project", "p", "--profile", "service", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"OperatingProfile service", "scheduling:", "budgets:", "verification:", "landing:", "release: enabled=false", "recovery:"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("dry-run missing %q:\n%s", want, out.String())
		}
	}
	if _, err := os.Stat(profileFile(w, "p")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run persisted policy: %v", err)
	}
}

func setProjectCodebaseMap(t *testing.T, w *workspace.Workspace, languages ...string) {
	t.Helper()
	p, err := store.LoadProject(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	body.WriteString("**Languages:**\n")
	for _, language := range languages {
		body.WriteString("- " + language + " (1 files)\n")
	}
	p.Doc.SetSection("Codebase map", body.String())
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

// Issue #801's public Python/Vue monorepo: repeating --profile selects the
// operating mode but must not regenerate omitted verification or landing
// fields. Only --width changes scheduling/task limits and their derived token
// total; the saved repository policy remains the source of truth.
func TestStartProfileOverridesPreservePersistedPythonVuePolicy(t *testing.T) {
	w := loopEnv(t)
	setProjectCodebaseMap(t, w, "Python", "TypeScript", "Vue")
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdStart(ctx, []string{"--project", "p", "--profile", "loop", "--configure"}); err != nil {
		t.Fatal(err)
	}
	p, err := loadProfile(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(p.Verification.Commands, " "), "go test") || p.Landing.AutoMerge {
		t.Fatalf("non-Go profile received Go/auto-merge defaults: %+v", p)
	}
	p.Verification.Commands = []string{"pytest backend/tests", "npm --prefix frontend test", "npm --prefix sdk test"}
	p.Landing = LandingPolicy{Mode: "project", ProtectedBranch: "dev", ChecksRequired: true, AutoMerge: false}
	if err := saveProfile(w, p); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	ctx = &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
	if err := cmdStart(ctx, []string{"--project", "p", "--profile", "loop", "--width", "1", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	var plan ProfilePlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatalf("decode preview: %v\n%s", err, out.String())
	}
	if !slices.Equal(plan.Policy.Verification.Commands, p.Verification.Commands) || plan.Policy.Landing != p.Landing {
		t.Fatalf("explicit profile/width regenerated omitted policy:\n got %+v\nwant %+v", plan.Policy, p)
	}
	if plan.Policy.Scheduling.Width != 1 || plan.Policy.Execution.TaskLimit != 1 {
		t.Fatalf("explicit width did not override its fields: %+v", plan.Policy)
	}
}

func TestStartConfigureRefusesUnknownStackInsteadOfGuessingGo(t *testing.T) {
	w := loopEnv(t)
	setProjectCodebaseMap(t, w, "Markdown")
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	err := cmdStart(ctx, []string{"--project", "p", "--profile", "loop", "--configure"})
	if clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "dacli adopt") || !strings.Contains(err.Error(), "verification") {
		t.Fatalf("unknown stack = %v (exit %d), want actionable refusal", err, clikit.ExitCode(err))
	}
}

func TestProfileLoopArgsCarryResolvedAutoMergePolicy(t *testing.T) {
	p, err := defaultProfile("p", "loop")
	if err != nil {
		t.Fatal(err)
	}
	if args := profileLoopArgs(p); !slices.Contains(args, "--no-auto-merge") || slices.Contains(args, "--auto-merge") {
		t.Fatalf("safe default not forwarded to real loop execution: %v", args)
	}
	p.Landing = LandingPolicy{Mode: "pr", ProtectedBranch: "dev", AutoMerge: true}
	if args := profileLoopArgs(p); !slices.Contains(args, "--auto-merge") || slices.Contains(args, "--no-auto-merge") || !slices.Contains(args, "--pr") || !containsAdjacent(args, "--into", "dev") {
		t.Fatalf("explicit auto-merge policy not forwarded: %v", args)
	}
}

func TestProfileLoopArgsCarryPersistedAdvisoryTokenPolicy(t *testing.T) {
	p, err := defaultProfile("p", "loop")
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(profileLoopArgs(p), "--allow-advisory-tokens") {
		t.Fatal("hard-cap default silently became advisory")
	}
	p.Budgets.AllowAdvisoryTokens = true
	if !slices.Contains(profileLoopArgs(p), "--allow-advisory-tokens") {
		t.Fatalf("persisted advisory policy was not forwarded: %v", profileLoopArgs(p))
	}
}

func TestStartPersistsExplicitAdvisoryTokenPolicy(t *testing.T) {
	w := loopEnv(t)
	setProjectCodebaseMap(t, w, "Go")
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := cmdStart(ctx, []string{"--project", "p", "--profile", "loop", "--allow-advisory-tokens", "--configure"}); err != nil {
		t.Fatal(err)
	}
	p, err := loadProfile(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Budgets.AllowAdvisoryTokens || p.Provenance.Overrides["allow_advisory_tokens"] != "true" {
		t.Fatalf("explicit advisory policy or provenance was not persisted: %+v", p)
	}
}

func TestProfilePlanExcludesRuntimesWithoutHardTokenCapability(t *testing.T) {
	w := loopEnv(t)
	if err := store.CreateRuntime(w, "a-root", store.Runtime{Name: "incapable", Binary: "agent", Mode: "stdin"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRole(w, "a-root", team.Role{Name: "incapable-role", Kind: "implementer", Runtime: "incapable", Grant: "rw"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(w, "a-root", "p", "Build capability-aware plan", store.TaskOpts{Accept: []string{"a"}, Estimate: "1,2,3"}); err != nil {
		t.Fatal(err)
	}
	p, _ := defaultProfile("p", "loop")
	plan, err := buildProfilePlan(w, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Tasks) != 1 || plan.Tasks[0].Role != "" || !strings.Contains(plan.Tasks[0].RoutingReason, "hard token policy excluded incapable-role/incapable") {
		t.Fatalf("hard-cap profile preview did not explain its capability exclusion: %+v", plan.Tasks)
	}
}

func containsAdjacent(values []string, first, second string) bool {
	for i := 0; i+1 < len(values); i++ {
		if values[i] == first && values[i+1] == second {
			return true
		}
	}
	return false
}

type failingServiceRunner struct{ calls int }

func (r *failingServiceRunner) run(string, ...string) (string, error) {
	r.calls++
	return "infrastructure unavailable\n", errors.New("unavailable")
}

type leaseLossRunner struct {
	w *workspace.Workspace
	p OperatingProfile
}

type refusalRunner struct{ calls int }

func (r *refusalRunner) run(string, ...string) (string, error) {
	r.calls++
	return "policy remedy\n", clikit.Refusedf("do not retry")
}

func (r leaseLossRunner) run(string, ...string) (string, error) {
	return "", os.Remove(servicePath(r.w, r.p, "lease"))
}

func TestServiceStopsAtCircuitBreakerCheckpoint(t *testing.T) {
	w := loopEnv(t)
	p, _ := defaultProfile("p", "service")
	p.Recovery.InfrastructureFailureLimit = 2
	p.Execution.ServiceInvocations = 5
	r := &failingServiceRunner{}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	err := runService(ctx, w, p, r)
	if err == nil || clikit.ExitCode(err) != 3 || r.calls != 2 {
		t.Fatalf("breaker result err=%v code=%d calls=%d", err, clikit.ExitCode(err), r.calls)
	}
	b, readErr := os.ReadFile(servicePath(w, p, "json"))
	if readErr != nil || !bytes.Contains(b, []byte(`"status": "halt"`)) || !bytes.Contains(b, []byte("circuit breaker")) {
		t.Fatalf("durable breaker checkpoint missing: %v %s", readErr, b)
	}
	if _, statErr := os.Stat(servicePath(w, p, "lease")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("lease not released: %v", statErr)
	}
}

func TestServiceNeverRetriesPolicyRefusal(t *testing.T) {
	w := loopEnv(t)
	p, _ := defaultProfile("p", "service")
	r := &refusalRunner{}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	err := runService(ctx, w, p, r)
	if err == nil || clikit.ExitCode(err) != 3 || r.calls != 1 {
		t.Fatalf("policy refusal err=%v code=%d calls=%d", err, clikit.ExitCode(err), r.calls)
	}
	b, readErr := os.ReadFile(servicePath(w, p, "json"))
	if readErr != nil || !bytes.Contains(b, []byte("refused by policy")) {
		t.Fatalf("durable refusal checkpoint missing: %v %s", readErr, b)
	}
}

func TestServiceStopAndLeaseLossAreDurable(t *testing.T) {
	w := loopEnv(t)
	p, _ := defaultProfile("p", "service")
	stop := filepath.Join(w.Root, p.Recovery.StopFile)
	if err := os.WriteFile(stop, []byte("stop"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &fakeRunner{}
	ctx := &clikit.Ctx{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Cwd: w.Root}
	if err := runService(ctx, w, p, r); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 0 {
		t.Fatalf("STOP launched work: %v", r.calls)
	}
	b, _ := os.ReadFile(servicePath(w, p, "json"))
	if !bytes.Contains(b, []byte("STOP requested")) {
		t.Fatalf("STOP checkpoint missing: %s", b)
	}

	if err := os.Remove(stop); err != nil {
		t.Fatal(err)
	}
	p.Execution.ServiceInvocations = 2
	err := runService(ctx, w, p, leaseLossRunner{w: w, p: p})
	if err == nil || clikit.ExitCode(err) != 3 || !strings.Contains(err.Error(), "lease lost") {
		t.Fatalf("lease loss was not a policy halt: %v", err)
	}
	b, _ = os.ReadFile(servicePath(w, p, "json"))
	if !bytes.Contains(b, []byte("lease lost")) {
		t.Fatalf("lease-loss checkpoint missing: %s", b)
	}
}
