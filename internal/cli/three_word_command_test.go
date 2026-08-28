package cli

import "testing"

func TestMatchDispatchesThreeWordCommandsThroughCLIAndMCP(t *testing.T) {
	cmd, rest := match([]string{"task", "acceptance", "migrate", "001", "--dry-run"})
	if cmd == nil || cmd.Path != "task acceptance migrate" {
		t.Fatalf("match = %#v, want task acceptance migrate", cmd)
	}
	if len(rest) != 2 || rest[0] != "001" || rest[1] != "--dry-run" {
		t.Fatalf("remaining argv = %#v, want task ref and flag", rest)
	}

	_, message, code := executor(t.TempDir())([]string{"task", "acceptance", "migrate", "--help"}, false)
	if code != 0 || message != "" {
		t.Fatalf("MCP executor returned code=%d message=%q", code, message)
	}
}
