package orchestration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/verifyroute"
)

func TestInferVerificationRulesUsesDashboardWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dashboard := filepath.Join(root, "internal", "features", "dashboard", "ui")
	if err := os.MkdirAll(dashboard, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{"scripts":{"test":"vitest run","build":"vite build"}}`
	if err := os.WriteFile(filepath.Join(dashboard, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, _, err := inferVerificationRules(root)
	if err != nil {
		t.Fatal(err)
	}
	var dashboardRules []verifyroute.Rule
	for _, rule := range rules {
		if len(rule.Argv) > 0 && rule.Argv[0] == "npm" {
			dashboardRules = append(dashboardRules, rule)
		}
	}
	if len(dashboardRules) != 2 {
		t.Fatalf("dashboard rules = %+v", dashboardRules)
	}
	for _, rule := range dashboardRules {
		if rule.Cwd != "internal/features/dashboard/ui" || rule.Cwd == "." {
			t.Fatalf("dashboard rule lost nested cwd: %+v", rule)
		}
		if !slices.Equal(rule.Argv[:2], []string{"npm", "run"}) {
			t.Fatalf("dashboard command is not structured argv: %+v", rule)
		}
	}
}

func TestDacliRepositoryInferenceNeverPlacesDashboardAtRoot(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	rules, _, err := inferVerificationRules(root)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, rule := range rules {
		if len(rule.Argv) == 0 || rule.Argv[0] != "npm" {
			continue
		}
		found++
		if rule.Cwd != "internal/features/dashboard/ui" {
			t.Fatalf("dacli dashboard rule resolved cwd %q: %+v", rule.Cwd, rule)
		}
	}
	if found == 0 {
		t.Fatal("dacli package.json scripts produced no dashboard rules")
	}
}

func TestExplicitVerificationRulesRoundTripByteStable(t *testing.T) {
	w := loopEnv(t)
	p, err := defaultProfile("p", "loop")
	if err != nil {
		t.Fatal(err)
	}
	p.Verification.Commands = nil
	p.Verification.Rules = []verifyroute.Rule{{ID: "explicit", Include: []string{"service/**"}, Cwd: ".", Argv: []string{"tool", "argument with spaces"}, Environment: verifyroute.EnvironmentPolicy{Inherit: []string{"PATH"}, Set: map[string]string{"MODE": "strict"}}, Gate: "test", Fanout: true, Required: true}}
	p.Verification.ContractGroups = []verifyroute.ContractGroup{{ID: "wire", Include: []string{"contracts/**"}, Rules: []string{"explicit"}}}
	want, err := json.Marshal(p.Verification)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveProfile(w, p); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadProfile(w, "p")
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(loaded.Verification)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("explicit verification changed across persistence:\n got %s\nwant %s", got, want)
	}
	for _, args := range [][]string{{"--project", "p", "--show"}, {"--project", "p", "--profile", "loop", "--dry-run"}} {
		out := &bytes.Buffer{}
		ctx := &clikit.Ctx{Stdout: out, Stderr: &bytes.Buffer{}, Cwd: w.Root, JSON: true}
		if err := cmdStart(ctx, args); err != nil {
			t.Fatalf("start %v: %v", args, err)
		}
		var plan ProfilePlan
		if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
			t.Fatal(err)
		}
		got, _ := json.Marshal(struct {
			Rules          []verifyroute.Rule          `json:"rules"`
			ContractGroups []verifyroute.ContractGroup `json:"contract_groups"`
		}{plan.Policy.Verification.Rules, plan.Policy.Verification.ContractGroups})
		wantRules, _ := json.Marshal(struct {
			Rules          []verifyroute.Rule          `json:"rules"`
			ContractGroups []verifyroute.ContractGroup `json:"contract_groups"`
		}{p.Verification.Rules, p.Verification.ContractGroups})
		if string(got) != string(wantRules) {
			t.Fatalf("start %v changed explicit rules:\n got %s\nwant %s", args, got, wantRules)
		}
	}
}
