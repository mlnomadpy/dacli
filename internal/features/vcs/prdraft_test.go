package vcs

import (
	"fmt"
	"strings"
	"testing"
)

// errNoPR simulates `gh pr view` finding no existing PR, so openPR falls
// through to `pr create`.
var errNoPR = fmt.Errorf("no pull requests found")

// findCreate returns the captured `pr create` invocation, or nil.
func findCreate(calls [][]string) []string {
	for _, c := range calls {
		if len(c) >= 2 && c[0] == "pr" && c[1] == "create" {
			return c
		}
	}
	return nil
}

// `dacli pr --draft` opens the PR as a GitHub draft — the work-in-progress PR a
// real project opens before the work is ready to review. openPR must pass
// --draft through to `gh pr create`.
func TestPRDraftPassesDraftFlag(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	calls := stubGH(t, func(dir string, args ...string) (string, error) {
		// No existing PR (openPRURL's `pr view` fails), so openPR falls through
		// to create.
		if len(args) >= 2 && args[0] == "pr" && args[1] == "view" {
			return "", errNoPR
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/9", nil
		}
		return "", nil
	})

	ctx, out := prCtx(dir)
	if _, _, err := openPR(ctx, w, "a-root", tk, "main", false, reviewComment, true); err != nil {
		t.Fatalf("openPR --draft: %v\n%s", err, out.String())
	}
	create := findCreate(*calls)
	if create == nil {
		t.Fatalf("no `pr create` was invoked: %v", *calls)
	}
	if !strings.Contains(strings.Join(create, " "), "--draft") {
		t.Fatalf("pr create missing --draft: %v", create)
	}
}

// Without --draft, the PR is a normal one: --draft must NOT leak in, or every PR
// would silently open unmergeable.
func TestPRWithoutDraftFlagIsNormal(t *testing.T) {
	dir, w, tk := prIntegrateEnv(t)
	calls := stubGH(t, func(dir string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "pr" && args[1] == "view" {
			return "", errNoPR
		}
		if len(args) >= 2 && args[0] == "pr" && args[1] == "create" {
			return "https://github.com/acme/widgets/pull/9", nil
		}
		return "", nil
	})

	ctx, _ := prCtx(dir)
	if _, _, err := openPR(ctx, w, "a-root", tk, "main", false, reviewComment, false); err != nil {
		t.Fatalf("openPR: %v", err)
	}
	create := findCreate(*calls)
	if create == nil {
		t.Fatalf("no `pr create` was invoked: %v", *calls)
	}
	if strings.Contains(strings.Join(create, " "), "--draft") {
		t.Fatalf("pr create must not carry --draft without the flag: %v", create)
	}
}
