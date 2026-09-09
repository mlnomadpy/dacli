// Package envelopeworker implements the tenant-scoped durable boundary for
// signed control-plane envelopes. It verifies authority before persistence and
// never interprets accepted payloads into local workspace mutations.
package envelopeworker

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/internal/cloudsync"
	"github.com/mlnomadpy/dacli/internal/ulid"
)

const maxEnvelopePayloadBytes = 1 << 20

var (
	ErrDenied      = errors.New("control-plane envelope denied")
	ErrUnavailable = errors.New("control-plane envelope unavailable")
)

type Request struct {
	Credential string             `json:"-"`
	Envelope   cloudsync.Envelope `json:"envelope"`
}

type Authenticator interface {
	VerifyWork(context.Context, string) (tenant.VerifiedIdentity, error)
}

// KeySource must resolve keys inside the verified tenant scope. Implementations
// must not use Envelope.TenantID as authority.
type KeySource interface {
	TrustedKeys(context.Context, tenant.Scope, tenant.ProjectID) (map[string]ed25519.PublicKey, error)
}

type Audit struct {
	CorrelationID string
	Identity      tenant.VerifiedIdentity
	Project       tenant.ProjectID
	EventIdentity string
	Decision      cloudsync.Outcome
	Reason        string
	OccurredAt    time.Time
}

type InboxStore interface {
	Accept(context.Context, tenant.VerifiedIdentity, cloudsync.Envelope, Audit) (cloudsync.Outcome, error)
	RecordDecision(context.Context, Audit) error
}

type Service struct {
	auth  Authenticator
	keys  KeySource
	store InboxStore
	now   func() time.Time
	id    func() string
}

func NewService(auth Authenticator, keys KeySource, store InboxStore) (*Service, error) {
	if auth == nil || keys == nil || store == nil {
		return nil, errors.New("envelope worker requires authenticator, key source, and store")
	}
	return &Service{auth: auth, keys: keys, store: store, now: time.Now, id: ulid.New}, nil
}

// Receive applies the protocol order after the outer request-size boundary:
// authenticate, route, verify signature, negotiate schema, validate the closed
// payload, then atomically check duplicate/replay/sequence and persist.
func (s *Service) Receive(ctx context.Context, request Request) (cloudsync.Outcome, error) {
	if s == nil || s.auth == nil || s.keys == nil || s.store == nil {
		return "", ErrUnavailable
	}
	identity, err := s.auth.VerifyWork(ctx, request.Credential)
	if err != nil || tenant.ValidateVerifiedIdentity(identity) != nil {
		return "", ErrDenied
	}
	envelope := request.Envelope
	project, routeOK := route(identity, envelope)
	if !routeOK || len(envelope.Payload) == 0 || len(envelope.Payload) > maxEnvelopePayloadBytes {
		return s.refuse(ctx, identity, project, envelope.EventID, cloudsync.OutcomeTampered, "route_or_size")
	}
	keys, err := s.keys.TrustedKeys(ctx, identity.Scope, project)
	if err != nil {
		return "", fmt.Errorf("%w: resolve signing keys", ErrUnavailable)
	}
	if !cloudsync.Verify(envelope, keys) {
		return s.refuse(ctx, identity, project, envelope.EventID, cloudsync.OutcomeTampered, "signature")
	}
	if envelope.SchemaVersion != 1 {
		return s.refuse(ctx, identity, project, envelope.EventID, cloudsync.OutcomeIncompatible, "schema")
	}
	if err := cloudsync.ValidatePayload(envelope.EventType, envelope.Payload); err != nil {
		return s.refuse(ctx, identity, project, envelope.EventID, cloudsync.OutcomeTampered, "payload")
	}
	audit := s.audit(identity, project, envelope.EventID, "", "accepted")
	outcome, err := s.store.Accept(ctx, identity, envelope, audit)
	if err != nil {
		return "", fmt.Errorf("%w: persist inbox transaction", ErrUnavailable)
	}
	if outcome == cloudsync.OutcomeReplay || outcome == cloudsync.OutcomeIncompatible || outcome == cloudsync.OutcomeTampered {
		return outcome, ErrDenied
	}
	return outcome, nil
}

func (s *Service) refuse(ctx context.Context, identity tenant.VerifiedIdentity, project tenant.ProjectID, eventID string, decision cloudsync.Outcome, reason string) (cloudsync.Outcome, error) {
	audit := s.audit(identity, project, eventID, decision, reason)
	if err := s.store.RecordDecision(ctx, audit); err != nil {
		return "", fmt.Errorf("%w: persist refusal audit", ErrUnavailable)
	}
	return decision, ErrDenied
}

func (s *Service) audit(identity tenant.VerifiedIdentity, project tenant.ProjectID, eventID string, decision cloudsync.Outcome, reason string) Audit {
	digest := sha256.Sum256([]byte(eventID))
	return Audit{CorrelationID: s.id(), Identity: identity, Project: project, EventIdentity: hex.EncodeToString(digest[:]), Decision: decision, Reason: reason, OccurredAt: s.now().UTC()}
}

func route(identity tenant.VerifiedIdentity, envelope cloudsync.Envelope) (tenant.ProjectID, bool) {
	project, err := tenant.NewProjectID(envelope.ProjectID)
	if err != nil {
		project = "invalid"
	}
	if err != nil || envelope.TenantID != string(identity.Scope.Organization) || !bounded(envelope.EventID, 128) || !bounded(envelope.IdempotencyKey, 256) || !bounded(envelope.ProducerVersion, 128) || !bounded(envelope.Integrity.KeyID, 128) || envelope.ProducerSequence == 0 || envelope.OccurredAt.IsZero() {
		return project, false
	}
	return project, true
}

func bounded(value string, maximum int) bool { return len(value) > 0 && len(value) <= maximum }

// MarshalJSON is intentionally explicit so a future field addition cannot
// accidentally expose the credential at a queue or diagnostic boundary.
func (r Request) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Envelope cloudsync.Envelope `json:"envelope"`
	}{Envelope: r.Envelope})
}
