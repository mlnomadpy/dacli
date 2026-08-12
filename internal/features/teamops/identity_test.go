package teamops

import (
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
)

func TestMain(m *testing.M) {
	ambient, hadAmbient := os.LookupEnv(agentid.EnvVar)
	_ = os.Unsetenv(agentid.EnvVar)
	code := m.Run()
	if hadAmbient {
		_ = os.Setenv(agentid.EnvVar, ambient)
	} else {
		_ = os.Unsetenv(agentid.EnvVar)
	}
	os.Exit(code)
}

func TestPackageHarnessClearsForeignAgentIdentity(t *testing.T) {
	if value, ok := os.LookupEnv(agentid.EnvVar); ok {
		t.Fatalf("package harness leaked %s=%q into temporary-workspace tests", agentid.EnvVar, value)
	}
}
