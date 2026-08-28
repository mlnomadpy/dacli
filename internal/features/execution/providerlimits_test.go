package execution

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/commandresult"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/providerpolicy"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

func TestRunGuardianPersistsRuntimeExitCode(t *testing.T) {
	exitFile := filepath.Join(t.TempDir(), "runtime-exit.txt")
	guardian := exec.Command(os.Args[0], "__run-guardian", "--exit-file", exitFile, "sh", "-c", "exit 9")
	guardian.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := guardian.Run(); err == nil {
		t.Fatal("guardian returned success, want runtime exit 9")
	} else if exitErr := new(exec.ExitError); !errors.As(err, &exitErr) || exitErr.ExitCode() != 9 {
		t.Fatalf("guardian error = %v, want exit 9", err)
	}
	raw, err := os.ReadFile(exitFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := strconv.Atoi(strings.TrimSpace(string(raw))); got != 9 {
		t.Fatalf("persisted exit = %d, want 9", got)
	}
}

func TestRunGuardianPrintsSafeActionableStartFailure(t *testing.T) {
	outside := "/private/operator/dacli-provider-that-does-not-exist"
	guardian := exec.Command(os.Args[0], "__run-guardian", outside)
	out, err := guardian.CombinedOutput()
	if err == nil {
		t.Fatal("guardian returned success for a missing runtime")
	}
	text := string(out)
	if strings.Contains(text, outside) || !strings.Contains(text, "install the executable or correct PATH") {
		t.Fatalf("guardian start diagnostic was not actionable and redacted: %q", text)
	}
}

func TestFinalizeDetachedProviderFailureRecordsCooldown(t *testing.T) {
	w := newExecWS(t)
	task := mustTask(t, w, "detached provider outcome", store.TaskOpts{})
	runID := "01DETACHEDPROVIDER"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "transcript.log"), []byte("rate limit retry_after: 17\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "runtime-exit.txt"), []byte("9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	summary := finalizeRun(w, procmon.Record{RunID: runID, Child: "a-detached", Task: task.ID, Runtime: "provider-rt", Started: time.Now()})
	if !strings.Contains(summary, "source=provider-rt destination=none reason=rate limit retry_after: 17 cooldown=17s") {
		t.Fatalf("provider transition not printed by wait summary: %q", summary)
	}
	if _, open, err := store.LoadRuntimeLimits(w).Open("provider-rt"); err != nil || !open {
		t.Fatalf("detached breaker open = %v, err = %v", open, err)
	}
}

func TestWaitReconstructsTypedDetachedProviderFailure(t *testing.T) {
	const secret = "detached-provider-secret"
	t.Setenv("DACLI_TEST_API_TOKEN", secret)
	w := newExecWS(t)
	task := mustTask(t, w, "detached typed failure", store.TaskOpts{})
	provider := filepath.Join(w.Root, "provider-fixture")
	mustRuntime(t, w, store.Runtime{Name: "provider-rt", Binary: provider, Mode: "stdin", SandboxRO: []string{"--ro"}})
	runID := "01DETACHEDTYPEDFAILURE"
	runDir := mkRun(t, w, runID, detachedRunningPlaceholder+"\nchild: a-detached\ntask: "+task.ID+"\n")
	transcript := strings.Repeat("discarded provider output\n", 1<<16) + fmt.Sprintf("setup line\nauthentication failed: %s at /private/operator/credentials\n", secret)
	if err := os.WriteFile(filepath.Join(runDir, "transcript.log"), []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "runtime-exit.txt"), []byte("19\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := procmon.Record{RunID: runID, Child: "a-detached", Task: task.ID, Runtime: "provider-rt", PID: 1 << 30, PGID: 1 << 30, Started: time.Now().Add(-time.Minute)}
	if err := procmon.WriteRecord(filepath.Join(runDir, "proc.txt"), rec); err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := newCtx(w.Root)
	waitErr := cmdWait(ctx, []string{runID})
	diagnostic, ok := commandresult.AsDiagnostic(waitErr)
	if !ok || diagnostic.ExitCode == nil || *diagnostic.ExitCode != 19 || diagnostic.Kind != "authentication" || diagnostic.Executable != "provider-fixture" {
		t.Fatalf("detached diagnostic = %#v, present=%v, error=%v", diagnostic, ok, waitErr)
	}
	if diagnostic.CwdScope != "." || len(diagnostic.StderrTail) > 4200 || !strings.Contains(diagnostic.NextAction, "authenticate") {
		t.Fatalf("detached diagnostic scope/bounds/action = %#v", diagnostic)
	}
	if strings.Contains(waitErr.Error(), secret) || strings.Contains(waitErr.Error(), "/private/operator") || !strings.Contains(diagnostic.StderrTail, "<redacted>") || !strings.Contains(diagnostic.StderrTail, "setup line") {
		t.Fatalf("detached diagnostic leaked protected output: %#v; %v", diagnostic, waitErr)
	}
	if !strings.Contains(out.String(), "a-detached") {
		t.Fatalf("wait did not print finalized identity before returning failure: %q", out)
	}
	result, resultOK := ctx.Result.(commandresult.Wait)
	if !resultOK || len(result.Runs) != 1 || result.Runs[0].RunID != runID {
		t.Fatalf("wait lost successful finalization identity: %#v", ctx.Result)
	}
}

func failingProviderBinary(t *testing.T, message string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho '"+message+"'\nexit 9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSpawnClassifiesProviderFailureAndRecordsOnlyFallbackableCooldown(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		wantOpen      bool
	}{
		{"rate-limit", "rate limit retry_after: 17", true},
		{"permanent-input", "invalid_request malformed prompt", false},
		{"policy", "content policy refusal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newExecWS(t)
			mustTask(t, w, "provider outcome", store.TaskOpts{})
			mustRuntime(t, w, store.Runtime{Name: "provider-rt", Binary: failingProviderBinary(t, tc.message), Mode: "stdin", SandboxRO: []string{"--ro"}})
			ctx, _, _ := newCtx(w.Root)
			spawnErr := cmdSpawn(ctx, []string{"--task", "001", "--runtime", "provider-rt", "--cooperative"})
			if spawnErr == nil {
				t.Fatal("failing provider spawn returned nil")
			}
			diagnostic, ok := commandresult.AsDiagnostic(spawnErr)
			if !ok || diagnostic.ExitCode == nil || *diagnostic.ExitCode != 9 || diagnostic.Executable != "provider" {
				t.Fatalf("spawn caller collapsed provider diagnostic: %#v, present=%v, error=%v", diagnostic, ok, spawnErr)
			}
			_, open, err := store.LoadRuntimeLimits(w).Open("provider-rt")
			if err != nil {
				t.Fatal(err)
			}
			if open != tc.wantOpen {
				t.Fatalf("breaker open = %v, want %v", open, tc.wantOpen)
			}
		})
	}
}

func TestResolveLaunchUsesExplicitFallbackForOpenRuntime(t *testing.T) {
	w := newExecWS(t)
	mustTask(t, w, "provider fallback", store.TaskOpts{})
	mustRuntime(t, w, store.Runtime{Name: "primary-rt", Binary: fakeBinary(t), Mode: "stdin", SandboxRO: []string{"--ro"}})
	mustRuntime(t, w, store.Runtime{Name: "secondary-rt", Binary: fakeBinary(t), Mode: "stdin", SandboxRO: []string{"--ro"}})
	mustRole(t, w, team.Role{Name: "primary", Scope: []string{"internal/**"}, Runtime: "primary-rt", Grant: "ro", Profile: team.ModelProfile{CapabilityTags: []string{"code"}}, FallbackTo: []string{"secondary"}})
	mustRole(t, w, team.Role{Name: "secondary", Scope: []string{"internal/**"}, Runtime: "secondary-rt", Grant: "rw", Profile: team.ModelProfile{CapabilityTags: []string{"code", "vision"}}})
	limits := store.LoadRuntimeLimits(w)
	if _, err := limits.Record("primary-rt", providerpolicy.Outcome{Kind: providerpolicy.RateLimited, Reason: "429"}, time.Hour); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	f, _ := clikit.ParseFlags([]string{"--task", "001", "--role", "primary", "--cooperative"})
	plan, err := resolveLaunch(ctx, w, f, "001")
	if err != nil {
		t.Fatal(err)
	}
	if plan.RoleName != "secondary" || plan.Runtime.Name != "secondary-rt" || plan.Grant != "rw" {
		t.Fatalf("fallback plan = role %q runtime %q grant %q", plan.RoleName, plan.Runtime.Name, plan.Grant)
	}
	if got := out.String(); !strings.Contains(got, "source=primary-rt destination=secondary-rt reason=429 cooldown=") {
		t.Fatalf("fallback transition not printed: %q", got)
	}
}

func TestResolveLaunchPermanentAndPolicyCooldownsNeverFallback(t *testing.T) {
	for _, kind := range []providerpolicy.Kind{providerpolicy.PermanentInput, providerpolicy.PolicyRefusal} {
		t.Run(string(kind), func(t *testing.T) {
			w := newExecWS(t)
			mustTask(t, w, "provider pause", store.TaskOpts{})
			mustRuntime(t, w, store.Runtime{Name: "primary-rt", Binary: fakeBinary(t), Mode: "stdin", SandboxRO: []string{"--ro"}})
			mustRuntime(t, w, store.Runtime{Name: "secondary-rt", Binary: fakeBinary(t), Mode: "stdin", SandboxRO: []string{"--ro"}})
			mustRole(t, w, team.Role{Name: "primary", Scope: []string{"internal/**"}, Runtime: "primary-rt", Grant: "ro", FallbackTo: []string{"secondary"}})
			mustRole(t, w, team.Role{Name: "secondary", Scope: []string{"internal/**"}, Runtime: "secondary-rt", Grant: "rw"})
			limits := store.LoadRuntimeLimits(w)
			if _, err := limits.Record("primary-rt", providerpolicy.Outcome{Kind: kind, Reason: string(kind)}, time.Hour); err != nil {
				t.Fatal(err)
			}

			ctx, out, _ := newCtx(w.Root)
			f, _ := clikit.ParseFlags([]string{"--task", "001", "--role", "primary", "--cooperative"})
			_, err := resolveLaunch(ctx, w, f, "001")
			if clikit.ExitCode(err) != 3 {
				t.Fatalf("resolve error = %v (exit %d), want policy pause", err, clikit.ExitCode(err))
			}
			if got := out.String(); !strings.Contains(got, "source=primary-rt destination=none") {
				t.Fatalf("pause transition not printed: %q", got)
			}
		})
	}
}
