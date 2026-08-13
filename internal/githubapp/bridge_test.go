package githubapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func signedRequest(t *testing.T, secret []byte, delivery string, body []byte) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	r.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	r.Header.Set("X-GitHub-Delivery", delivery)
	r.Header.Set("X-GitHub-Event", "issues")
	return r
}

func issuePayload(installation, repository int64, action string) []byte {
	b, _ := json.Marshal(map[string]any{
		"action":       action,
		"installation": map[string]any{"id": installation},
		"repository":   map[string]any{"id": repository, "name": "repo", "owner": map[string]any{"login": "acme"}},
		"sender":       map[string]any{"id": 9, "login": "human"},
		"issue":        map[string]any{"id": 33, "number": 7, "title": "remote text is data"},
	})
	return b
}

func testBridge(t *testing.T) *Bridge {
	t.Helper()
	b, err := Open(filepath.Join(t.TempDir(), "bridge"), Config{
		WebhookSecret: []byte("correct horse battery staple"),
		Tenants: map[int64]TenantBinding{41: {
			TenantID: "tenant-a", State: InstallationActive,
			Repositories: map[int64]RepositoryBinding{51: {ProjectID: "project-a", Owner: "acme", Name: "repo", Permissions: Permissions{Metadata: "read", Issues: "read", PullRequests: "read", Checks: "write"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestWebhookVerifiesRawBodyBeforeInboxOrParsing(t *testing.T) {
	b := testBridge(t)
	body := issuePayload(41, 51, "closed")
	r := signedRequest(t, []byte("wrong"), "delivery-1", body)
	result, err := b.Ingest(context.Background(), r)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("error = %v, want ErrInvalidSignature", err)
	}
	if result != (IngestResult{}) {
		t.Fatalf("result = %+v, want zero result", result)
	}
	entries, err := os.ReadDir(filepath.Join(b.root, "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("forged delivery reached inbox: %v", entries)
	}
}

func TestDeliveryIsIdempotentAndOnlyCreatesProposal(t *testing.T) {
	b := testBridge(t)
	body := issuePayload(41, 51, "closed")
	first, err := b.Ingest(context.Background(), signedRequest(t, b.secret, "delivery-2", body))
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.Ingest(context.Background(), signedRequest(t, b.secret, "delivery-2", body))
	if err != nil {
		t.Fatal(err)
	}
	if !first.New || second.New || first.ProposalID == "" || second.ProposalID != first.ProposalID {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	proposals, err := b.Proposals()
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 || proposals[0].Kind != "propose-status" || proposals[0].TenantID != "tenant-a" {
		t.Fatalf("proposals = %+v", proposals)
	}
}

func TestRedeliveryRejectsCollisionAndRepairsPendingEffect(t *testing.T) {
	b := testBridge(t)
	body := issuePayload(41, 51, "closed")
	first, err := b.Ingest(context.Background(), signedRequest(t, b.secret, "delivery-retry", body))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(b.root, "proposals", first.ProposalID+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Ingest(context.Background(), signedRequest(t, b.secret, "delivery-retry", body)); err != nil {
		t.Fatalf("redelivery did not repair pending effect: %v", err)
	}
	changed := issuePayload(41, 51, "reopened")
	if _, err := b.Ingest(context.Background(), signedRequest(t, b.secret, "delivery-retry", changed)); !errors.Is(err, ErrDeliveryCollision) {
		t.Fatalf("collision error = %v, want ErrDeliveryCollision", err)
	}
}

func TestTenantMappingReplayAndRevocation(t *testing.T) {
	b := testBridge(t)
	for _, tc := range []struct {
		name string
		body []byte
		want error
	}{
		{"wrong installation", issuePayload(99, 51, "closed"), ErrUnknownInstallation},
		{"confused deputy repository", issuePayload(41, 99, "closed"), ErrUnknownRepository},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := b.Ingest(context.Background(), signedRequest(t, b.secret, "delivery-"+tc.name, tc.body))
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
	b.SetInstallationState(41, InstallationSuspended)
	_, err := b.Ingest(context.Background(), signedRequest(t, b.secret, "delivery-revoked", issuePayload(41, 51, "closed")))
	if !errors.Is(err, ErrInstallationRevoked) {
		t.Fatalf("error = %v, want ErrInstallationRevoked", err)
	}
}

func TestOutboxIsIdempotentClosedMetadataOnly(t *testing.T) {
	b := testBridge(t)
	u := StatusUpdate{TenantID: "tenant-a", ProjectID: "project-a", InstallationID: 41, RepositoryID: 51, ObjectID: "task-1", HeadSHA: "0123456789abcdef0123456789abcdef01234567", State: "in_progress", IdempotencyKey: "task-1:running:1"}
	first, err := b.EnqueueStatus(u)
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.EnqueueStatus(u)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("duplicate created two records: %q != %q", first.ID, second.ID)
	}
	raw, err := os.ReadFile(filepath.Join(b.root, "outbox", first.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte("source"), []byte("prompt"), []byte("transcript"), []byte("environment"), []byte("secret"), []byte("command_output")} {
		if bytes.Contains(raw, forbidden) {
			t.Fatalf("outbox contains forbidden class %q: %s", forbidden, raw)
		}
	}
}

func TestDispatchRechecksInstallationAndPermission(t *testing.T) {
	b := testBridge(t)
	u := StatusUpdate{TenantID: "tenant-a", ProjectID: "project-a", InstallationID: 41, RepositoryID: 51, ObjectID: "task-1", HeadSHA: "0123456789abcdef0123456789abcdef01234567", State: "in_progress", IdempotencyKey: "dispatch-1"}
	if _, err := b.EnqueueStatus(u); err != nil {
		t.Fatal(err)
	}
	poster := &countingPoster{}
	if err := b.Dispatch(context.Background(), poster); err != nil {
		t.Fatal(err)
	}
	if err := b.Dispatch(context.Background(), poster); err != nil {
		t.Fatal(err)
	}
	if poster.calls != 1 {
		t.Fatalf("poster calls = %d, want 1", poster.calls)
	}

	b2 := testBridge(t)
	if _, err := b2.EnqueueStatus(StatusUpdate{TenantID: "tenant-a", ProjectID: "project-a", InstallationID: 41, RepositoryID: 51, ObjectID: "task-2", HeadSHA: "0123456789abcdef0123456789abcdef01234567", State: "in_progress", IdempotencyKey: "dispatch-2"}); err != nil {
		t.Fatal(err)
	}
	b2.SetInstallationState(41, InstallationRemoved)
	if err := b2.Dispatch(context.Background(), poster); !errors.Is(err, ErrInstallationRevoked) {
		t.Fatalf("revoked dispatch error = %v", err)
	}
}

type countingPoster struct{ calls int }

func (p *countingPoster) UpsertCheck(context.Context, OutboxRecord, RepositoryBinding) error {
	p.calls++
	return nil
}

func TestReconcileConvergesRenameRemovalAndPermissions(t *testing.T) {
	b := testBridge(t)
	fetch := staticReconciler{snapshot: InstallationSnapshot{
		InstallationID: 41, State: InstallationActive,
		Repositories: []RepositorySnapshot{{ID: 51, Owner: "acme", Name: "renamed", Permissions: Permissions{Metadata: "read", Issues: "write", PullRequests: "write", Checks: "write"}}},
	}}
	if err := b.Reconcile(context.Background(), fetch); err != nil {
		t.Fatal(err)
	}
	got := b.Binding(41)
	if got.Repositories[51].Name != "renamed" || got.Repositories[51].Permissions.Checks != "write" {
		t.Fatalf("binding did not converge: %+v", got)
	}
	fetch.snapshot.Repositories = nil
	if err := b.Reconcile(context.Background(), fetch); err != nil {
		t.Fatal(err)
	}
	if !b.Binding(41).Repositories[51].Removed {
		t.Fatal("removed repository remained active")
	}
}

type staticReconciler struct{ snapshot InstallationSnapshot }

func (s staticReconciler) Installation(context.Context, int64) (InstallationSnapshot, error) {
	return s.snapshot, nil
}
