// Package ghmirror is the GitHub-projection slice (docs/GITHUB.md): local
// markdown is the source of truth, GitHub is a projection that can be
// deleted and regenerated. Sync is explicit and never on the hot path.
//
// The two properties that matter, both from the spec: idempotency by marker
// (a retried sync after a timeout must converge with ZERO duplicate issues —
// the characteristic failure of naive syncers), and the disclosure gate (a
// public repository makes every mirrored artifact public; pushing there
// requires a RECORDED per-project confirmation, not a flag someone once
// passed in a script).
//
// The zero-duplicate guarantee is load-bearing, so recovery does NOT lean on
// GitHub's search index (eventually consistent — a fast retry after a
// create-then-crash would find nothing and duplicate). searchByMarker reads
// issue bodies via the strongly-consistent list endpoint and matches the
// marker by exact substring, so a just-created issue is adopted on the very
// next run. See searchByMarker for the full rationale.
package ghmirror

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "github doctor", Brief: "Probe gh, auth, the repo, and its visibility", Run: cmdDoctor},
	{Path: "github link", Brief: "Bind a project to the repo (--allow-public records the disclosure consent)", Run: cmdLink},
	{Path: "github push", Brief: "Outbound mirror: tasks to issues, marker-idempotent", Run: cmdPush},
	{Path: "github sync", Brief: "Bidirectional sync", Run: clikit.Planned("inbound humans-as-events; outbound push works today", "docs/GITHUB.md § 3")},
	{Path: "github pull", Brief: "Inbound only: fetch remote changes as events", Run: clikit.Planned("inbound humans-as-events", "docs/GITHUB.md § 3")},
}

// gh runs the GitHub CLI in the workspace root. Credentials are gh's own —
// dacli never handles a token. The exact subcommands used here are
// assumptions until doctor probes them, per the standing doctrine.
func gh(w *workspace.Workspace, args ...string) (string, error) {
	// gh is network- and auth-bound; a deadline keeps a hung request (no
	// network, an interactive auth prompt) from blocking the caller — and,
	// under `dacli mcp serve`, the entire stdio loop.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = w.Root
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), fmt.Errorf("gh %s timed out", strings.Join(args, " "))
	}
	return strings.TrimSpace(string(out)), err
}

type repoInfo struct {
	NameWithOwner string `json:"nameWithOwner"`
	Visibility    string `json:"visibility"`
}

func repoView(w *workspace.Workspace) (repoInfo, error) {
	var info repoInfo
	out, err := gh(w, "repo", "view", "--json", "nameWithOwner,visibility")
	if err != nil {
		return info, fmt.Errorf("gh repo view failed: %v (%s)", err, out)
	}
	return info, json.Unmarshal([]byte(out), &info)
}

func cmdDoctor(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh not on PATH — the mirror needs the GitHub CLI")
	}
	if out, err := gh(w, "auth", "status"); err != nil {
		return fmt.Errorf("gh is not authenticated: %s", out)
	}
	info, err := repoView(w)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "gh ✓ authenticated · repo %s · visibility %s\n", info.NameWithOwner, info.Visibility)
	if strings.EqualFold(info.Visibility, "PUBLIC") {
		fmt.Fprintln(ctx.Stdout, "note: PUBLIC repo — pushing mirrors findings and reasoning to the world; `github link --allow-public` records that consent per project")
	}
	return nil
}

func cmdLink(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	_ = id
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli github link <project> [--allow-public]")
	}
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	info, err := repoView(w)
	if err != nil {
		return err
	}

	public := strings.EqualFold(info.Visibility, "PUBLIC")
	if public && !f.Bool("allow-public") {
		return clikit.Refusedf("repo %s is PUBLIC: mirroring is a disclosure event — findings and internal reasoning become world-readable. Re-run with --allow-public to record that consent on the project", info.NameWithOwner)
	}

	p.Doc.Front.Set("github_repo", info.NameWithOwner)
	if public {
		// The recorded confirmation: in the project file, committed, blamed —
		// not a flag that evaporates with the shell history.
		p.Doc.Front.Set("github_public_confirmed", "true")
	}
	if err := mdstore.WriteFile(p.Path, p.Doc); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "linked %s → %s (%s)\n", p.Slug, info.NameWithOwner, strings.ToLower(info.Visibility))
	if public {
		fmt.Fprintln(ctx.Stdout, "public-push consent recorded on the project")
	}
	return nil
}

func cmdPush(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli github push <project>")
	}
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	repo, _ := p.Doc.Front.Get("github_repo")
	if repo == "" {
		return clikit.Usagef("project %s is not linked — `dacli github link %s` first", p.Slug, p.Slug)
	}

	// Visibility is re-checked LIVE at every push: a repo flipped public
	// after linking must re-trip the disclosure gate.
	info, err := repoView(w)
	if err != nil {
		return err
	}
	if strings.EqualFold(info.Visibility, "PUBLIC") {
		if ok, _ := p.Doc.Front.Get("github_public_confirmed"); ok != "true" {
			return clikit.Refusedf("repo is PUBLIC and project %s has no recorded consent — `dacli github link %s --allow-public`", p.Slug, p.Slug)
		}
	}

	tasks, err := store.ListTasks(w, p.Slug, "")
	if err != nil {
		return err
	}
	created, adopted, closed, kept := 0, 0, 0, 0
	for _, t := range tasks {
		num := mappedIssue(t)

		// The idempotent create path, per GITHUB.md § 4: frontmatter first,
		// then SEARCH BY MARKER, and only then create. A crash between the
		// remote create and the local mapping write must converge on re-run
		// by adoption, never by a duplicate.
		if num == 0 {
			if found := searchByMarker(w, marker(w, t)); found > 0 {
				num = found
				adopted++
			}
		}
		if num == 0 {
			body := issueBody(w, t)
			out, err := gh(w, "issue", "create", "--title", fmt.Sprintf("%03d: %s", t.Seq, t.Title), "--body", body)
			if err != nil {
				return fmt.Errorf("issue create for %03d-%s: %v (%s)", t.Seq, t.Slug, err, out)
			}
			num = trailingInt(out)
			if num == 0 {
				return fmt.Errorf("could not parse issue number from gh output %q", out)
			}
			created++
		} else if mappedIssue(t) != 0 {
			kept++
		}

		// Write the mapping back — after the remote exists, so the failure
		// window leaves an adoptable issue, not a dangling mapping.
		t.Doc.Front.SetBlock("github", fmt.Sprintf("  issue: %d\n  repo: %s", num, repo))
		if err := store.SaveTask(t); err != nil {
			return err
		}
		if t.Status == model.StatusDone {
			// Best-effort status mirror; closing a closed issue is not an
			// error worth failing a push over.
			if _, err := gh(w, "issue", "close", strconv.Itoa(num)); err == nil {
				closed++
			}
		}
	}
	fmt.Fprintf(ctx.Stdout, "push: %d created, %d adopted-by-marker, %d unchanged, %d closed (of %d tasks)\n",
		created, adopted, kept, closed, len(tasks))
	return nil
}

// marker is the recovery key embedded in every mirrored issue body: a lost
// or corrupted mapping is recoverable by SEARCH rather than by duplication.
func marker(w *workspace.Workspace, t *store.Task) string {
	return fmt.Sprintf("<!-- dacli:%s ws:%s -->", t.ID, w.ID)
}

func mappedIssue(t *store.Task) int {
	block, ok := t.Doc.Front.GetBlock("github")
	if !ok {
		return 0
	}
	for _, line := range strings.Split(block, "\n") {
		if k, v, found := strings.Cut(strings.TrimSpace(line), ":"); found && strings.TrimSpace(k) == "issue" {
			n, _ := strconv.Atoi(strings.TrimSpace(v))
			return n
		}
	}
	return 0
}

// searchByMarker is the crash-recovery path: a create that succeeded before its
// local mapping write must be ADOPTED on re-run, never duplicated. It fetches
// issue bodies via the plain list endpoint and matches the marker by exact
// SUBSTRING — deliberately NOT `gh issue list --search`.
//
// `--search` hits GitHub's code/issue search index, which is (a) EVENTUALLY
// CONSISTENT — a just-created issue is not indexed for seconds-to-minutes, so a
// fast retry after a create-then-crash finds nothing and duplicates — and (b)
// TOKENIZED, stripping the angle brackets and colons in the marker so a match
// is not even guaranteed once indexed. The list endpoint reflects a
// just-created issue immediately and we compare bytes, so recovery converges on
// the first retry regardless of index lag. This is what makes the docstring's
// zero-duplicate guarantee hold.
func searchByMarker(w *workspace.Workspace, mk string) int {
	out, err := gh(w, "issue", "list", "--state", "all", "--limit", "1000", "--json", "number,body")
	if err != nil {
		return 0
	}
	var hits []struct {
		Number int    `json:"number"`
		Body   string `json:"body"`
	}
	if json.Unmarshal([]byte(out), &hits) != nil {
		return 0
	}
	for _, h := range hits {
		if strings.Contains(h.Body, mk) {
			return h.Number
		}
	}
	return 0
}

func issueBody(w *workspace.Workspace, t *store.Task) string {
	var b strings.Builder
	b.WriteString(marker(w, t) + "\n\n")
	if s, ok := t.Doc.Section("So that"); ok && strings.TrimSpace(s.Content) != "" {
		b.WriteString("So that " + strings.TrimSpace(s.Content) + "\n\n")
	}
	if s, ok := t.Doc.Section("Acceptance"); ok {
		b.WriteString("### Acceptance\n" + s.Content + "\n")
	}
	b.WriteString("\n_Mirrored by dacli; the workspace is the source of truth._\n")
	return b.String()
}

func trailingInt(s string) int {
	parts := strings.Split(strings.TrimSpace(s), "/")
	n, _ := strconv.Atoi(parts[len(parts)-1])
	return n
}
