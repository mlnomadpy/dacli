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
	if p.Base != "" && strings.TrimSpace(p.Base) == "" {
		return fmt.Errorf("landing base must be a non-empty branch when configured")
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
