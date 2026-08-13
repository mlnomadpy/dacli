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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var Commands = []clikit.Command{
	{Path: "github doctor", Brief: "Probe gh, auth, the repo, and its visibility", Usage: "dacli github doctor", Run: cmdDoctor},
	{Path: "github link", Brief: "Bind a project to the repo (--allow-public records the disclosure consent)", Mutates: true, Usage: "dacli github link <project> [--allow-public]", Run: cmdLink},
	{Path: "github push", Brief: "Outbound mirror: tasks to issues (+finding comments; --findings-as-issues files each finding as its own issue), marker-idempotent; decision issues are filed CLOSED (a decision is a record, not open work); window with explicit task refs and/or --since; --dry-run previews what it would create/adopt/close", Mutates: true, Usage: "dacli github push <project> [task-ref...] [--since <dur>] [--findings-as-issues] [--dry-run]", Run: cmdPush},
	{Path: "github sync", Brief: "Bidirectional sync: pull then push (--dry-run previews both halves)", Mutates: true, Usage: "dacli github sync", Run: cmdSync},
	{Path: "github pull", Brief: "Inbound: adopt human-authored issues as local tasks (--dry-run previews the adoptions)", Mutates: true, Usage: "dacli github pull <project>", Run: cmdPull},
	{Path: "github project", Brief: "Sync mirrored issues into a Project v2 board with mapped Status/Severity/Area fields (idempotent; --dry-run previews the board/items)", Mutates: true, Usage: "dacli github project <project> [--dry-run]", Run: cmdProject},
	{Path: "github release", Brief: "Cut a tagged GitHub release with generated notes on the linked repo (--notes overrides; idempotent — an existing release is reported, never duplicated; --dry-run previews the release)", Mutates: true, Usage: "dacli github release <project> <tag> [--title t] [--notes text] [--target ref] [--draft] [--prerelease] [--dry-run]", Run: cmdRelease},
	{Path: "github codeowners", Brief: "Emit .github/CODEOWNERS from role scopes (owner from the linked repo or --owner; --dry-run previews the file)", Mutates: true, Usage: "dacli github codeowners <project> [--owner org] [--stdout] [--dry-run]", Run: cmdCodeowners},
}

// gh runs the GitHub CLI in the workspace root. Credentials are gh's own —
// dacli never handles a token. The exact subcommands used here are
// assumptions until doctor probes them, per the standing doctrine.
// A package variable so a test can force gh to FAIL. The bugs this package has
// had are all in the failure path — an unchecked error becoming a silent wrong
// outcome — and that path is unreachable while gh keeps succeeding (dacli 208).
var gh = ghExec

func ghExec(w *workspace.Workspace, args ...string) (string, error) {
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

// ghRepo runs a gh subcommand explicitly against the project's LINKED repo
// (github_repo), not whatever the workspace-root git remote happens to resolve
// to. cmd.Dir = w.Root means a bare gh call targets the checkout's own remote —
// but a dacli workspace can manage several projects, each `github link`ed to a
// DIFFERENT repo, while the root has exactly ONE remote. Without --repo, every
// project's issues would be created/edited/closed on that one cwd repo, so all
// but one project's mirror lands in the WRONG repository (dacli 221). Passing
// --repo makes the write target the repo the project is actually linked to.
//
// --repo is a per-command (cobra-inherited) flag, invalid at the root `gh`
// level, so it is APPENDED after the subcommand verb (the same placement
// selfreport and catalog use). It routes through the stubbable `gh` var so the
// failure-path tests still intercept it. An empty repo falls back to cwd
// resolution, so the pre-link discovery paths (doctor, link) keep working.
func ghRepo(w *workspace.Workspace, repo string, args ...string) (string, error) {
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	return gh(w, args...)
}

type repoInfo struct {
	NameWithOwner string `json:"nameWithOwner"`
	Visibility    string `json:"visibility"`
}

// repoView probes a repo's live visibility. A non-empty repo queries THAT repo
// explicitly so the disclosure gate judges the repo the push actually writes
// to, not the cwd remote (dacli 221, mirroring catalog's dacli 167 fix); an
// empty repo falls back to the cwd repo, which is how doctor/link discover the
// repo to report or store in the first place.
//
// `gh repo view` is the ONE call site here that cannot take --repo: the
// repository is its positional argument, and passing the flag makes gh exit 1
// with `unknown flag: --repo`. dacli 221 routed every gh call through ghRepo
// uniformly and broke `github push` at its first call — the disclosure probe —
// so the whole outbound mirror failed before it wrote anything (dacli 297).
// Verified against the installed gh: issue list/create/edit/close/comment/view,
// label create and release view all accept the inherited --repo; repo view does
// not. Uniformity was the bug; this one takes it positionally.
func repoView(w *workspace.Workspace, repo string) (repoInfo, error) {
	var info repoInfo
	args := []string{"repo", "view"}
	if repo != "" {
		args = append(args, repo)
	}
	args = append(args, "--json", "nameWithOwner,visibility")
	out, err := gh(w, args...)
	if err != nil {
		return info, fmt.Errorf("gh repo view failed: %w (%s)", err, out)
	}
	return info, json.Unmarshal([]byte(out), &info)
}

func cmdDoctor(ctx *clikit.Ctx, args []string) error {
	// Validate flags BEFORE probing gh: a typo should be answered
	// immediately, not after a network round-trip that fails for an
	// unrelated reason and buries it.
	if f, ferr := clikit.ParseFlags(args); ferr != nil {
		return ferr
	} else if err := f.Reject(); err != nil {
		return err
	}
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
	// doctor is a health check of the CURRENT checkout's repo (it takes no
	// project), so it deliberately resolves via cwd — hence the empty repo.
	info, err := repoView(w, "")
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
	if err := f.Reject("allow-public"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli github link <project> [--allow-public]")
	}
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	// link DISCOVERS the repo to bind from the current checkout's remote, so it
	// resolves via cwd (empty repo) — this is the one place github_repo is set.
	info, err := repoView(w, "")
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
		// not a flag that evaporates with the shell history. Scoped to THIS
		// repo (its nameWithOwner), so consent for one public repo never
		// silently authorizes a push to a different repo the project is later
		// relinked to — the disclosure gate compares this to the live repo.
		p.Doc.Front.Set("github_public_confirmed", info.NameWithOwner)
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
	if err := f.Reject("findings-as-issues", "with-tasks", "since", "dry-run"); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli github push <project> [task-ref...] [--since <dur>] [--findings-as-issues] [--dry-run]")
	}
	// --dry-run: run the real read + decision path below but ELIDE every write —
	// each remote mutation and local mapping write is swapped for a "would ..."
	// line, so the preview is the same code path a real push runs, never a
	// parallel re-description of it (task 294).
	dry := f.Bool("dry-run")
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	repo, _ := p.Doc.Front.Get("github_repo")
	if repo == "" {
		return unlinkedRefusal(w.Root, p.Slug)
	}
	// G5: --findings-as-issues files each finding note as its OWN standalone,
	// severity-labeled, triageable issue (distinct from G4, which posts findings
	// as COMMENTS on a task's issue). The flag selects the standalone-issue mode;
	// without it, the default finding-comment path is unchanged.
	findingsAsIssues := f.Bool("findings-as-issues")

	// Visibility is re-checked LIVE at every push: a repo flipped public
	// after linking must re-trip the disclosure gate. Findings ride this same
	// gate below — a finding comment on a public issue is the risk-rank-2 leak.
	if err := disclosureGate(w, repo, p); err != nil {
		return err
	}

	// G6: pre-create the full static label set (type:*, severity:*, finding,
	// decision, status:*) with stable colors ONCE, before any issue-create
	// references a label — so a not-yet-created label never fails a push under a
	// flaky network. The project's area: label is dynamic, so it rides the extras.
	taskArea := areaLabel(p.Slug)
	// Pre-creating labels is a remote write, so a dry-run skips it — labels are
	// cosmetic and outside the create/adopt/close the preview reports.
	if !dry {
		precreateLabels(w, repo, taskArea)
	}

	// dacli 224: a project maps to ONE milestone, so a mirrored repo groups its
	// task issues under a planning milestone the way a hand-run project does.
	// Ensure it exists BEFORE any issue-create references it — `gh issue create
	// --milestone` hard-fails on an unknown milestone, so haveMilestone is true
	// ONLY once the milestone is positively confirmed present; a create that
	// could not be confirmed is skipped, not passed as a poison --milestone.
	milestone := milestoneTitle(p)
	// A dry-run must not POST a milestone, so it probes existence read-only
	// (milestoneExists) instead of ensureMilestone, which creates one. haveMilestone
	// then reflects the LIVE state the preview reasons about.
	haveMilestone := false
	if dry {
		if ok, merr := milestoneExists(w, repo, milestone); merr == nil {
			haveMilestone = ok
		}
	} else {
		haveMilestone = ensureMilestone(w, repo, milestone)
	}

	// One issue-list snapshot for the ENTIRE push: adoption scans it in memory
	// instead of a full `gh issue list` per task/decision/finding (the old
	// behaviour cost one list call for every unmapped note in the loop). Scoped
	// to the linked repo so adoption reads the repo push writes to (dacli 221).
	idx := newMarkerIndex(w, repo)

	// Refuse a truncated index BEFORE the first create, not after the last one.
	// Both other cap-readers (pull's listIssues, project's itemSnapshot) stop on
	// a partial page; push writes to a live repository and so has the most to
	// lose by continuing (dacli 205).
	if err := idx.preflight(); err != nil {
		return err
	}

	// --findings-as-issues is a FINDINGS-ONLY push: skip the task/decision mirror
	// entirely so a run can publish just an audit's findings without also filing
	// an issue for every task in the project. (Pass --with-tasks to do both.)
	if findingsAsIssues && !f.Bool("with-tasks") {
		return mirrorFindingsOnly(w, p, repo, f, idx, dry, ctx.Stdout)
	}

	tasks, err := store.ListTasks(w, p.Slug, "")
	if err != nil {
		return err
	}

	// Task window (task 275): a mature project's done set can be hundreds of
	// tasks, and mirroring the whole backlog files (or adopts and closes) an
	// issue for every one — the operator cannot publish a single wave without
	// reaching for raw gh. Narrow the mirror to the requested window: the
	// explicit refs after the project and/or a --since cutoff. With neither, the
	// full backlog mirrors exactly as before. Computed once here and reused by
	// the --with-tasks finding-issue mirror below so both scope identically.
	since, err := sinceWindow(f)
	if err != nil {
		return err
	}
	tasks, err = selectTaskWindow(tasks, f.Pos[1:], since)
	if err != nil {
		return err
	}
	// task 298: the window must scope the decision and finding-issue mirrors too,
	// not just the tasks — a decision rode along on EVERY push regardless of the
	// window, so `github push core 275` published task 275's issue AND every other
	// decision in the project, an unbounded disclosure on a public repo. refTasks
	// is the ref axis of the window (the tasks the explicit refs named); the
	// decision/finding mirrors scope their `about`-match against it so a windowed
	// push publishes only the notes attached to the named tasks. Empty when no refs
	// were given, so a whole-project or --since-only push is unchanged.
	refTasks := refMatchedTasks(tasks, f.Pos[1:])

	// The project's finding notes are read ONCE for the whole push (dacli 245).
	// store.ListNotes used to live INSIDE the task loop below, which made the
	// finding mirror O(tasks × notes): measured on this workspace (238 tasks)
	// the per-task shape costs 579,551,265 ns/op, 341 MB, 1,007,954 allocs
	// against 2,433,990 ns/op, 1.4 MB, 4,235 allocs hoisted — 0.58s of pure
	// local file I/O and 341 MB of garbage per push, ~10s at 1000 tasks. The
	// only per-task work is the about-match, which needs no re-read; do not put
	// the read back in the loop. Skipped in --findings-as-issues mode, where
	// mirrorFindings is not called at all.
	var findingCommentNotes []*mdstore.Doc
	if !findingsAsIssues {
		// Errors are swallowed exactly as before: an unreadable notes dir meant
		// "no findings to mirror", never a failed push.
		findingCommentNotes, _ = store.ListNotes(w, p.Slug, model.NoteFinding)
		findingCommentNotes = canonicalFindingDocs(findingCommentNotes)
	}

	// The decision notes (and, in --findings-as-issues mode, the finding notes) are
	// read ONCE here so the blast-radius plan below and the mirrors that follow
	// share one traversal — the same read-once discipline the finding-comment path
	// uses (dacli 245).
	decNotes, err := decisionNotes(w, p.Slug)
	if err != nil {
		return err
	}
	var findIssueNotes []noteFile
	if findingsAsIssues {
		if findIssueNotes, err = findingNotes(w, p.Slug); err != nil {
			return err
		}
	}
	decNotes = canonicalNoteFiles(decNotes)
	findIssueNotes = canonicalNoteFiles(findIssueNotes)

	// task 298: state the blast radius — how many NEW issues of each kind this push
	// will create — BEFORE creating any, so an operator sees the disclosure size on
	// a public repo while it can still be aborted. Counts only genuine creates (a
	// note already mapped or adoptable by marker files no new issue), and only
	// notes inside the window, so the number matches what the mirrors below do.
	taskCreates := plannedTaskCreates(w, tasks, idx)
	decCreates := plannedNoteCreates(w, decNotes, refTasks, since, idx, decisionMarker, false)
	findCreates := 0
	if findingsAsIssues {
		findCreates = plannedNoteCreates(w, findIssueNotes, refTasks, since, idx, findingIssueMarker, true)
	}
	fmt.Fprintf(ctx.Stdout, "plan: will create %d task, %d decision, %d finding issue(s) on %s\n",
		taskCreates, decCreates, findCreates, repo)

	created, adopted, closed, kept, commented := 0, 0, 0, 0, 0
	for _, t := range tasks {
		num := mappedIssue(t)

		// The idempotent create path, per GITHUB.md § 4: frontmatter first,
		// then SEARCH BY MARKER, and only then create. A crash between the
		// remote create and the local mapping write must converge on re-run
		// by adoption, never by a duplicate.
		if num == 0 {
			if found := idx.find(marker(w, t)); found > 0 {
				num = found
				adopted++
				if dry {
					fmt.Fprintf(ctx.Stdout, "would adopt issue #%d by marker for task %03d-%s\n", num, t.Seq, t.Slug)
				}
			} else if found := idx.findByTitle(taskIssueTitle(t)); found > 0 {
				// task 275: an issue an operator filed by hand carries the
				// canonical `NNN: <title>` but no dacli marker, so the marker
				// search above misses it. Adopt it by exact title rather than
				// filing a duplicate. The mapping written back below binds it
				// locally, so the next push skips it via mappedIssue — the issue
				// body is never edited, exactly as pull leaves an adopted issue.
				num = found
				adopted++
				if dry {
					fmt.Fprintf(ctx.Stdout, "would adopt issue #%d by title for task %03d-%s\n", num, t.Seq, t.Slug)
				}
			}
		}
		// wouldCreate marks a task whose issue does not exist yet: in a dry-run it
		// leaves num == 0 (there is no real issue number), so the per-issue writes
		// below — mapping, labels, close, comments — are skipped for it and folded
		// into the single "would create" line instead.
		wouldCreate := false
		if num == 0 {
			if dry {
				fmt.Fprintf(ctx.Stdout, "would create issue %q\n", taskIssueTitle(t))
				created++
				wouldCreate = true
			} else {
				body := issueBody(w, t)
				createArgs := []string{"issue", "create", "--title", taskIssueTitle(t), "--body", body, "--label", "type:task"}
				if taskArea != "" {
					createArgs = append(createArgs, "--label", taskArea)
				}
				// Only pass --milestone when it is CONFIRMED to exist — an unknown
				// milestone would hard-fail this create and abort the whole push.
				if haveMilestone {
					createArgs = append(createArgs, "--milestone", milestone)
				}
				out, err := ghRepo(w, repo, createArgs...)
				if err != nil {
					return fmt.Errorf("issue create for %03d-%s: %w (%s)", t.Seq, t.Slug, err, out)
				}
				num = trailingInt(out)
				if num == 0 {
					return fmt.Errorf("could not parse issue number from gh output %q", out)
				}
				created++
			}
		} else if mappedIssue(t) != 0 {
			kept++
		}

		// A dry-run writes nothing — remote or local — for a would-be-created
		// issue: it has no real number to map, label, comment on or close.
		if wouldCreate {
			// Findings on a to-be-created issue ride the create; count nothing
			// here (the issue does not exist yet), and close is folded into the
			// create line below.
			if t.Status == model.StatusDone {
				closed++
			}
			continue
		}

		// Write the mapping back — after the remote exists, so the failure
		// window leaves an adoptable issue, not a dangling mapping. Skipped when
		// the mapping is already current, so an idempotent re-push does not
		// rewrite every task file (churning mtimes and git blame for a no-op).
		// A dry-run performs no local write, so the mapping is left untouched.
		if desired := githubBlock(num, repo); !dry && mappedBlockChanged(t.Doc, desired) {
			if err := store.WithTask(w, t, func(fresh *store.Task) error {
				if mappedBlockChanged(fresh.Doc, desired) {
					fresh.Doc.Front.SetBlock("github", desired)
				}
				return store.SaveTask(fresh)
			}); err != nil {
				return err
			}
		}
		// G1/G6 taxonomy and milestone assignment are best-effort remote writes
		// outside the create/adopt/close the preview reports, so a dry-run skips
		// them entirely rather than mutating labels on the live repo.
		if !dry {
			// One diffed edit per issue, covering the status label (G1), the
			// type:/area: taxonomy (G6) and the milestone (dacli 224).
			//
			// These were three unconditional calls that issued five gh
			// invocations between them — add, three separate removes, plus the
			// taxonomy — for EVERY mapped issue on every push, whether or not
			// anything had changed. At this repo's ~300 mirrored tasks an
			// idempotent re-push spent ~2,100 network round-trips relabelling
			// issues to the values they already had. syncIssueTaxonomy diffs
			// against the snapshot the marker index has already fetched and
			// makes zero calls when the issue is current.
			syncIssueTaxonomy(w, repo, idx, num, t.Status, taskArea, milestone, haveMilestone)
		}

		// Findings backlink to the issue a human sees: each finding note about
		// this task becomes an issue comment, idempotent by a per-finding marker
		// so a re-push never duplicates. Behind the disclosure gate tripped above.
		// Skipped in --findings-as-issues mode, where findings become standalone
		// issues instead (mirrored once, after the task loop).
		if !findingsAsIssues {
			if dry {
				// findingsToPost is the SHARED decision both the real mirror and
				// this preview run, so a would-comment can never drift from what a
				// real push comments (task 294).
				if todo, terr := findingsToPost(w, repo, num, t, findingCommentNotes); terr == nil {
					for _, n := range todo {
						id, _ := n.Front.Get("id")
						fmt.Fprintf(ctx.Stdout, "would comment on issue #%d: finding %s\n", num, id)
						commented++
					}
				}
			} else {
				posted, ferr := mirrorFindings(w, repo, num, t, findingCommentNotes)
				commented += posted
				if ferr != nil {
					return fmt.Errorf("github push incomplete: task stage stopped at %03d-%s; closure stage and decision stage were not completed: %w", t.Seq, t.Slug, ferr)
				}
			}
		}

		if t.Status == model.StatusDone {
			if dry {
				fmt.Fprintf(ctx.Stdout, "would close issue #%d (%s)\n", num, taskIssueTitle(t))
				closed++
			} else if closeOut, err := ghRepo(w, repo, "issue", "close", strconv.Itoa(num)); err != nil {
				// A failed close is a partial apply, not cosmetic drift. Returning
				// here leaves later decisions untouched; the next marker-idempotent
				// push retries this closure before entering that stage (task 394).
				return fmt.Errorf("github push incomplete: task stage stopped at %03d-%s; closure stage failed and decision stage was not completed: %w (%s)", t.Seq, t.Slug, err, closeOut)
			} else {
				closed++
			}
		}
	}
	if dry {
		fmt.Fprintf(ctx.Stdout, "dry-run: push would create %d, adopt %d, leave %d unchanged, close %d, add %d finding comment(s) (of %d tasks); nothing was written\n",
			created, adopted, kept, closed, commented, len(tasks))
	} else {
		fmt.Fprintf(ctx.Stdout, "push: %d created, %d adopted-by-marker, %d unchanged, %d closed, %d finding comment(s) (of %d tasks)\n",
			created, adopted, kept, closed, commented, len(tasks))
	}
	if haveMilestone {
		if dry {
			fmt.Fprintf(ctx.Stdout, "milestone: %s (task issues would be assigned)\n", milestone)
		} else {
			fmt.Fprintf(ctx.Stdout, "milestone: %s (task issues assigned)\n", milestone)
		}
	}

	// G2: decisions ride the SAME explicit push and the SAME disclosure gate
	// (already tripped above), never auto-run on ship. They share the one
	// issue-list snapshot, so decision adoption costs no extra list call. task 298:
	// scoped to the SAME window as the tasks — refTasks + since — so a windowed
	// push publishes only the decisions attached to the named tasks, never every
	// project decision unscoped.
	if err := mirrorDecisions(w, repo, decNotes, refTasks, since, idx, dry, ctx.Stdout); err != nil {
		return err
	}

	// With --with-tasks, findings-as-issues runs AFTER the task mirror above.
	// (The findings-ONLY path returned earlier before the task loop.) It shares
	// the same window — refTasks + since — computed for the task window above, so
	// the standalone finding issues scope identically to the tasks (task 298).
	if findingsAsIssues {
		if err := mirrorFindingIssues(w, repo, findIssueNotes, refTasks, since, idx, dry, ctx.Stdout); err != nil {
			return err
		}
	}
	if !dry {
		fmt.Fprintf(ctx.Stdout, "applied: github push completed task, closure, decision, and finding stages on %s\n", repo)
	}
	return nil
}

// mirrorFindingsOnly is the FINDINGS-ONLY push (--findings-as-issues without
// --with-tasks): the disclosure gate has already tripped; project just the
// finding notes as issues, scoped by --since. No task/decision issues are
// touched — so an audit can publish its findings without filing an issue per
// task in the project.
func mirrorFindingsOnly(w *workspace.Workspace, p *store.Project, repo string, f *clikit.Flags, idx *markerIndex, dry bool, out io.Writer) error {
	since, err := sinceWindow(f)
	if err != nil {
		return err
	}
	// task 298: honor an explicit task-ref window here too — scope the finding
	// issues to those about the named tasks — rather than silently ignoring the
	// refs an operator typed (the invisible-drop failure mode). The refs are
	// validated against the project's tasks via selectTaskWindow (zero since, so it
	// returns exactly the ref-matched tasks), so an unknown ref is a not-found
	// error, never a silent empty scope.
	var refTasks []*store.Task
	if refs := f.Pos[1:]; len(refs) > 0 {
		tasks, err := store.ListTasks(w, p.Slug, "")
		if err != nil {
			return err
		}
		if refTasks, err = selectTaskWindow(tasks, refs, time.Time{}); err != nil {
			return err
		}
	}
	notes, err := findingNotes(w, p.Slug)
	if err != nil {
		return err
	}
	notes = canonicalNoteFiles(notes)
	// State the blast radius before creating any issue (task 298).
	fmt.Fprintf(out, "plan: will create %d finding issue(s) on %s\n",
		plannedNoteCreates(w, notes, refTasks, since, idx, findingIssueMarker, true), repo)
	if err := mirrorFindingIssues(w, repo, notes, refTasks, since, idx, dry, out); err != nil {
		return err
	}
	if !dry {
		fmt.Fprintf(out, "applied: github push completed finding stages on %s\n", repo)
	}
	return nil
}

// refMatchedTasks returns the subset of tasks that the explicit window refs name —
// the ref axis of the push window (task 298), as distinct from the --since axis.
// It is the set the decision and finding-issue mirrors scope their `about`-match
// against, so a windowed push publishes only the notes attached to the named
// tasks. Empty when no refs were given, so a whole-project or --since-only push
// leaves the note mirrors unscoped by refs (the since axis still applies).
func refMatchedTasks(tasks []*store.Task, refs []string) []*store.Task {
	if len(refs) == 0 {
		return nil
	}
	var out []*store.Task
	for _, t := range tasks {
		for _, ref := range refs {
			if taskMatchesRef(t, ref) {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// noteInWindow reports whether a decision or finding note falls inside the push
// window (task 298). With no window — no refs (empty refTasks) and a zero since —
// every note is in, the default whole-project mirror. Otherwise a note is in the
// window when it was created at or after the --since cutoff (the temporal axis,
// the rule the task window and the finding-issue --since filter both apply), OR
// its `about` field names one of the tasks the explicit refs selected (the ref
// axis) — so `github push core 275` mirrors the decision about 275 and leaves
// every other project decision unpublished, the scoping the task mirror already
// had and the decision mirror lacked.
func noteInWindow(doc *mdstore.Doc, refTasks []*store.Task, since time.Time) bool {
	if len(refTasks) == 0 && since.IsZero() {
		return true
	}
	if !since.IsZero() {
		if cs, ok := doc.Front.Get("created"); ok {
			if ct, err := time.Parse(time.RFC3339, cs); err == nil && !ct.Before(since) {
				return true
			}
		}
	}
	for _, rt := range refTasks {
		if findingAboutTask(doc, rt) {
			return true
		}
	}
	return false
}

// plannedTaskCreates counts the windowed tasks that would file a BRAND-NEW issue —
// neither already mapped nor adoptable by marker or by canonical title — using the
// in-memory issue-list snapshot, so the blast-radius plan (task 298) matches what
// the task loop actually creates without a second network call.
func plannedTaskCreates(w *workspace.Workspace, tasks []*store.Task, idx *markerIndex) int {
	n := 0
	for _, t := range tasks {
		if mappedIssue(t) != 0 {
			continue
		}
		if idx.find(marker(w, t)) > 0 {
			continue
		}
		if idx.findByTitle(taskIssueTitle(t)) > 0 {
			continue
		}
		n++
	}
	return n
}

// plannedNoteCreates counts the in-window notes that would file a brand-new issue —
// keyed on the same conditions the decision/finding-issue mirrors use (has an id,
// non-empty text when requireText, in the window, not already mapped, not adoptable
// by marker) — so the blast-radius plan (task 298) never over- or under-states the
// creates. mk is the note's marker function (decisionMarker or findingIssueMarker).
func plannedNoteCreates(w *workspace.Workspace, notes []noteFile, refTasks []*store.Task, since time.Time, idx *markerIndex, mk func(*workspace.Workspace, string) string, requireText bool) int {
	n := 0
	for _, dn := range notes {
		if dn.id == "" {
			continue
		}
		if requireText && findingText(dn.doc) == "" {
			continue
		}
		if !noteFileInWindow(dn, refTasks, since) {
			continue
		}
		if mappedIssueDoc(dn.doc) != 0 {
			continue
		}
		if findNoteMarker(w, idx, dn, mk) > 0 {
			continue
		}
		n++
	}
	return n
}

// sinceWindow parses --since <dur> (e.g. 2h, 90m) into a cutoff time; the zero
// time means "no window — every finding". Shared so the findings-only and
// --with-tasks paths scope identically.
func sinceWindow(f *clikit.Flags) (time.Time, error) {
	v := f.Get("since")
	if v == "" {
		return time.Time{}, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return time.Time{}, clikit.Usagef("--since wants a duration like 2h or 90m: %v", err)
	}
	return time.Now().Add(-d), nil
}

// selectTaskWindow narrows the tasks a push mirrors to the requested window
// (task 275): the explicit refs after the project and/or a --since cutoff. With
// no refs and a zero since it returns tasks unchanged, so the default push still
// mirrors the whole backlog.
//
// When both are given the window is the UNION — the named refs PLUS everything
// created since the cutoff — so `push core 275 --since 2h` mirrors task 275 even
// if it predates the window. A task selected by neither is left out.
//
// A ref that names no task in the project is a NOT-FOUND error (exit 4), never a
// silent no-op: an operator who asked to mirror a specific task must hear that it
// was not found, not watch the push report success having filed nothing for it —
// the invisible-success failure mode this tool guards against hardest.
func selectTaskWindow(tasks []*store.Task, refs []string, since time.Time) ([]*store.Task, error) {
	if len(refs) == 0 && since.IsZero() {
		return tasks, nil
	}
	matched := make(map[string]bool, len(refs))
	var out []*store.Task
	for _, t := range tasks {
		include := false
		for _, ref := range refs {
			if taskMatchesRef(t, ref) {
				include = true
				matched[ref] = true
			}
		}
		if !include && !since.IsZero() && taskCreatedSince(t, since) {
			include = true
		}
		if include {
			out = append(out, t)
		}
	}
	var missing []string
	for _, ref := range refs {
		if !matched[ref] {
			missing = append(missing, ref)
		}
	}
	if len(missing) > 0 {
		return nil, store.ErrNotFound{Ref: strings.Join(missing, ", ")}
	}
	return out, nil
}

// taskCreatedSince reports whether task t was created at or after the cutoff. A
// task with no parseable `created` stamp is treated as OUTSIDE the window — a
// --since window means "demonstrably created since", never "assume recent" — the
// same rule the finding-issue --since filter applies (mirrorFindingIssues).
func taskCreatedSince(t *store.Task, since time.Time) bool {
	cs, ok := t.Doc.Front.Get("created")
	if !ok {
		return false
	}
	ct, err := time.Parse(time.RFC3339, cs)
	if err != nil {
		return false
	}
	return !ct.Before(since)
}

// taskIssueTitle is the canonical GitHub issue title a task mirrors to,
// `NNN: <title>`. One definition so the create path and the title-adoption path
// (task 275, idx.findByTitle) compare against the identical string and never
// drift — a drift would either duplicate (adopt misses, create fires) or adopt
// the wrong issue.
func taskIssueTitle(t *store.Task) string {
	return fmt.Sprintf("%03d: %s", t.Seq, t.Title)
}

// disclosureGate re-checks the repo's LIVE visibility and refuses an outbound
// mirror onto a PUBLIC repo without recorded per-project consent. Factored out
// so push and its finding-comment path share one gate — a public repo flipped
// after linking re-trips it, and there is exactly one place the consent is read.
func disclosureGate(w *workspace.Workspace, repo string, p *store.Project) error {
	// Judge the LINKED repo's visibility (the repo push actually writes to),
	// not the cwd remote's — a public linked repo must trip the gate even from a
	// private checkout, and a private linked repo must not be blocked by a public
	// cwd (dacli 221).
	info, err := repoView(w, repo)
	if err != nil {
		return err
	}
	if strings.EqualFold(info.Visibility, "PUBLIC") {
		confirmed, _ := p.Doc.Front.Get("github_public_confirmed")
		if !consentCoversRepo(confirmed, info.NameWithOwner) {
			return clikit.Refusedf("repo %s is PUBLIC and project %s has no recorded consent for it — `dacli github link %s --allow-public`", info.NameWithOwner, p.Slug, p.Slug)
		}
	}
	return nil
}

// consentCoversRepo reports whether the recorded public-push consent applies to
// the repo currently being pushed to. Consent is SCOPED to the exact repo it was
// granted for (stored as nameWithOwner, not a bare boolean), so a project
// relinked to a DIFFERENT public repo must consent afresh — a "true" flag would
// have silently leaked to any repo the project was later pointed at. A legacy
// bare-boolean "true" no longer matches any repo, so it fails closed (re-link).
func consentCoversRepo(confirmed, repo string) bool {
	return confirmed != "" && strings.EqualFold(confirmed, repo)
}

// --- inbound: github pull (G4) ---

// ghIssue is the subset of a remote issue that pull reads to seed a task.
type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	// Labels and Milestone come from the SAME snapshot the marker index
	// already fetches, so the push can diff before writing instead of issuing
	// an unconditional edit per issue. They cost nothing extra: gh returns
	// them in the one `issue list` call (dacli 2026-08-06 audit).
	Labels    []ghLabel   `json:"labels"`
	Milestone ghMilestone `json:"milestone"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghMilestone struct {
	Title string `json:"title"`
}

// labelSet returns the issue's labels as a set for cheap diffing.
func (i ghIssue) labelSet() map[string]bool {
	out := make(map[string]bool, len(i.Labels))
	for _, l := range i.Labels {
		out[l.Name] = true
	}
	return out
}

// markerPrefix leads every issue/decision body dacli itself authors
// (`<!-- dacli:… -->`, `<!-- dacli-decision:… -->`). An inbound issue carrying
// it is one WE mirrored outbound — not a human-authored issue to adopt — so
// pull skips it and never round-trips its own projection back into a task.
const markerPrefix = "<!-- dacli"

// shouldImport reports whether a remote issue should seed a new local task. It
// is the pure skip logic pull applies (unit-tested without gh): adopt an issue
// only when it is human-authored (no dacli marker in the body) AND not already
// mapped to a local task. The mapped-set is what makes pull idempotent — a
// re-pull finds the issue already bound to a task (the issue body itself never
// gains a marker, since pull does not edit the remote), so number-mapping, not
// a body marker, prevents re-import.
//
// A closed, unmapped issue is also skipped: a maintainer closing an issue as
// wontfix/duplicate/resolved is a settled human decision, and pull adopting it
// as a fresh open task would resurrect work the maintainer already ended.
func shouldImport(is ghIssue, mapped map[int]bool) bool {
	if mapped[is.Number] {
		return false
	}
	if strings.Contains(is.Body, markerPrefix) {
		return false
	}
	if strings.EqualFold(is.State, "closed") {
		return false
	}
	return true
}

// ghIssueListLimit caps every `gh issue list` fetch in this package. gh
// paginates transparently up to --limit in one call, so a result landing
// EXACTLY on the cap is indistinguishable from a repo with precisely that
// many issues — the signal that older issues past the page may exist and
// were silently left out (dacli 205).
const ghIssueListLimit = 1000

// fetchAllIssues lists every issue (open and closed) via the strongly-
// consistent list endpoint — the same one searchByMarker trusts over the
// search index — reporting whether the fetch landed exactly on
// ghIssueListLimit, the "may be more than this" signal a caller trusting the
// result as the whole repo must not ignore.
// ghLabelListLimit bounds the label list the same way ghIssueListLimit bounds
// issues, and for the same reason: a silently truncated page must be
// detectable, never mistaken for the complete set.
const ghLabelListLimit = 200

func fetchAllIssues(w *workspace.Workspace, repo, jsonFields string) ([]ghIssue, bool, error) {
	out, err := ghRepo(w, repo, "issue", "list", "--state", "all", "--limit", strconv.Itoa(ghIssueListLimit), "--json", jsonFields)
	if err != nil {
		return nil, false, fmt.Errorf("gh issue list failed: %w (%s)", err, out)
	}
	var issues []ghIssue
	if err := json.Unmarshal([]byte(out), &issues); err != nil {
		return nil, false, fmt.Errorf("parse issue list: %w", err)
	}
	return issues, len(issues) >= ghIssueListLimit, nil
}

// listIssues fetches every issue for cmdPull. A hit-limit fetch errors rather
// than handing pull a partial page to silently adopt as "every issue" — a
// mature repo with more than ghIssueListLimit issues must not have its tail
// silently skipped (dacli 205).
func listIssues(w *workspace.Workspace, repo string) ([]ghIssue, error) {
	issues, truncated, err := fetchAllIssues(w, repo, "number,title,body,state")
	if err != nil {
		return nil, err
	}
	if truncated {
		return nil, fmt.Errorf("gh issue list hit the --limit %d cap — this repo may have more issues than that, and pull would silently adopt only the first page; prune closed issues or raise the limit before retrying", ghIssueListLimit)
	}
	return issues, nil
}

// mappedIssues returns the set of remote issue numbers already bound to a local
// task in this project, so pull skips anything it has already adopted.
func mappedIssues(tasks []*store.Task) map[int]bool {
	mapped := map[int]bool{}
	for _, t := range tasks {
		if n := mappedIssue(t); n > 0 {
			mapped[n] = true
		}
	}
	return mapped
}

// cmdPull adopts human-authored GitHub issues as local tasks — the inbound half
// of the bidirectional loop. It is operator-triggered and read-only against the
// remote (it never edits an issue), so it is NOT gated on public visibility:
// importing an issue discloses nothing. Each adopted issue seeds a task titled
// and bodied from the issue, with the `github: issue/repo` block written back so
// the next pull (and any push) treats it as linked, not re-imported.
func cmdPull(ctx *clikit.Ctx, args []string) error { return pull(ctx, args, nil) }

// pushOnlyFlags are the flags `github push` accepts that pull has no use for.
// `github sync` forwards ONE arg list to both halves, so pull sees them and
// must ignore them rather than refuse — otherwise `github sync <proj> --since
// 2h` exits 2 at pull and the push half never runs at all.
//
// They are tolerated only on that path. A direct `github pull <proj> --since
// 2h` still fails, because there the flag is a real mistake: pull would
// silently ignore it and the caller would believe a window was applied. This is
// the whole point of the Reject guard, and widening pull's own allowlist to fix
// sync would have traded a loud bug for a quiet one.
var pushOnlyFlags = []string{"findings-as-issues", "with-tasks", "since"}

func pull(ctx *clikit.Ctx, args []string, alsoAllow []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	// Reject unknown flags: a typo used to be dropped silently and the
	// command ran as if the caller had meant the default.
	if err := f.Reject(append([]string{"dry-run"}, alsoAllow...)...); err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli github pull <project>")
	}
	// --dry-run previews the adoptions without creating any local task.
	dry := f.Bool("dry-run")
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	repo, _ := p.Doc.Front.Get("github_repo")
	if repo == "" {
		return unlinkedRefusal(w.Root, p.Slug)
	}

	issues, err := listIssues(w, repo)
	if err != nil {
		return err
	}
	tasks, err := store.ListTasks(w, p.Slug, "")
	if err != nil {
		return err
	}
	mapped := mappedIssues(tasks)

	imported, skipped := 0, 0
	for _, is := range issues {
		if !shouldImport(is, mapped) {
			skipped++
			continue
		}
		// The shouldImport decision above is identical in both modes; a dry-run
		// only elides the CreateTask/SaveTask writes and reports what it would
		// adopt (task 294).
		if dry {
			fmt.Fprintf(ctx.Stdout, "would adopt issue #%d → new task %q\n", is.Number, is.Title)
			mapped[is.Number] = true // count a duplicate issue number in one run once
			imported++
			continue
		}
		nt, err := store.CreateTask(w, id.ID, p.Slug, is.Title, store.TaskOpts{
			Context: issueContext(is),
		})
		if err != nil {
			return fmt.Errorf("create task from issue #%d: %w", is.Number, err)
		}
		// Link the new task back to its issue so it is neither re-imported on
		// the next pull nor re-created on push (mappedIssue reads this block).
		if err := store.WithTask(w, nt, func(fresh *store.Task) error {
			fresh.Doc.Front.SetBlock("github", githubBlock(is.Number, repo))
			return store.SaveTask(fresh)
		}); err != nil {
			return err
		}
		mapped[is.Number] = true // guard against a duplicate issue number in one run
		imported++
		fmt.Fprintf(ctx.Stdout, "adopted issue #%d → task %03d-%s\n", is.Number, nt.Seq, nt.Slug)
	}
	if dry {
		fmt.Fprintf(ctx.Stdout, "dry-run: pull would adopt %d, skip %d (of %d issues); nothing was written\n", imported, skipped, len(issues))
	} else {
		fmt.Fprintf(ctx.Stdout, "pull: %d adopted, %d skipped (of %d issues)\n", imported, skipped, len(issues))
	}
	return nil
}

// issueContext seeds the adopted task's Context section: a backlink to the
// issue and its body, so the seed carries the human's original framing.
func issueContext(is ghIssue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Adopted from GitHub issue #%d.\n", is.Number)
	if body := strings.TrimSpace(is.Body); body != "" {
		b.WriteString("\n" + body + "\n")
	}
	return b.String()
}

// cmdSync is the bidirectional convenience: pull adopts human issues first, then
// push projects local state (and finding comments) back out. Each half already
// carries its own linkage/disclosure checks; running pull first means a freshly
// adopted task is mirrored on the same invocation.
func cmdSync(ctx *clikit.Ctx, args []string) error {
	// One arg list, both halves. pull is told which of push's flags to tolerate
	// so a legitimate `github sync <proj> --since 2h` reaches push instead of
	// being refused by the inbound half — while an actual typo is still caught,
	// by whichever half does not recognize it.
	if err := pull(ctx, args, pushOnlyFlags); err != nil {
		return err
	}
	return cmdPush(ctx, args)
}

// --- findings → issue comments (G4) ---

// findingMarker is the per-finding recovery key embedded in every mirrored
// finding comment, keyed on the note id AND the workspace id — a distinct
// prefix from the task/decision markers so it is never mistaken for one and
// (crucially) not seen as a body marker by pull. A comment already carrying it
// is skipped, so a re-push never duplicates a finding.
func findingMarker(w *workspace.Workspace, noteID string) string {
	return fmt.Sprintf("<!-- dacli-finding:%s ws:%s -->", noteID, w.ID)
}

// findingAboutTask reports whether a finding note names this task in its `about`
// field. The `about` value is one or more `[[ref]]` wikilinks (store.CreateNote
// wraps the --about value in brackets); we match each ref EXACTLY against the
// task — never as a loose substring. A substring test would cross-match: a
// finding about task `1007` (or a ULID that happens to contain the digits)
// would look like it belonged to task `007`, mirroring another task's finding
// onto the wrong issue.
func findingAboutTask(n *mdstore.Doc, t *store.Task) bool {
	about, _ := n.Front.Get("about")
	for _, ref := range aboutRefs(about) {
		if taskMatchesRef(t, ref) {
			return true
		}
	}
	return false
}

// aboutRefs extracts the reference tokens a note's `about` field names. The
// field is stored as one or more `[[ref]]` wikilinks; we return each bracketed
// target, falling back to the whole trimmed string when no wikilink is present.
func aboutRefs(about string) []string {
	var refs []string
	rest := about
	for {
		i := strings.Index(rest, "[[")
		if i < 0 {
			break
		}
		rest = rest[i+2:]
		j := strings.Index(rest, "]]")
		if j < 0 {
			break
		}
		if ref := strings.TrimSpace(rest[:j]); ref != "" {
			refs = append(refs, ref)
		}
		rest = rest[j+2:]
	}
	if len(refs) == 0 {
		if s := strings.TrimSpace(about); s != "" {
			refs = append(refs, s)
		}
	}
	return refs
}

// taskMatchesRef reports whether ref names this task EXACTLY — the ULID (with or
// without the `t-` prefix), the slug, the bare or zero-padded sequence, or the
// `NNN-slug` form. It mirrors store.matchesRef (unexported there) so the mirror
// resolves a finding's target the same way the rest of dacli resolves a task
// ref, and never by a loose substring that cross-matches a sibling task.
func taskMatchesRef(t *store.Task, ref string) bool {
	if ref == "" {
		return false
	}
	if t.ID == ref || t.Slug == ref || strings.TrimPrefix(t.ID, "t-") == ref {
		return true
	}
	if strconv.Itoa(t.Seq) == ref {
		return true
	}
	seq3 := fmt.Sprintf("%03d", t.Seq)
	return seq3 == ref || seq3+"-"+t.Slug == ref
}

// findingText collects the note's rendered body — the same rule the brief and PR
// assemblers use: content runs from the level-1 title to the next heading, so we
// concatenate every section's content.
func findingText(n *mdstore.Doc) string {
	var b strings.Builder
	for _, s := range n.Sections {
		b.WriteString(s.Content)
	}
	return strings.TrimSpace(b.String())
}

// findingComment renders the comment body a finding becomes: the marker leads
// (for idempotency), the severity is surfaced, then the finding text and a
// backlink to the dacli note.
func findingComment(mk, severity, id, text string) string {
	var b strings.Builder
	b.WriteString(mk + "\n\n")
	if severity != "" {
		b.WriteString("**" + severity + "** ")
	}
	b.WriteString(text + "\n\n")
	b.WriteString("_Filed as dacli finding " + id + "; the workspace is the source of truth._\n")
	return b.String()
}

func findingDocIDs(n *mdstore.Doc) []string {
	id, _ := n.Front.Get("id")
	aliases, _ := n.Front.Get("mirror_aliases")
	ids := []string{id}
	for _, alias := range strings.Split(aliases, ",") {
		if alias = strings.TrimSpace(alias); alias != "" && alias != id {
			ids = append(ids, alias)
		}
	}
	return ids
}

// commentsHaveMarker reports whether any existing comment already carries the
// marker — the idempotency check that stops a re-push from re-posting a finding.
func commentsHaveMarker(comments []string, mk string) bool {
	for _, c := range comments {
		if strings.Contains(c, mk) {
			return true
		}
	}
	return false
}

// issueComments fetches the bodies of an issue's existing comments so the mirror
// can skip a finding it already posted (idempotency by marker substring). It
// returns an error on a fetch or parse failure so the caller can tell "the issue
// has no comments" (safe to post) from "we could not read the comments" (must
// NOT post) — dacli 220: an unparseable list returned as an empty slice made
// mirrorFindings believe nothing had been posted yet and re-post every finding
// on the next push, so a transient gh/JSON hiccup duplicated every comment.
func issueComments(w *workspace.Workspace, repo string, num int) ([]string, error) {
	out, err := ghRepo(w, repo, "issue", "view", strconv.Itoa(num), "--json", "comments")
	if err != nil {
		return nil, err
	}
	var v struct {
		Comments []struct {
			Body string `json:"body"`
		} `json:"comments"`
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return nil, fmt.Errorf("parse issue %d comments: %w", num, err)
	}
	bodies := make([]string, 0, len(v.Comments))
	for _, c := range v.Comments {
		bodies = append(bodies, c.Body)
	}
	return bodies, nil
}

// mirrorFindings posts each finding note about this task as a comment on the
// mirrored issue, idempotently (a finding whose marker is already present is
// skipped), and returns the count actually posted. A read or post failure is a
// partial apply and therefore fails the push. Existing comments are fetched once
// per task so N findings cost one extra read, not N.
//
// notes is the project's finding notes, read ONCE by the caller before the task
// loop (dacli 245). Reading them here made the mirror O(tasks × notes) —
// 579,551,265 ns/op and 341 MB per push at 238 tasks, versus 2,433,990 ns/op
// and 1.4 MB hoisted. Take the slice; never re-read it per task.
func mirrorFindings(w *workspace.Workspace, repo string, num int, t *store.Task, notes []*mdstore.Doc) (int, error) {
	todo, err := findingsToPost(w, repo, num, t, notes)
	if err != nil {
		// If we cannot read the existing comments we cannot tell which findings
		// are already posted; posting anyway would duplicate every one (dacli 220).
		// Fail loudly; the next push retries once the read succeeds.
		return 0, fmt.Errorf("read finding comments for issue #%d: %w", num, err)
	}
	posted := 0
	for _, n := range todo {
		id, _ := n.Front.Get("id")
		markers := make([]string, 0, len(findingDocIDs(n)))
		for _, sourceID := range findingDocIDs(n) {
			markers = append(markers, findingMarker(w, sourceID))
		}
		mk := strings.Join(markers, "\n")
		sev, _ := n.Front.Get("severity")
		body := findingComment(mk, sev, id, findingText(n))
		commentOut, err := ghRepo(w, repo, "issue", "comment", strconv.Itoa(num), "--body", body)
		if err != nil {
			return posted, fmt.Errorf("post finding %s on issue #%d: %w (%s)", id, num, err, commentOut)
		}
		posted++
	}
	return posted, nil
}

// findingsToPost is the SHARED decision that names which finding notes about
// task t are NOT yet posted as a comment on issue num — the notes a push would
// comment. Both the real mirror (mirrorFindings, which posts them) and the
// --dry-run preview (which prints them) run this one function, so the preview
// can never drift from what a real push would comment (task 294). An error means
// the existing comments could not be read (dacli 220): the caller treats that as
// "post nothing" so a transient read failure never duplicates every comment.
func findingsToPost(w *workspace.Workspace, repo string, num int, t *store.Task, notes []*mdstore.Doc) ([]*mdstore.Doc, error) {
	if num == 0 || len(notes) == 0 {
		return nil, nil
	}
	var about []*mdstore.Doc
	for _, n := range notes {
		if findingAboutTask(n, t) && findingText(n) != "" {
			about = append(about, n)
		}
	}
	if len(about) == 0 {
		return nil, nil
	}
	existing, err := issueComments(w, repo, num)
	if err != nil {
		return nil, err
	}
	var todo []*mdstore.Doc
	for _, n := range about {
		posted := false
		for _, id := range findingDocIDs(n) {
			if commentsHaveMarker(existing, findingMarker(w, id)) {
				posted = true
				break
			}
		}
		if !posted {
			todo = append(todo, n)
		}
	}
	return todo, nil
}

// canonicalFindingDocs applies the same semantic grouping to task comments.
// These docs are read-only projection inputs, so the representative may carry
// the merged evidence without changing any local note on disk.
func canonicalFindingDocs(notes []*mdstore.Doc) []*mdstore.Doc {
	files := make([]noteFile, 0, len(notes))
	for _, doc := range notes {
		id, _ := doc.Front.Get("id")
		title := ""
		for _, section := range doc.Sections {
			if section.Level == 1 {
				title = section.Title
				break
			}
		}
		files = append(files, noteFile{doc: doc, id: id, title: title})
	}
	groups := canonicalNoteFiles(files)
	out := make([]*mdstore.Doc, 0, len(groups))
	for _, group := range groups {
		group.doc.Front.Set("mirror_aliases", strings.Join(group.aliases, ","))
		if len(group.evidence) > 1 {
			group.doc.Sections = append(group.doc.Sections, mdstore.Section{Content: "\n---\n\n" + strings.Join(group.evidence[1:], "\n\n---\n\n") + "\n"})
		}
		out = append(out, group.doc)
	}
	return out
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

// ensureLabel creates a label if missing, with a stable color. Best-effort:
// --force turns an "already exists" into a harmless update (also re-applying the
// canonical color) instead of an error, so a repeated push never fails on label
// creation and the label set stays visually consistent across pushes.
func ensureLabel(w *workspace.Workspace, repo, name string) {
	_, _ = ghRepo(w, repo, "label", "create", name, "--color", labelColor(name), "--force")
}

// labelColor returns a stable GitHub label color (6 hex digits, no leading #)
// for any label dacli emits. Colors are fixed per category so a pre-created set
// is visually consistent and a re-push (label create --force) never randomizes
// them: type: labels share a hue, severities run hot→cool by seriousness, and
// area: labels share one neutral tint.
func labelColor(name string) string {
	switch name {
	case "finding":
		return "d73a4a" // red — a discovered problem
	case "decision":
		return "0075ca" // blue — a recorded choice
	case "type:finding":
		return "5319e7" // purple family (type:)
	case "type:task":
		return "8a63d2"
	case "type:decision":
		return "6f42c1"
	case "severity:major":
		return "b60205" // dark red — most serious
	case "severity:moderate":
		return "d93f0b" // orange
	case "severity:minor":
		return "fbca04" // yellow
	case "severity:unspecified":
		return "cccccc" // gray — no severity carried
	}
	switch {
	case strings.HasPrefix(name, "status:"):
		return "0e8a16" // green family — lifecycle
	case strings.HasPrefix(name, "area:"):
		return "bfd4f2" // light blue — code slice
	}
	return "ededed"
}

// baseLabels is the full STATIC label set every push pre-creates once up front,
// so no issue-create ever races a not-yet-created label (the ensureLabel
// flakiness the G6 spec targets). area: labels are dynamic (derived per note or
// task) and ensured just-in-time before the issue that carries them.
func baseLabels() []string {
	labels := []string{
		"finding", "decision",
		"type:finding", "type:task", "type:decision",
		"severity:major", "severity:moderate", "severity:minor", "severity:unspecified",
	}
	for _, s := range model.AllStatuses {
		labels = append(labels, statusLabel(s))
	}
	return labels
}

// precreateLabels creates the base label set (plus any dynamic extras, e.g. the
// area: label a task project derives) with stable colors ONCE, before any
// issue-create references them — so under a flaky network a missing label never
// fails a push mid-loop. Best-effort per label; a create that later carries the
// label still fails loudly if the label genuinely could not be made.
func precreateLabels(w *workspace.Workspace, repo string, extra ...string) {
	// One list, then create only what is missing. This used to fire an
	// unconditional `label create --force` per label — 13 base labels plus the
	// extras — at the top of EVERY push, so a repo whose labels were created
	// on the first push paid ~13 network round-trips on every push after it to
	// recreate them identically. A repo already set up now costs one call.
	//
	// A failed or unparseable list degrades to the old behavior rather than
	// skipping creation: an unknown label set must not mean "assume they all
	// exist", because a missing label makes every later --add-label fail.
	existing, err := listLabels(w, repo)
	for _, name := range append(baseLabels(), extra...) {
		if name == "" {
			continue
		}
		if err == nil && existing[name] {
			continue
		}
		ensureLabel(w, repo, name)
	}
}

// listLabels returns the repo's current label names as a set.
func listLabels(w *workspace.Workspace, repo string) (map[string]bool, error) {
	out, err := ghRepo(w, repo, "label", "list", "--limit", strconv.Itoa(ghLabelListLimit), "--json", "name")
	if err != nil {
		return nil, fmt.Errorf("gh label list: %w (%s)", err, out)
	}
	var labels []ghLabel
	if err := json.Unmarshal([]byte(out), &labels); err != nil {
		return nil, fmt.Errorf("parse label list: %w", err)
	}
	// A hit cap means the tail is unknown, so treat the whole set as unknown
	// and fall back to unconditional creation. Reading a partial page as the
	// complete set is the milestone-pagination bug (dacli 266) in a new place.
	if len(labels) >= ghLabelListLimit {
		return nil, fmt.Errorf("label list hit the %d cap — cannot tell which labels exist", ghLabelListLimit)
	}
	set := make(map[string]bool, len(labels))
	for _, l := range labels {
		set[l.Name] = true
	}
	return set, nil
}

// otherSeverityLabels are the severity labels an issue must NOT carry given its
// current severity — the stale ones to strip so a finding issue first filed when
// its note had no severity (published as severity:unspecified on the public
// repo) is corrected on the next push instead of accumulating two severity
// labels. Mirrors otherStatusLabels for the status family.
func otherSeverityLabels(sevLabel string) []string {
	var out []string
	for _, s := range []string{"severity:major", "severity:moderate", "severity:minor", "severity:unspecified"} {
		if s != sevLabel {
			out = append(out, s)
		}
	}
	return out
}

// areaSlice derives a best-effort code-slice name from the first `internal/<...>`
// path mentioned in text: the LAST directory segment names the package
// (internal/features/ghmirror → ghmirror, internal/store → store). Trailing
// filename segments (containing a dot, e.g. ghmirror.go) are dropped so the area
// is the package, not the file. Returns "" when no internal path is present, so
// the caller skips the area label cleanly.
func areaSlice(text string) string {
	m := internalPathRe.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	last := ""
	for _, seg := range strings.Split(m[1], "/") {
		if seg == "" || strings.Contains(seg, ".") {
			continue // skip empties and filename-ish segments
		}
		last = seg
	}
	return sanitizeSlice(last)
}

// internalPathRe captures the path following an `internal/` mention. The colon
// and whitespace that end a `file.go:44` reference are not in the class, so a
// trailing line number is naturally excluded.
var internalPathRe = regexp.MustCompile(`internal/([A-Za-z0-9_./-]+)`)

// sanitizeSlice lowercases a slice name and keeps only label-safe characters, so
// a derived area label is always a valid, stable GitHub label.
func sanitizeSlice(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// areaLabel renders the area: label for a code slice, or "" for an empty slice
// (the signal to skip the label). Shared so findings (slice from the finding
// body) and tasks (slice from the project) produce the identical label form.
func areaLabel(slice string) string {
	s := sanitizeSlice(slice)
	if s == "" {
		return ""
	}
	return "area:" + s
}

// --- milestones (dacli 224) ---

// milestoneTitle is the milestone a project maps to: its human title when set,
// else its slug. One milestone per project, so a mirrored repo's task issues
// group under a planning milestone the way a hand-run project's do.
func milestoneTitle(p *store.Project) string {
	if t := strings.TrimSpace(p.Title); t != "" {
		return t
	}
	return p.Slug
}

// milestoneTitles splits the newline-separated titles a `gh api … --jq
// '.[].title'` milestone list emits into a slice, dropping blank lines.
func milestoneTitles(jqOut string) []string {
	var out []string
	for _, line := range strings.Split(jqOut, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// milestonesHave reports whether title is already among the repo's milestones
// (exact match), so ensureMilestone creates one only when it is genuinely absent.
func milestonesHave(titles []string, title string) bool {
	for _, t := range titles {
		if t == title {
			return true
		}
	}
	return false
}

// ghMilestoneListLimit caps the milestones milestoneExists reads in a single
// call. The REST milestones endpoint paginates at 30 per page by default, so a
// bare list saw only the FIRST page: a repo past 30 milestones never found an
// existing one that had fallen onto a later page, and ensureMilestone re-created
// it on every push, accumulating a duplicate each time (dacli 266). per_page
// tops out at 100 on this endpoint, so 100 is both the page size we request and
// the cap — a list landing exactly on it may have a tail we did not see, the
// same "may be more than this" signal fetchAllIssues guards for issues (205).
const ghMilestoneListLimit = 100

// milestoneExists lists the repo's milestones (open AND closed) and reports
// whether title is among them. Milestones live on the REST endpoint, not the
// issue surface, so this calls `gh api` with the repo in the PATH — not ghRepo's
// `--repo`, which `gh api` does not accept — but still through the stubbable `gh`
// var so the failure-path tests intercept it.
//
// A list that lands exactly on ghMilestoneListLimit WITHOUT the title on it is
// not a trustworthy "absent": the title may sit on an unread page, so this
// errors rather than reporting a false absence — a partial page read as the
// whole repo is exactly what let ensureMilestone duplicate a milestone (266). A
// positive find, by contrast, is definitive at any length.
func milestoneExists(w *workspace.Workspace, repo, title string) (bool, error) {
	out, err := gh(w, "api", fmt.Sprintf("repos/%s/milestones?state=all&per_page=%d", repo, ghMilestoneListLimit), "--jq", ".[].title")
	if err != nil {
		return false, err
	}
	titles := milestoneTitles(out)
	if milestonesHave(titles, title) {
		return true, nil
	}
	if len(titles) >= ghMilestoneListLimit {
		return false, fmt.Errorf("gh milestone list hit the per_page %d cap and %q was not on it — milestones beyond that page cannot be checked, and creating against a partial list would duplicate an existing one; prune milestones before retrying", ghMilestoneListLimit, title)
	}
	return false, nil
}

// ensureMilestone makes the project's milestone exist and returns whether it is
// CONFIRMED present. That confirmation is load-bearing: `gh issue create
// --milestone <title>` hard-fails on an unknown milestone and would abort the
// whole push, so a caller passes --milestone ONLY when this returned true — a
// milestone that could not be confirmed (a gh/network failure, or a create that
// did not land) is skipped, exactly like the best-effort labels, rather than
// poisoning every issue-create in the loop.
//
// gh has no `milestone create` verb, so creation is a POST to the REST
// milestones endpoint; a re-run finds it already present and creates nothing,
// and a create that races another push (a 422 already-exists) still confirms on
// the re-list — so the re-check, not the POST's exit code, is the real gate.
//
// A list that could not be trusted (a gh/network failure, or a hit-cap page on
// which the title was absent) is REFUSED, not created against: creating a
// milestone we merely failed to see is the duplicate this task exists to stop,
// so an unconfirmable existence check returns false and skips --milestone rather
// than POSTing a fresh milestone that may already exist on an unread page (266).
func ensureMilestone(w *workspace.Workspace, repo, title string) bool {
	if title == "" || repo == "" {
		return false
	}
	exists, err := milestoneExists(w, repo, title)
	if err != nil {
		return false
	}
	if exists {
		return true
	}
	_, _ = gh(w, "api", "--method", "POST", "repos/"+repo+"/milestones", "-f", "title="+title)
	exists, err = milestoneExists(w, repo, title)
	if err != nil {
		return false
	}
	return exists
}

// --- decisions → GitHub (G2) ---

// decisionMarker is the recovery key embedded in every mirrored decision issue,
// keyed on the note id AND the workspace id — the same marker-idempotency
// machinery tasks use, but a distinct prefix so a decision issue is never
// adopted as a task mirror (and vice versa).
func decisionMarker(w *workspace.Workspace, noteID string) string {
	return fmt.Sprintf("<!-- dacli-decision:%s ws:%s -->", noteID, w.ID)
}

// noteFile is a note read from disk WITH its on-disk path — ListNotes yields
// docs without paths, but a mirror that writes the issue number back onto the
// note frontmatter needs the exact file. Shared by the decision mirror (G2) and
// the finding-issue mirror (G5).
type noteFile struct {
	path  string
	doc   *mdstore.Doc
	id    string
	title string
	// aliases are semantically equivalent local records folded into this one.
	// Their ids remain recovery keys so a partial push that published any member
	// of the group resumes by adoption instead of creating a second issue.
	aliases  []string
	evidence []string
	members  []*mdstore.Doc
}

// canonicalNoteFiles collapses repeated operational records before either the
// blast-radius plan or a remote write is considered. Title token sets make the
// key insensitive to case, punctuation, word order and light inflection. A
// deliberately stricter threshold than task filing avoids merging records that
// merely share an operational prefix (for example DNS vs authentication).
func canonicalNoteFiles(notes []noteFile) []noteFile {
	var out []noteFile
	for _, n := range notes {
		merged := false
		for i := range out {
			if store.TitleSimilarity(out[i].title, n.title) < 0.65 {
				continue
			}
			out[i].aliases = append(out[i].aliases, n.id)
			out[i].members = append(out[i].members, n.doc)
			if evidence := noteEvidence(n); evidence != "" && !containsString(out[i].evidence, evidence) {
				out[i].evidence = append(out[i].evidence, evidence)
			}
			merged = true
			break
		}
		if !merged {
			n.aliases = []string{n.id}
			n.evidence = []string{noteEvidence(n)}
			n.members = []*mdstore.Doc{n.doc}
			out = append(out, n)
		}
	}
	return out
}

func noteFileInWindow(n noteFile, refTasks []*store.Task, since time.Time) bool {
	if len(n.members) == 0 {
		return noteInWindow(n.doc, refTasks, since)
	}
	for _, member := range n.members {
		if noteInWindow(member, refTasks, since) {
			return true
		}
	}
	return false
}

func noteEvidence(n noteFile) string {
	return findingText(n.doc)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func noteFileText(n noteFile) string {
	if len(n.evidence) == 0 {
		return findingText(n.doc)
	}
	return strings.TrimSpace(strings.Join(n.evidence, "\n\n---\n\n"))
}

func findNoteMarker(w *workspace.Workspace, idx *markerIndex, n noteFile, mk func(*workspace.Workspace, string) string) int {
	for _, id := range n.aliases {
		if found := idx.find(mk(w, id)); found > 0 {
			return found
		}
	}
	return 0
}

// noteFiles reads a project's notes of one kind with their on-disk paths and
// level-1 titles — the reader both the decision mirror and the finding-issue
// mirror build on, so the two share one traversal and one write-back contract.
func noteFiles(w *workspace.Workspace, project string, kind model.NoteKind) ([]noteFile, error) {
	dir := w.NotesDir(project, kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no notes dir yet is not an error
		}
		return nil, err
	}
	var out []noteFile
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
		out = append(out, noteFile{path: path, doc: d, id: id, title: title})
	}
	return out, nil
}

// decisionNotes reads the project's decision notes with their on-disk paths.
func decisionNotes(w *workspace.Workspace, project string) ([]noteFile, error) {
	return noteFiles(w, project, model.NoteDecision)
}

// findingNotes reads the project's finding notes with their on-disk paths, so
// the finding-issue mirror (G5) can write the issue number back onto each note.
func findingNotes(w *workspace.Workspace, project string) ([]noteFile, error) {
	return noteFiles(w, project, model.NoteFinding)
}

// decisionBody renders the WHY that is the whole point of mirroring a decision:
// the choice, the rejected alternative, and the because. The marker leads (for
// crash-recovery adoption) and the note id trails (the backlink to the dacli
// decision).
func decisionBody(w *workspace.Workspace, dn noteFile) string {
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
	if len(dn.evidence) > 1 {
		b.WriteString("**Additional evidence:**\n\n" + strings.Join(dn.evidence[1:], "\n\n---\n\n") + "\n\n")
	}
	b.WriteString("_Mirrored from dacli decision " + dn.id + "; the workspace is the source of truth._\n")
	return b.String()
}

// mirrorDecisions projects each decision note to an issue labeled `decision`,
// reusing the marker/searchByMarker/write-back idempotency the task mirror uses:
// frontmatter mapping first, then SEARCH BY MARKER, and only then create — so a
// crash between the remote create and the local write converges by adoption,
// never a duplicate.
func mirrorDecisions(w *workspace.Workspace, repo string, notes []noteFile, refTasks []*store.Task, since time.Time, idx *markerIndex, dry bool, out io.Writer) error {
	if len(notes) == 0 {
		return nil
	}
	// The `decision`/`type:decision` labels are pre-created by precreateLabels at
	// the start of the push, so no create here races a missing label.

	created, adopted, kept, skipped := 0, 0, 0, 0
	for _, dn := range notes {
		// task 298: a windowed push mirrors ONLY the decisions inside the window
		// (about a named task, or created since the cutoff); every other decision
		// is left unpublished rather than riding along on a scoped push.
		if !noteFileInWindow(dn, refTasks, since) {
			skipped++
			continue
		}
		if dn.id == "" {
			// A note with no id cannot be keyed idempotently; skip rather than
			// risk creating a duplicate on every push.
			continue
		}
		num := mappedIssueDoc(dn.doc)
		if num == 0 {
			if found := findNoteMarker(w, idx, dn, decisionMarker); found > 0 {
				num = found
				adopted++
				if dry {
					fmt.Fprintf(out, "would adopt issue #%d by marker for decision %s\n", num, dn.id)
				}
			}
		}
		// A dry-run cannot obtain a real issue number for a would-be-created
		// decision, so it prints the create and leaves the mapping/labels alone.
		if num == 0 {
			if dry {
				fmt.Fprintf(out, "would create issue %q\n", "decision: "+dn.title)
				created++
				continue
			}
			ghout, err := ghRepo(w, repo, "issue", "create",
				"--title", "decision: "+dn.title,
				"--body", decisionBody(w, dn),
				"--label", "decision",
				"--label", "type:decision")
			if err != nil {
				return fmt.Errorf("issue create for decision %s: %w (%s)", dn.id, err, ghout)
			}
			num = trailingInt(ghout)
			if num == 0 {
				return fmt.Errorf("could not parse issue number from gh output %q", ghout)
			}
			// FILE IT CLOSED. A decision is a RECORD of a choice already made,
			// not work anyone can action, and leaving it open put it in the
			// queue reviewers read as "things to do". They accumulate with
			// nothing ever closing them: this repo reached 15 open decision
			// issues crowding out 4 real ones, and clearing them was a manual
			// sweep (dacli 336).
			//
			// Closed on CREATE rather than on every push, deliberately. Closing
			// an existing issue on each push would fight a human who reopened
			// one to discuss it — the mirror publishes records, it does not get
			// to overrule someone reading them.
			//
			// Best-effort: the record is already published, and a close that
			// fails leaves an open issue rather than a lost decision.
			if _, cerr := ghRepo(w, repo, "issue", "close", strconv.Itoa(num), "--reason", "completed"); cerr != nil {
				fmt.Fprintf(out, "note: filed decision issue #%d but could not close it (%v) — close it by hand; it is a record, not open work\n", num, cerr)
			}
			created++
		} else if mappedIssueDoc(dn.doc) != 0 {
			kept++
		}
		if dry {
			// Nothing else to preview for an existing decision issue: its taxonomy
			// re-label is a best-effort cosmetic write outside create/adopt.
			continue
		}
		// G6: keep the decision taxonomy current on adopted/existing issues too
		// (best-effort, idempotent) so a re-push enriches issues filed before G6.
		_, _ = ghRepo(w, repo, "issue", "edit", strconv.Itoa(num), "--add-label", "decision", "--add-label", "type:decision")

		// Write the mapping back after the remote exists, so the failure window
		// leaves an adoptable issue, not a dangling mapping — mirrors the task
		// path, and likewise skipped when unchanged so a re-push rewrites no file.
		if desired := githubBlock(num, repo); mappedBlockChanged(dn.doc, desired) {
			dn.doc.Front.SetBlock("github", desired)
			if err := mdstore.WriteFile(dn.path, dn.doc); err != nil {
				return err
			}
		}
	}
	if dry {
		fmt.Fprintf(out, "dry-run: decisions would create %d, adopt %d, leave %d unchanged, %d out-of-window (of %d)\n",
			created, adopted, kept, skipped, len(notes))
	} else {
		fmt.Fprintf(out, "decisions: %d created, %d adopted-by-marker, %d unchanged, %d out-of-window (of %d)\n",
			created, adopted, kept, skipped, len(notes))
	}
	return nil
}

// --- findings → standalone issues (G5) ---

// findingIssueMarker is the recovery key embedded in every mirrored finding
// ISSUE body, keyed on the note id AND the workspace id. It has a distinct
// prefix from the task (`dacli:`), decision (`dacli-decision:`) and finding
// COMMENT (`dacli-finding:`) markers, so searchByMarker/adoption never crosses
// between the standalone-issue mirror and any other mirror. (A finding issue
// carries this in its body; a finding comment carries findingMarker in a
// comment — never the same location, so the two modes are independently
// idempotent.)
func findingIssueMarker(w *workspace.Workspace, noteID string) string {
	return fmt.Sprintf("<!-- dacli-finding-issue:%s ws:%s -->", noteID, w.ID)
}

// severityLabel maps a finding's severity to its GitHub label. An empty or
// unrecognized severity maps to `severity:unspecified` — an honest, still-valid
// label rather than a silently missing one — so the mapping is total and
// unit-testable without a live gh.
func severityLabel(severity string) string {
	s := strings.ToLower(strings.TrimSpace(severity))
	switch s {
	case "major", "moderate", "minor":
		return "severity:" + s
	default:
		return "severity:unspecified"
	}
}

// findingIssueBody renders the body of a standalone finding issue: the marker
// leads (for crash-recovery adoption), the severity is surfaced, then the
// finding detail and a backlink to the local dacli note (the note id is the
// backlink — the workspace is the source of truth).
func findingIssueBody(w *workspace.Workspace, dn noteFile, severity string) string {
	var b strings.Builder
	b.WriteString(findingIssueMarker(w, dn.id) + "\n\n")
	if s := strings.TrimSpace(severity); s != "" {
		b.WriteString("**Severity:** " + s + "\n\n")
	}
	b.WriteString(noteFileText(dn) + "\n\n")
	b.WriteString("_Filed as dacli finding " + dn.id + "; the workspace is the source of truth._\n")
	return b.String()
}

// mirrorFindingIssues projects each finding note to ONE standalone GitHub issue
// labeled `finding` + `severity:<...>`, reusing the exact marker/searchByMarker/
// write-back idempotency the task and decision mirrors use: frontmatter mapping
// first, then SEARCH BY MARKER, and only then create — so a crash between the
// remote create and the local write converges by adoption on the next push,
// never a duplicate. The issue number is written back onto the finding note.
func mirrorFindingIssues(w *workspace.Workspace, repo string, notes []noteFile, refTasks []*store.Task, since time.Time, idx *markerIndex, dry bool, out io.Writer) error {
	if len(notes) == 0 {
		return nil
	}
	// `finding`, `type:finding` and every `severity:*` label are pre-created by
	// precreateLabels at the start of the push; area: labels are dynamic and
	// ensured just-in-time below, so no create here races a missing label.

	created, adopted, kept, skipped := 0, 0, 0, 0
	for _, dn := range notes {
		// task 298: skip findings outside the window — created before the --since
		// cutoff AND not about a task the explicit refs named. A --since-only push
		// still targets just a recent audit; an explicit-ref push now scopes the
		// standalone finding issues to the named tasks the same way the task mirror
		// is scoped, instead of filing every finding in the project.
		if !noteFileInWindow(dn, refTasks, since) {
			skipped++
			continue
		}
		if dn.id == "" || noteFileText(dn) == "" {
			// A note with no id cannot be keyed idempotently, and an empty
			// finding has no detail to file; skip rather than risk a duplicate.
			continue
		}
		severity, _ := dn.doc.Front.Get("severity")
		sevLabel := severityLabel(severity)
		// G6: a best-effort area: label from the first internal/<...> path named
		// in the finding detail (skipped cleanly when none is present). Ensured
		// just-in-time (it is dynamic, so not in the pre-created static set). A
		// dry-run skips the label create — it is a remote write outside the
		// create/adopt the preview reports.
		area := areaLabel(areaSlice(noteFileText(dn)))
		if area != "" && !dry {
			ensureLabel(w, repo, area)
		}

		num := mappedIssueDoc(dn.doc)
		if num == 0 {
			if found := findNoteMarker(w, idx, dn, findingIssueMarker); found > 0 {
				num = found
				adopted++
				if dry {
					fmt.Fprintf(out, "would adopt issue #%d by marker for finding %s\n", num, dn.id)
				}
			}
		}
		// A dry-run cannot obtain a real issue number for a would-be-created
		// finding issue, so it prints the create and leaves the mapping/labels alone.
		if num == 0 {
			if dry {
				fmt.Fprintf(out, "would create issue %q\n", dn.title)
				created++
				continue
			}
			createArgs := []string{"issue", "create",
				"--title", dn.title,
				"--body", findingIssueBody(w, dn, severity),
				"--label", "finding",
				"--label", "type:finding",
				"--label", sevLabel}
			if area != "" {
				createArgs = append(createArgs, "--label", area)
			}
			ghout, err := ghRepo(w, repo, createArgs...)
			if err != nil {
				return fmt.Errorf("issue create for finding %s: %w (%s)", dn.id, err, ghout)
			}
			num = trailingInt(ghout)
			if num == 0 {
				return fmt.Errorf("could not parse issue number from gh output %q", ghout)
			}
			created++
		} else {
			if mappedIssueDoc(dn.doc) != 0 {
				kept++
			}
			if dry {
				// Nothing else to preview for an existing finding issue: the label
				// correction is a best-effort cosmetic write outside create/adopt.
				continue
			}
			// An adopted or already-mapped issue keeps its labels current: add the
			// correct taxonomy AND strip stale severity labels, so a finding issue
			// first filed as severity:unspecified (the public-repo bug) is corrected
			// rather than left carrying two severity labels. Best-effort/idempotent.
			applyFindingLabels(w, repo, num, sevLabel, area)
		}

		// Write the mapping back after the remote exists, so the failure window
		// leaves an adoptable issue, not a dangling mapping — mirrors the task
		// path, and likewise skipped when unchanged so a re-push rewrites no file.
		if desired := githubBlock(num, repo); mappedBlockChanged(dn.doc, desired) {
			dn.doc.Front.SetBlock("github", desired)
			if err := mdstore.WriteFile(dn.path, dn.doc); err != nil {
				return err
			}
		}
	}
	if dry {
		fmt.Fprintf(out, "dry-run: findings-as-issues would create %d, adopt %d, leave %d unchanged, skip %d by --since (of %d); nothing was written\n",
			created, adopted, kept, skipped, len(notes))
	} else {
		fmt.Fprintf(out, "findings-as-issues: %d created, %d adopted-by-marker, %d unchanged, %d skipped-by-since (of %d)\n",
			created, adopted, kept, skipped, len(notes))
	}
	return nil
}

// applyFindingLabels keeps a finding issue's G6 taxonomy current: it adds
// finding, type:finding, the correct severity label and (if any) the area label,
// then strips the OTHER severity labels so exactly one severity: label survives.
// This is what corrects an issue first filed as severity:unspecified — the
// public-repo bug — instead of leaving it carrying two severity labels. All gh
// calls are best-effort (a --remove-label for an absent label errors, ignored).
func applyFindingLabels(w *workspace.Workspace, repo string, num int, sevLabel, area string) {
	if num == 0 {
		return
	}
	args := []string{"issue", "edit", strconv.Itoa(num), "--add-label", "finding", "--add-label", "type:finding", "--add-label", sevLabel}
	if area != "" {
		args = append(args, "--add-label", area)
	}
	_, _ = ghRepo(w, repo, args...)
	for _, stale := range otherSeverityLabels(sevLabel) {
		_, _ = ghRepo(w, repo, "issue", "edit", strconv.Itoa(num), "--remove-label", stale)
	}
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

// githubBlock renders the `github:` frontmatter block that binds a task or note
// to its mirrored issue. One definition so every write-back — and the
// unchanged-check that skips a needless file rewrite — produces the identical
// bytes, and a re-push compares like against like.
func githubBlock(num int, repo string) string {
	return fmt.Sprintf("  issue: %d\n  repo: %s", num, repo)
}

// mappedBlockChanged reports whether the doc's current `github:` block differs
// from the desired one — the guard that lets an idempotent re-push skip a file
// write (and its mtime/git-blame churn) when the issue mapping is already
// current.
func mappedBlockChanged(d *mdstore.Doc, desired string) bool {
	cur, _ := d.Front.GetBlock("github")
	return cur != desired
}

// markerIndex is the strongly-consistent issue list fetched ONCE per push and
// scanned in memory, so marker adoption is not a full `gh issue list` per
// task/decision/finding — the previous behaviour, which cost one list call for
// every unmapped note in the push loop. Built lazily on first lookup and reused
// for the rest of the push.
//
// Adoption is the crash-recovery path: a create that succeeded before its local
// mapping write must be ADOPTED on re-run, never duplicated. It matches the
// marker by exact SUBSTRING over issue bodies from the plain list endpoint —
// deliberately NOT `gh issue list --search`.
//
// `--search` hits GitHub's code/issue search index, which is (a) EVENTUALLY
// CONSISTENT — a just-created issue is not indexed for seconds-to-minutes, so a
// fast retry after a create-then-crash finds nothing and duplicates — and (b)
// TOKENIZED, stripping the angle brackets and colons in the marker so a match
// is not even guaranteed once indexed. The list endpoint reflects a
// just-created issue immediately and we compare bytes, so recovery converges on
// the first retry regardless of index lag. This is what makes the docstring's
// zero-duplicate guarantee hold.
//
// A single snapshot per push is safe: adoption only ever targets issues from a
// PRIOR run — every note created this run writes its mapping back locally before
// the next note is searched — so a mid-push create never needs to be found by a
// later lookup in the same run.
// syncIssueTaxonomy brings ONE issue's labels and milestone to their desired
// state in at most one `gh issue edit`, and in zero calls when they already
// match.
//
// It replaced three unconditional writers (status label, type:/area: labels,
// milestone) that between them issued five gh invocations per mapped issue on
// every push — an add, three separate removes, and the taxonomy edit — with no
// comparison against the issue's current state. On a repo with ~300 mirrored
// tasks that made an idempotent, change-nothing re-push cost roughly 2,100
// network round-trips.
//
// The comparison uses the snapshot the marker index already loaded, so the
// diff is free. When the index could not load (a transient gh failure), the
// snapshot is empty and every issue looks unlabelled: the edit is then issued
// unconditionally, which is exactly the old behavior and still correct, just
// not cheap. Best-effort throughout — a taxonomy write must never fail a push.
func syncIssueTaxonomy(w *workspace.Workspace, repo string, idx *markerIndex, num int, st model.Status, area, milestone string, haveMilestone bool) {
	if num == 0 {
		return
	}
	want := statusLabel(st)
	have := idx.labelsFor(num)

	var args []string
	add := func(label string) {
		if label != "" && !have[label] {
			args = append(args, "--add-label", label)
		}
	}
	add(want)
	add("type:task")
	add(area)
	for _, stale := range otherStatusLabels(st) {
		if have[stale] {
			args = append(args, "--remove-label", stale)
		}
	}
	if haveMilestone && milestone != "" && idx.milestoneFor(num) != milestone {
		args = append(args, "--milestone", milestone)
	}
	if len(args) == 0 {
		return // already current: the common case on a re-push
	}
	// The label must exist before it can be attached; only ensure the ones
	// this edit actually adds.
	for i := 0; i+1 < len(args); i += 2 {
		if args[i] == "--add-label" {
			ensureLabel(w, repo, args[i+1])
		}
	}
	_, _ = ghRepo(w, repo, append([]string{"issue", "edit", strconv.Itoa(num)}, args...)...)
	idx.forget(num)
}

type markerIndex struct {
	w         *workspace.Workspace
	repo      string // the linked repo the snapshot is scoped to (dacli 221)
	loaded    bool
	truncated bool
	issues    []ghIssue
}

func newMarkerIndex(w *workspace.Workspace, repo string) *markerIndex {
	return &markerIndex{w: w, repo: repo}
}

// load fetches the issue list once. Memoize only a SUCCESSFUL fetch. Setting
// loaded before the call cached a failure as "the repo has no issues", so one
// transient gh error made every later find() miss and the push re-created every
// issue as a duplicate — on the exact path (adoption, when the local mapping is
// gone) that the marker exists to protect. A parse failure is a failure too:
// unparseable output is not an empty repository (dacli 208).
func (m *markerIndex) load() {
	if m.loaded {
		return
	}
	// title is fetched alongside body so title-based adoption (task 275) can
	// match a hand-filed issue that carries no marker in its body.
	if issues, truncated, err := fetchAllIssues(m.w, m.repo, "number,title,body,labels,milestone"); err == nil {
		m.issues = issues
		m.truncated = truncated
		m.loaded = true
	}
}

// labelsFor returns the labels the snapshot saw on issue num. An unknown issue
// (freshly created this push, or an index that failed to load) returns an
// empty set, so the caller writes unconditionally rather than skipping a
// needed edit — wrong-but-cheap is not a trade this makes.
func (m *markerIndex) labelsFor(num int) map[string]bool {
	m.load()
	for _, h := range m.issues {
		if h.Number == num {
			return h.labelSet()
		}
	}
	return map[string]bool{}
}

// milestoneFor returns the milestone title the snapshot saw, or "".
func (m *markerIndex) milestoneFor(num int) string {
	m.load()
	for _, h := range m.issues {
		if h.Number == num {
			return h.Milestone.Title
		}
	}
	return ""
}

// forget drops an issue from the snapshot after it has been edited, so a later
// read in the same push does not diff against stale labels and skip a real
// change. Dropping (rather than patching) keeps this honest: an unknown issue
// is treated as needing the write.
func (m *markerIndex) forget(num int) {
	for i, h := range m.issues {
		if h.Number == num {
			m.issues = append(m.issues[:i], m.issues[i+1:]...)
			return
		}
	}
}

// find returns the issue number whose body contains the marker, or 0. The issue
// list is fetched on first use and reused for the rest of the push; a fetch
// failure yields an empty index, so adoption simply finds nothing and the create
// path still guards duplicates by the local mapping written back after create.
func (m *markerIndex) find(mk string) int {
	m.load()
	for _, h := range m.issues {
		if strings.Contains(h.Body, mk) {
			return h.Number
		}
	}
	return 0
}

// findByTitle returns the number of the lowest-numbered issue whose title
// EXACTLY equals title, or 0. It is push's SECOND adoption path (task 275): an
// issue an operator filed by hand — titled `NNN: <task title>` but carrying no
// dacli marker — is adopted into the mapping rather than duplicated.
//
// The match is the FULL canonical title, never a prefix, so a coincidental
// `275: ...` cannot cross-adopt. It runs only AFTER the marker search misses, so
// a dacli-mirrored issue (which carries its marker) is always adopted by marker
// first and never reached here. The lowest number wins on the off chance the
// repo already holds two identically-titled issues, so the choice is a
// deterministic tie-break — never a guess — and a re-push converges on the same
// one instead of oscillating.
func (m *markerIndex) findByTitle(title string) int {
	if title == "" {
		return 0
	}
	m.load()
	best := 0
	for _, h := range m.issues {
		if h.Title == title && (best == 0 || h.Number < best) {
			best = h.Number
		}
	}
	return best
}

// preflight loads the index and refuses a push whose issue-list fetch hit
// ghIssueListLimit, BEFORE the first create.
//
// This used to warn at the end of the push instead. The warning was accurate
// and useless: by the time it printed, every issue past the fetched page had
// already been re-created as a duplicate, because none of them was in the
// index to be adopted. Push is the only one of the three cap-readers that
// writes to a live repository, and it was the only one that did not refuse —
// listIssues (pull) and itemSnapshot (project) both stop.
//
// The truncation is knowable here: load runs on the first find(), find() is
// the first idempotency check, and that check precedes the first create. So
// refusing costs no extra fetch — only the decision to make it before the
// writes rather than after them (dacli 205).
//
// A fetch FAILURE is deliberately not an error here. find() is fail-soft by
// design after dacli 208: a transient gh error leaves the index unloaded so a
// later find() retries, and the create path still guards duplicates by the
// local mapping. Truncation is the opposite case — the fetch succeeded, and
// what it returned is a confident, wrong answer.
func (m *markerIndex) preflight() error {
	m.find("") // forces the fetch; the empty marker matches nothing
	if m.truncated {
		return fmt.Errorf("gh issue list hit the --limit %d cap while indexing markers — issues beyond that page cannot be checked for an existing marker, and pushing against a partial index would re-create each of them as a duplicate; prune closed issues or raise the limit before retrying", ghIssueListLimit)
	}
	return nil
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

// originRepo derives `owner/name` from the `origin` remote's URL, with no
// network call. It handles the two forms git writes: SSH (git@host:owner/name)
// and HTTPS (https://host/owner/name), with or without a .git suffix.
//
// It exists so an unlinked project's refusal can name the exact command that
// fixes it. Deliberately NOT an auto-link: binding a project to a repository
// is a consent decision — the disclosure gate exists because pushing to a
// PUBLIC repo publishes the backlog — and inferring it from a remote would
// make that decision silently, on the operator's behalf, at the first push.
// Naming the repo turns a dead end into a copy-paste without taking the
// decision away (task 306).
func originRepo(root string) string {
	out, err := gitx.Run(root, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	u := strings.TrimSpace(out)
	u = strings.TrimSuffix(u, ".git")
	if i := strings.LastIndex(u, ":"); i >= 0 && !strings.Contains(u[i+1:], "/") {
		// SSH scp-style: git@github.com:owner/name — but only when the part
		// after the colon is not itself a path (which would be a port or a
		// URL scheme).
		u = u[i+1:]
	}
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
		if j := strings.Index(u, "/"); j >= 0 {
			u = u[j+1:]
		}
	}
	parts := strings.Split(strings.Trim(u, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

// unlinkedRefusal names the fix, filling in the repo when `origin` reveals it.
func unlinkedRefusal(root, slug string) error {
	if repo := originRepo(root); repo != "" {
		return clikit.Usagef("project %s is not linked to a GitHub repo, so nothing can be mirrored — this repo's origin is %s: run `dacli github link %s --repo %s` (linking is a consent step, so it is never inferred for you)", slug, repo, slug, repo)
	}
	return clikit.Usagef("project %s is not linked — `dacli github link %s` first (and this repo has no `origin` remote to infer one from)", slug, slug)
}
