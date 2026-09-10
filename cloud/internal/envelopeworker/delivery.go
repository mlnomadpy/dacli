package envelopeworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/ratelimit"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/internal/cloudsync"
)

const maxDeliveryBatch = 100

type Delivery struct {
	Tenant   tenant.OrganizationID
	Project  tenant.ProjectID
	ID       string
	Envelope cloudsync.Envelope
	Attempts int
}

type DeadLetter struct {
	Project        tenant.ProjectID
	ID             string
	IdempotencyKey string
	Attempts       int
	LastErrorCode  string
	CreatedAt      time.Time
}

type OutboxStore interface {
	ClaimDue(context.Context, tenant.Scope, time.Time, time.Time, int) ([]Delivery, error)
	MarkDelivered(context.Context, tenant.Scope, Delivery, time.Time) error
	Reschedule(context.Context, tenant.Scope, Delivery, time.Time, string) error
	MarkDeadLetter(context.Context, tenant.Scope, Delivery, time.Time, string) error
}

type Sender interface {
	Send(context.Context, cloudsync.Envelope) error
}

type DeliveryConfig struct {
	BatchSize   int
	MaxAttempts int
	Lease       time.Duration
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	Jitter      func(time.Duration) time.Duration
	Now         func() time.Time
	Limiter     *ratelimit.Limiter
	LimitSecret []byte
}

type Deliverer struct {
	store  OutboxStore
	sender Sender
	config DeliveryConfig
}

func NewDeliverer(store OutboxStore, sender Sender, config DeliveryConfig) (*Deliverer, error) {
	if store == nil || sender == nil || config.BatchSize < 1 || config.BatchSize > maxDeliveryBatch || config.MaxAttempts < 1 || config.Lease <= 0 || config.BaseBackoff <= 0 || config.MaxBackoff < config.BaseBackoff || (config.Limiter != nil && len(config.LimitSecret) < 32) {
		return nil, errors.New("invalid bounded outbox delivery configuration")
	}
	if config.Jitter == nil {
		config.Jitter = func(value time.Duration) time.Duration { return value }
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Deliverer{store: store, sender: sender, config: config}, nil
}

func (d *Deliverer) RunTenant(ctx context.Context, scope tenant.Scope) error {
	if d == nil || d.store == nil || d.sender == nil {
		return errors.New("outbox deliverer is not configured")
	}
	now := d.config.Now().UTC()
	if d.config.Limiter != nil {
		decision, err := d.config.Limiter.Allow(ctx, ratelimit.SyncKey(d.config.LimitSecret, scope, "", "delivery"))
		if err != nil {
			return err
		}
		if !decision.Allowed {
			return RateLimitError{RetryAfter: decision.RetryAfter}
		}
	}
	rows, err := d.store.ClaimDue(ctx, scope, now, now.Add(d.config.Lease), d.config.BatchSize)
	if err != nil {
		return fmt.Errorf("claim outbox: %w", err)
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := d.sender.Send(ctx, row.Envelope)
		at := d.config.Now().UTC()
		if err == nil {
			if err := d.store.MarkDelivered(ctx, scope, row, at); err != nil {
				return fmt.Errorf("acknowledge outbox %s: %w", row.ID, err)
			}
			continue
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		code := "delivery_failed"
		if row.Attempts+1 >= d.config.MaxAttempts {
			if markErr := d.store.MarkDeadLetter(ctx, scope, row, at, code); markErr != nil {
				return fmt.Errorf("dead-letter outbox %s: %w", row.ID, markErr)
			}
			continue
		}
		delay := d.backoff(row.Attempts + 1)
		if retry := d.config.Jitter(delay); retry > 0 && retry <= d.config.MaxBackoff {
			delay = retry
		}
		if retryErr := d.store.Reschedule(ctx, scope, row, at.Add(delay), code); retryErr != nil {
			return fmt.Errorf("reschedule outbox %s: %w", row.ID, retryErr)
		}
	}
	return nil
}

func (d *Deliverer) backoff(attempt int) time.Duration {
	delay := d.config.BaseBackoff
	for step := 1; step < attempt && delay < d.config.MaxBackoff; step++ {
		if delay > d.config.MaxBackoff/2 {
			return d.config.MaxBackoff
		}
		delay *= 2
	}
	if delay > d.config.MaxBackoff {
		return d.config.MaxBackoff
	}
	return delay
}
