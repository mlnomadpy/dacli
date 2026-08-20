package orchestration

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
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
