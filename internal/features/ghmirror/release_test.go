package ghmirror

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// linkRepo binds a project to a repo (writing github_repo) so the release/push
// paths resolve the linked repo they must target with --repo (dacli 221).
func linkRepo(t *testing.T, w *workspace.Workspace, slug, repo string) {
	t.Helper()
	p, err := store.LoadProject(w, slug)
	if err != nil {
		t.Fatalf("load project %s: %v", slug, err)
	}
	p.Doc.Front.Set("github_repo", repo)
	if err := mdstore.WriteFile(p.Path, p.Doc); err != nil {
		t.Fatalf("write project: %v", err)
	}
}

// findCall returns the recorded gh call whose leading verbs match, or nil.
func findCall(calls [][]string, verbs ...string) []string {
	for _, c := range calls {
		if len(c) < len(verbs) {
			continue
		}
		match := true
		for i, v := range verbs {
			if c[i] != v {
				match = false
				break
			}
		}
		if match {
			return c
		}
	}
	return nil
}

// releaseCtx builds a Ctx rooted at the workspace, with DACLI_AGENT cleared so
// the acting identity is root (rw) regardless of who runs the suite.
func releaseCtx(t *testing.T, w *workspace.Workspace) (*clikit.Ctx, *bytes.Buffer) {
	t.Helper()
	t.Setenv("DACLI_AGENT", "")
	var out bytes.Buffer
	return &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: w.Root}, &out
}

// A release with no --notes defaults to gh --generate-notes, carries the tag as
// its title, and targets the LINKED repo explicitly (--repo) — the whole point
// of task 223: a generated repo carries a real release history.
func TestReleaseCutsWithGeneratedNotes(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		// The idempotency probe (`release view`) must MISS so create runs.
		if len(args) >= 2 && args[0] == "release" && args[1] == "view" {
			return "release not found", fmt.Errorf("not found")
		}
		return "https://github.com/owner/repo/releases/tag/v1.0.0", nil
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdRelease(ctx, []string{"core", "v1.0.0"}); err != nil {
		t.Fatalf("release: %v\n%s", err, out.String())
	}

	create := findCall(calls, "release", "create")
	if create == nil {
		t.Fatalf("no gh release create call; calls=%v", calls)
	}
	joined := strings.Join(create, " ")
	for _, want := range []string{"release create v1.0.0", "--title v1.0.0", "--generate-notes", "--repo owner/repo"} {
		if !strings.Contains(joined, want) {
			t.Errorf("create call %q missing %q", joined, want)
		}
	}
	if !strings.Contains(out.String(), "released v1.0.0") {
		t.Errorf("expected a released notice with the URL, got %q", out.String())
	}
}

// --notes overrides the generated notes with an explicit body, and the two are
// mutually exclusive in gh, so --generate-notes must NOT also be passed.
func TestReleaseExplicitNotesSkipsGenerate(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		if len(args) >= 2 && args[0] == "release" && args[1] == "view" {
			return "", fmt.Errorf("not found")
		}
		return "https://github.com/owner/repo/releases/tag/v2", nil
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdRelease(ctx, []string{"core", "v2", "--notes", "hand written notes"}); err != nil {
		t.Fatalf("release: %v\n%s", err, out.String())
	}
	create := findCall(calls, "release", "create")
	if create == nil {
		t.Fatalf("no gh release create call; calls=%v", calls)
	}
	joined := strings.Join(create, " ")
	if !strings.Contains(joined, "--notes hand written notes") {
		t.Errorf("create call %q missing the explicit --notes body", joined)
	}
	if strings.Contains(joined, "--generate-notes") {
		t.Errorf("create call %q passed both --notes and --generate-notes (gh rejects that)", joined)
	}
}

// Idempotency: an existing release is REPORTED and the command succeeds — it
// does NOT run `release create` again (no duplicate, no raw gh error), so a
// re-run of ship's release step converges.
func TestReleaseIdempotentWhenExists(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")

	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		// `release view` SUCCEEDS → the release already exists.
		return "v1.0.0\nLatest", nil
	}

	ctx, out := releaseCtx(t, w)
	if err := cmdRelease(ctx, []string{"core", "v1.0.0"}); err != nil {
		t.Fatalf("release: %v\n%s", err, out.String())
	}
	if c := findCall(calls, "release", "create"); c != nil {
		t.Fatalf("release create ran despite an existing release: %v", c)
	}
	if !strings.Contains(out.String(), "already exists") {
		t.Errorf("expected an 'already exists' notice, got %q", out.String())
	}
}

// An unlinked project has no repo to release against — a usage error (exit 2),
// pointing at `github link`, rather than a confusing gh failure.
func TestReleaseUnlinkedProjectRefused(t *testing.T) {
	w := mirrorWorkspace(t)
	ctx, out := releaseCtx(t, w)
	err := cmdRelease(ctx, []string{"core", "v1.0.0"})
	if err == nil {
		t.Fatalf("expected an error for an unlinked project\n%s", out.String())
	}
	if code := clikit.ExitCode(err); code != 2 {
		t.Errorf("unlinked-project error exit = %d, want 2 (usage)", code)
	}
}

// A missing tag argument is a usage error before any gh call.
func TestReleaseRequiresTag(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	ctx, _ := releaseCtx(t, w)
	if err := cmdRelease(ctx, []string{"core"}); err == nil {
		t.Fatalf("expected a usage error when the tag is omitted")
	} else if code := clikit.ExitCode(err); code != 2 {
		t.Errorf("missing-tag exit = %d, want 2 (usage)", code)
	}
}
