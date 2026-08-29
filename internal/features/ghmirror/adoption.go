package ghmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

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

type pullOutcome string

const (
	pullCreate            pullOutcome = "create"
	pullAlreadyMapped     pullOutcome = "already-mapped"
	pullExactMatch        pullOutcome = "exact-match/link"
	pullPossibleDuplicate pullOutcome = "possible-duplicate"
	pullRefused           pullOutcome = "refused"
)

type pullPlanItem struct {
	issue      ghIssue
	outcome    pullOutcome
	match      *store.Task
	score      float64
	reason     string
	acceptance acceptanceExtraction
}

func planPull(w *workspace.Workspace, project string, issues []ghIssue, mapped map[int]bool) ([]pullPlanItem, bool, error) {
	plan := make([]pullPlanItem, 0, len(issues))
	plannedLinks := map[string]int{}
	refuse := false
	for _, is := range issues {
		item := pullPlanItem{issue: is, acceptance: extractIssueAcceptance(is.Body)}
		if !shouldImport(is, mapped) {
			switch {
			case mapped[is.Number]:
				item.outcome = pullAlreadyMapped
			case strings.Contains(is.Body, markerPrefix):
				item.outcome, item.reason = pullRefused, "dacli marker"
			default:
				item.outcome, item.reason = pullRefused, "closed issue"
			}
		} else if len(item.acceptance.Ambiguities) > 0 {
			item.outcome = pullRefused
			item.reason = "ambiguous acceptance markup; edit the issue before retrying"
			refuse = true
		} else {
			context := issueContext(is.Number, is.Body)
			match, score, err := store.FindNearDuplicateTaskContent(w, project, store.TaskSimilarityInput{
				Title: is.Title, Problem: context, Acceptance: item.acceptance.Criteria,
			})
			if err != nil {
				return nil, false, err
			}
			item.match, item.score = match, score
			switch {
			case match == nil:
				item.outcome = pullCreate
			case store.TitleSimilarity(is.Title, match.Title) == 1 && mappedIssue(match) == 0 && plannedLinks[match.ID] == 0:
				item.outcome = pullExactMatch
				plannedLinks[match.ID] = is.Number
			case store.TitleSimilarity(is.Title, match.Title) == 1 && plannedLinks[match.ID] != 0:
				item.outcome, item.reason = pullRefused, fmt.Sprintf("matching task is also claimed by issue #%d", plannedLinks[match.ID])
				refuse = true
			case store.TitleSimilarity(is.Title, match.Title) == 1:
				item.outcome, item.reason = pullRefused, fmt.Sprintf("matching task already maps issue #%d", mappedIssue(match))
				refuse = true
			default:
				item.outcome = pullPossibleDuplicate
				refuse = true
			}
		}
		plan = append(plan, item)
	}
	return plan, refuse, nil
}

func printPullPlanItem(out io.Writer, item pullPlanItem) {
	switch item.outcome {
	case pullExactMatch, pullPossibleDuplicate:
		t := item.match
		fmt.Fprintf(out, "issue #%d: %s task %03d-%s %q (%.0f%%)\n", item.issue.Number, item.outcome, t.Seq, t.Slug, t.Title, item.score*100)
	case pullRefused:
		fmt.Fprintf(out, "issue #%d: %s (%s)\n", item.issue.Number, item.outcome, item.reason)
	case pullCreate:
		fmt.Fprintf(out, "issue #%d: %s (would adopt issue #%d → new task %q)\n", item.issue.Number, item.outcome, item.issue.Number, item.issue.Title)
	default:
		fmt.Fprintf(out, "issue #%d: %s\n", item.issue.Number, item.outcome)
	}
	printAcceptanceExtraction(out, item.acceptance)
}

func printAcceptanceExtraction(out io.Writer, plan acceptanceExtraction) {
	fmt.Fprintf(out, "  acceptance source: %s\n", plan.BodyDigest)
	for _, criterion := range plan.Criteria {
		fmt.Fprintf(out, "  acceptance criterion: %q (unchecked)\n", criterion)
	}
	for _, ambiguity := range plan.Ambiguities {
		fmt.Fprintf(out, "  acceptance ambiguity: %s\n", ambiguity)
	}
	for _, skipped := range plan.Skipped {
		fmt.Fprintf(out, "  acceptance skipped: %s\n", skipped)
	}
	if len(plan.Criteria) == 0 && len(plan.Ambiguities) == 0 {
		fmt.Fprintln(out, "  acceptance: zero usable criteria; add top-level checkboxes or an Acceptance section on GitHub, or add criteria locally after adoption")
	}
}

// cmdPull adopts human-authored GitHub issues as local tasks — the inbound half
// of the bidirectional loop. It is operator-triggered and read-only against the
// remote (it never edits an issue), so it is NOT gated on public visibility:
// importing an issue discloses nothing. Each adopted issue seeds a task titled
// and bodied from the issue, with the `github: issue/repo` block written back so
// the next pull (and any push) treats it as linked, not re-imported. Duplicate
// reconciliation uses store's task-creation policy and is project/live-task
// scoped: done/history and cross-project tasks are documented by exclusion,
// because completed work must not have its mapping silently reassigned and
// ownership remains local to one project (issue #718).
func cmdPull(ctx *clikit.Ctx, args []string) error {
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	// A direct pull must reject push-only flags: accepting --since here would
	// tell the caller an inbound window exists when it does not.
	if err := f.Reject("dry-run"); err != nil {
		return err
	}
	return pullParsed(ctx, f)
}

// pullParsed executes the inbound plan after its caller has validated the
// complete command surface. cmdPull supplies the narrow inbound schema;
// cmdSync supplies the documented union before entering either half. Keeping
// validation outside this mutating function makes the ordering enforceable.
func pullParsed(ctx *clikit.Ctx, f *clikit.Flags) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli github pull <project> [--dry-run]")
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
	plan, refused, err := planPull(w, p.Slug, issues, mapped)
	if err != nil {
		return err
	}
	if dry {
		for _, item := range plan {
			printPullPlanItem(ctx.Stdout, item)
		}
		fmt.Fprintln(ctx.Stdout, "dry-run: nothing was written")
		return nil
	}
	if refused {
		for _, item := range plan {
			if item.outcome == pullPossibleDuplicate || item.outcome == pullRefused {
				printPullPlanItem(ctx.Stderr, item)
			}
		}
		return clikit.Refusedf("github pull found an ambiguous or conflicting duplicate; resolve it explicitly before retrying")
	}

	imported, linked, skipped := 0, 0, 0
	for _, item := range plan {
		is := item.issue
		if item.outcome == pullAlreadyMapped || item.outcome == pullRefused {
			skipped++
			continue
		}
		if item.outcome == pullExactMatch {
			if err := store.WithTask(w, item.match, func(fresh *store.Task) error {
				fresh.Doc.Front.SetBlock("github", githubBlock(is.Number, repo))
				mergeAcceptanceCriteria(fresh, item.acceptance.Criteria)
				recordAcceptanceImport(fresh, is.Number, item.acceptance.BodyDigest, id.ID)
				return store.SaveTask(fresh)
			}); err != nil {
				return err
			}
			linked++
			fmt.Fprintf(ctx.Stdout, "linked issue #%d → existing task %03d-%s\n", is.Number, item.match.Seq, item.match.Slug)
			continue
		}
		// Planning above is shared by preview and mutation; this path only
		// performs the create selected by that read-only decision (task 294).
		nt, err := store.CreateTask(w, id.ID, p.Slug, is.Title, store.TaskOpts{
			Context: issueContext(is.Number, is.Body),
			Accept:  item.acceptance.Criteria,
		})
		if err != nil {
			return fmt.Errorf("create task from issue #%d: %w", is.Number, err)
		}
		// Link the new task back to its issue so it is neither re-imported on
		// the next pull nor re-created on push (mappedIssue reads this block).
		if err := store.WithTask(w, nt, func(fresh *store.Task) error {
			fresh.Doc.Front.SetBlock("github", githubBlock(is.Number, repo))
			recordAcceptanceImport(fresh, is.Number, item.acceptance.BodyDigest, id.ID)
			return store.SaveTask(fresh)
		}); err != nil {
			return err
		}
		mapped[is.Number] = true // guard against a duplicate issue number in one run
		imported++
		fmt.Fprintf(ctx.Stdout, "adopted issue #%d → task %03d-%s\n", is.Number, nt.Seq, nt.Slug)
		if len(item.acceptance.Criteria) == 0 {
			fmt.Fprintf(ctx.Stdout, "issue #%d has zero usable acceptance criteria; add top-level checkboxes or an Acceptance section on GitHub, or add criteria locally before implementation\n", is.Number)
		}
	}
	fmt.Fprintf(ctx.Stdout, "pull: %d adopted, %d linked, %d skipped (of %d issues)\n", imported, linked, skipped, len(issues))
	return nil
}

// issueTaskContent preserves the complete human-authored issue body while
// returning a conservative, always-unchecked acceptance projection. GitHub
// checkbox state is not local verification evidence (issue #875).
func issueTaskContent(is ghIssue) (string, []mdstore.Checkbox) {
	plan := extractIssueAcceptance(is.Body)
	boxes := make([]mdstore.Checkbox, 0, len(plan.Criteria))
	if len(plan.Ambiguities) == 0 {
		for _, criterion := range plan.Criteria {
			boxes = append(boxes, mdstore.Checkbox{Text: criterion})
		}
	}
	return issueContext(is.Number, is.Body), boxes
}

type acceptanceExtraction struct {
	Criteria    []string
	Ambiguities []string
	Skipped     []string
	BodyDigest  string
}

// extractIssueAcceptance recognizes explicit top-level task lists anywhere and
// plain bullets only below an Acceptance / Acceptance criteria heading. It is
// intentionally not a general Markdown interpretation engine: fenced code,
// examples, non-goals, and nested lists are never promoted to requirements.
// Nested candidate markup inside an acceptance section fails the whole import
// closed so partial extraction cannot silently rewrite human intent.
func extractIssueAcceptance(body string) acceptanceExtraction {
	return extractIssueAcceptanceFrom(body, "")
}

func extractIssueAcceptanceFrom(body, onlySection string) acceptanceExtraction {
	plan := acceptanceExtraction{BodyDigest: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(body)))}
	seen := map[string]bool{}
	onlySection = normalizedHeading(onlySection)
	acceptanceLevel, excludedLevel := 0, 0
	inFence := false
	for i, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			if looksLikeCriterion(trimmed) {
				plan.Skipped = append(plan.Skipped, fmt.Sprintf("line %d (fenced code): %q", i+1, trimmed))
			}
			continue
		}
		if level, title, ok := markdownHeading(trimmed); ok {
			if acceptanceLevel > 0 && level <= acceptanceLevel {
				acceptanceLevel = 0
			}
			if excludedLevel > 0 && level <= excludedLevel {
				excludedLevel = 0
			}
			heading := normalizedHeading(title)
			switch heading {
			case "acceptance", "acceptance criteria":
				if excludedLevel == 0 && (onlySection == "" || heading == onlySection) {
					acceptanceLevel = level
				}
			case "example", "examples", "non goals", "non-goals":
				excludedLevel = level
			}
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		text, checkbox, bullet := criterionLine(trimmed)
		if excludedLevel > 0 {
			if checkbox || (acceptanceLevel > 0 && bullet) {
				plan.Skipped = append(plan.Skipped, fmt.Sprintf("line %d (excluded section): %q", i+1, trimmed))
			}
			continue
		}
		if acceptanceLevel > 0 && indent > 0 && (checkbox || bullet) {
			plan.Ambiguities = append(plan.Ambiguities, fmt.Sprintf("line %d nested list item %q", i+1, trimmed))
			continue
		}
		if indent > 0 && checkbox {
			plan.Skipped = append(plan.Skipped, fmt.Sprintf("line %d (nested checklist outside acceptance): %q", i+1, trimmed))
			continue
		}
		if acceptanceLevel > 0 && looksLikeNumberedList(trimmed) {
			plan.Ambiguities = append(plan.Ambiguities, fmt.Sprintf("line %d numbered list item %q", i+1, trimmed))
			continue
		}
		if indent > 0 || (!checkbox && !(acceptanceLevel > 0 && bullet)) || (onlySection != "" && acceptanceLevel == 0) {
			continue
		}
		text = normalizeCriterion(text)
		if text == "" {
			plan.Ambiguities = append(plan.Ambiguities, fmt.Sprintf("line %d has an empty criterion", i+1))
			continue
		}
		key := strings.ToLower(text)
		if seen[key] {
			plan.Skipped = append(plan.Skipped, fmt.Sprintf("line %d (duplicate): %q", i+1, text))
			continue
		}
		seen[key] = true
		plan.Criteria = append(plan.Criteria, text)
	}
	if len(plan.Ambiguities) > 0 {
		plan.Criteria = nil
	}
	return plan
}

func markdownHeading(line string) (int, string, bool) {
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	title := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line[level+1:]), strings.Repeat("#", level)))
	return level, title, title != ""
}

func normalizedHeading(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

func normalizeCriterion(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func criterionLine(trimmed string) (text string, checkbox, bullet bool) {
	if len(trimmed) == 5 && strings.Contains("-*+", trimmed[:1]) && trimmed[1:3] == " [" && strings.Contains(" xX", trimmed[3:4]) && trimmed[4:] == "]" {
		return "", true, true
	}
	if len(trimmed) >= 6 && strings.Contains("-*+", trimmed[:1]) && trimmed[1:3] == " [" && strings.Contains(" xX", trimmed[3:4]) && trimmed[4:6] == "] " {
		return trimmed[6:], true, true
	}
	if len(trimmed) >= 2 && strings.Contains("-*+", trimmed[:1]) && trimmed[1] == ' ' {
		return trimmed[2:], false, true
	}
	return "", false, false
}

func looksLikeCriterion(trimmed string) bool {
	_, checkbox, bullet := criterionLine(trimmed)
	return checkbox || bullet
}

func looksLikeNumberedList(trimmed string) bool {
	dot := strings.Index(trimmed, ". ")
	if dot <= 0 {
		return false
	}
	_, err := strconv.Atoi(trimmed[:dot])
	return err == nil
}

func mergeAcceptanceCriteria(task *store.Task, criteria []string) int {
	if len(criteria) == 0 {
		return 0
	}
	seen := map[string]bool{}
	for _, box := range task.Acceptance() {
		seen[strings.ToLower(normalizeCriterion(box.Text))] = true
	}
	sec, _ := task.Doc.Section("Acceptance")
	content := strings.TrimRight(sec.Content, "\n")
	added := 0
	for _, criterion := range criteria {
		key := strings.ToLower(normalizeCriterion(criterion))
		if key == "" || seen[key] {
			continue
		}
		if content != "" {
			content += "\n"
		}
		content += "- [ ] " + criterion
		seen[key] = true
		added++
	}
	task.Doc.SetSection("Acceptance", content+"\n")
	return added
}

func recordAcceptanceImport(task *store.Task, issue int, digest, actor string) {
	task.Doc.Front.SetBlock("github_acceptance_import", fmt.Sprintf("  issue: %d\n  body_digest: %s\n  actor: %s\n  imported_at: %s", issue, digest, actor, time.Now().UTC().Format(time.RFC3339)))
}

type acceptanceMigrationPlan struct {
	Version      int      `json:"version"`
	ID           string   `json:"id"`
	TaskID       string   `json:"task_id"`
	Issue        int      `json:"issue"`
	Repo         string   `json:"repo"`
	BodyDigest   string   `json:"body_digest"`
	FromSection  string   `json:"from_section,omitempty"`
	Criteria     []string `json:"criteria"`
	Ambiguities  []string `json:"ambiguities,omitempty"`
	SkippedLines []string `json:"skipped_lines,omitempty"`
}

// cmdTaskAcceptanceMigrate is the explicit recovery path for tasks adopted
// before acceptance extraction existed. Preview is a pure computation and
// emits a content-addressed plan id. Apply re-fetches the mapped issue and
// refuses unless the exact body and extraction still produce that id, then
// persists the immutable plan before its idempotent task merge (issue #875).
func cmdTaskAcceptanceMigrate(ctx *clikit.Ctx, args []string) error {
	w, id, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("dry-run", "apply", "from-section"); err != nil {
		return err
	}
	if len(f.Pos) != 1 || (f.Bool("dry-run") == (f.Get("apply") != "")) {
		return clikit.Usagef("usage: dacli task acceptance migrate <ref> [--from-section \"Acceptance criteria\"] (--dry-run | --apply plan-id)")
	}
	section := normalizedHeading(f.Get("from-section"))
	if section != "" && section != "acceptance" && section != "acceptance criteria" {
		return clikit.Usagef("--from-section must be Acceptance or Acceptance criteria")
	}
	task, err := store.FindTask(w, f.Pos[0])
	if err != nil {
		return err
	}
	if !id.CanMutate(task.Owner()) {
		return clikit.Refusedf("%03d-%s is owned by %s; only that owner or read-write root may migrate its acceptance", task.Seq, task.Slug, clikit.OrDash(task.Owner()))
	}
	issue, repo := mappedIssue(task), mappedRepo(task)
	if issue == 0 || repo == "" {
		return clikit.Refusedf("%03d-%s has no complete GitHub issue mapping; link or adopt the issue before migrating acceptance", task.Seq, task.Slug)
	}
	remote, err := fetchIssueBody(w, repo, issue)
	if err != nil {
		return err
	}
	extraction := extractIssueAcceptanceFrom(remote.Body, section)
	plan, err := newAcceptanceMigrationPlan(task.ID, repo, issue, section, extraction)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "acceptance migration plan %s for task %03d-%s from %s#%d\n", plan.ID, task.Seq, task.Slug, repo, issue)
	printAcceptanceExtraction(ctx.Stdout, extraction)
	if len(plan.Ambiguities) > 0 {
		return clikit.Refusedf("acceptance migration is ambiguous; edit the GitHub issue or choose an explicit acceptance section, then preview again")
	}
	if len(plan.Criteria) == 0 {
		return clikit.Refusedf("acceptance migration found zero usable criteria; edit the GitHub issue or add criteria locally")
	}
	if f.Bool("dry-run") {
		fmt.Fprintf(ctx.Stdout, "dry-run: nothing was written; apply this immutable plan with --apply %s\n", plan.ID)
		return nil
	}
	if f.Get("apply") != plan.ID {
		return clikit.Refusedf("migration plan changed (requested %s, current %s); review a new --dry-run before applying", f.Get("apply"), plan.ID)
	}
	if err := persistAcceptanceMigrationPlan(w, plan); err != nil {
		return err
	}
	added := 0
	if err := store.WithTask(w, task, func(fresh *store.Task) error {
		added = mergeAcceptanceCriteria(fresh, plan.Criteria)
		if prior, ok := fresh.Doc.Front.GetBlock("github_acceptance_migration"); added == 0 && ok && strings.Contains(prior, "plan: "+plan.ID) {
			return nil
		}
		fresh.Doc.Front.SetBlock("github_acceptance_migration", fmt.Sprintf("  plan: %s\n  issue: %d\n  repo: %s\n  body_digest: %s\n  actor: %s\n  applied_at: %s", plan.ID, issue, repo, plan.BodyDigest, id.ID, time.Now().UTC().Format(time.RFC3339)))
		return store.SaveTask(fresh)
	}); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "applied acceptance migration plan %s: %d criterion/criteria added, %d preserved\n", plan.ID, added, len(plan.Criteria)-added)
	return nil
}

func newAcceptanceMigrationPlan(taskID, repo string, issue int, section string, extraction acceptanceExtraction) (acceptanceMigrationPlan, error) {
	plan := acceptanceMigrationPlan{
		Version: 1, TaskID: taskID, Issue: issue, Repo: repo, BodyDigest: extraction.BodyDigest,
		FromSection: section, Criteria: extraction.Criteria, Ambiguities: extraction.Ambiguities, SkippedLines: extraction.Skipped,
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return acceptanceMigrationPlan{}, err
	}
	plan.ID = fmt.Sprintf("%x", sha256.Sum256(raw))
	return plan, nil
}

func persistAcceptanceMigrationPlan(w *workspace.Workspace, plan acceptanceMigrationPlan) error {
	dir := filepath.Join(w.Root, workspace.Dir, "plans", "acceptance")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := filepath.Join(dir, plan.ID+".json")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		if _, writeErr := f.Write(raw); writeErr != nil {
			_ = f.Close()
			return writeErr
		}
		if syncErr := f.Sync(); syncErr != nil {
			_ = f.Close()
			return syncErr
		}
		return f.Close()
	}
	if !os.IsExist(err) {
		return err
	}
	existing, readErr := os.ReadFile(path)
	if readErr != nil {
		return readErr
	}
	if !bytes.Equal(existing, raw) {
		return clikit.Refusedf("persisted acceptance migration plan %s does not match its content address; inspect the workspace record", plan.ID)
	}
	return nil
}

func fetchIssueBody(w *workspace.Workspace, repo string, issue int) (ghIssue, error) {
	out, err := ghRepo(w, repo, "issue", "view", strconv.Itoa(issue), "--json", "number,title,body,state")
	if err != nil {
		return ghIssue{}, err
	}
	var remote ghIssue
	if err := json.Unmarshal([]byte(out), &remote); err != nil {
		return ghIssue{}, fmt.Errorf("decode GitHub issue #%d: %w", issue, err)
	}
	if remote.Number != issue {
		return ghIssue{}, fmt.Errorf("GitHub returned issue #%d while #%d was requested", remote.Number, issue)
	}
	return remote, nil
}

func mappedRepo(t *store.Task) string {
	block, ok := t.Doc.Front.GetBlock("github")
	if !ok {
		return ""
	}
	for _, line := range strings.Split(block, "\n") {
		if key, value, found := strings.Cut(strings.TrimSpace(line), ":"); found && strings.TrimSpace(key) == "repo" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// issueContext seeds the adopted task's Context section with its backlink and
// the issue body remaining after any canonical acceptance extraction.
func issueContext(number int, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Adopted from GitHub issue #%d.\n", number)
	if body := strings.TrimSpace(body); body != "" {
		b.WriteString("\n" + body + "\n")
	}
	return b.String()
}

// cmdSync is the bidirectional convenience: pull adopts human issues first, then
// push projects local state (and finding comments) back out. Each half already
// carries its own linkage/disclosure checks; running pull first means a freshly
// adopted task is mirrored on the same invocation.
func cmdSync(ctx *clikit.Ctx, args []string) error {
	// Validate the complete sync surface before either half runs. Pull is
	// locally mutating, so relying on cmdPush to reject a typo is too late: the
	// inbound half may already have adopted issues by then (issue #760). Keep
	// the direct pull/push allowlists narrow; this union belongs only to the
	// command that deliberately forwards one argv to both halves.
	f, err := clikit.ParseFlags(args)
	if err != nil {
		return err
	}
	if err := f.Reject("findings-as-issues", "with-tasks", "since", "include-internal", "dry-run"); err != nil {
		return err
	}
	// One arg list, both halves. pull is told which of push's flags to tolerate
	// so a legitimate `github sync <proj> --since 2h` reaches push instead of
	// being refused by the inbound half — while an actual typo is still caught,
	// by whichever half does not recognize it.
	if err := pullParsed(ctx, f); err != nil {
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
