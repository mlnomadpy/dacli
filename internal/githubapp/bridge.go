// Package githubapp implements the untrusted GitHub-facing edge of dacli's
// optional App adapter. It deliberately has no workspace/store dependency:
// webhook text can become a proposal record, never a local command or write.
package githubapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidSignature    = errors.New("invalid GitHub webhook signature")
	ErrUnknownInstallation = errors.New("unknown GitHub App installation")
	ErrUnknownRepository   = errors.New("repository is not assigned to this installation")
	ErrInstallationRevoked = errors.New("GitHub App installation is not active")
	ErrDeliveryCollision   = errors.New("GitHub delivery id was reused with different content")
)

type InstallationState string

const (
	InstallationActive    InstallationState = "active"
	InstallationSuspended InstallationState = "suspended"
	InstallationRemoved   InstallationState = "removed"
)

// Permissions is the reconciled grant GitHub reports for one installation.
// Values are read/write/none; endpoints must check the relevant field before
// minting an installation token or attempting a remote write.
type Permissions struct {
	Metadata     string `json:"metadata"`
	Issues       string `json:"issues"`
	PullRequests string `json:"pull_requests"`
	Checks       string `json:"checks"`
}

type RepositoryBinding struct {
	ProjectID   string      `json:"project_id"`
	Owner       string      `json:"owner"`
	Name        string      `json:"name"`
	Permissions Permissions `json:"permissions"`
	Removed     bool        `json:"removed"`
}

type TenantBinding struct {
	TenantID     string                      `json:"tenant_id"`
	State        InstallationState           `json:"state"`
	Repositories map[int64]RepositoryBinding `json:"repositories"`
}

type Config struct {
	WebhookSecret []byte
	Tenants       map[int64]TenantBinding
}

type Bridge struct {
	mu      sync.Mutex
	root    string
	secret  []byte
	tenants map[int64]TenantBinding
}

func Open(root string, cfg Config) (*Bridge, error) {
	if len(cfg.WebhookSecret) < 16 {
		return nil, errors.New("webhook secret must contain at least 16 bytes")
	}
	for _, dir := range []string{"inbox", "proposals", "outbox"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			return nil, err
		}
	}
	b := &Bridge{root: root, secret: append([]byte(nil), cfg.WebhookSecret...), tenants: cloneTenants(cfg.Tenants)}
	if raw, err := os.ReadFile(filepath.Join(root, "bindings.json")); err == nil {
		var saved map[int64]TenantBinding
		if json.Unmarshal(raw, &saved) == nil {
			// Tenant/project assignment is local policy. Only hydrate remote state
			// for installations and repositories still present in local config.
			for id, local := range b.tenants {
				if old, ok := saved[id]; ok {
					local.State = old.State
					for repoID, repo := range local.Repositories {
						if prior, ok := old.Repositories[repoID]; ok {
							prior.ProjectID = repo.ProjectID
							local.Repositories[repoID] = prior
						}
					}
					b.tenants[id] = local
				}
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return b, nil
}

type IngestResult struct {
	New        bool
	ProposalID string
}

type webhookEnvelope struct {
	Action       string `json:"action"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Repository struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Sender struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	} `json:"sender"`
	Issue struct {
		ID     int64 `json:"id"`
		Number int   `json:"number"`
	} `json:"issue"`
	PullRequest struct {
		ID     int64 `json:"id"`
		Number int   `json:"number"`
	} `json:"pull_request"`
}

type inboxRecord struct {
	DeliveryID string          `json:"delivery_id"`
	Event      string          `json:"event"`
	ReceivedAt time.Time       `json:"received_at"`
	Payload    json.RawMessage `json:"payload"`
	ProposalID string          `json:"proposal_id"`
}

type Proposal struct {
	ID             string    `json:"id"`
	DeliveryID     string    `json:"delivery_id"`
	TenantID       string    `json:"tenant_id"`
	ProjectID      string    `json:"project_id"`
	InstallationID int64     `json:"installation_id"`
	RepositoryID   int64     `json:"repository_id"`
	Kind           string    `json:"kind"`
	ObjectKind     string    `json:"object_kind"`
	ObjectID       int64     `json:"object_id"`
	ObjectNumber   int       `json:"object_number"`
	Action         string    `json:"action"`
	ActorID        int64     `json:"actor_id"`
	ActorLogin     string    `json:"actor_login"`
	OccurredAt     time.Time `json:"occurred_at"`
}

const maxWebhookBody = 10 << 20

func (b *Bridge) Ingest(_ context.Context, r *http.Request) (IngestResult, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
	if err != nil {
		return IngestResult{}, err
	}
	if len(body) > maxWebhookBody {
		return IngestResult{}, errors.New("webhook body exceeds 10 MiB")
	}
	if !validSignature(b.secret, r.Header.Get("X-Hub-Signature-256"), body) {
		return IngestResult{}, ErrInvalidSignature
	}
	delivery := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
	event := strings.TrimSpace(r.Header.Get("X-GitHub-Event"))
	if delivery == "" || event == "" {
		return IngestResult{}, errors.New("missing GitHub delivery or event header")
	}
	var payload webhookEnvelope
	if err := json.Unmarshal(body, &payload); err != nil {
		return IngestResult{}, fmt.Errorf("parse verified webhook: %w", err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	binding, repo, err := b.route(payload.Installation.ID, payload.Repository.ID)
	if err != nil {
		return IngestResult{}, err
	}
	key := digest(delivery)
	inboxPath := filepath.Join(b.root, "inbox", key+".json")
	if raw, err := os.ReadFile(inboxPath); err == nil {
		var record inboxRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return IngestResult{}, fmt.Errorf("read inbox record: %w", err)
		}
		if record.Event != event || !hmac.Equal(record.Payload, body) {
			return IngestResult{}, ErrDeliveryCollision
		}
		// An inbox row is the transaction's durable intent. Redelivery repairs a
		// crash after that commit but before the idempotent proposal write.
		if _, err := os.Stat(filepath.Join(b.root, "proposals", record.ProposalID+".json")); errors.Is(err, os.ErrNotExist) {
			proposal, err := makeProposal(delivery, event, payload, binding, repo)
			if err != nil {
				return IngestResult{}, err
			}
			if err := createJSON(filepath.Join(b.root, "proposals", proposal.ID+".json"), proposal); err != nil && !errors.Is(err, os.ErrExist) {
				return IngestResult{}, err
			}
		} else if err != nil {
			return IngestResult{}, err
		}
		return IngestResult{ProposalID: record.ProposalID}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return IngestResult{}, err
	}

	proposal, err := makeProposal(delivery, event, payload, binding, repo)
	if err != nil {
		return IngestResult{}, err
	}
	record := inboxRecord{DeliveryID: delivery, Event: event, ReceivedAt: time.Now().UTC(), Payload: body, ProposalID: proposal.ID}
	// The inbox is committed before its effect. A crash between these writes is
	// repaired by RecoverPending, while O_EXCL makes concurrent redelivery one row.
	if err := createJSON(inboxPath, record); err != nil {
		if errors.Is(err, os.ErrExist) {
			return IngestResult{ProposalID: proposal.ID}, nil
		}
		return IngestResult{}, err
	}
	if err := createJSON(filepath.Join(b.root, "proposals", proposal.ID+".json"), proposal); err != nil && !errors.Is(err, os.ErrExist) {
		return IngestResult{}, err
	}
	return IngestResult{New: true, ProposalID: proposal.ID}, nil
}

func validSignature(secret []byte, header string, body []byte) bool {
	if !strings.HasPrefix(header, "sha256=") {
		return false
	}
	want, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil || len(want) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return hmac.Equal(want, mac.Sum(nil))
}

func (b *Bridge) route(installationID, repositoryID int64) (TenantBinding, RepositoryBinding, error) {
	tenant, ok := b.tenants[installationID]
	if !ok {
		return TenantBinding{}, RepositoryBinding{}, ErrUnknownInstallation
	}
	if tenant.State != InstallationActive {
		return TenantBinding{}, RepositoryBinding{}, ErrInstallationRevoked
	}
	repo, ok := tenant.Repositories[repositoryID]
	if !ok || repo.Removed {
		return TenantBinding{}, RepositoryBinding{}, ErrUnknownRepository
	}
	return tenant, repo, nil
}

func makeProposal(delivery, event string, p webhookEnvelope, tenant TenantBinding, repo RepositoryBinding) (Proposal, error) {
	proposal := Proposal{ID: digest("proposal:" + delivery), DeliveryID: delivery, TenantID: tenant.TenantID, ProjectID: repo.ProjectID, InstallationID: p.Installation.ID, RepositoryID: p.Repository.ID, Action: p.Action, ActorID: p.Sender.ID, ActorLogin: p.Sender.Login, OccurredAt: time.Now().UTC()}
	switch event {
	case "issues":
		proposal.ObjectKind, proposal.ObjectID, proposal.ObjectNumber = "issue", p.Issue.ID, p.Issue.Number
		if p.Action == "closed" || p.Action == "reopened" {
			proposal.Kind = "propose-status"
		} else {
			proposal.Kind = "comment"
		}
	case "issue_comment":
		proposal.Kind, proposal.ObjectKind, proposal.ObjectID, proposal.ObjectNumber = "comment", "issue", p.Issue.ID, p.Issue.Number
	case "pull_request":
		proposal.Kind, proposal.ObjectKind, proposal.ObjectID, proposal.ObjectNumber = "pull-request", "pull_request", p.PullRequest.ID, p.PullRequest.Number
	default:
		return Proposal{}, fmt.Errorf("unsupported GitHub event %q", event)
	}
	return proposal, nil
}

func (b *Bridge) Proposals() ([]Proposal, error) {
	entries, err := os.ReadDir(filepath.Join(b.root, "proposals"))
	if err != nil {
		return nil, err
	}
	result := make([]Proposal, 0, len(entries))
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(b.root, "proposals", entry.Name()))
		if err != nil {
			return nil, err
		}
		var p Proposal
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

type StatusUpdate struct {
	TenantID       string `json:"tenant_id"`
	ProjectID      string `json:"project_id"`
	InstallationID int64  `json:"installation_id"`
	RepositoryID   int64  `json:"repository_id"`
	ObjectID       string `json:"object_id"`
	HeadSHA        string `json:"head_sha"`
	State          string `json:"state"`
	IdempotencyKey string `json:"idempotency_key"`
}

type OutboxRecord struct {
	ID        string       `json:"id"`
	CreatedAt time.Time    `json:"created_at"`
	Status    StatusUpdate `json:"status"`
}

// CheckPoster is implemented by the server-side GitHub client. UpsertCheck
// must use Status.IdempotencyKey as its remote marker; a local client never
// receives the installation token used inside this call.
type CheckPoster interface {
	UpsertCheck(context.Context, OutboxRecord, RepositoryBinding) error
}

// Dispatch posts pending records and writes a separate acknowledgement. The
// outbox row remains immutable, so a crash or redelivery can safely retry it.
func (b *Bridge) Dispatch(ctx context.Context, poster CheckPoster) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	entries, err := os.ReadDir(filepath.Join(b.root, "outbox"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".ack.json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(b.root, "outbox", entry.Name()))
		if err != nil {
			return err
		}
		var record OutboxRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return err
		}
		ack := filepath.Join(b.root, "outbox", record.ID+".ack.json")
		if _, err := os.Stat(ack); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		tenant, repo, err := b.route(record.Status.InstallationID, record.Status.RepositoryID)
		if err != nil {
			return err
		}
		if record.Status.TenantID != tenant.TenantID || record.Status.ProjectID != repo.ProjectID {
			return errors.New("outbox route no longer matches installation binding")
		}
		if repo.Permissions.Checks != "write" {
			return errors.New("installation no longer grants checks:write")
		}
		if err := poster.UpsertCheck(ctx, record, repo); err != nil {
			return err
		}
		if err := createJSON(ack, struct {
			ID     string    `json:"id"`
			SentAt time.Time `json:"sent_at"`
		}{ID: record.ID, SentAt: time.Now().UTC()}); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	return nil
}

func (b *Bridge) EnqueueStatus(update StatusUpdate) (OutboxRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	tenant, repo, err := b.route(update.InstallationID, update.RepositoryID)
	if err != nil {
		return OutboxRecord{}, err
	}
	if update.TenantID != tenant.TenantID || update.ProjectID != repo.ProjectID {
		return OutboxRecord{}, errors.New("status route does not match installation binding")
	}
	if update.IdempotencyKey == "" || update.ObjectID == "" || !validCommitSHA(update.HeadSHA) || !validCheckState(update.State) {
		return OutboxRecord{}, errors.New("status requires idempotency key, object, a commit SHA, and queued/in_progress/completed state")
	}
	record := OutboxRecord{ID: digest("outbox:" + update.TenantID + ":" + update.IdempotencyKey), CreatedAt: time.Now().UTC(), Status: update}
	path := filepath.Join(b.root, "outbox", record.ID+".json")
	if raw, err := os.ReadFile(path); err == nil {
		var old OutboxRecord
		if err := json.Unmarshal(raw, &old); err != nil {
			return OutboxRecord{}, err
		}
		return old, nil
	}
	if err := createJSON(path, record); err != nil && !errors.Is(err, os.ErrExist) {
		return OutboxRecord{}, err
	}
	return record, nil
}

type RepositorySnapshot struct {
	ID          int64
	Owner, Name string
	Permissions Permissions
}
type InstallationSnapshot struct {
	InstallationID int64
	State          InstallationState
	Repositories   []RepositorySnapshot
}
type Reconciler interface {
	Installation(context.Context, int64) (InstallationSnapshot, error)
}

func (b *Bridge) Reconcile(ctx context.Context, remote Reconciler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for installationID, tenant := range b.tenants {
		snapshot, err := remote.Installation(ctx, installationID)
		if err != nil {
			return err
		}
		if snapshot.InstallationID != installationID {
			return errors.New("reconciler returned a different installation")
		}
		tenant.State = snapshot.State
		seen := make(map[int64]bool)
		for _, current := range snapshot.Repositories {
			local, ok := tenant.Repositories[current.ID]
			if !ok {
				continue
			} // Remote discovery cannot expand local tenant policy.
			local.Owner, local.Name, local.Permissions, local.Removed = current.Owner, current.Name, current.Permissions, false
			tenant.Repositories[current.ID] = local
			seen[current.ID] = true
		}
		for id, local := range tenant.Repositories {
			if !seen[id] {
				local.Removed = true
				tenant.Repositories[id] = local
			}
		}
		b.tenants[installationID] = tenant
	}
	return writeJSONAtomic(filepath.Join(b.root, "bindings.json"), b.tenants)
}

func (b *Bridge) SetInstallationState(id int64, state InstallationState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	t := b.tenants[id]
	t.State = state
	b.tenants[id] = t
}

func (b *Bridge) Binding(id int64) TenantBinding {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneTenant(b.tenants[id])
}

func createJSON(path string, value any) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".record-*")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	encErr := json.NewEncoder(f).Encode(value)
	if encErr == nil {
		encErr = f.Sync()
	}
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	if closeErr != nil {
		return closeErr
	}
	// Link publishes a fully written inode without replacing an existing key.
	// That gives inbox/outbox inserts atomic create-if-absent semantics.
	return os.Link(tmpName, path)
}

func writeJSONAtomic(path string, value any) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bindings-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := json.NewEncoder(tmp).Encode(value); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func digest(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }

func validCommitSHA(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validCheckState(value string) bool {
	return value == "queued" || value == "in_progress" || value == "completed"
}

func cloneTenants(in map[int64]TenantBinding) map[int64]TenantBinding {
	out := make(map[int64]TenantBinding, len(in))
	for id, tenant := range in {
		out[id] = cloneTenant(tenant)
	}
	return out
}

func cloneTenant(in TenantBinding) TenantBinding {
	out := in
	out.Repositories = make(map[int64]RepositoryBinding, len(in.Repositories))
	for id, repo := range in.Repositories {
		out.Repositories[id] = repo
	}
	return out
}
