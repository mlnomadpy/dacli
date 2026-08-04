package ghmirror

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// A project maps to one milestone titled by its human title (falling back to the
// slug), so a mirrored repo groups its task issues the way a hand-run project's do.
func TestMilestoneTitle(t *testing.T) {
	if got := milestoneTitle(&store.Project{Title: "Core Backlog", Slug: "core"}); got != "Core Backlog" {
		t.Fatalf("milestoneTitle with a title = %q, want the title", got)
	}
	if got := milestoneTitle(&store.Project{Title: "  ", Slug: "core"}); got != "core" {
		t.Fatalf("milestoneTitle with a blank title = %q, want the slug fallback", got)
	}
}

// milestonesHave is exact-match: a title that is a substring of another must not
// count as present, or ensureMilestone would skip creating a genuinely missing one.
func TestMilestonesHaveIsExact(t *testing.T) {
	titles := milestoneTitles("core\ncore backlog\n\n  extra  \n")
	if len(titles) != 3 {
		t.Fatalf("milestoneTitles dropped/kept wrong lines: %v", titles)
	}
	if !milestonesHave(titles, "core") || !milestonesHave(titles, "extra") {
		t.Fatalf("milestonesHave missed an exact title in %v", titles)
	}
	if milestonesHave(titles, "cor") {
		t.Fatalf("milestonesHave matched a substring 'cor' — must be exact")
	}
}

// ensureMilestone must return true ONLY when it can positively confirm the
// milestone exists, because a false positive would pass a poison --milestone to
// `gh issue create` and abort the whole push. Here the list lacks it, the POST
// lands, and the re-list confirms it — so the create-then-confirm path returns true.
func TestEnsureMilestoneCreatesThenConfirms(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	posted := false
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "--method POST"):
			posted = true
			return `{"title":"Core"}`, nil
		case strings.Contains(joined, "milestones"): // a list
			if posted {
				return "Core", nil
			}
			return "", nil // not there yet
		}
		return "", nil
	}
	if !ensureMilestone(w, "o/r", "Core") {
		t.Fatalf("ensureMilestone: a milestone confirmed present after create must return true")
	}
	if !posted {
		t.Fatalf("ensureMilestone did not POST to create the missing milestone")
	}
}

// The load-bearing failure case: if the milestone cannot be confirmed after the
// create attempt (a flaky gh, a create that did not land), ensureMilestone MUST
// return false so the caller skips --milestone rather than poisoning issue-create.
func TestEnsureMilestoneRefusesWhenUnconfirmed(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "--method POST") {
			return "", nil // pretend the POST "worked" but nothing lands
		}
		return "", fmt.Errorf("gh: could not connect") // every list fails
	}
	if ensureMilestone(w, "o/r", "Core") {
		t.Fatalf("ensureMilestone returned true without ever confirming the milestone — that would abort a real push")
	}
}

// A present milestone is confirmed on the first list, with no POST at all
// (idempotent re-push).
func TestEnsureMilestoneReusesExisting(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	posts := 0
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "--method POST") {
			posts++
		}
		return "Core\nOther", nil
	}
	if !ensureMilestone(w, "o/r", "Core") {
		t.Fatalf("ensureMilestone must reuse an existing milestone")
	}
	if posts != 0 {
		t.Fatalf("ensureMilestone POSTed %d times for an existing milestone; must create none", posts)
	}
}

// The duplicate-milestone bug (266): the REST milestones list paginates at 30
// per page, so a bare fetch saw only the first page. A repo whose target
// milestone has fallen past that page must still find it — and must NOT POST a
// duplicate. Here the milestone sits at position 45 of 60 (well past the old
// 30-item first page); the fetch must request a bigger page (per_page) and the
// find must be reported present with zero creates.
func TestEnsureMilestoneFindsTargetPastFirstPage(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })

	titles := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		if i == 44 {
			titles = append(titles, "Core")
		} else {
			titles = append(titles, fmt.Sprintf("milestone-%02d", i))
		}
	}
	listBody := strings.Join(titles, "\n")

	posts, askedForPage := 0, false
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--method POST") {
			posts++
			return "", nil
		}
		if strings.Contains(joined, "milestones") { // a list read
			if !strings.Contains(joined, "per_page=") {
				t.Fatalf("milestone list did not request a page beyond the default 30: %v", args)
			}
			askedForPage = true
			return listBody, nil
		}
		return "", nil
	}
	if !ensureMilestone(w, "o/r", "Core") {
		t.Fatalf("ensureMilestone missed a milestone sitting past the first page — that is the duplicate bug")
	}
	if !askedForPage {
		t.Fatalf("ensureMilestone never issued the paginated list read")
	}
	if posts != 0 {
		t.Fatalf("ensureMilestone POSTed %d times for a milestone that already exists past the first page — a duplicate", posts)
	}
}

// A list landing exactly on the per_page cap WITHOUT the title on it is not a
// trustworthy "absent": the title may sit on an unread page. ensureMilestone
// must REFUSE (return false, create nothing) rather than treat the partial page
// as the whole repo and POST a milestone that may already exist (266).
func TestEnsureMilestoneRefusesAtListCap(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })

	capped := make([]string, ghMilestoneListLimit) // exactly the cap, none is "Core"
	for i := range capped {
		capped[i] = fmt.Sprintf("milestone-%03d", i)
	}
	listBody := strings.Join(capped, "\n")

	posts := 0
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "--method POST") {
			posts++
			return "", nil
		}
		return listBody, nil
	}
	if _, err := milestoneExists(w, "o/r", "Core"); err == nil {
		t.Fatalf("milestoneExists treated a hit-cap page missing the title as a complete answer; want an error")
	}
	if ensureMilestone(w, "o/r", "Core") {
		t.Fatalf("ensureMilestone returned true off a partial page it could not trust")
	}
	if posts != 0 {
		t.Fatalf("ensureMilestone POSTed %d times against a hit-cap list it could not trust — a possible duplicate", posts)
	}
}

// An empty title or repo is a no-op false — never a create against a bad path.
func TestEnsureMilestoneEmptyInputs(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		t.Fatalf("gh must not be called for empty inputs: %v", args)
		return "", nil
	}
	if ensureMilestone(w, "", "Core") || ensureMilestone(w, "o/r", "") {
		t.Fatalf("ensureMilestone with an empty repo/title must return false without calling gh")
	}
}
