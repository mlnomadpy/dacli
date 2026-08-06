package stagegate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gates"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func newWS(t *testing.T) *workspace.Workspace {
	t.Helper()
	if v, ok := os.LookupEnv(agentid.EnvVar); ok {
		t.Setenv(agentid.EnvVar, v)
		_ = os.Unsetenv(agentid.EnvVar)
	}
	w, err := workspace.Init(t.TempDir(), "stagegate-test")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func newCtx(cwd string) (*clikit.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &errb, Cwd: cwd}, &out, &errb
}

func mustProject(t *testing.T, w *workspace.Workspace, slug string) *store.Project {
	t.Helper()
	p, err := store.CreateProject(w, agentid.RootID, "Test project", slug, "Ship it.", "")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// attachTemplate puts the project at a template's first stage, the way
// `project add --template` does: template + template_stage in the frontmatter.
func attachTemplate(t *testing.T, w *workspace.Workspace, slug string, tpl gates.Template) {
	t.Helper()
	p, err := store.LoadProject(w, slug)
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.Front.Set("template", tpl.Name)
	p.Doc.Front.Set("template_stage", tpl.Stages[0].Name)
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

func becomeReadOnlyChild(t *testing.T, w *workspace.Workspace) {
	t.Helper()
	_, token, err := agentid.Spawn(w, &agentid.Identity{ID: agentid.RootID, Grant: model.GrantRW}, "junior", model.GrantRO)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agentid.EnvVar, token)
}

// firstTemplate picks a shipped template by name so the gate tests do not
// hard-code a manifest that may be renamed.
func firstTemplate(t *testing.T, w *workspace.Workspace) gates.Template {
	t.Helper()
	ts, err := gates.Load(w)
	if err != nil {
		t.Fatal(err)
	}
	for _, tpl := range ts {
		if len(tpl.Stages) > 0 && tpl.Name != "solo" {
			return tpl
		}
	}
	t.Skip("no multi-stage template shipped")
	return gates.Template{}
}

// `template list` must report every shipped template WITH its stated cost —
// the cost line is how an operator decides whether a template's ceremony is
// worth it, so a template listed without one is a template chosen blind.
func TestTemplateListReportsOriginAndCost(t *testing.T) {
	w := newWS(t)
	ctx, out, _ := newCtx(w.Root)
	if err := cmdList(ctx, nil); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if got == "" {
		t.Fatal("no templates listed; dacli ships embedded ones")
	}
	if !strings.Contains(got, "embedded") {
		t.Errorf("template origin not reported:\n%s", got)
	}
	if !strings.Contains(got, "cost:") {
		t.Errorf("template cost not reported:\n%s", got)
	}
}

func TestTemplateShowAndVendor(t *testing.T) {
	w := newWS(t)
	tpl := firstTemplate(t, w)

	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdShow(ctx, nil)); code != 2 {
		t.Error("template show with no name must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdShow(ctx2, []string{"no-such-template"})); code != 4 {
		t.Error("showing an unknown template must be a not-found")
	}
	ctx3, out, _ := newCtx(w.Root)
	if err := cmdShow(ctx3, []string{tpl.Name}); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Errorf("template show printed nothing for %q", tpl.Name)
	}

	// Vendoring copies the manifest into the workspace so local edits win over
	// the embedded default — the nearest-wins rule the rest of dacli uses.
	ctx4, out4, _ := newCtx(w.Root)
	if err := cmdVendor(ctx4, []string{tpl.Name}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(w.TemplatesDir(), tpl.Name+".md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("vendored file missing: %v", err)
	}
	if !strings.Contains(out4.String(), "edits there win over the embedded default") {
		t.Errorf("vendor report = %q", out4)
	}
	// Vendoring twice must refuse rather than overwrite local edits.
	ctx5, _, _ := newCtx(w.Root)
	if err := cmdVendor(ctx5, []string{tpl.Name}); err == nil {
		t.Error("re-vendoring must refuse rather than clobber the local copy")
	}
	ctx6, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdVendor(ctx6, []string{"no-such-template"})); code != 4 {
		t.Error("vendoring an unknown template must be a not-found")
	}
	ctx7, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdVendor(ctx7, nil)); code != 2 {
		t.Error("vendor with no name must be a usage error")
	}
}

// An untemplated (solo) project has no gates, and must SAY so — with the
// remedy — rather than printing a bare "complete" that reads as "all gates
// passed" when in truth none exist.
func TestStageOnASoloProjectExplainsThereAreNoGates(t *testing.T) {
	w := newWS(t)
	mustProject(t, w, "solo-proj")

	ctx, out, _ := newCtx(w.Root)
	if err := cmdStage(ctx, []string{"solo-proj"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "no template (solo): no gates") {
		t.Errorf("solo project reported %q", got)
	}
	if !strings.Contains(got, "--template") {
		t.Errorf("the solo message must name the remedy: %q", got)
	}
}

func TestStageUsageAndNotFound(t *testing.T) {
	w := newWS(t)
	ctx, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdStage(ctx, nil)); code != 2 {
		t.Error("stage with no project must be a usage error")
	}
	ctx2, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdStage(ctx2, []string{"no-such-project"})); code != 4 {
		t.Error("stage on an unknown project must be a not-found")
	}
	ctx3, _, _ := newCtx(w.Root)
	if code := clikit.ExitCode(cmdAdvance(ctx3, nil)); code != 2 {
		t.Error("stage advance with no project must be a usage error")
	}
}

// A stage report lists every check with a ✓/✗ and, for the failures, WHY. A
// gate that says only "closed" gets argued past; one that names the missing
// artifact gets filled in.
func TestStageReportsPerCheckStatusAndReasons(t *testing.T) {
	w := newWS(t)
	tpl := firstTemplate(t, w)
	mustProject(t, w, "gated")
	attachTemplate(t, w, "gated", tpl)

	ctx, out, _ := newCtx(w.Root)
	if err := cmdStage(ctx, []string{"gated"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "template "+tpl.Name) || !strings.Contains(got, "stage "+tpl.Stages[0].Name) {
		t.Fatalf("stage line missing the template/stage:\n%s", got)
	}
	if !strings.Contains(got, "✗") {
		t.Fatalf("a fresh project cannot already satisfy the first gate:\n%s", got)
	}
	// Every failing check must carry a reason after the em dash.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "✗") && !strings.Contains(line, " — ") {
			t.Errorf("failing check gives no reason: %q", line)
		}
	}
}

// A closed gate is an ANSWER, not a retryable error: exit 3, the full unmet
// list, and an explicit "do not retry". The stage must not move.
func TestStageAdvanceRefusesWithTheUnmetList(t *testing.T) {
	w := newWS(t)
	tpl := firstTemplate(t, w)
	mustProject(t, w, "gated")
	attachTemplate(t, w, "gated", tpl)

	ctx, _, _ := newCtx(w.Root)
	err := cmdAdvance(ctx, []string{"gated"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("a closed gate: exit %d, want 3 (err %v)", code, err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "gate closed — unmet:") {
		t.Errorf("refusal %q does not lead with the unmet list", msg)
	}
	if !strings.Contains(msg, "✗") {
		t.Errorf("refusal %q does not enumerate the failing checks", msg)
	}
	if !strings.Contains(msg, "do not retry") {
		t.Errorf("refusal %q does not say a closed gate is an answer", msg)
	}

	st, err := gates.Status(w, "gated")
	if err != nil {
		t.Fatal(err)
	}
	if st.Stage != tpl.Stages[0].Name {
		t.Errorf("a refused advance moved the stage to %q", st.Stage)
	}
}

// Advancing rewrites the project file, so it needs an rw grant — and the
// grant check must come BEFORE the gate evaluation, so a read-only agent gets
// the grant refusal rather than a misleading list of unmet checks.
func TestStageAdvanceRequiresRW(t *testing.T) {
	w := newWS(t)
	tpl := firstTemplate(t, w)
	mustProject(t, w, "gated")
	attachTemplate(t, w, "gated", tpl)
	becomeReadOnlyChild(t, w)

	ctx, _, _ := newCtx(w.Root)
	err := cmdAdvance(ctx, []string{"gated"})
	if code := clikit.ExitCode(err); code != 3 {
		t.Fatalf("a ro agent advancing: exit %d, want 3 (err %v)", code, err)
	}
	if !strings.Contains(err.Error(), "rw grant") {
		t.Errorf("refusal %q must name the missing grant, not the unmet gate", err)
	}
	if strings.Contains(err.Error(), "gate closed") {
		t.Errorf("the grant check must precede gate evaluation; got %q", err)
	}
}

// A solo project has nothing to advance THROUGH; advancing it reports the
// template complete rather than inventing a stage.
func TestStageAdvanceOnASoloProjectIsComplete(t *testing.T) {
	w := newWS(t)
	mustProject(t, w, "solo-proj")

	ctx, out, _ := newCtx(w.Root)
	if err := cmdAdvance(ctx, []string{"solo-proj"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "complete") {
		t.Errorf("advancing a solo project reported %q", out)
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	want := map[string]bool{
		"template list": false, "template show": false, "template add": false,
		"stage": false, "stage advance": false,
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
