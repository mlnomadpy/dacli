package envelopeworker

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/ratelimit"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/internal/cloudsync"
)

type testAuth struct {
	identity tenant.VerifiedIdentity
	err      error
	order    *[]string
}

func (a testAuth) VerifyWork(context.Context, string) (tenant.VerifiedIdentity, error) {
	if a.order != nil {
		*a.order = append(*a.order, "authenticate")
	}
	return a.identity, a.err
}

type testKeys struct {
	keys  map[string]ed25519.PublicKey
	err   error
	order *[]string
}

func (k testKeys) TrustedKeys(context.Context, tenant.Scope, tenant.ProjectID) (map[string]ed25519.PublicKey, error) {
	if k.order != nil {
		*k.order = append(*k.order, "keys")
	}
	return k.keys, k.err
}

type testStore struct {
	outcome cloudsync.Outcome
	err     error
	accepts int
	audits  []Audit
	order   *[]string
}

func (s *testStore) Accept(_ context.Context, _ tenant.VerifiedIdentity, _ cloudsync.Envelope, audit Audit) (cloudsync.Outcome, error) {
	s.accepts++
	s.audits = append(s.audits, audit)
	if s.order != nil {
		*s.order = append(*s.order, "store")
	}
	return s.outcome, s.err
}

func (s *testStore) RecordDecision(_ context.Context, audit Audit) error {
	s.audits = append(s.audits, audit)
	if s.order != nil {
		*s.order = append(*s.order, "audit")
	}
	return s.err
}

func TestReceiveAuthenticatesRoutesAndVerifiesBeforePersistence(t *testing.T) {
	identity, public, private := testIdentityAndKeys(t)
	order := []string{}
	store := &testStore{outcome: cloudsync.OutcomeAccepted, order: &order}
	service, err := NewService(testAuth{identity: identity, order: &order}, testKeys{keys: map[string]ed25519.PublicKey{"key-a": public}, order: &order}, store)
	if err != nil {
		t.Fatal(err)
	}
	service.id = func() string { return "operation-a" }
	service.now = func() time.Time { return time.Unix(10, 0) }
	outcome, err := service.Receive(context.Background(), Request{Credential: "secret", Envelope: testEnvelope(t, private)})
	if err != nil || outcome != cloudsync.OutcomeAccepted || strings.Join(order, ",") != "authenticate,keys,store" || store.accepts != 1 {
		t.Fatalf("outcome=%q err=%v order=%v accepts=%d", outcome, err, order, store.accepts)
	}
	if len(store.audits) != 1 || store.audits[0].EventIdentity == "event-a" || len(store.audits[0].EventIdentity) != 64 {
		t.Fatalf("audit leaked or lost event identity: %+v", store.audits)
	}
}

func TestReceiveRateLimitsOnlyAfterVerifiedIdentity(t *testing.T) {
	identity, public, private := testIdentityAndKeys(t)
	limiter, _ := ratelimit.New(ratelimit.Policy{Capacity: 1, RefillInterval: time.Hour, MaxKeys: 4, IdleTTL: 2 * time.Hour}, nil)
	store := &testStore{outcome: cloudsync.OutcomeAccepted}
	service, err := NewLimitedService(testAuth{identity: identity}, testKeys{keys: map[string]ed25519.PublicKey{"key-a": public}}, store, limiter, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Credential: "secret", Envelope: testEnvelope(t, private)}
	if _, err := service.Receive(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Receive(context.Background(), request); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("second ingestion = %v", err)
	}
	if store.accepts != 1 {
		t.Fatalf("rate-limited ingestion reached persistence %d times", store.accepts)
	}
	if _, err := NewLimitedService(testAuth{}, testKeys{}, store, limiter, []byte("short")); err == nil {
		t.Fatal("short limiter key was accepted")
	}
}

func TestReceiveFailsClosedAtEachBoundary(t *testing.T) {
	identity, public, private := testIdentityAndKeys(t)
	for name, mutate := range map[string]func(*Request, *testAuth, *testKeys){
		"authentication": func(_ *Request, auth *testAuth, _ *testKeys) { auth.err = errors.New("revoked") },
		"claimed tenant": func(request *Request, _ *testAuth, _ *testKeys) { request.Envelope.TenantID = "tenant-b" },
		"signature":      func(request *Request, _ *testAuth, _ *testKeys) { request.Envelope.Payload[1] ^= 1 },
		"schema": func(request *Request, _ *testAuth, _ *testKeys) {
			request.Envelope.SchemaVersion = 2
			_ = cloudsync.Sign(&request.Envelope, private)
		},
		"payload": func(request *Request, _ *testAuth, _ *testKeys) {
			request.Envelope.Payload = json.RawMessage(`{"run_id":"secret","unexpected":"value"}`)
			_ = cloudsync.Sign(&request.Envelope, private)
		},
	} {
		t.Run(name, func(t *testing.T) {
			request := Request{Credential: "secret", Envelope: testEnvelope(t, private)}
			auth := testAuth{identity: identity}
			keys := testKeys{keys: map[string]ed25519.PublicKey{"key-a": public}}
			mutate(&request, &auth, &keys)
			store := &testStore{}
			service, _ := NewService(auth, keys, store)
			outcome, err := service.Receive(context.Background(), request)
			if !errors.Is(err, ErrDenied) || store.accepts != 0 {
				t.Fatalf("outcome=%q err=%v accepts=%d", outcome, err, store.accepts)
			}
			if name == "authentication" {
				if len(store.audits) != 0 {
					t.Fatal("unverified identity reached tenant audit")
				}
			} else if len(store.audits) != 1 || outcome == "" {
				t.Fatalf("authenticated refusal was not audited: outcome=%q audits=%+v", outcome, store.audits)
			}
		})
	}
}

func TestReceiveGoldenOutcomes(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "contracts", "controlplane", "v1", "testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	identity, public, private := testIdentityAndKeys(t)
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Outcome string `json:"outcome"`
				Event   struct {
					EventID          string `json:"event_id"`
					IdempotencyKey   string `json:"idempotency_key"`
					ProducerSequence uint64 `json:"producer_sequence"`
					SchemaVersion    int    `json:"schema_version"`
					SignatureValid   bool   `json:"signature_valid"`
				} `json:"event"`
			}
			if err := json.Unmarshal(raw, &fixture); err != nil {
				t.Fatal(err)
			}
			envelope := testEnvelope(t, private)
			envelope.EventID, envelope.IdempotencyKey, envelope.ProducerSequence, envelope.SchemaVersion = fixture.Event.EventID, fixture.Event.IdempotencyKey, fixture.Event.ProducerSequence, fixture.Event.SchemaVersion
			_ = cloudsync.Sign(&envelope, private)
			if !fixture.Event.SignatureValid {
				envelope.Payload[1] ^= 1
			}
			storeOutcome := cloudsync.Outcome(fixture.Outcome)
			store := &testStore{outcome: storeOutcome}
			service, _ := NewService(testAuth{identity: identity}, testKeys{keys: map[string]ed25519.PublicKey{"key-a": public}}, store)
			outcome, err := service.Receive(context.Background(), Request{Credential: "secret", Envelope: envelope})
			if string(outcome) != fixture.Outcome {
				t.Fatalf("outcome=%q err=%v want=%q", outcome, err, fixture.Outcome)
			}
			refused := outcome == cloudsync.OutcomeTampered || outcome == cloudsync.OutcomeIncompatible || outcome == cloudsync.OutcomeReplay
			if refused != errors.Is(err, ErrDenied) {
				t.Fatalf("outcome=%q err=%v refused=%v", outcome, err, refused)
			}
		})
	}
}

func TestRequestJSONNeverContainsCredential(t *testing.T) {
	raw, err := json.Marshal(Request{Credential: "bearer-secret", Envelope: cloudsync.Envelope{EventID: "event-a"}})
	if err != nil || strings.Contains(string(raw), "bearer-secret") || strings.Contains(string(raw), "Credential") {
		t.Fatalf("request JSON leaked credential: %s, %v", raw, err)
	}
	if _, err := NewService(nil, testKeys{}, &testStore{}); err == nil {
		t.Fatal("nil authenticator accepted")
	}
}

func TestReceiveDoesNotClaimSuccessWhenInfrastructureCannotCommit(t *testing.T) {
	identity, public, private := testIdentityAndKeys(t)
	request := Request{Credential: "secret", Envelope: testEnvelope(t, private)}
	for name, tc := range map[string]struct {
		keys  testKeys
		store *testStore
	}{
		"keys":  {keys: testKeys{err: errors.New("key store offline")}, store: &testStore{}},
		"inbox": {keys: testKeys{keys: map[string]ed25519.PublicKey{"key-a": public}}, store: &testStore{err: errors.New("database offline")}},
	} {
		t.Run(name, func(t *testing.T) {
			service, _ := NewService(testAuth{identity: identity}, tc.keys, tc.store)
			if outcome, err := service.Receive(context.Background(), request); outcome != "" || !errors.Is(err, ErrUnavailable) {
				t.Fatalf("outcome=%q err=%v", outcome, err)
			}
		})
	}
	service, _ := NewService(testAuth{identity: identity}, testKeys{keys: map[string]ed25519.PublicKey{"key-a": public}}, &testStore{err: errors.New("audit offline")})
	tampered := request
	tampered.Envelope.Payload[1] ^= 1
	if outcome, err := service.Receive(context.Background(), tampered); outcome != "" || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unaudited refusal outcome=%q err=%v", outcome, err)
	}
	if outcome, err := (*Service)(nil).Receive(context.Background(), request); outcome != "" || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil service outcome=%q err=%v", outcome, err)
	}
}

func testIdentityAndKeys(t *testing.T) (tenant.VerifiedIdentity, ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	scope, _ := tenant.NewScope("tenant-a")
	return tenant.VerifiedIdentity{Scope: scope, Account: "account-a", Device: "device-a", MembershipVersion: 1, PolicyRevision: 1}, public, private
}

func testEnvelope(t *testing.T, private ed25519.PrivateKey) cloudsync.Envelope {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"run_id": "run-a", "task_id": "task-a", "agent_id": "agent-a", "status": "succeeded",
		"started_at": "2026-09-09T10:00:00Z", "finished_at": "2026-09-09T10:01:00Z",
		"commit_ids": []string{"0123456789abcdef0123456789abcdef01234567"},
		"checks":     []map[string]string{{"name": "go test", "result": "pass"}},
	})
	envelope := cloudsync.Envelope{SchemaVersion: 1, EventType: "run_summary", TenantID: "tenant-a", ProjectID: "project-a", EventID: "event-a", ProducerSequence: 1, OccurredAt: time.Unix(1, 0).UTC(), IdempotencyKey: "idem-a", ProducerVersion: "test", Payload: payload, Integrity: cloudsync.Integrity{KeyID: "key-a"}}
	if err := cloudsync.Sign(&envelope, private); err != nil {
		t.Fatal(err)
	}
	return envelope
}
