package ghmirror

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/publication"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func authorizePublic(t *testing.T, w *workspace.Workspace, internal bool) {
	t.Helper()
	p, err := store.LoadProject(w, "core")
	if err != nil {
		t.Fatal(err)
	}
	p.Doc.Front.Set("github_repo", "owner/repo")
	p.Doc.Front.Set("github_public_confirmed", "owner/repo")
	if internal {
		p.Doc.Front.Set("github_internal_disclosure", "owner/repo")
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
}

func TestProjectionCommandTextAndJSONExposeSameTypedPolicy(t *testing.T) {
	w := mirrorWorkspace(t)
	authorizePublic(t, w, false)
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return `{"nameWithOwner":"owner/repo","visibility":"PUBLIC"}`, nil
	}
	var textOut, jsonOut bytes.Buffer
	if err := cmdProjection(&clikit.Ctx{Cwd: w.Root, Stdout: &textOut}, []string{"core"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdProjection(&clikit.Ctx{Cwd: w.Root, Stdout: &jsonOut, JSON: true}, []string{"core"}); err != nil {
		t.Fatal(err)
	}
	var p publication.Policy
	if err := json.Unmarshal(jsonOut.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Schema != publication.Schema || p.Mode != "public-safe" || !strings.Contains(textOut.String(), p.Mode) || !strings.Contains(textOut.String(), "withheld findings") {
		t.Fatalf("projection surfaces drifted: text=%q json=%+v", textOut.String(), p)
	}
}

func TestPublicPushWithholdsInternalNotesByDefault(t *testing.T) {
	w := mirrorWorkspace(t)
	authorizePublic(t, w, false)
	tk, err := store.CreateTask(w, "a-root", "core", "safe task", store.TaskOpts{Accept: []string{"a-secret verifies /private/operator/token.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteFinding, "private path", store.NoteOpts{About: tk.ID, Severity: "major", Body: "/private/operator/token.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteDecision, "private decision", store.NoteOpts{Rejected: "secret", Because: "internal", Body: "agent a-secret"}); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PUBLIC"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return `[]`, nil
		default:
			return "", nil
		}
	}
	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, forbidden := range []string{"private path", "private decision", "/private/operator", "a-secret"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("public preview leaked %q:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"publication policy: public-safe", "withheld findings:", "withheld decisions:", taskIssueTitle(tk), "### Acceptance"} {
		if !strings.Contains(got, want) {
			t.Fatalf("public preview omitted %q:\n%s", want, got)
		}
	}
	assertNoWrites(t, calls)
}

func TestPublicInternalProjectionNeedsSeparateRecordedAuthority(t *testing.T) {
	w := mirrorWorkspace(t)
	authorizePublic(t, w, false)
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
			return `{"nameWithOwner":"owner/repo","visibility":"PUBLIC"}`, nil
		}
		return `[]`, nil
	}
	ctx, _ := releaseCtx(t, w)
	err := cmdPush(ctx, []string{"core", "--include-internal", "--dry-run"})
	if err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("unrecorded disclosure = %v", err)
	}

	authorizePublic(t, w, true)
	ctx, out := releaseCtx(t, w)
	if err := cmdPush(ctx, []string{"core", "--include-internal", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "publication policy: public-disclosed") {
		t.Fatalf("authority not reflected:\n%s", out.String())
	}
}
