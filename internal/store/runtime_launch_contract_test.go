package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlnomadpy/dacli/internal/model"
)

func TestRuntimeLaunchContractBindsEveryReviewLaunchDimension(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(binary, []byte("fixture-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{
		Name: "codex-review", Harness: "codex", Binary: binary, Mode: "stdin",
		GlobalArgs: []string{"exec"}, Args: []string{"--json"}, SandboxRO: []string{"--sandbox", "read-only"},
		ModelFlag: "--model", BehavioralPreflight: BehavioralPreflightCodexExecJSONV2,
	}
	base, err := BuildRuntimeLaunchContract(runtime, binary, model.GrantRO, "gpt-5.6", false, runtime.SandboxRO, IndependentReviewChannel)
	if err != nil {
		t.Fatal(err)
	}
	if base.Schema != RuntimeLaunchContractSchema || base.Harness != "codex" || base.Adapter != BehavioralPreflightCodexExecJSONV2 || base.Grant != "ro" || base.Runtime != "codex-review" || base.Model != "gpt-5.6" || base.ResultChannel != IndependentReviewChannel || base.Fingerprint == "" {
		t.Fatalf("contract = %+v", base)
	}

	tests := map[string]func() (RuntimeLaunchContract, error){
		"harness": func() (RuntimeLaunchContract, error) {
			changed := runtime
			changed.Harness = "hybrid"
			return BuildRuntimeLaunchContract(changed, binary, model.GrantRO, "gpt-5.6", false, changed.SandboxRO, IndependentReviewChannel)
		},
		"adapter": func() (RuntimeLaunchContract, error) {
			changed := runtime
			changed.BehavioralPreflight = BehavioralPreflightCodexExecJSONV1
			return BuildRuntimeLaunchContract(changed, binary, model.GrantRO, "gpt-5.6", false, changed.SandboxRO, IndependentReviewChannel)
		},
		"sandbox": func() (RuntimeLaunchContract, error) {
			return BuildRuntimeLaunchContract(runtime, binary, model.GrantRO, "gpt-5.6", false, []string{"--sandbox", "workspace-write"}, IndependentReviewChannel)
		},
		"grant": func() (RuntimeLaunchContract, error) {
			return BuildRuntimeLaunchContract(runtime, binary, model.GrantRW, "gpt-5.6", false, nil, IndependentReviewChannel)
		},
		"runtime": func() (RuntimeLaunchContract, error) {
			changed := runtime
			changed.Name = "other"
			return BuildRuntimeLaunchContract(changed, binary, model.GrantRO, "gpt-5.6", false, changed.SandboxRO, IndependentReviewChannel)
		},
		"model": func() (RuntimeLaunchContract, error) {
			return BuildRuntimeLaunchContract(runtime, binary, model.GrantRO, "gpt-5.6-mini", false, runtime.SandboxRO, IndependentReviewChannel)
		},
		"result-channel": func() (RuntimeLaunchContract, error) {
			return BuildRuntimeLaunchContract(runtime, binary, model.GrantRO, "gpt-5.6", false, runtime.SandboxRO, CommandResultChannel)
		},
	}
	for name, build := range tests {
		t.Run(name, func(t *testing.T) {
			changed, err := build()
			if err != nil {
				t.Fatal(err)
			}
			if changed.Fingerprint == base.Fingerprint {
				t.Fatalf("%s did not change fingerprint %s", name, base.Fingerprint)
			}
		})
	}
	raw, _ := json.Marshal(base)
	parsed, err := ParseRuntimeLaunchContract("preflight ok\n" + RuntimeLaunchContractMarker + string(raw) + "\n")
	if err != nil || !parsed.Equal(base) {
		t.Fatalf("parsed=%+v err=%v", parsed, err)
	}
}
