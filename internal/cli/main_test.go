package cli

import (
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
)

// TestMain strips DACLI_AGENT from the environment before any test runs.
// The commands resolve identity from the process env, so when the suite is
// executed *by a spawned dacli agent* (the tool developing itself), that
// child's token would leak in and make root-only operations like `project
// add` fail with "agent token not recognized". Clearing it once here makes
// the cli tests hermetic regardless of who launches them.
func TestMain(m *testing.M) {
	_ = os.Unsetenv(agentid.EnvVar)
	os.Exit(m.Run())
}
