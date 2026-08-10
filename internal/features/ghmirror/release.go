package ghmirror

import (
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// cmdRelease cuts a tagged GitHub release with generated notes on the project's
// LINKED repo (github_repo), so a generated repo carries a real release history
// (dacli 223). Like every outbound gh write in this slice it targets the linked
// repo EXPLICITLY (--repo, dacli 221) rather than whatever the cwd remote
// happens to resolve to, so a multi-project workspace releases the right repo.
//
// Unlike `github push`, a release is deliberately NOT run through
// disclosureGate. That gate exists because pushing MIRRORS workspace finding and
// decision notes — internal reasoning that is not otherwise in the repo — onto a
// public issue tracker. Generated release notes are the opposite: gh assembles
// them from the repo's OWN merged-PR and commit history, which is already at the
// repo's visibility, so cutting a release discloses nothing new. It still needs
// an rw grant, because it writes a tag and a release to the remote (Method
// axiom 4: any command that writes to a remote gets a grant check).
//
// Idempotency, in the mirror's spirit: an existing release for the tag is
// REPORTED and the command succeeds, so a re-run converges instead of surfacing
// gh's raw "release already exists" error or cutting a second release. Generated
// notes are the default (`--generate-notes`); --notes overrides with an explicit
// body (and then --generate-notes is NOT also passed — gh rejects the two
// together).
func cmdRelease(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("title", "notes", "target", "draft", "prerelease", "dry-run"); err != nil {
		return err
	}
	if len(f.Pos) < 2 {
		return clikit.Usagef("usage: dacli github release <project> <tag> [--title t] [--notes text] [--target ref] [--draft] [--prerelease] [--dry-run]")
	}
	// --dry-run previews the release without writing a tag or release to the
	// remote, so it does NOT require the rw grant a real cut does — it writes
	// nothing (task 294).
	dry := f.Bool("dry-run")
	// Cutting a release writes a tag and a release to the remote — a
	// state-mutating remote write, so it needs an rw grant (Method axiom 4).
	if !dry && id.Grant != model.GrantRW {
		return clikit.Refusedf("cutting a release writes a tag and release to the remote; that needs an rw grant (yours is %s)", id.Grant)
	}
	project, tag := f.Pos[0], f.Pos[1]
	p, err := store.LoadProject(w, project)
	if err != nil {
		return err
	}
	repo, _ := p.Doc.Front.Get("github_repo")
	if repo == "" {
		return clikit.Usagef("project %s is not linked — `dacli github link %s` first", p.Slug, p.Slug)
	}

	// Idempotency: an existing release for this tag is reported, not duplicated.
	// A re-run of `github release <tag>` on a repo that already carries it
	// converges (prints the existing release) instead of erroring or filing a
	// second one — and never prints "released" on a path that wrote nothing.
	if releaseExists(w, repo, tag) {
		fmt.Fprintf(ctx.Stdout, "release %s already exists on %s — nothing to do\n", tag, repo)
		return nil
	}

	createArgs := []string{"release", "create", tag}
	if t := f.Get("title"); t != "" {
		createArgs = append(createArgs, "--title", t)
	} else {
		createArgs = append(createArgs, "--title", tag)
	}
	// Generated notes are the default (dacli 223): gh builds them from the
	// merged PRs and commits since the previous release. --notes overrides with
	// an explicit body; the two are mutually exclusive in gh, so only one is
	// ever passed.
	if notes := f.Get("notes"); notes != "" {
		createArgs = append(createArgs, "--notes", notes)
	} else {
		createArgs = append(createArgs, "--generate-notes")
	}
	// --target pins the release to a commitish (the branch ship just pushed, so
	// the release tags the state that actually reached the remote). Absent, gh
	// tags the repo's default branch head.
	if target := f.Get("target"); target != "" {
		createArgs = append(createArgs, "--target", target)
	}
	if f.Bool("draft") {
		createArgs = append(createArgs, "--draft")
	}
	if f.Bool("prerelease") {
		createArgs = append(createArgs, "--prerelease")
	}

	// --dry-run: report the release the same createArgs would cut, then stop
	// before the write. The preview is the real code path — releaseExists ran, the
	// exact createArgs were built — with only the final gh write elided (task 294).
	if dry {
		fmt.Fprintf(ctx.Stdout, "dry-run: would cut release %s on %s", tag, repo)
		if notes := f.Get("notes"); notes != "" {
			fmt.Fprint(ctx.Stdout, " with explicit --notes")
		} else {
			fmt.Fprint(ctx.Stdout, " with generated notes")
		}
		if target := f.Get("target"); target != "" {
			fmt.Fprintf(ctx.Stdout, ", targeting %s", target)
		}
		if f.Bool("draft") {
			fmt.Fprint(ctx.Stdout, ", as a draft")
		}
		if f.Bool("prerelease") {
			fmt.Fprint(ctx.Stdout, ", as a prerelease")
		}
		fmt.Fprintln(ctx.Stdout, "; nothing was written")
		return nil
	}

	out, err := ghRepo(w, repo, createArgs...)
	if err != nil {
		return fmt.Errorf("gh release create %s: %w (%s)", tag, err, out)
	}
	// gh prints the release URL on a successful create; surface it so the
	// operator has the link, and only after the write succeeded (Method axiom 5).
	fmt.Fprintf(ctx.Stdout, "released %s on %s", tag, repo)
	if url := strings.TrimSpace(out); url != "" {
		fmt.Fprintf(ctx.Stdout, " — %s", url)
	}
	fmt.Fprintln(ctx.Stdout)
	return nil
}

// releaseExists reports whether a release for tag already exists on repo — a
// clean `gh release view` success. A view FAILURE is treated as "does not
// exist" so the create proceeds: if the failure was actually a transient
// network error rather than a genuine not-found, `gh release create` fails the
// same way and surfaces it honestly, so a transient error can never be mistaken
// for a green create nor duplicate an existing release (create refuses an
// existing tag itself).
func releaseExists(w *workspace.Workspace, repo, tag string) bool {
	_, err := ghRepo(w, repo, "release", "view", tag)
	return err == nil
}
