package docs_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultAgentWalkthroughIsExecutableAndNamesTheLifecycle(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(here), ".."))
	script := filepath.Join(root, "docs", "examples", "default-agent-workflow.sh")
	info, err := os.Stat(script)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatal("default workflow must be executable")
	}
	body, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--roster agents", "task claim", "--verify", "task done", "integrate", "test -f result.txt"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("walkthrough missing lifecycle step %q", want)
		}
	}
	if out, err := exec.Command("bash", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("walkthrough shell syntax: %v: %s", err, out)
	}
}
