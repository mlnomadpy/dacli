package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
)

const (
	RuntimeLaunchContractSchema = "runtime-launch-contract/v1"
	CommandResultChannel        = "command-result/v1"
	IndependentReviewChannel    = "independent-review-result/v1:stdout-marker"
	RuntimeLaunchContractMarker = "launch-contract: "
)

// RuntimeLaunchContract is the safe, versioned identity of one provider
// launch. Fingerprint also covers the adapter's private argv/env declaration
// and installed binary bytes without exposing either in logs.
type RuntimeLaunchContract struct {
	Schema          string   `json:"schema"`
	Fingerprint     string   `json:"fingerprint"`
	Harness         string   `json:"harness"`
	Adapter         string   `json:"adapter"`
	SandboxFlags    []string `json:"sandbox_flags"`
	Grant           string   `json:"grant"`
	Runtime         string   `json:"runtime"`
	Model           string   `json:"model"`
	ResultChannel   string   `json:"result_channel"`
	AllowUserConfig bool     `json:"allow_user_config"`
}

type runtimeLaunchContractDigest struct {
	Contract       RuntimeLaunchContract `json:"contract"`
	BinaryPath     string                `json:"binary_path"`
	BinaryDigest   string                `json:"binary_digest"`
	Mode           string                `json:"mode"`
	Flag           string                `json:"flag"`
	GlobalArgs     []string              `json:"global_args"`
	Args           []string              `json:"args"`
	Environment    []string              `json:"environment_names"`
	ModelFlag      string                `json:"model_flag"`
	TokenLimitFlag string                `json:"token_limit_flag"`
}

// BuildRuntimeLaunchContract binds preflight evidence to the exact launch
// shape later handed to the provider. sandboxFlags are the flags actually
// selected by sandboxFor, not merely the adapter's declaration.
func BuildRuntimeLaunchContract(rt Runtime, binaryPath string, grant model.Grant, selectedModel string, allowUserConfig bool, sandboxFlags []string, resultChannel string) (RuntimeLaunchContract, error) {
	if strings.TrimSpace(resultChannel) == "" {
		resultChannel = CommandResultChannel
	}
	contract := RuntimeLaunchContract{
		Schema: RuntimeLaunchContractSchema, Harness: rt.Harness,
		Adapter: rt.BehavioralPreflight, SandboxFlags: append([]string(nil), sandboxFlags...),
		Grant: string(grant), Runtime: rt.Name, Model: selectedModel,
		ResultChannel: resultChannel, AllowUserConfig: allowUserConfig,
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return RuntimeLaunchContract{}, err
	}
	binarySum := sha256.Sum256(binary)
	digest := runtimeLaunchContractDigest{
		Contract: contract, BinaryPath: binaryPath, BinaryDigest: hex.EncodeToString(binarySum[:]),
		Mode: rt.Mode, Flag: rt.Flag, GlobalArgs: append([]string(nil), rt.GlobalArgs...),
		Args: append([]string(nil), rt.Args...), Environment: append([]string(nil), rt.Env...),
		ModelFlag: rt.ModelFlag, TokenLimitFlag: rt.TokenLimitFlag,
	}
	raw, err := json.Marshal(digest)
	if err != nil {
		return RuntimeLaunchContract{}, err
	}
	sum := sha256.Sum256(raw)
	contract.Fingerprint = "sha256:" + hex.EncodeToString(sum[:])
	return contract, nil
}

func (c RuntimeLaunchContract) Validate() error {
	if c.Schema != RuntimeLaunchContractSchema || !strings.HasPrefix(c.Fingerprint, "sha256:") || c.Runtime == "" || c.Grant == "" || c.ResultChannel == "" {
		return fmt.Errorf("invalid %s contract", RuntimeLaunchContractSchema)
	}
	return nil
}

func (c RuntimeLaunchContract) Equal(other RuntimeLaunchContract) bool {
	return c.Schema == other.Schema && c.Fingerprint == other.Fingerprint && c.Harness == other.Harness && c.Adapter == other.Adapter &&
		slices.Equal(c.SandboxFlags, other.SandboxFlags) && c.Grant == other.Grant && c.Runtime == other.Runtime && c.Model == other.Model &&
		c.ResultChannel == other.ResultChannel && c.AllowUserConfig == other.AllowUserConfig
}

func ParseRuntimeLaunchContract(output string) (RuntimeLaunchContract, error) {
	var contract RuntimeLaunchContract
	index := strings.LastIndex(output, RuntimeLaunchContractMarker)
	if index < 0 {
		return contract, fmt.Errorf("missing %s", strings.TrimSpace(RuntimeLaunchContractMarker))
	}
	raw := output[index+len(RuntimeLaunchContractMarker):]
	if end := strings.IndexByte(raw, '\n'); end >= 0 {
		raw = raw[:end]
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &contract); err != nil {
		return contract, fmt.Errorf("decode %s: %w", RuntimeLaunchContractSchema, err)
	}
	return contract, contract.Validate()
}
