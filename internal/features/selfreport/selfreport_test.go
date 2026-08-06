package selfreport

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/buildinfo"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	if v, ok := os.LookupEnv(agentid.EnvVar); ok {
		t.Setenv(agentid.EnvVar, v)
		_ = os.Unsetenv(agentid.EnvVar)
	}
	w, err := workspace.Init(t.TempDir(), "selfreport-test")
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

func TestCmdVersion(t *testing.T) {
	ctx, out, _ := newCtx(t.TempDir())
	if err := cmdVersion(ctx, nil); err != nil {
		t.Fatal(err)
	}
	want := "dacli " + buildinfo.Version + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")"
	if !strings.Contains(out.String(), want) {
		t.Errorf("version output missing %q:\n%s", want, out)
	}
}

func TestCmdReportUsageAndRejects(t *testing.T) {
	ctx, _, _ := newCtx(t.TempDir())
	if code := clikit.ExitCode(cmdReport(ctx, nil)); code != 2 {
		t.Error("report with no title must be a usage error")
	}
	ctx2, _, _ := newCtx(t.TempDir())
	if code := clikit.ExitCode(cmdReport(ctx2, []string{"title", "--bogus", "x"})); code != 2 {
		t.Error("a typo'd flag must be a usage error, not silently dropped")
	}
}

// Fail CLOSED: the privileged levers (--repo, --disclose) refuse when no
// workspace identity resolves at all, rather than silently proceeding.
func TestCmdReportPrivilegedLeversNeedAResolvableWorkspace(t *testing.T) {
	outsideDir := t.TempDir()

	ctx, _, _ := newCtx(outsideDir)
	err := cmdReport(ctx, []string{"bug", "--repo", "someone/else"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("--repo with no workspace: exit %d, want 3 (err %v)", code, err)
	}

	ctx2, _, _ := newCtx(outsideDir)
	err2 := cmdReport(ctx2, []string{"bug", "--disclose"})
	if code := clikit.ExitCode(err2); code != 3 {
		t.Fatalf("--disclose with no workspace: exit %d, want 3 (err %v)", code, err2)
	}
}

// Even with a resolvable workspace, --repo/--disclose need an rw grant: a
// read-only agent must not redirect the target repo or attach workspace
// internals.
func TestCmdReportPrivilegedLeversNeedRW(t *testing.T) {
	w := newWS(t)
	becomeChild(t, w, "auditor", model.GrantRO)

	ctx, _, _ := newCtx(w.Root)
	err := cmdReport(ctx, []string{"bug", "--repo", "someone/else"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("ro --repo: exit %d, want 3 (err %v)", code, err)
	}

	ctx2, _, _ := newCtx(w.Root)
	err2 := cmdReport(ctx2, []string{"bug", "--disclose"})
	if code := clikit.ExitCode(err2); code != 3 {
		t.Fatalf("ro --disclose: exit %d, want 3 (err %v)", code, err2)
	}
}

// A plain --dry-run withholds workspace/run internals by default and targets
// the tool's own repo, never the user's.
func TestCmdReportDryRunWithholdsByDefault(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdReport(ctx, []string{"something", "broke", "--body", "detail", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "would file to "+buildinfo.Repo+":") {
		t.Errorf("dry-run must target the tool's own repo by default:\n%s", got)
	}
	if !strings.Contains(got, "[agent-report] something broke") {
		t.Errorf("dry-run must show the prefixed title:\n%s", got)
	}
	if !strings.Contains(got, "withheld") {
		t.Errorf("dry-run without --disclose must state workspace/run are withheld:\n%s", got)
	}
	if strings.Contains(got, w.Name) {
		t.Errorf("workspace name leaked without --disclose:\n%s", got)
	}
}

// --disclose (with rw) attaches the workspace name and, when --run names one,
// the transcript's tail — capped at 30 lines.
func TestCmdReportDryRunWithDiscloseAttachesWorkspaceAndRunTail(t *testing.T) {
	w := newWS(t)
	runID := "01RUNAAAAAAAAAAAAAAAAAAAAA"
	runDir := w.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var lines []string
	for i := 1; i <= 40; i++ {
		lines = append(lines, "line "+string(rune('a'+i%26))+string(rune(i)))
	}
	transcript := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(runDir, "transcript.log"), []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, out, _ := newCtx(w.Root)
	if err := cmdReport(ctx, []string{"broke", "--disclose", "--run", runID, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "workspace: "+w.Name) {
		t.Errorf("--disclose must attach the workspace name:\n%s", got)
	}
	if strings.Contains(got, lines[0]) {
		t.Errorf("excerpt must be TAILED to 30 lines, but the first line leaked:\n%s", got)
	}
	if !strings.Contains(got, lines[len(lines)-1]) {
		t.Errorf("excerpt must include the last line of the transcript:\n%s", got)
	}
}

// Real filing needs gh on PATH; absent it, the command fails before touching
// the network.
func TestCmdReportRequiresGhOnPath(t *testing.T) {
	w := newWS(t)
	t.Setenv("PATH", "")
	ctx, _, _ := newCtx(w.Root)
	err := cmdReport(ctx, []string{"broke"})
	if err == nil || !strings.Contains(err.Error(), "gh not on PATH") {
		t.Fatalf("want a gh-not-on-PATH error, got %v", err)
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	want := map[string]bool{"report": false, "version": false}
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
