package store

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestRuntimeLimitTransitionPrintsAndRecordsSameDetail(t *testing.T) {
	w := providerLimitsWorkspace(t)
	limits := LoadRuntimeLimits(w)
	transition := providerpolicy.Transition{Source: "codex", Destination: "claude", Reason: "quota_exhausted", Cooldown: time.Hour}
	var printed bytes.Buffer
	if err := limits.Report(&printed, transition); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(filepath.Join(w.RunsDir(), "runtime-cooldowns", "transitions.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(printed.String()) != strings.TrimSpace(string(recorded)) {
		t.Fatalf("printed %q recorded %q", printed.String(), recorded)
	}
	for _, want := range []string{"source=codex", "destination=claude", "reason=quota_exhausted", "cooldown=1h0m0s"} {
		if !strings.Contains(printed.String(), want) {
			t.Errorf("transition missing %q", want)
		}
	}
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

func TestRoleFallbackChainSurvivesPersistence(t *testing.T) {
	w := providerLimitsWorkspace(t)
	want := team.Role{Name: "primary", Scope: []string{"internal/**"}, FallbackTo: []string{"secondary", "last-resort"}}
	if err := CreateRole(w, "a-root", want); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadRole(w, want.Name)
	if !ok {
		t.Fatal("persisted role was not found")
	}
	if !reflect.DeepEqual(got.FallbackTo, want.FallbackTo) {
		t.Fatalf("fallback_to = %v, want %v", got.FallbackTo, want.FallbackTo)
	}
}
