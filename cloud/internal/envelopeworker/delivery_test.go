package envelopeworker

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/ratelimit"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/internal/cloudsync"
)

type deliveryStore struct {
	rows        []Delivery
	claimErr    error
	delivered   []Delivery
	rescheduled []Delivery
	dead        []Delivery
	retryAt     time.Time
}

func (s *deliveryStore) ClaimDue(context.Context, tenant.Scope, time.Time, time.Time, int) ([]Delivery, error) {
	return append([]Delivery(nil), s.rows...), s.claimErr
}
func (s *deliveryStore) MarkDelivered(_ context.Context, _ tenant.Scope, row Delivery, _ time.Time) error {
	s.delivered = append(s.delivered, row)
	return nil
}
func (s *deliveryStore) Reschedule(_ context.Context, _ tenant.Scope, row Delivery, at time.Time, _ string) error {
	s.rescheduled = append(s.rescheduled, row)
	s.retryAt = at
	return nil
}
func (s *deliveryStore) MarkDeadLetter(_ context.Context, _ tenant.Scope, row Delivery, _ time.Time, _ string) error {
	s.dead = append(s.dead, row)
	return nil
}

type sequenceSender struct {
	errors []error
	seen   []cloudsync.Envelope
	cancel context.CancelFunc
}

func (s *sequenceSender) Send(_ context.Context, envelope cloudsync.Envelope) error {
	s.seen = append(s.seen, envelope)
	if s.cancel != nil {
		s.cancel()
	}
	if len(s.errors) == 0 {
		return nil
	}
	err := s.errors[0]
	s.errors = s.errors[1:]
	return err
}

func TestDelivererAcknowledgesRetriesAndDeadLettersWithoutChangingEnvelope(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	envelopes := []cloudsync.Envelope{{EventID: "ok"}, {EventID: "retry"}, {EventID: "dead"}}
	store := &deliveryStore{rows: []Delivery{
		{Tenant: "tenant-a", Project: "project-a", ID: "one", Envelope: envelopes[0]},
		{Tenant: "tenant-a", Project: "project-a", ID: "two", Envelope: envelopes[1], Attempts: 1},
		{Tenant: "tenant-a", Project: "project-a", ID: "three", Envelope: envelopes[2], Attempts: 2},
	}}
	sender := &sequenceSender{errors: []error{nil, errors.New("offline"), errors.New("offline")}}
	deliverer, err := NewDeliverer(store, sender, DeliveryConfig{BatchSize: 3, MaxAttempts: 3, Lease: time.Minute, BaseBackoff: time.Second, MaxBackoff: 8 * time.Second, Jitter: func(value time.Duration) time.Duration { return value + time.Second }, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	scope, _ := tenant.NewScope("tenant-a")
	if err := deliverer.RunTenant(context.Background(), scope); err != nil {
		t.Fatal(err)
	}
	if len(store.delivered) != 1 || len(store.rescheduled) != 1 || len(store.dead) != 1 || !reflect.DeepEqual(sender.seen, envelopes) {
		t.Fatalf("delivered=%v retry=%v dead=%v seen=%v", store.delivered, store.rescheduled, store.dead, sender.seen)
	}
	// The second delivery is attempt 2: exponential delay 2s plus bounded jitter 1s.
	if !store.retryAt.Equal(now.Add(3 * time.Second)) {
		t.Fatalf("retry at %v", store.retryAt)
	}
}

func TestDelivererHonorsCancellationAndBounds(t *testing.T) {
	if _, err := NewDeliverer(nil, &sequenceSender{}, DeliveryConfig{}); err == nil {
		t.Fatal("invalid configuration accepted")
	}
	store := &deliveryStore{rows: []Delivery{{ID: "one"}}}
	deliverer, _ := NewDeliverer(store, &sequenceSender{}, DeliveryConfig{BatchSize: 1, MaxAttempts: 1, Lease: time.Second, BaseBackoff: time.Second, MaxBackoff: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scope, _ := tenant.NewScope("tenant-a")
	if err := deliverer.RunTenant(ctx, scope); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled run = %v", err)
	}
	store.claimErr = errors.New("database unavailable")
	if err := deliverer.RunTenant(context.Background(), scope); err == nil {
		t.Fatal("claim failure was hidden")
	}
}

func TestBackoffCapsBeforeOverflow(t *testing.T) {
	deliverer := &Deliverer{config: DeliveryConfig{BaseBackoff: time.Second, MaxBackoff: 4 * time.Second}}
	if got := deliverer.backoff(100); got != 4*time.Second {
		t.Fatalf("backoff=%v", got)
	}
}

func TestDelivererLeavesLeaseForRecoveryWhenSendIsCancelled(t *testing.T) {
	store := &deliveryStore{rows: []Delivery{{Tenant: "tenant-a", Project: "project-a", ID: "one"}}}
	ctx, cancel := context.WithCancel(context.Background())
	sender := &sequenceSender{errors: []error{context.Canceled}, cancel: cancel}
	deliverer, _ := NewDeliverer(store, sender, DeliveryConfig{BatchSize: 1, MaxAttempts: 2, Lease: time.Second, BaseBackoff: time.Second, MaxBackoff: time.Second})
	scope, _ := tenant.NewScope("tenant-a")
	if err := deliverer.RunTenant(ctx, scope); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled send = %v", err)
	}
	if len(store.rescheduled)+len(store.dead)+len(store.delivered) != 0 {
		t.Fatal("cancelled send changed durable delivery state")
	}
}

func TestDelivererRateLimitsBeforeClaimingTenantWork(t *testing.T) {
	store := &deliveryStore{rows: []Delivery{{Tenant: "tenant-a", Project: "project-a", ID: "one"}}}
	sender := &sequenceSender{}
	limiter, _ := ratelimit.New(ratelimit.Policy{Capacity: 1, RefillInterval: time.Hour, MaxKeys: 4, IdleTTL: 2 * time.Hour}, nil)
	if _, err := NewDeliverer(store, sender, DeliveryConfig{BatchSize: 1, MaxAttempts: 1, Lease: time.Second, BaseBackoff: time.Second, MaxBackoff: time.Second, Limiter: limiter, LimitSecret: []byte("short")}); err == nil {
		t.Fatal("short delivery limiter key was accepted")
	}
	deliverer, err := NewDeliverer(store, sender, DeliveryConfig{BatchSize: 1, MaxAttempts: 1, Lease: time.Second, BaseBackoff: time.Second, MaxBackoff: time.Second, Limiter: limiter, LimitSecret: []byte("0123456789abcdef0123456789abcdef")})
	if err != nil {
		t.Fatal(err)
	}
	scope, _ := tenant.NewScope("tenant-a")
	if err := deliverer.RunTenant(context.Background(), scope); err != nil {
		t.Fatal(err)
	}
	if err := deliverer.RunTenant(context.Background(), scope); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("second delivery cycle = %v", err)
	}
	if len(sender.seen) != 1 {
		t.Fatalf("rate-limited delivery sent %d envelopes", len(sender.seen))
	}
}
