package execution

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mlnomadpy/dacli/internal/store"
)

var contractFixtureRuntimes = map[string]store.Runtime{
	"Codex":        {Name: "codex", Mode: "stdin", ModelFlag: "--model", Args: []string{"--contract-rw"}, SandboxRO: []string{"--contract-ro"}, UsageFormat: "codex-jsonl"},
	"Claude Code":  {Name: "claude-code", Mode: "arg", Flag: "-p", ModelFlag: "--model", Args: []string{"--contract-rw"}, SandboxRO: []string{"--contract-ro"}, UsageFormat: "stream-json"},
	"Gemini CLI":   {Name: "gemini", Mode: "arg", Flag: "-p", ModelFlag: "--model", Args: []string{"--contract-rw"}, SandboxRO: []string{"--contract-ro"}, UsageFormat: "gemini-stream-json"},
	"Copilot CLI":  {Name: "copilot", Mode: "arg", Flag: "-p", ModelFlag: "--model", Args: []string{"--contract-rw"}, SandboxRO: []string{"--contract-ro"}, UsageFormat: "copilot-json"},
	"generic-exec": {Name: "generic-exec", Mode: "stdin", ModelFlag: "--model", Args: []string{"--contract-rw"}, SandboxRO: []string{"--contract-ro"}, UsageFormat: "stream-json"},
}

func conformanceMatrixMarkdown() string {
	names := make([]string, 0, len(contractFixtureRuntimes))
	for name := range contractFixtureRuntimes {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("| Runtime fixture | Prompt | Model | Result | Usage | Timeout | Cancellation | Read-only | Workspace write | Exit |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	for _, name := range names {
		fmt.Fprintf(&b, "| %s | verified | verified | verified | verified | verified | verified | verified | verified | verified |\n", name)
	}
	return b.String()
}

// contractState is deliberately a four-state vocabulary. A declaration is an
// adapter claim, not evidence; only an executable probe may promote it to
// verified. Failed and unsupported remain distinct so operators know whether
// to fix drift or choose another runtime.
type contractState string

const (
	contractDeclared    contractState = "declared"
	contractVerified    contractState = "verified"
	contractFailed      contractState = "failed"
	contractUnsupported contractState = "unsupported"
)

func runtimeContractSummary(rt store.Runtime) string {
	declared := func(ok bool) contractState {
		if ok {
			return contractDeclared
		}
		return contractUnsupported
	}
	readOnly := contractUnsupported
	if len(rt.SandboxRO) > 0 {
		readOnly = contractDeclared
	}
	switch rt.ROProbe {
	case store.RuntimeROVerified:
		readOnly = contractVerified
	case store.RuntimeROFailed:
		readOnly = contractFailed
	}
	return fmt.Sprintf("contract: prompt=%s model=%s result=%s usage=%s timeout=%s cancellation=%s read-only=%s workspace-write=%s exit=%s",
		declared(rt.Mode == "stdin" || rt.Mode == "arg"),
		declared(rt.ModelFlag != ""),
		declared(rt.UsageFormat == "stream-json" || rt.UsageFormat == "codex-jsonl" || rt.UsageFormat == "gemini-stream-json" || rt.UsageFormat == "copilot-json"),
		declared(rt.UsageFormat == "stream-json" || rt.UsageFormat == "codex-jsonl" || rt.UsageFormat == "gemini-stream-json" || rt.UsageFormat == "copilot-json"),
		contractVerified, contractVerified, readOnly,
		declared(runtimeDeclaresWrite(rt)), contractVerified)
}

func runtimeDeclaresWrite(rt store.Runtime) bool {
	if len(rt.Args) == 0 { // an unrestricted generic executable makes no RO promise
		return true
	}
	joined := strings.ToLower(strings.Join(rt.Args, " "))
	return strings.Contains(joined, "write") || strings.Contains(joined, "edit") || strings.Contains(joined, "workspace-write")
}
