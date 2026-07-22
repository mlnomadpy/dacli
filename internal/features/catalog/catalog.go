// Package catalog projects the .dacli role/skill roster into ONE browsable
// markdown catalog for humans (docs/ROSTER.md, or the repo wiki). The source of
// truth stays in .dacli/ — the catalog is a generated, one-way read view: you
// change a role or skill by editing its file (via PR), never by editing the
// catalog. This mirrors the ghmirror doctrine: local markdown is canonical,
// the projection is deletable and regenerable.
//
// The wiki publish is disclosure-gated exactly like `github push`: a PUBLIC
// repo makes the projection world-readable, so it needs recorded per-project
// consent. It is best-effort — a wiki failure never fails the docs write, and
// nothing here requires a live network call to be unit-tested.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/skills"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "catalog", Brief: "Generate a browsable role/skill catalog to docs/ROSTER.md (--publish-wiki mirrors it to the repo wiki, disclosure-gated)", Run: cmdCatalog},
}

// defaultOut is the versioned, reliable catalog: committed with the repo so the
// read view travels with the source of truth it is generated from.
const defaultOut = "docs/ROSTER.md"

// resolveOut turns the --out flag into an absolute destination. An empty flag
// falls back to defaultOut, and any relative path resolves against the caller's
// cwd — NOT the workspace root — so an agent running in an isolated worktree
// writes the catalog into its own tree. An absolute --out is honored verbatim.
func resolveOut(cwd, out string) string {
	if out == "" {
		out = defaultOut
	}
	if filepath.IsAbs(out) {
		return out
	}
	return filepath.Join(cwd, out)
}

// wikiPage is the wiki file the roster publishes to. GitHub serves
// `Roster.md` at the wiki path `/Roster`.
const wikiPage = "Roster.md"

// roleRow / skillRow are the flat, render-ready projections of a role/skill.
// Splitting the pure render (renderCatalog) from the workspace reads keeps the
// markdown deterministically testable without a git repo or a live gh.
type roleRow struct {
	Name, Version, Grant, Kind, Model, Purpose, LastChanged string
	Skills                                                  []string
}

type skillRow struct {
	Name, Version, Purpose, LastChanged string
	EstTokens                           int
}

func cmdCatalog(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)

	roles := collectRoles(w)
	skls := collectSkills(w)
	md := renderCatalog(roles, skls)

	// A relative --out resolves against the CALLER's working directory, not the
	// workspace root: a worktree agent's catalog must land in its own tree, not
	// the shared main checkout that workspace.Find redirects to.
	out := resolveOut(ctx.Cwd, f.Get("out"))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
		return err
	}
	rel := out
	if r, err := filepath.Rel(ctx.Cwd, out); err == nil && !strings.HasPrefix(r, "..") {
		rel = r
	}
	fmt.Fprintf(ctx.Stdout, "wrote %s — %d roles, %d skills (generated from .dacli/; edit roles/skills via PR, never here)\n", rel, len(roles), len(skls))

	// --publish-wiki is a SEPARATE, best-effort outbound projection. The docs
	// write above has already succeeded and is never rolled back: a wiki
	// failure degrades to a warning, per the acceptance ("a wiki failure must
	// not fail the docs write").
	if f.Bool("publish-wiki") {
		if err := publishWiki(ctx, w, f, md); err != nil {
			// A refusal (disclosure gate) is an answer, not a crash: surface it
			// and still exit 0, because the reliable catalog is already written.
			fmt.Fprintf(ctx.Stderr, "wiki publish skipped: %v\n", err)
		}
	}
	return nil
}

// collectRoles reads every role and annotates it with its version and the
// most-recent commit that touched its file (the "last changed" column).
func collectRoles(w *workspace.Workspace) []roleRow {
	roles, _ := store.LoadRoles(w)
	rows := make([]roleRow, 0, len(roles))
	for _, r := range roles {
		path := w.RolePath(r.Name)
		rows = append(rows, roleRow{
			Name:        r.Name,
			Version:     store.FileVersion(path),
			Grant:       r.Grant,
			Kind:        r.Kind,
			Model:       r.Model,
			Purpose:     r.Summary,
			LastChanged: lastChanged(path),
			Skills:      r.Skills,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}

func collectSkills(w *workspace.Workspace) []skillRow {
	list, _ := skills.LoadSkills(w)
	rows := make([]skillRow, 0, len(list))
	for _, s := range list {
		manifest := skillManifest(s.Dir)
		rows = append(rows, skillRow{
			Name:        s.Name,
			Version:     store.FileVersion(manifest),
			Purpose:     firstLine(s.Desc),
			EstTokens:   s.EstTokens,
			LastChanged: lastChanged(manifest),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}

// skillManifest returns the skill's manifest path (skill.md or the native
// SKILL.md) so version and changelog read the same file the loader parsed. It
// falls back to skill.md so an absent directory still yields a stable path
// (FileVersion/lastChanged degrade to their defaults on a missing file).
func skillManifest(dir string) string {
	for _, name := range []string{"skill.md", "SKILL.md"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(dir, "skill.md")
}

// lastChanged renders the newest commit that touched path as "when · subject",
// or a dash when there is no committed history (untracked, no git). It reuses
// the H1 changelog helper so the catalog and `role show` agree on history.
func lastChanged(path string) string {
	changes, _ := store.FileChangelog(path, 1)
	if len(changes) == 0 {
		return "—"
	}
	c := changes[0]
	return c.When + " · " + c.Subject
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// renderCatalog is the pure, deterministic markdown projection — no workspace,
// no git, no clock — so it is exhaustively unit-testable. Every cell is escaped
// so a pipe or newline in a purpose can never break the table.
func renderCatalog(roles []roleRow, skls []skillRow) string {
	var b strings.Builder
	b.WriteString("# Team Roster\n\n")
	b.WriteString("_Generated from `.dacli/` by `dacli catalog` — do **not** edit this page. " +
		"It is a one-way read view: to change a role or skill, edit its file under `.dacli/` (via PR), " +
		"then regenerate. Versions and last-changed come from git history._\n\n")

	fmt.Fprintf(&b, "## Roles (%d)\n\n", len(roles))
	if len(roles) == 0 {
		b.WriteString("_No roles defined._\n\n")
	} else {
		b.WriteString("| Role | Version | Grant | Kind | Model | Skills | Purpose | Last changed |\n")
		b.WriteString("|------|---------|-------|------|-------|--------|---------|--------------|\n")
		for _, r := range roles {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
				cell(r.Name), cell(r.Version), dash(r.Grant), dash(r.Kind), dash(r.Model),
				cell(strings.Join(r.Skills, ", ")), dash(r.Purpose), dash(r.LastChanged))
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "## Skills (%d)\n\n", len(skls))
	if len(skls) == 0 {
		b.WriteString("_No skills in the library._\n")
	} else {
		b.WriteString("| Skill | Version | Est. tokens | Purpose | Last changed |\n")
		b.WriteString("|-------|---------|-------------|---------|--------------|\n")
		for _, s := range skls {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				cell(s.Name), cell(s.Version), strconv.Itoa(s.EstTokens), dash(s.Purpose), dash(s.LastChanged))
		}
	}
	return b.String()
}

// cell makes any string safe inside a markdown table cell: pipes would end the
// cell early and newlines would end the row, so both are neutralised.
func cell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(s)
}

// dash renders an empty value as an em dash so a table cell is never blank.
func dash(s string) string {
	if c := cell(s); c != "" {
		return c
	}
	return "—"
}

// --- wiki publish (best-effort, disclosure-gated) ---

// publishWiki mirrors the rendered catalog to the repo's wiki, which is itself
// a git repo at <owner/repo>.wiki.git. It honors the SAME disclosure gate as
// `github push`: a PUBLIC repo needs recorded per-project consent, because the
// wiki of a public repo is world-readable. Everything is best-effort — a clone,
// write, or push failure returns an error the caller turns into a warning.
func publishWiki(ctx *clikit.Ctx, w *workspace.Workspace, f *clikit.Flags, md string) error {
	slug := f.Get("project")
	if slug == "" {
		return clikit.Usagef("--publish-wiki needs --project <slug> to know which repo's wiki to write (its linked repo)")
	}
	p, err := store.LoadProject(w, slug)
	if err != nil {
		return err
	}
	repo, _ := p.Doc.Front.Get("github_repo")
	if repo == "" {
		return fmt.Errorf("project %s is not linked to a repo — `dacli github link %s` first", p.Slug, p.Slug)
	}
	// Live-visibility disclosure gate, identical in intent to ghmirror's: a
	// repo flipped public after linking must re-trip it before we publish.
	if err := disclosureGate(w, repo, p); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "dacli-wiki-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	url := "https://github.com/" + repo + ".wiki.git"
	if out, err := git(w, tmp, "clone", "--depth", "1", url, "."); err != nil {
		// A wiki that has never been initialized has no clonable repo; say so
		// plainly rather than leaking git's raw message.
		return fmt.Errorf("clone %s failed (create the first wiki page in the browser once to initialize it): %s", url, out)
	}
	if err := os.WriteFile(filepath.Join(tmp, wikiPage), []byte(md), 0o644); err != nil {
		return err
	}
	if out, err := git(w, tmp, "add", wikiPage); err != nil {
		return fmt.Errorf("git add: %s", out)
	}
	// Nothing to commit means the wiki already matches — a success, not a
	// failure, so report it and stop before an empty-commit error.
	if out, _ := git(w, tmp, "status", "--porcelain"); strings.TrimSpace(out) == "" {
		fmt.Fprintf(ctx.Stdout, "wiki already up to date at %s/wiki/Roster\n", repoWebBase(repo))
		return nil
	}
	if out, err := git(w, tmp, "commit", "-m", "dacli catalog: update Roster"); err != nil {
		return fmt.Errorf("git commit: %s", out)
	}
	if out, err := git(w, tmp, "push"); err != nil {
		return fmt.Errorf("git push: %s", out)
	}
	fmt.Fprintf(ctx.Stdout, "published roster to %s/wiki/Roster\n", repoWebBase(repo))
	return nil
}

func repoWebBase(repo string) string { return "https://github.com/" + repo }

// git runs a git subcommand in dir with a deadline so a hung network/auth
// prompt cannot block the caller (or the mcp stdio loop). It returns combined
// output for diagnostics.
func git(w *workspace.Workspace, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	full := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), fmt.Errorf("git %s timed out", strings.Join(args, " "))
	}
	return strings.TrimSpace(string(out)), err
}

type repoInfo struct {
	NameWithOwner string `json:"nameWithOwner"`
	Visibility    string `json:"visibility"`
}

// repoView probes the repo's LIVE visibility via gh, so the disclosure gate
// decides on current reality, not a value cached at link time.
func repoView(w *workspace.Workspace) (repoInfo, error) {
	var info repoInfo
	c, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, "gh", "repo", "view", "--json", "nameWithOwner,visibility")
	cmd.Dir = w.Root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return info, fmt.Errorf("gh repo view failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return info, json.Unmarshal(out, &info)
}

// disclosureGate refuses to publish onto a PUBLIC repo's wiki without recorded
// per-project consent — the same gate `github push` applies, reimplemented here
// because a feature slice may not import another slice (arch_test).
func disclosureGate(w *workspace.Workspace, repo string, p *store.Project) error {
	info, err := repoView(w)
	if err != nil {
		return err
	}
	if strings.EqualFold(info.Visibility, "PUBLIC") {
		confirmed, _ := p.Doc.Front.Get("github_public_confirmed")
		if !consentCoversRepo(confirmed, info.NameWithOwner) {
			return clikit.Refusedf("repo %s is PUBLIC and project %s has no recorded consent for it — publishing the roster to its wiki is a disclosure event; `dacli github link %s --allow-public` first", info.NameWithOwner, p.Slug, p.Slug)
		}
	}
	return nil
}

// consentCoversRepo mirrors ghmirror's scoped-consent rule: consent is stored
// as the exact nameWithOwner, so it never leaks to a different repo the project
// is later relinked to, and a legacy bare-boolean "true" fails closed.
func consentCoversRepo(confirmed, repo string) bool {
	return confirmed != "" && strings.EqualFold(confirmed, repo)
}
