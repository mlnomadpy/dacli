// Package devicesession owns short-lived, revocable device authorization.
// Raw credentials exist only at call boundaries; stores receive one-way digests.
package devicesession

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

const minimumCredentialBytes = 32

var ErrDenied = errors.New("device session authorization denied")

type ID string

type State uint8

const (
	StateUnknown State = iota
	StateActive
	StateRevoked
)

type Record struct {
	Tenant           tenant.OrganizationID `json:"tenant_id"`
	ID               ID                    `json:"session_id"`
	Account          tenant.AccountID      `json:"account_id"`
	Device           tenant.DeviceID       `json:"device_id"`
	CredentialDigest [32]byte              `json:"-"`
	State            State                 `json:"state"`
	Version          tenant.Version        `json:"version"`
	ExpiresUnix      int64                 `json:"expires_unix"`
}

type Snapshot struct {
	Session    Record
	Device     tenant.Device
	Membership tenant.Membership
}

type Mutation struct {
	Actor         tenant.AccountID
	Device        tenant.DeviceID
	CorrelationID string
	OccurredAt    time.Time
}

type Store interface {
	Issue(context.Context, tenant.Scope, Mutation, Record) error
	Lookup(context.Context, tenant.Scope, [32]byte) (Snapshot, error)
	Revoke(context.Context, tenant.Scope, Mutation, [32]byte, tenant.Version) error
	Rotate(context.Context, tenant.Scope, Mutation, [32]byte, tenant.Version, Record) error
}

type Manager struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) (*Manager, error) {
	if store == nil {
		return nil, errors.New("device session manager requires a store")
	}
	if now == nil {
		now = time.Now
	}
	return &Manager{store: store, now: now}, nil
}

func (m *Manager) Issue(ctx context.Context, scope tenant.Scope, mutation Mutation, id ID, account tenant.AccountID, device tenant.DeviceID, credential []byte, expires time.Time) (Record, error) {
	if len(credential) < minimumCredentialBytes || expires.Unix() <= m.now().Unix() || id == "" {
		return Record{}, ErrDenied
	}
	record := Record{Tenant: scope.Organization, ID: id, Account: account, Device: device, CredentialDigest: sha256.Sum256(credential), State: StateActive, Version: 1, ExpiresUnix: expires.Unix()}
	if err := validate(scope, record); err != nil {
		return Record{}, ErrDenied
	}
	if err := m.store.Issue(ctx, scope, mutation, record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (m *Manager) Authorize(ctx context.Context, scope tenant.Scope, credential []byte) (Snapshot, error) {
	if len(credential) < minimumCredentialBytes {
		return Snapshot{}, ErrDenied
	}
	digest := sha256.Sum256(credential)
	snapshot, err := m.store.Lookup(ctx, scope, digest)
	if err != nil || subtle.ConstantTimeCompare(digest[:], snapshot.Session.CredentialDigest[:]) != 1 {
		return Snapshot{}, ErrDenied
	}
	if err := validateSnapshot(scope, snapshot, m.now()); err != nil {
		return Snapshot{}, ErrDenied
	}
	return snapshot, nil
}

func (m *Manager) Revoke(ctx context.Context, scope tenant.Scope, mutation Mutation, credential []byte, expected tenant.Version) error {
	if len(credential) < minimumCredentialBytes || expected == 0 {
		return ErrDenied
	}
	return m.store.Revoke(ctx, scope, mutation, sha256.Sum256(credential), expected)
}

func (m *Manager) Rotate(ctx context.Context, scope tenant.Scope, mutation Mutation, credential []byte, expected tenant.Version, nextID ID, nextCredential []byte, expires time.Time) (Record, error) {
	if len(credential) < minimumCredentialBytes || len(nextCredential) < minimumCredentialBytes || expected == 0 || expires.Unix() <= m.now().Unix() || nextID == "" {
		return Record{}, ErrDenied
	}
	current, err := m.Authorize(ctx, scope, credential)
	if err != nil || current.Session.Version != expected {
		return Record{}, ErrDenied
	}
	next := Record{Tenant: scope.Organization, ID: nextID, Account: current.Session.Account, Device: current.Session.Device, CredentialDigest: sha256.Sum256(nextCredential), State: StateActive, Version: 1, ExpiresUnix: expires.Unix()}
	if err := m.store.Rotate(ctx, scope, mutation, current.Session.CredentialDigest, expected, next); err != nil {
		return Record{}, err
	}
	return next, nil
}

func validate(scope tenant.Scope, record Record) error {
	if _, err := tenant.NewScope(scope.Organization); err != nil || record.Tenant != scope.Organization {
		return ErrDenied
	}
	if _, err := tenant.NewAccountID(string(record.Account)); err != nil {
		return ErrDenied
	}
	if _, err := tenant.NewDeviceID(string(record.Device)); err != nil || record.ID == "" || record.CredentialDigest == [32]byte{} || record.State < StateActive || record.State > StateRevoked || record.Version == 0 || record.ExpiresUnix <= 0 {
		return ErrDenied
	}
	return nil
}

func validateSnapshot(scope tenant.Scope, value Snapshot, now time.Time) error {
	if err := validate(scope, value.Session); err != nil || value.Session.State != StateActive || value.Session.ExpiresUnix <= now.Unix() {
		return ErrDenied
	}
	if err := tenant.ValidateDevice(scope, value.Device); err != nil || value.Device.State != tenant.LifecycleActive || value.Device.ID != value.Session.Device || value.Device.Account != value.Session.Account {
		return ErrDenied
	}
	if err := tenant.ValidateMembership(scope, value.Membership); err != nil || value.Membership.State != tenant.LifecycleActive || value.Membership.Account != value.Session.Account || (value.Membership.ExpiresUnix != 0 && value.Membership.ExpiresUnix <= now.Unix()) {
		return ErrDenied
	}
	return nil
}
