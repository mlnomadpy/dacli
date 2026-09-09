package devicesession

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

var errConflict = errors.New("session conflict")

type memoryStore struct {
	mu       sync.Mutex
	snapshot Snapshot
	lookups  int
}

func (s *memoryStore) Issue(_ context.Context, _ tenant.Scope, _ Mutation, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.Session = record
	return nil
}

func (s *memoryStore) Lookup(_ context.Context, scope tenant.Scope, digest [32]byte) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lookups++
	if s.snapshot.Session.Tenant != scope.Organization || s.snapshot.Session.CredentialDigest != digest {
		return Snapshot{}, ErrDenied
	}
	return s.snapshot, nil
}

func (s *memoryStore) Revoke(_ context.Context, scope tenant.Scope, _ Mutation, digest [32]byte, expected tenant.Version) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snapshot.Session.Tenant != scope.Organization || s.snapshot.Session.CredentialDigest != digest || s.snapshot.Session.Version != expected || s.snapshot.Session.State != StateActive {
		return errConflict
	}
	s.snapshot.Session.State = StateRevoked
	s.snapshot.Session.Version++
	return nil
}

func (s *memoryStore) Rotate(_ context.Context, scope tenant.Scope, _ Mutation, digest [32]byte, expected tenant.Version, next Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snapshot.Session.Tenant != scope.Organization || s.snapshot.Session.CredentialDigest != digest || s.snapshot.Session.Version != expected || s.snapshot.Session.State != StateActive {
		return errConflict
	}
	s.snapshot.Session = next
	return nil
}

func TestAuthorizeReloadsRevocationOnEveryOperation(t *testing.T) {
	manager, store, scope, credential := fixture(t)
	if _, err := manager.Authorize(context.Background(), scope, credential); err != nil {
		t.Fatal(err)
	}
	if err := manager.Revoke(context.Background(), scope, mutation(), credential, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Authorize(context.Background(), scope, credential); !errors.Is(err, ErrDenied) {
		t.Fatalf("revoked authorization = %v", err)
	}
	if store.lookups != 2 {
		t.Fatalf("membership/session snapshot was not reloaded: lookups=%d", store.lookups)
	}
}

func TestAuthorizeImmediatelyObservesMembershipAndDeviceChanges(t *testing.T) {
	for name, change := range map[string]func(*Snapshot){
		"membership removed": func(value *Snapshot) { value.Membership.State = tenant.LifecycleRevoked; value.Membership.Version++ },
		"device revoked":     func(value *Snapshot) { value.Device.State = tenant.LifecycleRevoked; value.Device.Version++ },
		"membership expired": func(value *Snapshot) { value.Membership.ExpiresUnix = 100 },
		"session expired":    func(value *Snapshot) { value.Session.ExpiresUnix = 100 },
	} {
		t.Run(name, func(t *testing.T) {
			manager, store, scope, credential := fixture(t)
			change(&store.snapshot)
			if _, err := manager.Authorize(context.Background(), scope, credential); !errors.Is(err, ErrDenied) {
				t.Fatalf("Authorize = %v", err)
			}
		})
	}
}

func TestWrongTenantAndCredentialFailIdentically(t *testing.T) {
	manager, _, scope, credential := fixture(t)
	otherOrg, _ := tenant.NewOrganizationID("tenant-b")
	other, _ := tenant.NewScope(otherOrg)
	wrong := append([]byte(nil), credential...)
	wrong[len(wrong)-1] ^= 1
	for name, attempt := range map[string]func() error{
		"tenant": func() error { _, err := manager.Authorize(context.Background(), other, credential); return err },
		"secret": func() error { _, err := manager.Authorize(context.Background(), scope, wrong); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := attempt(); !errors.Is(err, ErrDenied) || err.Error() != ErrDenied.Error() {
				t.Fatalf("denial = %v", err)
			}
		})
	}
}

func TestRotationInvalidatesOldCredentialAndRefusesReplay(t *testing.T) {
	manager, _, scope, credential := fixture(t)
	nextCredential := secret("next")
	next, err := manager.Rotate(context.Background(), scope, mutation(), credential, 1, "session-2", nextCredential, time.Unix(4000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if next.Version != 1 || next.ID != "session-2" {
		t.Fatalf("next session = %+v", next)
	}
	if _, err := manager.Authorize(context.Background(), scope, credential); !errors.Is(err, ErrDenied) {
		t.Fatalf("old credential = %v", err)
	}
	if _, err := manager.Authorize(context.Background(), scope, nextCredential); err != nil {
		t.Fatalf("new credential = %v", err)
	}
	if _, err := manager.Rotate(context.Background(), scope, mutation(), credential, 1, "session-3", secret("third"), time.Unix(4000, 0)); !errors.Is(err, ErrDenied) {
		t.Fatalf("replayed rotation = %v", err)
	}
}

func TestRecordJSONAndErrorsNeverExposeCredential(t *testing.T) {
	manager, _, scope, credential := fixture(t)
	snapshot, err := manager.Authorize(context.Background(), scope, credential)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(snapshot.Session)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{string(credential), string(snapshot.Session.CredentialDigest[:]), "credential_digest"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("session JSON leaked credential material: %s", raw)
		}
	}
	if _, err := manager.Authorize(context.Background(), scope, []byte("short")); err == nil || strings.Contains(err.Error(), string(credential)) {
		t.Fatalf("short credential error = %v", err)
	}
}

func TestIssuePersistsOnlyDigest(t *testing.T) {
	_, store, scope, _ := fixture(t)
	store.snapshot = Snapshot{Device: store.snapshot.Device, Membership: store.snapshot.Membership}
	manager, err := New(store, func() time.Time { return time.Unix(1000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	raw := secret(t.Name())
	record, err := manager.Issue(context.Background(), scope, mutation(), "issued", "account-1", "device-1", raw, time.Unix(2000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if record.CredentialDigest != sha256.Sum256(raw) || store.snapshot.Session.CredentialDigest != record.CredentialDigest {
		t.Fatal("store did not receive the one-way credential digest")
	}
}

func fixture(t *testing.T) (*Manager, *memoryStore, tenant.Scope, []byte) {
	t.Helper()
	organization, _ := tenant.NewOrganizationID("tenant-a")
	scope, _ := tenant.NewScope(organization)
	roles, _ := tenant.Roles(tenant.RoleDeveloper)
	credential := secret(t.Name())
	digest := sha256.Sum256(credential)
	store := &memoryStore{snapshot: Snapshot{
		Session:    Record{Tenant: organization, ID: "session-1", Account: "account-1", Device: "device-1", CredentialDigest: digest, State: StateActive, Version: 1, ExpiresUnix: 2000},
		Device:     tenant.Device{Tenant: organization, ID: "device-1", Account: "account-1", Name: "build", Platform: tenant.PlatformLinux, PublicKey: [32]byte{1}, State: tenant.LifecycleActive, Version: 1},
		Membership: tenant.Membership{Tenant: organization, Account: "account-1", Roles: roles, State: tenant.LifecycleActive, Version: 1},
	}}
	manager, err := New(store, func() time.Time { return time.Unix(1000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	return manager, store, scope, credential
}

func secret(seed string) []byte {
	first := sha256.Sum256([]byte(seed))
	second := sha256.Sum256(first[:])
	return append(first[:], second[:]...)
}

func mutation() Mutation {
	return Mutation{Actor: "account-1", Device: "device-1", CorrelationID: "correlation-1", OccurredAt: time.Unix(1000, 0)}
}
