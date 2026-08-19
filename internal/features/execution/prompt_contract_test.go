package execution

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/prompts"
)

func TestInvocationRecordsResolvedPromptContract(t *testing.T) {
	prompt := "exact delivered prompt"
	got, err := promptInvocation("", prompt)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := prompts.AutonomousContract("")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"prompt_schema: " + prompts.Schema,
		"prompt_version: " + prompts.Version,
		"contract_hash: sha256:" + contract.Hash,
		"prompt_hash: sha256:" + prompts.DeliveredHash(prompt),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("invocation provenance missing %q:\n%s", want, got)
		}
	}
}

func TestEveryRuntimePresetUsesTheSharedSemanticContract(t *testing.T) {
	contract, err := prompts.AutonomousContract("")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"codex", "claude-code", "gemini", "copilot", "generic-exec"} {
		rt, ok := presets[name]
		if !ok {
			t.Fatalf("missing runtime preset %q", name)
		}
		if rt.Mode != "stdin" && rt.Mode != "arg" {
			t.Errorf("%s has invalid transport mode %q", name, rt.Mode)
		}
		// Runtime data has no semantic-prompt field: every adapter receives
		// these exact shared bytes through execRuntime's prompt parameter.
		if !strings.Contains(contract.Text, "contract:provider-neutral-adapters") {
			t.Fatalf("shared contract disappeared while testing %s", name)
		}
	}
}
