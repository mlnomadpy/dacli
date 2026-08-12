package execution

import (
	"os"
	"testing"
)

// TestMain lets guardian tests re-enter the package test binary through the
// same private argv contract used by the production dacli executable.
func TestMain(m *testing.M) {
	if len(os.Args) >= 2 && os.Args[1] == "__run-guardian" {
		os.Exit(RunGuardian(os.Args[2:]))
	}
	os.Exit(m.Run())
}
