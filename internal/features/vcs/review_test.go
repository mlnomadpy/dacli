package vcs

import (
	"errors"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

// errStub is the failure a stubbed gh returns; the tests never run a real gh
// and never touch the network.
var errStub = errors.New("gh exited 1")

// ghArgs joins one stubbed gh invocation so a test can assert on the shape of
// the call without matching argument indices.
func ghArgs(call []string) string { return strings.Join(call, "\x00") }

// reviewCall returns the single `gh api .../reviews` invocation the stub
// captured, failing when there is not exactly one.
func reviewCall(t *testing.T, calls [][]string) []string {
	t.Helper()
	var found [][]string
	for _, c := range calls {
		if len(c) >= 2 && c[0] == "api" && strings.Contains(c[1], "/reviews") {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly 1 review API call, got %d: %v", len(found), calls)
	}
	return found[0]
}

// A finding whose origin names a file:line must reach the PR as a LINE
// comment — path and line carried in the API payload — while a finding with no
// parseable location must still reach the PR in the summary body rather than
// being dropped (dacli 194).
func TestPostReviewAnchorsFindingsAndKeepsTheRest(t *testing.T) {
	w, tk := prEnv(t)

	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "double free",
		store.NoteOpts{About: tk.ID, Severity: "major", Origin: "file:internal/features/vcs/lifecycle.go:200",
			Body: "the second close runs on an already-closed handle"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "no home",
		store.NoteOpts{About: tk.ID, Severity: "minor", Body: "the whole retry design is wrong"}); err != nil {
		t.Fatal(err)
	}

	calls := stubGH(t, func(dir string, args ...string) (string, error) { return "{}", nil })
	ctx, out := prCtx(w.Root)
	if err := postReview(ctx, w, tk, BranchFor(tk), "https://github.com/acme/widgets/pull/7", reviewComment); err != nil {
		t.Fatalf("postReview: %v", err)
	}

	call := ghArgs(reviewCall(t, *calls))
	if !strings.Contains(call, "repos/{owner}/{repo}/pulls/7/reviews") {
		t.Errorf("review posted to the wrong endpoint: %q", call)
	}
	if !strings.Contains(call, "-f\x00event=COMMENT") {
		t.Errorf("review missing the COMMENT event: %q", call)
	}
	if !strings.Contains(call, "comments[][path]=internal/features/vcs/lifecycle.go") {
		t.Errorf("review missing the anchored path: %q", call)
	}
	if !strings.Contains(call, "comments[][line]=200") {
		t.Errorf("review missing the anchored line: %q", call)
	}
	if !strings.Contains(call, "already-closed handle") {
		t.Errorf("review missing the anchored finding's text: %q", call)
	}
	// Every comments[][...] field must ride on -F, never -f: gh parses the two
	// in separate passes and mixing them scrambles the array objects.
	if strings.Contains(call, "-f\x00comments[][") {
		t.Errorf("comment fields must use -F, not -f: %q", call)
	}
	// The locationless finding lands in the summary body, not on the floor.
	body := reviewBodyArg(t, reviewCall(t, *calls))
	if !strings.Contains(body, "the whole retry design is wrong") {
		t.Errorf("finding without a location was dropped from the summary:\n%s", body)
	}
	if !strings.Contains(body, "Findings without a code location") {
		t.Errorf("summary missing the no-location section:\n%s", body)
	}
	if !strings.Contains(out.String(), "1 line comment(s)") {
		t.Errorf("expected a posted-review report, got:\n%s", out.String())
	}
}

// reviewBodyArg extracts the value of the review's `-f body=` field.
func reviewBodyArg(t *testing.T, call []string) string {
	t.Helper()
	for _, a := range call {
		if strings.HasPrefix(a, "body=") {
			return strings.TrimPrefix(a, "body=")
		}
	}
	t.Fatalf("no body field in %v", call)
	return ""
}

func TestFindingLocationParsing(t *testing.T) {
	cases := []struct {
		origin string
		path   string
		line   int
		ok     bool
	}{
		{"file:internal/x.go:42", "internal/x.go", 42, true},
		{"internal/x.go:42", "internal/x.go", 42, true},
		{"file:internal/x.go#L7", "internal/x.go", 7, true},
		{"file:./cmd/main.go:3", "cmd/main.go", 3, true},
		{"file:internal/x.go", "", 0, false}, // no line to anchor to
		{"agent", "", 0, false},
		{"external:attacker", "", 0, false},
		{"", "", 0, false},
		{"file:/abs/path.go:9", "", 0, false}, // absolute paths 422 the review
		{"file:internal/x.go:0", "", 0, false},
		{"file:internal/x.go:notanumber", "", 0, false},
	}
	for _, c := range cases {
		path, line, ok := findingLocation(c.origin)
		if ok != c.ok || path != c.path || line != c.line {
			t.Errorf("findingLocation(%q) = (%q, %d, %v), want (%q, %d, %v)",
				c.origin, path, line, ok, c.path, c.line, c.ok)
		}
	}
}

// --approve / --request-changes must map onto real GitHub review states, and
// the default must stay COMMENT — dacli never silently approves its own work.
func TestReviewEventMapping(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{nil, reviewComment},
		{[]string{"--approve"}, reviewApprove},
		{[]string{"--request-changes"}, reviewRequestChanges},
	}
	for _, c := range cases {
		f, _ := clikit.ParseFlags(c.args)
		got, err := reviewEventFor(f)
		if err != nil {
			t.Fatalf("reviewEventFor(%v): %v", c.args, err)
		}
		if got != c.want {
			t.Errorf("reviewEventFor(%v) = %q, want %q", c.args, got, c.want)
		}
	}
	f, _ := clikit.ParseFlags([]string{"--approve", "--request-changes"})
	if _, err := reviewEventFor(f); err == nil {
		t.Error("expected a usage error when both review states are asked for")
	}
}

// The mapped event must actually reach the API payload.
func TestPostReviewCarriesRequestedState(t *testing.T) {
	for _, event := range []string{reviewApprove, reviewRequestChanges} {
		w, tk := prEnv(t)
		if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "confirmed defect",
			store.NoteOpts{About: tk.ID, Severity: "major", Origin: "file:internal/x.go:12",
				Body: "the retry loses the error"}); err != nil {
			t.Fatal(err)
		}
		calls := stubGH(t, func(dir string, args ...string) (string, error) { return "{}", nil })
		ctx, _ := prCtx(w.Root)
		if err := postReview(ctx, w, tk, BranchFor(tk), "https://github.com/acme/widgets/pull/9", event); err != nil {
			t.Fatalf("postReview(%s): %v", event, err)
		}
		call := ghArgs(reviewCall(t, *calls))
		if !strings.Contains(call, "-f\x00event="+event) {
			t.Errorf("review did not carry event=%s: %q", event, call)
		}
		if !strings.Contains(call, "/pulls/9/reviews") {
			t.Errorf("review posted to the wrong PR: %q", call)
		}
	}
}

// An explicit APPROVE with nothing filed still posts — the state is the
// message, and GitHub rejects an empty body.
func TestPostReviewApprovesWithoutFindings(t *testing.T) {
	w, tk := prEnv(t)
	calls := stubGH(t, func(dir string, args ...string) (string, error) { return "{}", nil })
	ctx, _ := prCtx(w.Root)
	if err := postReview(ctx, w, tk, BranchFor(tk), "https://github.com/acme/widgets/pull/3", reviewApprove); err != nil {
		t.Fatalf("postReview: %v", err)
	}
	call := reviewCall(t, *calls)
	if body := reviewBodyArg(t, call); strings.TrimSpace(body) == "" {
		t.Error("an APPROVE review must carry a non-empty body")
	}
}

// Nothing filed and nothing verified: post nothing at all. A review that says
// nothing is noise on the PR.
func TestPostReviewSilentWithoutFindings(t *testing.T) {
	w, tk := prEnv(t)
	calls := stubGH(t, func(dir string, args ...string) (string, error) {
		t.Errorf("gh must not be called with no findings and no verdicts: %v", args)
		return "", nil
	})
	ctx, out := prCtx(w.Root)
	if err := postReview(ctx, w, tk, BranchFor(tk), "https://github.com/acme/widgets/pull/1", reviewComment); err != nil {
		t.Fatalf("postReview: %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("expected no gh calls, got %v", *calls)
	}
	if !strings.Contains(out.String(), "no findings or recorded verdicts") {
		t.Errorf("expected an explanation on stdout, got:\n%s", out.String())
	}
}

// Recorded verify verdicts alone (no findings) still post — the existing
// --with-verdicts behaviour, preserved through the new endpoint.
func TestPostReviewStillPostsVerdicts(t *testing.T) {
	w, tk := prEnv(t)
	if _, err := eventlog.Append(w, "a-seat1", model.EventComment, tk.ID, "",
		"verify-verdict: confirmed — claude-code (a-seat1) on claim: race in the merge path"); err != nil {
		t.Fatal(err)
	}
	calls := stubGH(t, func(dir string, args ...string) (string, error) { return "{}", nil })
	ctx, _ := prCtx(w.Root)
	if err := postReview(ctx, w, tk, BranchFor(tk), "https://github.com/acme/widgets/pull/5", reviewComment); err != nil {
		t.Fatalf("postReview: %v", err)
	}
	body := reviewBodyArg(t, reviewCall(t, *calls))
	if !strings.Contains(body, "dacli verify panel") || !strings.Contains(body, "confirmed — claude-code") {
		t.Errorf("verdict review body lost its verdicts:\n%s", body)
	}
}

// GitHub 422s the whole review when a comment names a line outside the diff.
// The retry drops the anchors, never the findings.
func TestPostReviewRetriesWithoutAnchorsOn422(t *testing.T) {
	w, tk := prEnv(t)
	if _, err := store.CreateNote(w, "a-child", "p", model.NoteFinding, "stale anchor",
		store.NoteOpts{About: tk.ID, Severity: "major", Origin: "file:internal/x.go:9999",
			Body: "line moved since the finding was filed"}); err != nil {
		t.Fatal(err)
	}
	var n int
	calls := stubGH(t, func(dir string, args ...string) (string, error) {
		n++
		if n == 1 {
			return "line must be part of the diff (HTTP 422)", errStub
		}
		return "{}", nil
	})
	ctx, _ := prCtx(w.Root)
	if err := postReview(ctx, w, tk, BranchFor(tk), "https://github.com/acme/widgets/pull/8", reviewComment); err != nil {
		t.Fatalf("postReview: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected a retry after the 422, got %d call(s)", n)
	}
	retry := (*calls)[1]
	if strings.Contains(ghArgs(retry), "comments[][") {
		t.Errorf("retry must carry no line anchors: %v", retry)
	}
	if body := reviewBodyArg(t, retry); !strings.Contains(body, "internal/x.go:9999") ||
		!strings.Contains(body, "line moved since the finding was filed") {
		t.Errorf("retry dropped the finding instead of the anchor:\n%s", body)
	}
}

func TestNumberFromURL(t *testing.T) {
	cases := map[string]int{
		"https://github.com/acme/widgets/pull/7":          7,
		"warning: 3 uncommitted\nhttps://x/y/pull/42":     42,
		"https://github.com/acme/widgets/pull/notanumber": 0,
		"": 0,
		"Creating pull request https://x/y/pull/11 for main": 0, // trailing words: not a URL tail
	}
	for in, want := range cases {
		if got := numberFromURL(in); got != want {
			t.Errorf("numberFromURL(%q) = %d, want %d", in, got, want)
		}
	}
}
