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
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
		// G1 residual: reflect the task's status folder as a single
		// `status:<folder>` label so the issue tracker shows dacli's own
		// lifecycle. Best-effort and idempotent — see applyStatusLabel.
		applyStatusLabel(w, num, t.Status)

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

	// G2: decisions ride the SAME explicit push and the SAME disclosure gate
	// (already tripped above), never auto-run on ship.
	if err := mirrorDecisions(w, p.Slug, repo, ctx.Stdout); err != nil {
		return err
	}
	return nil
}

// --- status labels (G1 residual) ---

// statusLabel is the per-status label a mirrored issue carries, tracking the
// task's status folder (status:open | status:active | status:blocked |
// status:done).
func statusLabel(s model.Status) string { return "status:" + string(s) }

// otherStatusLabels are the status labels a mirrored issue must NOT carry given
// its current status — the stale labels to strip so a task that changed folders
// doesn't accumulate a second status: label.
func otherStatusLabels(s model.Status) []string {
	var out []string
	for _, o := range model.AllStatuses {
		if o != s {
			out = append(out, statusLabel(o))
		}
	}
	return out
}

// ensureLabel creates a label if missing. Best-effort: --force turns an
// "already exists" into a harmless update instead of an error, so a repeated
// push never fails on label creation.
func ensureLabel(w *workspace.Workspace, name string) {
	_, _ = gh(w, "label", "create", name, "--force")
}

// applyStatusLabel gives issue num EXACTLY ONE status: label. gh --add-label is
// itself idempotent (re-adding an existing label is a no-op), and stripping the
// other three status labels means a re-run never stacks duplicates and a moved
// task never carries two conflicting status labels. All calls are best-effort:
// a --remove-label for a label the issue doesn't carry errors, which we ignore.
func applyStatusLabel(w *workspace.Workspace, num int, s model.Status) {
	if num == 0 {
		return
	}
	ensureLabel(w, statusLabel(s))
	_, _ = gh(w, "issue", "edit", strconv.Itoa(num), "--add-label", statusLabel(s))
	for _, stale := range otherStatusLabels(s) {
		_, _ = gh(w, "issue", "edit", strconv.Itoa(num), "--remove-label", stale)
	}
}

// --- decisions → GitHub (G2) ---

// decisionMarker is the recovery key embedded in every mirrored decision issue,
// keyed on the note id AND the workspace id — the same marker-idempotency
// machinery tasks use, but a distinct prefix so a decision issue is never
// adopted as a task mirror (and vice versa).
func decisionMarker(w *workspace.Workspace, noteID string) string {
	return fmt.Sprintf("<!-- dacli-decision:%s ws:%s -->", noteID, w.ID)
}

type decisionNote struct {
	path  string
	doc   *mdstore.Doc
	id    string
	title string
}

// decisionNotes reads the project's decision notes with their on-disk paths, so
// the mirror can write the issue number back onto the exact note file (ListNotes
// yields docs without paths, which the write-back needs).
func decisionNotes(w *workspace.Workspace, project string) ([]decisionNote, error) {
	dir := w.NotesDir(project, model.NoteDecision)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no decisions dir yet is not an error
		}
		return nil, err
	}
	var out []decisionNote
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		d, err := mdstore.ReadFile(path)
		if err != nil {
			continue
		}
		id, _ := d.Front.Get("id")
		title := ""
		for _, s := range d.Sections {
			if s.Level == 1 {
				title = s.Title
				break
			}
		}
		out = append(out, decisionNote{path: path, doc: d, id: id, title: title})
	}
	return out, nil
}

// decisionBody renders the WHY that is the whole point of mirroring a decision:
// the choice, the rejected alternative, and the because. The marker leads (for
// crash-recovery adoption) and the note id trails (the backlink to the dacli
// decision).
func decisionBody(w *workspace.Workspace, dn decisionNote) string {
	var b strings.Builder
	b.WriteString(decisionMarker(w, dn.id) + "\n\n")
	b.WriteString("**Decision:** " + dn.title + "\n\n")
	if s, ok := dn.doc.Section("Chose"); ok && strings.TrimSpace(s.Content) != "" {
		b.WriteString("**Chose:** " + strings.TrimSpace(s.Content) + "\n\n")
	}
	if s, ok := dn.doc.Section("Rejected"); ok && strings.TrimSpace(s.Content) != "" {
		b.WriteString("**Rejected:** " + strings.TrimSpace(s.Content) + "\n\n")
	}
	if s, ok := dn.doc.Section("Because"); ok && strings.TrimSpace(s.Content) != "" {
		b.WriteString("**Because:** " + strings.TrimSpace(s.Content) + "\n\n")
	}
	b.WriteString("_Mirrored from dacli decision " + dn.id + "; the workspace is the source of truth._\n")
	return b.String()
}

// mirrorDecisions projects each decision note to an issue labeled `decision`,
// reusing the marker/searchByMarker/write-back idempotency the task mirror uses:
// frontmatter mapping first, then SEARCH BY MARKER, and only then create — so a
// crash between the remote create and the local write converges by adoption,
// never a duplicate.
func mirrorDecisions(w *workspace.Workspace, project, repo string, out io.Writer) error {
	notes, err := decisionNotes(w, project)
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		return nil
	}
	// The `decision` label must exist before an issue can be created with it.
	ensureLabel(w, "decision")

	created, adopted, kept := 0, 0, 0
	for _, dn := range notes {
		if dn.id == "" {
			// A note with no id cannot be keyed idempotently; skip rather than
			// risk creating a duplicate on every push.
			continue
		}
		num := mappedIssueDoc(dn.doc)
		if num == 0 {
			if found := searchByMarker(w, decisionMarker(w, dn.id)); found > 0 {
				num = found
				adopted++
			}
		}
		if num == 0 {
			ghout, err := gh(w, "issue", "create",
				"--title", "decision: "+dn.title,
				"--body", decisionBody(w, dn),
				"--label", "decision")
			if err != nil {
				return fmt.Errorf("issue create for decision %s: %v (%s)", dn.id, err, ghout)
			}
			num = trailingInt(ghout)
			if num == 0 {
				return fmt.Errorf("could not parse issue number from gh output %q", ghout)
			}
			created++
		} else if mappedIssueDoc(dn.doc) != 0 {
			kept++
		}

		// Write the mapping back after the remote exists, so the failure window
		// leaves an adoptable issue, not a dangling mapping — mirrors the task path.
		dn.doc.Front.SetBlock("github", fmt.Sprintf("  issue: %d\n  repo: %s", num, repo))
		if err := mdstore.WriteFile(dn.path, dn.doc); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "decisions: %d created, %d adopted-by-marker, %d unchanged (of %d)\n",
		created, adopted, kept, len(notes))
	return nil
}

// marker is the recovery key embedded in every mirrored issue body: a lost
// or corrupted mapping is recoverable by SEARCH rather than by duplication.
func marker(w *workspace.Workspace, t *store.Task) string {
	return fmt.Sprintf("<!-- dacli:%s ws:%s -->", t.ID, w.ID)
}

func mappedIssue(t *store.Task) int { return mappedIssueDoc(t.Doc) }

// mappedIssueDoc reads the mirrored issue number from any doc's `github:` block
// (tasks and decision notes store the mapping the same way), so a doc already
// bound to an issue skips creation on the next push — the local half of the
// idempotency guarantee.
func mappedIssueDoc(d *mdstore.Doc) int {
	block, ok := d.Front.GetBlock("github")
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
