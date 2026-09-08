package ghmirror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func TestRenderPullPlanJSONClassifiesAndPagesWithoutLeakingAcceptance(t *testing.T) {
	match := &store.Task{ID: "t-match"}
	plan := []pullPlanItem{
		{issue: ghIssue{Number: 1, Title: "create"}, outcome: pullCreate, acceptance: acceptanceExtraction{Criteria: []string{"secret detail"}}},
		{issue: ghIssue{Number: 2, Title: "link"}, outcome: pullExactMatch, match: match},
		{issue: ghIssue{Number: 3, Title: "mapped"}, outcome: pullAlreadyMapped, match: match},
		{issue: ghIssue{Number: 4, Title: "duplicate"}, outcome: pullPossibleDuplicate, match: match, reason: "similar"},
		{issue: ghIssue{Number: 5, Title: "marker"}, outcome: pullRefused, reason: "dacli marker"},
	}

	first := renderPullPlanJSON("core", "owner/repo", plan, 2, 0, false)
	if first.Schema != "github-adoption-plan/v1" || first.Total != 5 || !first.Truncated || first.NextCursor != 2 {
		t.Fatalf("first page metadata = %+v", first)
	}
	if first.Counts["create"] != 1 || first.Counts["link"] != 1 || first.Counts["already_mapped"] != 1 || first.Counts["refused"] != 1 || first.Counts["skipped"] != 1 {
		t.Fatalf("classification counts = %+v", first.Counts)
	}
	if len(first.Create) != 1 || len(first.Link) != 1 || first.Create[0].Criteria != nil {
		t.Fatalf("first page leaked or misclassified details: %+v", first)
	}

	last := renderPullPlanJSON("core", "owner/repo", plan, 2, first.NextCursor, true)
	if len(last.AlreadyMapped) != 1 || len(last.Refused) != 1 || !last.Truncated || last.NextCursor != 4 {
		t.Fatalf("second page = %+v", last)
	}
	acceptance := renderPullPlanJSON("core", "owner/repo", plan, 1, 0, true)
	if got := acceptance.Create[0].Criteria; len(got) != 1 || got[0] != "secret detail" {
		t.Fatalf("explicit acceptance projection = %v", got)
	}
}

func TestPullDryRunJSONIsTypedBoundedAndReadOnly(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	before, err := snapshotTree(w.Root)
	if err != nil {
		t.Fatal(err)
	}
	stubPullIssues(t, `[
		{"number":1,"title":"one","body":"## Acceptance\n- [ ] first","state":"open"},
		{"number":2,"title":"two","body":"","state":"open"},
		{"number":3,"title":"three","body":"","state":"open"}
	]`)
	var out bytes.Buffer
	ctx := &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: w.Root, JSON: true}
	if err := cmdPull(ctx, []string{"core", "--dry-run", "--limit", "1"}); err != nil {
		t.Fatal(err)
	}
	var got pullPlanJSON
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.Total != 3 || got.Limit != 1 || len(got.Create) != 1 || !got.Truncated || got.NextCursor != 1 {
		t.Fatalf("bounded plan = %+v", got)
	}
	if strings.Contains(out.String(), "first") {
		t.Fatalf("acceptance leaked without --include-acceptance: %s", out.String())
	}
	after, err := snapshotTree(w.Root)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("JSON adoption preview changed local state")
	}
}

func TestPullDryRunJSONNoOpAndTransientFailure(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	stubPullIssues(t, `[]`)
	var out bytes.Buffer
	ctx := &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: w.Root, JSON: true}
	if err := cmdPull(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	var empty pullPlanJSON
	if err := json.Unmarshal(out.Bytes(), &empty); err != nil || empty.Total != 0 || empty.Create == nil || empty.Refused == nil {
		t.Fatalf("no-op plan must retain typed empty arrays: err=%v plan=%+v", err, empty)
	}

	orig := gh
	gh = func(_ *workspace.Workspace, _ ...string) (string, error) {
		return "", errors.New("temporary GitHub outage")
	}
	t.Cleanup(func() { gh = orig })
	out.Reset()
	err := cmdPull(ctx, []string{"core", "--dry-run"})
	if err == nil || !strings.Contains(err.Error(), "temporary GitHub outage") {
		t.Fatalf("transient error = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("failed JSON plan emitted partial prose: %q", out.String())
	}
}

func TestPullDryRunHumanOutputIsBoundedByDefault(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	stubPullIssues(t, issuesJSON(105))
	ctx, out := releaseCtx(t, w)
	if err := cmdPull(ctx, []string{"core", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	rows := 0
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "issue #") {
			rows++
		}
	}
	if got := rows; got != 100 {
		t.Fatalf("default human page printed %d issue rows, want 100", got)
	}
	if !strings.Contains(out.String(), "continue with --cursor 100") {
		t.Fatalf("bounded output omitted continuation cursor:\n%s", out.String())
	}
}

func TestSyncDryRunJSONComposesBoundedPlansWithoutWrites(t *testing.T) {
	w := mirrorWorkspace(t)
	linkRepo(t, w, "core", "owner/repo")
	if _, err := store.CreateTask(w, "a-root", "core", "local outbound", store.TaskOpts{}); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotTree(w.Root)
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	orig := gh
	t.Cleanup(func() { gh = orig })
	gh = func(_ *workspace.Workspace, args ...string) (string, error) {
		calls = append(calls, args)
		switch {
		case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
			return `{"nameWithOwner":"owner/repo","visibility":"PRIVATE"}`, nil
		case len(args) >= 2 && args[0] == "issue" && args[1] == "list":
			return `[{"number":42,"title":"human inbound","body":"","state":"open"}]`, nil
		default:
			return "", fmt.Errorf("unexpected gh call: %v", args)
		}
	}
	var out bytes.Buffer
	ctx := &clikit.Ctx{Stdout: &out, Stderr: &out, Cwd: w.Root, JSON: true}
	if err := cmdSync(ctx, []string{"core", "--dry-run", "--limit", "1"}); err != nil {
		t.Fatalf("sync JSON: %v", err)
	}
	var got struct {
		Schema   string       `json:"schema"`
		Inbound  pullPlanJSON `json:"inbound"`
		Outbound struct {
			Schema string `json:"schema"`
		} `json:"outbound"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid sync JSON: %v\n%s", err, out.String())
	}
	if got.Schema != "github-sync-plan/v1" || got.Inbound.Total != 1 || got.Outbound.Schema != "github-push-preview/v1" {
		t.Fatalf("sync envelope = %+v", got)
	}
	assertNoWrites(t, calls)
	after, err := snapshotTree(w.Root)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("sync JSON preview changed local state")
	}
}
