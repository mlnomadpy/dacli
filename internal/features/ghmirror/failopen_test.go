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
