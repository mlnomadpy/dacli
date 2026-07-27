package execution

import "testing"

// A runtime's env passthrough must never carry a credential to a child: the
// no-ANTHROPIC_API_KEY rule (children use the operator's own Claude Code login)
// has to be a checked invariant, not a default value one edit away from being
// undone (dacli 166).
func TestDeniedEnvPassthrough(t *testing.T) {
	allowed := []string{"PATH", "HOME", "TMPDIR", "WANDB_ENTITY"}
	if bad := deniedEnvPassthrough(allowed); bad != "" {
		t.Errorf("deniedEnvPassthrough(%v) = %q; want all allowed", allowed, bad)
	}

	for _, name := range []string{"ANTHROPIC_API_KEY", "anthropic_api_key", " ANTHROPIC_API_KEY ", "ANTHROPIC_AUTH_TOKEN"} {
		if deniedEnvPassthrough([]string{"PATH", name}) == "" {
			t.Errorf("deniedEnvPassthrough allowed a denied credential env %q", name)
		}
	}
}
