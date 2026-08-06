package skillforge

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// unsetAgentEnv clears DACLI_AGENT for the test, restoring whatever the
// process started with. t.Setenv cannot unset a variable, and since dacli 288
// a present-but-empty DACLI_AGENT is a lost token that fails closed rather
// than resolving to root — so a test wanting the root identity must remove
// the variable entirely, not blank it.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv(agentid.EnvVar); ok {
		t.Setenv(agentid.EnvVar, v)
		_ = os.Unsetenv(agentid.EnvVar)
	}
}

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	unsetAgentEnv(t)
	w, err := workspace.Init(t.TempDir(), "skillforge-test")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func newCtx(cwd string) (*clikit.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd}, &out, &errb
}

func becomeChild(t *testing.T, w *workspace.Workspace, role string, grant model.Grant) string {
	t.Helper()
	id, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, role, grant)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)
	return id
}

func TestCmdAddUsageAndRejects(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAdd(ctx, nil)); code != 2 {
		t.Error("skill add with no name/desc must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAdd(ctx2, []string{"x", "--desc", "y", "--bogus", "z"})); code != 2 {
		t.Error("a typo'd flag must be a usage error, not silently dropped")
	}
}

// A skill's body is compiled into every future agent's context, so
// authoring one is a write to standing instructions — rw only.
func TestCmdAddNeedsRW(t *testing.T) {
	w := newWS(t)
	becomeChild(t, w, "auditor", model.GrantRO)
	ctx, _, _ := newCtx(w.Root)
	err := cmdAdd(ctx, []string{"x", "--desc", "does x"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro skill add: exit %d, want 3 (err %v)", code, err)
	}
}

func TestCmdAddRejectsUnsafeName(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAdd(ctx, []string{"../escape", "--desc", "y"})); code != 2 {
		t.Error("a name that escapes the skills dir must be a usage error")
	}
}

func TestCmdAddCreatesASkill(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"go-review", "--desc", "reviews go diffs", "--body", "check errors", "--min-delivery", "context"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "skill go-review created at") {
		t.Errorf("must confirm creation: %q", out)
	}
	raw, err := os.ReadFile(filepath.Join(w.SkillsLibDir(), "go-review", "skill.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"go-review", "reviews go diffs", "check errors", "context", store.DefaultVersion} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("skill.md missing %q:\n%s", want, raw)
		}
	}

	// Re-creating the same skill must fail rather than clobber it.
	ctx2, _, _ := newCtx(w.Root)
	if err := cmdAdd(ctx2, []string{"go-review", "--desc", "again"}); err == nil {
		t.Error("re-adding an existing skill must fail")
	}
}

func TestCmdPromoteOnlyRoot(t *testing.T) {
	w := newWS(t)
	becomeChild(t, w, "auditor", model.GrantRW)
	ctx, _, _ := newCtx(w.Root)
	err := cmdPromote(ctx, []string{"some-lesson"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("non-root promote: exit %d, want 3 (err %v)", code, err)
	}
}

func TestCmdPromoteUsageAndNotFound(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdPromote(ctx, nil)); code != 2 {
		t.Error("skill promote with no ref must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdPromote(ctx2, []string{"nope"})); code != 4 {
		t.Error("promoting an unknown lesson must be a not-found")
	}
}

func TestCmdPromoteCreatesSkillFromLesson(t *testing.T) {
	w := newWS(t)
	if _, err := store.CreateProject(w, agentid.RootID, "Core", "core", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, agentid.RootID, "core", model.NoteFinding, "Reproduce before fixing", store.NoteOpts{
		Scope: "workspace", Origin: "file:internal/foo.go", Body: "always write the failing test first",
	}); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdPromote(ctx, []string{"Reproduce before fixing", "--name", "tdd-discipline"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "promoted lesson") {
		t.Errorf("must confirm the promotion: %q", out)
	}
	raw, err := os.ReadFile(filepath.Join(w.SkillsLibDir(), "tdd-discipline", "skill.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"tdd-discipline", "Reproduce before fixing", "always write the failing test first", "file:internal/foo.go"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("promoted skill.md missing %q:\n%s", want, raw)
		}
	}
}

func TestCmdListAndShow(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"go-review", "--desc", "reviews go diffs", "--body", "check errors"}); err != nil {
		t.Fatal(err)
	}

	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdList(ctx2, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "go-review") || !strings.Contains(out2.String(), "reviews go diffs") {
		t.Errorf("skill list missing the skill: %q", out2)
	}

	ctx3, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdShow(ctx3, nil)); code != 2 {
		t.Error("skill show with no name must be a usage error")
	}
	ctx4, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdShow(ctx4, []string{"nope"})); code != 4 {
		t.Error("showing an unknown skill must be a not-found")
	}

	ctx5, out5, _ := newCtx(w.Root)
	if err := cmdShow(ctx5, []string{"go-review"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"go-review", "reviews go diffs", store.DefaultVersion} {
		if !strings.Contains(out5.String(), want) {
			t.Errorf("skill show missing %q:\n%s", want, out5)
		}
	}
}

func TestCmdBump(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"go-review", "--desc", "reviews go diffs"}); err != nil {
		t.Fatal(err)
	}

	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdBump(ctx2, nil)); code != 2 {
		t.Error("skill bump with no name must be a usage error")
	}
	ctx3, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdBump(ctx3, []string{"nope"})); code != 4 {
		t.Error("bumping an unknown skill must be a not-found")
	}

	becomeChild(t, w, "auditor", model.GrantRO)
	ctx4, _, _ := newCtx(w.Root)
	err := cmdBump(ctx4, []string{"go-review"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro skill bump: exit %d, want 3 (err %v)", code, err)
	}
	unsetAgentEnv(t)

	ctx5, out5, _ := newCtx(w.Root)
	if err := cmdBump(ctx5, []string{"go-review"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out5.String(), "v1 → v2") {
		t.Errorf("bump must report the version transition: %q", out5)
	}
	if v := store.FileVersion(filepath.Join(w.SkillsLibDir(), "go-review", "skill.md")); v != "v2" {
		t.Errorf("skill.md version = %q, want v2", v)
	}
}

func TestCmdImport(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdImport(ctx, nil)); code != 2 {
		t.Error("skill import with no dir must be a usage error")
	}

	becomeChild(t, w, "auditor", model.GrantRO)
	ctx2, _, _ := newCtx(w.Root)
	err := cmdImport(ctx2, []string{t.TempDir()})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro skill import: exit %d, want 3 (err %v)", code, err)
	}
	unsetAgentEnv(t)

	// An empty source dir imports nothing.
	empty := t.TempDir()
	ctx3, _, _ := newCtx(w.Root)
	if err := cmdImport(ctx3, []string{empty}); err == nil {
		t.Error("importing a source with no skill directories must error")
	}

	// A source dir with one native SKILL.md imports it losslessly.
	src := t.TempDir()
	skillDir := filepath.Join(src, "native-thing")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: native-thing\ndescription: d\n---\n# native-thing\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx4, out4, _ := newCtx(w.Root)
	if err := cmdImport(ctx4, []string{src}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out4.String(), "imported 1 skill(s) losslessly: native-thing") {
		t.Errorf("import output: %q", out4)
	}
	if _, err := os.Stat(filepath.Join(w.SkillsLibDir(), "native-thing", "SKILL.md")); err != nil {
		t.Error("imported skill must keep its native SKILL.md filename verbatim")
	}
}

func TestCmdFetchNeedsRWAndValidatesRef(t *testing.T) {
	w := newWS(t)
	becomeChild(t, w, "auditor", model.GrantRO)
	ctx, _, _ := newCtx(w.Root)
	err := cmdFetch(ctx, []string{"owner/repo"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro skill fetch: exit %d, want 3 (err %v)", code, err)
	}
	unsetAgentEnv(t)

	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdFetch(ctx2, nil)); code != 2 {
		t.Error("skill fetch with no ref must be a usage error")
	}

	// A malformed owner/repo ref fails validation before any network call.
	ctx3, _, _ := newCtx(w.Root)
	err = cmdFetch(ctx3, []string{"not-a-valid-ref"})
	if err == nil || !strings.Contains(err.Error(), "owner/repo") {
		t.Errorf("a malformed ref must be rejected before touching the network: %v", err)
	}
}

func mustRuntime(t *testing.T, w *workspace.Workspace, name string) {
	t.Helper()
	if err := store.CreateRuntime(w, agentid.RootID, store.Runtime{
		Name: name, Binary: "true", Mode: "arg", Flag: "-p",
	}, ""); err != nil {
		t.Fatal(err)
	}
}

func TestCmdCompileUsageAndRejects(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdCompile(ctx, nil)); code != 2 {
		t.Error("skill compile with no runtime must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdCompile(ctx2, []string{"--runtime", "claude-code", "--bogus", "x"})); code != 2 {
		t.Error("a typo'd flag must be a usage error, not silently dropped")
	}
}

func TestCmdCompileUnknownRuntime(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdCompile(ctx, []string{"--runtime", "nope"})); code != 4 {
		t.Error("compiling for an unknown runtime must be a not-found")
	}
}

func TestCmdCompileUnknownRole(t *testing.T) {
	w := newWS(t)
	mustRuntime(t, w, "claude-code")
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdCompile(ctx, []string{"--runtime", "claude-code", "--role", "nope"})); code != 4 {
		t.Error("compiling for an unknown role must be a not-found")
	}
}

func TestCmdCompileRoleNamesUnknownSkill(t *testing.T) {
	w := newWS(t)
	mustRuntime(t, w, "claude-code")
	if err := store.CreateRole(w, agentid.RootID, team.Role{Name: "backend", Skills: []string{"missing-skill"}}); err != nil {
		t.Fatal(err)
	}
	ctx, _, _ := newCtx(w.Root)
	err := cmdCompile(ctx, []string{"--runtime", "claude-code", "--role", "backend"})
	if err == nil || !strings.Contains(err.Error(), "not in the library") {
		t.Errorf("a role naming a missing skill must error clearly: %v", err)
	}
}

func TestCmdCompileNothingToCompile(t *testing.T) {
	w := newWS(t)
	mustRuntime(t, w, "claude-code")
	ctx, _, _ := newCtx(w.Root)
	err := cmdCompile(ctx, []string{"--runtime", "claude-code"})
	if err == nil || !strings.Contains(err.Error(), "nothing to compile") {
		t.Errorf("compiling an empty library must error: %v", err)
	}
}

func TestCmdCompileDryRunReportsTaxWithoutWriting(t *testing.T) {
	w := newWS(t)
	mustRuntime(t, w, "claude-code")
	ctx, _, _ := newCtx(w.Root)
	// --body does not survive a disk round-trip once merged under the
	// skill's H1 on reparse, so a long --desc is what actually drives the
	// per-turn token estimate here.
	if err := cmdAdd(ctx, []string{"go-review", "--desc", strings.Repeat("x", 20000)}); err != nil {
		t.Fatal(err)
	}

	ctx2, out2, errb2 := newCtx(w.Root)
	if err := cmdCompile(ctx2, []string{"--runtime", "claude-code", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "(dry-run: nothing written)") {
		t.Errorf("dry-run must say nothing was written: %q", out2)
	}
	if !strings.Contains(out2.String(), "per-turn tax") {
		t.Errorf("dry-run must report the per-turn token tax: %q", out2)
	}
	if !strings.Contains(errb2.String(), "warning: heavy always-on payload") {
		t.Errorf("a >4000 token always-on payload must warn: %q", errb2)
	}
	entries, _ := os.ReadDir(w.BuildSkillsDir("claude-code", "_all"))
	if len(entries) != 0 {
		t.Error("dry-run must not write any compiled output")
	}
}

func TestCmdCompileWritesOutput(t *testing.T) {
	w := newWS(t)
	mustRuntime(t, w, "claude-code")
	ctx, _, _ := newCtx(w.Root)
	if err := cmdAdd(ctx, []string{"go-review", "--desc", "reviews go diffs"}); err != nil {
		t.Fatal(err)
	}
	ctx2, out2, _ := newCtx(w.Root)
	if err := cmdCompile(ctx2, []string{"--runtime", "claude-code"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "compiled to") {
		t.Errorf("must report where output landed: %q", out2)
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	want := map[string]bool{
		"skill add": false, "skill list": false, "skill show": false, "skill bump": false,
		"skill import": false, "skill fetch": false, "skill compile": false, "skill promote": false,
	}
	for _, c := range Commands {
		if _, ok := want[c.Path]; !ok {
			t.Errorf("unexpected command path %q", c.Path)
			continue
		}
		want[c.Path] = true
		if c.Run == nil || c.Brief == "" {
			t.Errorf("command %q is missing a Run or Brief", c.Path)
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("command %q is no longer registered", path)
		}
	}
}
