package tenantcache

import (
	"sync"
	"testing"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

func cacheKey() Key {
	return Key{Tenant: "tenant-a", Kind: ResourceProject, ResourceID: "same-id", Subject: "account-a", ResourceVersion: 3, MembershipVersion: 7, PolicyRevision: 2}
}

func TestKeySeparatesTenantResourceSubjectAndAuthorizationVersions(t *testing.T) {
	cache := New[string](20)
	base := cacheKey()
	if !cache.Put(base, "tenant-a") {
		t.Fatal("valid cache entry refused")
	}
	mutations := []func(*Key){
		func(k *Key) { k.Tenant = "tenant-b" },
		func(k *Key) { k.Kind = ResourceEnvironment },
		func(k *Key) { k.ResourceID = "other-id" },
		func(k *Key) { k.Subject = "account-b" },
		func(k *Key) { k.ResourceVersion++ },
		func(k *Key) { k.MembershipVersion++ },
		func(k *Key) { k.PolicyRevision++ },
	}
	for _, mutate := range mutations {
		key := base
		mutate(&key)
		if value, ok := cache.Get(key); ok {
			t.Fatalf("changed cache dimension hit value %q: %+v", value, key)
		}
	}
	if value, ok := cache.Get(base); !ok || value != "tenant-a" {
		t.Fatalf("base cache entry = %q, %v", value, ok)
	}
}

func TestCacheIsBoundedLRUAndTenantInvalidationIsScoped(t *testing.T) {
	cache := New[string](2)
	first := cacheKey()
	second, third := first, first
	second.ResourceID, third.ResourceID = "second", "third"
	cache.Put(first, "first")
	cache.Put(second, "second")
	_, _ = cache.Get(first)
	cache.Put(third, "third")
	if _, ok := cache.Get(second); ok {
		t.Fatal("least-recent entry survived capacity eviction")
	}
	other := first
	other.Tenant = "tenant-b"
	cache.Put(other, "other")
	cache.InvalidateTenant(first.Tenant)
	if _, ok := cache.Get(first); ok {
		t.Fatal("tenant entry survived invalidation")
	}
	if value, ok := cache.Get(other); !ok || value != "other" {
		t.Fatal("another tenant was invalidated")
	}
}

func TestInvalidAndConcurrentAccessFailsClosed(t *testing.T) {
	cache := New[int](0)
	if cache.Put(Key{}, 1) {
		t.Fatal("invalid cache key was accepted")
	}
	if _, ok := cache.Get(Key{}); ok {
		t.Fatal("invalid cache key hit")
	}
	key := cacheKey()
	var group sync.WaitGroup
	for index := 0; index < 20; index++ {
		group.Add(1)
		go func(value int) {
			defer group.Done()
			cache.Put(key, value)
			_, _ = cache.Get(key)
		}(index)
	}
	group.Wait()
	cache.InvalidateTenant(tenant.OrganizationID("tenant-a"))
}

func TestEveryResourceKindAndInvalidIdentityShape(t *testing.T) {
	cache := New[string](20_000)
	base := cacheKey()
	for _, kind := range []ResourceKind{ResourceProject, ResourceEnvironment, ResourceProjectAssignment, ResourceEnvironmentAssignment} {
		key := base
		key.Kind = kind
		if kind == ResourceEnvironment {
			key.ResourceID = "environment-a"
		}
		if kind == ResourceProjectAssignment || kind == ResourceEnvironmentAssignment {
			key.ResourceID = "assignment-a"
		}
		if !cache.Put(key, "value") {
			t.Fatalf("resource kind %d refused", kind)
		}
	}
	invalid := []Key{base, base, base, base, base, base}
	invalid[0].Tenant = "bad tenant"
	invalid[1].Subject = "bad account"
	invalid[2].ResourceID = "bad/resource"
	invalid[3].Kind = ResourceUnknown
	invalid[4].ResourceVersion = 0
	invalid[5].PolicyRevision = 0
	for _, key := range invalid {
		if cache.Put(key, "unsafe") {
			t.Fatalf("invalid key accepted: %+v", key)
		}
	}
	var nilCache *Cache[string]
	if nilCache.Put(base, "unsafe") {
		t.Fatal("nil cache accepted a value")
	}
	if _, ok := nilCache.Get(base); ok {
		t.Fatal("nil cache hit")
	}
	nilCache.InvalidateTenant("tenant-a")
	cache.InvalidateTenant("")
}
