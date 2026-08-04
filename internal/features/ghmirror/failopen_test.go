package ghmirror

import (
	"fmt"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

// The marker index is what stops `github push` from re-creating an issue that
// already exists: every mirrored issue carries an HTML marker, and find()
// searches for it. It memoizes the fetch — but set `loaded = true` BEFORE the
// fetch, so a single transient gh failure left an empty index marked as loaded
// and every subsequent find() returned "not found" for the whole push.
//
// The consequence is the one the marker exists to prevent: duplicate issues on
// a real repository, created faster than a human notices. Adoption — recovering
// when the local mapping was lost — is exactly the case that depends on it, so
// the failure lands precisely when the index matters most (dacli 208).
func TestMarkerIndexDoesNotFailOpen(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}

	calls := 0
	orig := gh
	t.Cleanup(func() { gh = orig })
	// First call fails (a wedged network, an expired auth token); later calls
	// succeed and would have found the marker.
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls++
		if calls == 1 {
			return "", fmt.Errorf("gh: could not connect")
		}
		return `[{"number":7,"body":"<!-- dacli:t-abc ws:w1 -->\nbody"}]`, nil
	}

	idx := &markerIndex{w: w}
	if got := idx.find("<!-- dacli:t-abc ws:w1 -->"); got != 0 {
		t.Fatalf("first find during the gh failure returned %d; it cannot know the answer", got)
	}
	// THE POINT: a failed fetch must not be cached as a successful empty one.
	// A retry must actually retry, and must find the existing issue.
	if got := idx.find("<!-- dacli:t-abc ws:w1 -->"); got != 7 {
		t.Fatalf("after the transient failure, find = %d, want 7 — the index cached a failure as an empty result, so the push would duplicate every issue", got)
	}
}

// A successful fetch must still be memoized: the index exists to make one
// network call, not one per marker.
func TestMarkerIndexMemoizesSuccess(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := 0
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls++
		return `[{"number":3,"body":"<!-- dacli:t-xyz ws:w1 -->"}]`, nil
	}

	idx := &markerIndex{w: w}
	for i := 0; i < 5; i++ {
		if got := idx.find("<!-- dacli:t-xyz ws:w1 -->"); got != 3 {
			t.Fatalf("find #%d = %d, want 3", i, got)
		}
	}
	if calls != 1 {
		t.Errorf("gh called %d times; a successful index must be fetched once", calls)
	}
}

// Malformed JSON is also a failure, not an empty repository.
func TestMarkerIndexTreatsBadJSONAsAFailure(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := 0
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls++
		if calls == 1 {
			return "not json at all", nil
		}
		return `[{"number":9,"body":"<!-- dacli:t-q ws:w1 -->"}]`, nil
	}

	idx := &markerIndex{w: w}
	idx.find("<!-- dacli:t-q ws:w1 -->")
	if got := idx.find("<!-- dacli:t-q ws:w1 -->"); got != 9 {
		t.Fatalf("after unparseable output, find = %d, want 9 — a parse failure must not be cached as 'no issues exist'", got)
	}
}

// A fetch landing exactly on --limit issues may be hiding older issues past
// the page — and an older issue is exactly the one whose marker a re-push
// needs to find, or it re-creates it as a duplicate. find() must flag this
// rather than silently treating the page as the whole repo (dacli 205).
func TestMarkerIndexDetectsHitLimit(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return issuesJSON(ghIssueListLimit), nil
	}

	idx := &markerIndex{w: w}
	idx.find("<!-- dacli:nonexistent -->")
	if !idx.truncated {
		t.Fatalf("a fetch landing exactly on the --limit %d cap must be flagged truncated", ghIssueListLimit)
	}
}

// Below the cap, the fetch is a complete picture and must not be flagged.
func TestMarkerIndexBelowLimitIsNotTruncated(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return issuesJSON(ghIssueListLimit - 1), nil
	}

	idx := &markerIndex{w: w}
	idx.find("<!-- dacli:nonexistent -->")
	if idx.truncated {
		t.Fatalf("a below-cap fetch must not be flagged truncated")
	}
}

// Detecting truncation is not enough — push has to STOP on it. This used to
// warn at the end of the push, which is after every issue past the fetched
// page has already been re-created as a duplicate, because none of them was in
// the index to be adopted. Pull and the project board both refuse on a partial
// page; push is the one that writes to a live repository (dacli 205).
func TestPreflightRefusesATruncatedIndex(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return issuesJSON(ghIssueListLimit), nil
	}

	idx := &markerIndex{w: w}
	if err := idx.preflight(); err == nil {
		t.Fatalf("preflight accepted an index that hit the --limit %d cap; the push would duplicate every issue past the page", ghIssueListLimit)
	}
}

// A complete fetch passes, and passes without a second network call — the
// index exists to make one.
func TestPreflightAcceptsACompleteIndex(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	calls := 0
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls++
		return issuesJSON(3), nil
	}

	idx := &markerIndex{w: w}
	if err := idx.preflight(); err != nil {
		t.Fatalf("preflight refused a complete index: %v", err)
	}
	idx.find("<!-- dacli:whatever -->")
	if calls != 1 {
		t.Errorf("gh called %d times; preflight must reuse the one snapshot find() would have taken", calls)
	}
}

// A fetch FAILURE is not a truncation. find() is fail-soft by design after
// dacli 208 — a transient gh error leaves the index unloaded so a later find()
// retries — and preflight must not convert that into a refused push.
func TestPreflightDoesNotRefuseOnAFetchFailure(t *testing.T) {
	w := &workspace.Workspace{Root: t.TempDir()}
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		return "", fmt.Errorf("gh: could not connect")
	}

	idx := &markerIndex{w: w}
	if err := idx.preflight(); err != nil {
		t.Fatalf("preflight refused on a fetch failure: %v — a failed fetch is retried, a truncated one is a confident wrong answer", err)
	}
}
