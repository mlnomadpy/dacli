package prompts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Section is a typed semantic unit of the shared delivery contract. Keeping
// the required set in code makes deletion observable instead of relying on a
// reviewer noticing that a paragraph disappeared from a large template.
type Section string

const (
	SectionIdentity     Section = "identity-deduplication"
	SectionScheduling   Section = "estimation-critical-path"
	SectionModels       Section = "model-economics"
	SectionIsolation    Section = "role-grant-isolation"
	SectionVerification Section = "verification-landing"
	SectionBudgets      Section = "budgets-recovery"
	SectionEvidence     Section = "honest-evidence"
	SectionAdapters     Section = "provider-neutral-adapters"
)

var RequiredSections = []Section{
	SectionIdentity, SectionScheduling, SectionModels, SectionIsolation,
	SectionVerification, SectionBudgets, SectionEvidence, SectionAdapters,
}

// RolePhase names semantic responsibilities, never a model vendor. A runtime
// adapter only transports the resulting bytes and configures its own CLI.
type RolePhase string

const (
	Implementer RolePhase = "implementer"
	Reviewer    RolePhase = "reviewer"
	Planner     RolePhase = "estimator/planner"
	LoopAuditor RolePhase = "loop-auditor"
	Recovery    RolePhase = "recovery"
)

var RolePhases = []RolePhase{Implementer, Reviewer, Planner, LoopAuditor, Recovery}

// Contract is the resolved, deterministic semantic prompt and its provenance.
type Contract struct {
	Schema   string
	Version  string
	Hash     string
	Text     string
	Sections []Section
}

// AutonomousContract renders the one shared semantic contract. Role-specific
// lifecycle instructions are branches inside this artifact, not copied into
// provider adapters or separate prompts where they can contradict each other.
func AutonomousContract(overrideDir string) (Contract, error) {
	text, err := Render(overrideDir, "autonomous_delivery", map[string]any{
		"Sections": RequiredSections,
		"Roles":    RolePhases,
	})
	if err != nil {
		return Contract{}, err
	}
	for _, section := range RequiredSections {
		if !strings.Contains(text, "contract:"+string(section)) {
			return Contract{}, fmt.Errorf("autonomous delivery prompt missing required section %q", section)
		}
	}
	for _, role := range RolePhases {
		if !strings.Contains(text, "role:"+string(role)) {
			return Contract{}, fmt.Errorf("autonomous delivery prompt missing role %q", role)
		}
	}
	sum := sha256.Sum256([]byte(text))
	return Contract{Schema: Schema, Version: Version, Hash: hex.EncodeToString(sum[:]), Text: text, Sections: append([]Section(nil), RequiredSections...)}, nil
}

// DeliveredHash fingerprints the exact bytes handed to the runtime. The
// contract hash answers "which semantics?"; this answers "what was sent?".
func DeliveredHash(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}
