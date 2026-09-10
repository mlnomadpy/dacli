// Package ratelimit provides a bounded, goroutine-free token bucket for
// control-plane trust boundaries. Keys are fixed digests, never raw identity.
package ratelimit

import (
	"container/list"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

type Policy struct {
	Capacity       int
	RefillInterval time.Duration
	MaxKeys        int
	IdleTTL        time.Duration
}

func (p Policy) Validate() error {
	if p.Capacity < 1 || p.Capacity > 1_000_000 || p.RefillInterval < time.Millisecond || p.RefillInterval > time.Hour || p.MaxKeys < 1 || p.MaxKeys > 1_000_000 || p.IdleTTL < p.RefillInterval || p.IdleTTL > 24*time.Hour {
		return errors.New("invalid bounded rate-limit policy")
	}
	return nil
}

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}

type bucket struct {
	key        [32]byte
	tokens     int
	lastRefill time.Time
	lastSeen   time.Time
}

type Limiter struct {
	mu      sync.Mutex
	policy  Policy
	now     func() time.Time
	entries map[[32]byte]*list.Element
	lru     list.List
}

func New(policy Policy, now func() time.Time) (*Limiter, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &Limiter{policy: policy, now: now, entries: make(map[[32]byte]*list.Element, policy.MaxKeys)}, nil
}

func (l *Limiter) Allow(ctx context.Context, key [32]byte) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	now := l.now()
	element := l.entries[key]
	if element == nil {
		l.expire(now)
		if len(l.entries) >= l.policy.MaxKeys {
			l.remove(l.lru.Back())
		}
		value := &bucket{key: key, tokens: l.policy.Capacity, lastRefill: now, lastSeen: now}
		element = l.lru.PushFront(value)
		l.entries[key] = element
	}
	value := element.Value.(*bucket)
	if now.Before(value.lastSeen) {
		now = value.lastSeen
	}
	l.refill(value, now)
	value.lastSeen = now
	l.lru.MoveToFront(element)
	if value.tokens > 0 {
		value.tokens--
		return Decision{Allowed: true}, nil
	}
	retry := l.policy.RefillInterval - now.Sub(value.lastRefill)
	if retry <= 0 || retry > l.policy.RefillInterval {
		retry = l.policy.RefillInterval
	}
	return Decision{RetryAfter: retry}, nil
}

func (l *Limiter) refill(value *bucket, now time.Time) {
	elapsed := now.Sub(value.lastRefill)
	if elapsed < l.policy.RefillInterval {
		return
	}
	added := int(elapsed / l.policy.RefillInterval)
	value.tokens = min(l.policy.Capacity, value.tokens+added)
	value.lastRefill = value.lastRefill.Add(time.Duration(added) * l.policy.RefillInterval)
}

func (l *Limiter) expire(now time.Time) {
	for element := l.lru.Back(); element != nil; element = l.lru.Back() {
		value := element.Value.(*bucket)
		if now.Before(value.lastSeen) || now.Sub(value.lastSeen) < l.policy.IdleTTL {
			return
		}
		l.remove(element)
	}
}

func (l *Limiter) remove(element *list.Element) {
	if element == nil {
		return
	}
	delete(l.entries, element.Value.(*bucket).key)
	l.lru.Remove(element)
}

// NetworkKey uses the direct peer address. Forwarded headers are intentionally
// ignored until a deployment configures a separately trusted proxy boundary.
func NetworkKey(secret []byte, remoteAddress string) [32]byte {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}
	return digest(secret, "network/v1", host)
}

func IdentityKey(secret []byte, identity tenant.VerifiedIdentity, purpose string) [32]byte {
	return digest(secret, "identity/v1", string(identity.Scope.Organization), string(identity.Account), string(identity.Device), purpose)
}

func SyncKey(secret []byte, scope tenant.Scope, project tenant.ProjectID, direction string) [32]byte {
	return digest(secret, "sync/v1", string(scope.Organization), string(project), direction)
}

func digest(secret []byte, parts ...string) [32]byte {
	hash := hmac.New(sha256.New, secret)
	for _, part := range parts {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(part)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(part))
	}
	var out [32]byte
	copy(out[:], hash.Sum(nil))
	return out
}
