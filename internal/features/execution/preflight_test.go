package execution

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// mustRoleWithPrompt writes a role file with a real markdown body — unlike
// store.CreateRole, which only ever writes the (short) Summary as the body
// and so can never produce a Prompt distinct from it (roleBody treats a body
// that equals the summary as empty). Mirrors the on-disk shape of a real role
// file like .dacli/roles/fixer.md: an H1 section titled after the role.
func mustRoleWithPrompt(t *testing.T, w *workspace.Workspace, r team.Role, prompt string) {
	t.Helper()
	d := &mdstore.Doc{}
	d.Front.Set("id", "role-"+r.Name)
	d.Front.Set("kind", "role")
	d.Front.Set("name", r.Name)
	if r.Runtime != "" {
		d.Front.Set("runtime", r.Runtime)
	}
	if r.Grant != "" {
		d.Front.Set("grant", r.Grant)
	}
	d.Sections = []mdstore.Section{{Level: 1, Title: r.Name, Content: prompt + "\n"}}
	if err := mdstore.WriteFile(w.RolePath(r.Name), d); err != nil {
		t.Fatal(err)
	}
}

// TestPreflightIssuesReportsEveryMismatchInOnePass is dacli 272's core claim:
// a role whose grant, binary-allowlist path, AND prompt-named tools all
// disagree with its runtime must surface all three mismatches, not just the
// first one a naive early-return would have hit.
func TestPreflightIssuesReportsEveryMismatchInOnePass(t *testing.T) {
	rt := store.Runtime{Name: "cc", Args: []string{"--allowedTools", "Read,Grep,Bash(/opt/dacli:*)"}}
	role := team.Role{Name: "junior", Prompt: "Use `WebFetch` to check the changelog before summarizing."}

	issues := preflightIssues(rt, role, true, model.GrantRW, false, "/repo/dacli")

	byClass := map[string]preflightIssue{}
	for _, iss := range issues {
		byClass[iss.class] = iss
	}
	if len(issues) != 3 {
		t.Fatalf("expected exactly 3 issues (one per class), got %d: %#v", len(issues), issues)
	}
	if iss, ok := byClass["grant-write"]; !ok {
		t.Errorf("missing grant-write issue: %#v", issues)
	} else if !iss.refuse {
		t.Errorf("grant-write without --cooperative must refuse (dacli 250)")
	}
	if iss, ok := byClass["binary-allowlist"]; !ok {
		t.Errorf("missing binary-allowlist issue: %#v", issues)
	} else if iss.refuse {
		t.Errorf("binary-allowlist must warn, not refuse (dacli 267)")
	}
	if iss, ok := byClass["prompt-tools"]; !ok {
		t.Errorf("missing prompt-tools issue: %#v", issues)
	} else if iss.refuse {
		t.Errorf("prompt-tools must warn, not refuse")
	} else if !strings.Contains(iss.message, "WebFetch") {
		t.Errorf("prompt-tools message must name the tool, got %q", iss.message)
	}
}

// TestPreflightIssuesCooperativeDowngradesGrantWriteToWarn matches sandboxFor's
// existing convention: --cooperative accepts the grant-write mismatch out
// loud rather than refusing it.
func TestPreflightIssuesCooperativeDowngradesGrantWriteToWarn(t *testing.T) {
	rt := store.Runtime{Name: "cc", SandboxRO: []string{"--allowedTools", "Read"}}
	issues := preflightIssues(rt, team.Role{}, false, model.GrantRW, true, "")
	if len(issues) != 1 || issues[0].class != "grant-write" || issues[0].refuse {
		t.Fatalf("expected one non-refusing grant-write issue under --cooperative, got %#v", issues)
	}
}

// TestPreflightIssuesNoMismatches proves a clean role/runtime/grant combo
// reports nothing — a preflight that always finds something to say is noise.
func TestPreflightIssuesNoMismatches(t *testing.T) {
	rt := store.Runtime{Name: "generic-exec", Args: []string{"-p"}}
	role := team.Role{Name: "junior", Prompt: "Write the failing test first, then read the code."}
	if issues := preflightIssues(rt, role, true, model.GrantRW, false, ""); len(issues) != 0 {
		t.Errorf("expected no issues, got %#v", issues)
	}
}

// TestCmdPreflightReportsAllAndRefuses drives the standalone command
// (dacli 272's "a single command") end to end: it must print every class's
// mismatch and exit 3 because one of them (grant-write) refuses.
func TestCmdPreflightReportsAllAndRefuses(t *testing.T) {
	w := newExecWS(t)
	// The allowlist lives in invoke_args (not just the ro sandbox) so it is
	// the EFFECTIVE args an rw child actually runs with — grant-write and
	// prompt-tools both read against it, exactly as exeAllowlistWarning does.
	mustRuntime(t, w, store.Runtime{Name: "cc", Binary: "sh", Args: []string{"--allowedTools", "Read,Grep"}})
	mustRoleWithPrompt(t, w, team.Role{Name: "junior", Runtime: "cc", Grant: "rw"},
		"Use `WebFetch` to check the changelog.")

	ctx, out, _ := newCtx(w.Root)
	err := cmdPreflight(ctx, []string{"--role", "junior"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("exit %d, want 3 (err %v)", code, err)
	}
	got := out.String()
	if !strings.Contains(got, "grant-write") {
		t.Errorf("report missing grant-write issue:\n%s", got)
	}
	if !strings.Contains(got, "prompt-tools") {
		t.Errorf("report missing prompt-tools issue:\n%s", got)
	}
}

// TestCmdPreflightNoMismatches proves the standalone command exits 0 and
// says so plainly when a role/runtime/grant combo is clean.
func TestCmdPreflightNoMismatches(t *testing.T) {
	w := newExecWS(t)
	mustRuntime(t, w, store.Runtime{Name: "rt", Binary: "sh", Args: []string{"-p"}})

	ctx, out, _ := newCtx(w.Root)
	if err := cmdPreflight(ctx, []string{"--runtime", "rt"}); err != nil {
		t.Fatalf("expected no issues, got %v", err)
	}
	if !strings.Contains(out.String(), "no mismatches") {
		t.Errorf("expected a clean report, got:\n%s", out.String())
	}
}

// TestCmdPreflightUnknownRoleIsNotFound matches `role show`'s existing idiom
// for a name that resolves to nothing.
func TestCmdPreflightUnknownRoleIsNotFound(t *testing.T) {
	w := newExecWS(t)
	ctx, _, _ := newCtx(w.Root)
	err := cmdPreflight(ctx, []string{"--role", "nope"})
	if code := clikit.ExitCode(err); code != 4 {
		t.Fatalf("exit %d, want 4 (err %v)", code, err)
	}
}

// TestSpawnReportsEveryPreflightMismatchBeforeRefusing is the regression this
// task exists to fix: before dacli 272, a grant-write refusal inside
// sandboxFor short-circuited resolveLaunch before the binary-allowlist or
// prompt-tools checks ever ran, so a role with several real mismatches only
// ever heard about the first one. Now every applicable warning must surface
// even though the launch still refuses for the (unchanged) grant-write
// reason — the existing refusal message is untouched, but nothing it would
// have hidden is dropped anymore.
func TestSpawnReportsEveryPreflightMismatchBeforeRefusing(t *testing.T) {
	w := newExecWS(t)
	mustTask(t, w, "some task", store.TaskOpts{})
	// invoke_args (not just the ro sandbox) carries the allowlist, so it is
	// what an rw child actually runs with.
	mustRuntime(t, w, store.Runtime{Name: "rt", Binary: fakeBinary(t), Mode: "arg", Flag: "-p",
		Args: []string{"--allowedTools", "Read,Grep,Glob,LS"}})
	mustRoleWithPrompt(t, w, team.Role{Name: "junior", Runtime: "rt"},
		"Use `WebFetch` to check the changelog before summarizing.")

	ctx, _, errb := newCtx(w.Root)
	err := cmdSpawn(ctx, []string{"--task", "001", "--role", "junior", "--grant", "rw"})
	if code := clikit.ExitCode(err); code != 3 || !strings.Contains(err.Error(), "no write tool") {
		t.Fatalf("expected the existing grant-write refusal, got exit %d err %v", code, err)
	}
	if !strings.Contains(errb.String(), "WebFetch") {
		t.Errorf("expected the prompt-tools warning to surface even though grant-write refused; stderr: %s", errb.String())
	}
}
