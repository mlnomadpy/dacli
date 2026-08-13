package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/providerpolicy"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
)

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
			if err := cmdSpawn(ctx, []string{"--task", "001", "--runtime", "provider-rt", "--cooperative"}); err == nil {
				t.Fatal("failing provider spawn returned nil")
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
