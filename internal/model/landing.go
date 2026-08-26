package model

import (
	"fmt"
	"strings"
)

// LandingMode selects how completed project work reaches its base branch.
type LandingMode string

const (
	LandingLocal LandingMode = "local"
	LandingPR    LandingMode = "pr"
)

// LandingPolicy is the typed project landing configuration. An empty Base
// preserves the legacy behaviour: the consuming command resolves its normal
// repository default rather than dacli inventing a branch at load time.
type LandingPolicy struct {
	Mode LandingMode `json:"mode"`
	Base string      `json:"base,omitempty"`
}

// LandingOverride is command-line intent. Nil fields distinguish an omitted
// flag from an explicitly supplied value, which is required to report whether
// config/default precedence was overridden.
type LandingOverride struct {
	Mode *LandingMode
	Base *string
}

func ValidateLandingPolicy(p LandingPolicy) error {
	if p.Mode != LandingLocal && p.Mode != LandingPR {
		return fmt.Errorf("unknown landing mode %q — expected local or pr", p.Mode)
	}
	if p.Base != "" {
		if err := validateLandingBase(p.Base); err != nil {
			return err
		}
	}
	return nil
}

// validateLandingBase accepts the branch-name subset safe to pass through to
// git. A project policy is re-used by ship and integrate, so unsafe ref syntax
// must be rejected at configuration time rather than when work is landing.
func validateLandingBase(base string) error {
	if strings.TrimSpace(base) == "" {
		return fmt.Errorf("landing base must be a non-empty branch when configured")
	}
	if strings.HasPrefix(base, "-") || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "/") || strings.HasSuffix(base, "/") || strings.HasSuffix(base, ".") || strings.Contains(base, "..") || strings.Contains(base, "//") || strings.Contains(base, "@{") || base == "HEAD" {
		return fmt.Errorf("landing base %q is not a safe branch name", base)
	}
	for _, r := range base {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return fmt.Errorf("landing base %q is not a safe branch name", base)
		}
	}
	for _, part := range strings.Split(base, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return fmt.Errorf("landing base %q is not a safe branch name", base)
		}
	}
	return nil
}

// ResolveLanding applies CLI > project config > legacy defaults and reports
// whether either CLI field supplied an explicit override.
func ResolveLanding(config LandingPolicy, override LandingOverride) (LandingPolicy, bool, error) {
	if config.Mode == "" {
		config.Mode = LandingLocal
	}
	if err := ValidateLandingPolicy(config); err != nil {
		return LandingPolicy{}, false, err
	}
	effective := config
	explicit := false
	if override.Mode != nil {
		effective.Mode = *override.Mode
		explicit = true
	}
	if override.Base != nil {
		if strings.TrimSpace(*override.Base) == "" {
			return LandingPolicy{}, true, fmt.Errorf("landing base must be a non-empty branch when configured")
		}
		effective.Base = *override.Base
		explicit = true
	}
	if err := ValidateLandingPolicy(effective); err != nil {
		return LandingPolicy{}, explicit, err
	}
	return effective, explicit, nil
}
