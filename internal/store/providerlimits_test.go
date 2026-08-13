package store

import (
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/internal/providerpolicy"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func providerLimitsWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "provider-limits")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestRuntimeLimitsPersistAcrossStoreInstances(t *testing.T) {
	w := providerLimitsWorkspace(t)
	first := LoadRuntimeLimits(w)
	if _, err := first.Record("codex", providerpolicy.Outcome{Kind: providerpolicy.QuotaExhausted, Reason: "monthly quota"}, time.Hour); err != nil {
		t.Fatal(err)
	}
	got, open, err := LoadRuntimeLimits(w).Open("codex")
	if err != nil || !open || got.Runtime != "codex" || got.Kind != providerpolicy.QuotaExhausted {
		t.Fatalf("cooldown = %+v, open=%v err=%v", got, open, err)
	}
}

func TestSelectFallbackUsesDeclaredOrderAndSecurityFloor(t *testing.T) {
	w := providerLimitsWorkspace(t)
	limits := LoadRuntimeLimits(w)
	source := team.Role{Name: "primary", Runtime: "one", Grant: "rw", Profile: team.ModelProfile{CapabilityTags: []string{"code", "vision"}}, FallbackTo: []string{"weak", "safe", "unlisted"}}
	roles := []team.Role{
		{Name: "unlisted", Runtime: "fast", Grant: "rw", Profile: team.ModelProfile{CapabilityTags: []string{"code", "vision"}}},
		{Name: "weak", Runtime: "cheap", Grant: "ro", Profile: team.ModelProfile{CapabilityTags: []string{"code"}}},
		{Name: "safe", Runtime: "backup", Grant: "rw", Profile: team.ModelProfile{CapabilityTags: []string{"code", "vision", "large-context"}}},
	}
	got, _, ok, err := SelectFallback(source, roles, limits)
	if err != nil || !ok || got.Name != "safe" {
		t.Fatalf("selected %+v, ok=%v err=%v", got, ok, err)
	}
}

func TestPermanentAndPolicyOutcomesNeverSelectFallback(t *testing.T) {
	for _, kind := range []providerpolicy.Kind{providerpolicy.PermanentInput, providerpolicy.PolicyRefusal} {
		if (providerpolicy.Outcome{Kind: kind}).Fallbackable() {
			t.Errorf("%s triggered fallback", kind)
		}
	}
}
