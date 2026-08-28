package verifyroute

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolveMonorepoPathsAndContractFanout(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"web app", "service", "shared", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	rules := []Rule{
		{ID: "go-test", Include: []string{"**/*.go", "go.mod"}, Cwd: ".", Argv: []string{"go", "test", "./..."}, Gate: "test", Fanout: true, Required: true},
		{ID: "web-test", Include: []string{"web app/**"}, Cwd: "web app", Argv: []string{"npm", "test"}, Gate: "test", Fanout: true},
		{ID: "python-test", Include: []string{"service/**"}, Cwd: "service", Argv: []string{"python", "-m", "pytest"}, Gate: "test", Fanout: true},
	}
	groups := []ContractGroup{{ID: "wire", Include: []string{"shared/**"}, Rules: []string{"go-test", "web-test", "python-test"}}}

	for _, tc := range []struct {
		name  string
		paths []string
		want  []string
	}{
		{"go", []string{"cmd/tool/main.go"}, []string{"go-test"}},
		{"web path with spaces", []string{"web app/src/main.ts"}, []string{"web-test"}},
		{"python", []string{"service/app.py"}, []string{"python-test"}},
		{"shared contract", []string{"shared/schema.json"}, []string{"go-test", "web-test", "python-test"}},
		{"docs only", []string{"docs/guide.md"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Resolve(root, rules, groups, tc.paths)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, command := range plan {
				got = append(got, command.RuleID)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("selected rules = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveFailsClosedForConflictAndUnmatchedRequiredPath(t *testing.T) {
	root := t.TempDir()
	duplicate := []Rule{
		{ID: "test", Include: []string{"a/**"}, Cwd: ".", Argv: []string{"go", "test"}, Gate: "test"},
		{ID: "test", Include: []string{"b/**"}, Cwd: ".", Argv: []string{"npm", "test"}, Gate: "test"},
	}
	if _, err := Resolve(root, duplicate, nil, []string{"a/a.go"}); err == nil || !strings.Contains(err.Error(), "conflicts") || !strings.Contains(err.Error(), "test") {
		t.Fatalf("duplicate rule diagnostic = %v", err)
	}

	rules := []Rule{{ID: "web", Include: []string{"web/**"}, Cwd: ".", Argv: []string{"npm", "test"}, Gate: "test", Required: true}}
	if _, err := Resolve(root, rules, nil, []string{"service/app.py"}); err == nil || !strings.Contains(err.Error(), "service/app.py") || !strings.Contains(err.Error(), "include matcher") {
		t.Fatalf("unmatched required diagnostic = %v", err)
	}
}

func TestExecuteUsesStructuredArgvCwdAndEnvironmentAllowlist(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "directory with spaces")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "assert argv.sh")
	body := "#!/bin/sh\npwd > executed-from\n[ \"$1\" = 'argument with spaces' ] && [ \"$VERIFY_ALLOWED\" = yes ] && [ -z \"$VERIFY_DENIED\" ]\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VERIFY_ALLOWED", "yes")
	t.Setenv("VERIFY_DENIED", "secret")
	commands := []Command{{RuleID: "structured", Cwd: "directory with spaces", Argv: []string{script, "argument with spaces"}, Environment: EnvironmentPolicy{Inherit: []string{"VERIFY_ALLOWED"}}, Gate: "test"}}
	results, err := Execute(context.Background(), root, commands)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Cwd != "directory with spaces" || !slices.Equal(results[0].Argv, commands[0].Argv) {
		t.Fatalf("execution result = %+v", results)
	}
	if _, err := os.Stat(filepath.Join(cwd, "executed-from")); err != nil {
		t.Fatalf("command did not execute from configured cwd: %v", err)
	}
}
