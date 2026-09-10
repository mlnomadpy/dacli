package ratelimit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

func TestLimiterRefillsBoundsKeysAndHandlesClockSkew(t *testing.T) {
	now := time.Unix(100, 0)
	limiter, err := New(Policy{Capacity: 2, RefillInterval: time.Second, MaxKeys: 2, IdleTTL: 2 * time.Second}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	first, second, third := [32]byte{1}, [32]byte{2}, [32]byte{3}
	for index := 0; index < 2; index++ {
		decision, err := limiter.Allow(context.Background(), first)
		if err != nil || !decision.Allowed {
			t.Fatalf("token %d = %+v, %v", index, decision, err)
		}
	}
	decision, _ := limiter.Allow(context.Background(), first)
	if decision.Allowed || decision.RetryAfter != time.Second {
		t.Fatalf("exhausted = %+v", decision)
	}
	now = now.Add(-time.Second)
	decision, _ = limiter.Allow(context.Background(), first)
	if decision.Allowed || decision.RetryAfter != time.Second {
		t.Fatalf("clock skew = %+v", decision)
	}
	now = now.Add(2 * time.Second)
	decision, _ = limiter.Allow(context.Background(), first)
	if !decision.Allowed {
		t.Fatalf("refill = %+v", decision)
	}
	_, _ = limiter.Allow(context.Background(), second)
	_, _ = limiter.Allow(context.Background(), third)
	if len(limiter.entries) != 2 || limiter.entries[first] != nil {
		t.Fatalf("deterministic LRU eviction failed: %#v", limiter.entries)
	}
	now = now.Add(3 * time.Second)
	fourth := [32]byte{4}
	_, _ = limiter.Allow(context.Background(), fourth)
	if len(limiter.entries) != 1 || limiter.entries[fourth] == nil {
		t.Fatalf("idle expiry failed: %#v", limiter.entries)
	}
}

func TestLimiterIsRaceSafeAndCancellationDoesNotConsume(t *testing.T) {
	limiter, _ := New(Policy{Capacity: 200, RefillInterval: time.Second, MaxKeys: 4, IdleTTL: time.Minute}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := limiter.Allow(ctx, [32]byte{1}); !errors.Is(err, context.Canceled) || len(limiter.entries) != 0 {
		t.Fatalf("cancelled allow = %v entries=%d", err, len(limiter.entries))
	}
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _ = limiter.Allow(context.Background(), [32]byte{2})
		}()
	}
	wait.Wait()
}

func TestKeysAreSecretScopedTenantSafeAndIgnoreForwardedClaims(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	network := NetworkKey(secret, "192.0.2.1:443")
	if network != NetworkKey(secret, "192.0.2.1:8443") || network == NetworkKey(secret, "192.0.2.2:443") || network == NetworkKey([]byte("other"), "192.0.2.1:443") {
		t.Fatal("network key lost peer or secret binding")
	}
	scopeA, _ := tenant.NewScope("tenant-a")
	scopeB, _ := tenant.NewScope("tenant-b")
	identityA := tenant.VerifiedIdentity{Scope: scopeA, Account: "account-a", Device: "device-a", MembershipVersion: 1, PolicyRevision: 1}
	identityB := identityA
	identityB.Scope = scopeB
	if IdentityKey(secret, identityA, "read") == IdentityKey(secret, identityB, "read") || SyncKey(secret, scopeA, "project-a", "ingest") == SyncKey(secret, scopeA, "project-a", "deliver") {
		t.Fatal("post-authentication keys collided across tenant or purpose")
	}
}

func TestPolicyRejectsUnboundedOrInvalidConfiguration(t *testing.T) {
	for _, policy := range []Policy{{}, {Capacity: 1, RefillInterval: time.Second, MaxKeys: 0, IdleTTL: time.Minute}, {Capacity: 1, RefillInterval: time.Second, MaxKeys: 1, IdleTTL: time.Millisecond}} {
		if _, err := New(policy, nil); err == nil {
			t.Fatalf("invalid policy accepted: %+v", policy)
		}
	}
}
