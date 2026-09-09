package tenantrepo

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mlnomadpy/dacli/cloud/internal/devicesession"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
)

func TestIssueSessionCommitsDigestAndAudit(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, mutation, record := sessionFixture(t)
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectExec(`INSERT INTO controlplane_device_sessions`).
		WithArgs(record.Tenant, record.ID, record.Account, record.Device, record.CredentialDigest[:], record.State, record.Version, record.ExpiresUnix).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Issue(context.Background(), scope, mutation, record); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestLookupSessionJoinsOnlyWithinTenant(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, _, record := sessionFixture(t)
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`JOIN controlplane_devices d ON d.tenant_id = s.tenant_id.*JOIN controlplane_memberships m ON m.tenant_id = s.tenant_id.*WHERE s.tenant_id = \$1 AND s.credential_digest = \$2`).
		WithArgs(scope.Organization, record.CredentialDigest[:]).
		WillReturnRows(sessionRows(record))
	mock.ExpectCommit()
	snapshot, err := repo.Lookup(context.Background(), scope, record.CredentialDigest)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Session.ID != record.ID || snapshot.Device.ID != record.Device || !snapshot.Membership.Roles.Has(tenant.RoleDeveloper) {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	assertExpectations(t, mock)
}

func TestRevokeSessionIsOptimisticAndAudited(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, mutation, record := sessionFixture(t)
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_device_sessions WHERE tenant_id = \$1 AND credential_digest = \$2`).
		WithArgs(scope.Organization, record.CredentialDigest[:]).WillReturnRows(sessionRecordRows(record))
	mock.ExpectExec(`UPDATE controlplane_device_sessions SET state = \$3, version = \$4`).
		WithArgs(scope.Organization, record.ID, devicesession.StateRevoked, tenant.Version(2), devicesession.StateActive, tenant.Version(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Revoke(context.Background(), scope, mutation, record.CredentialDigest, 1); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestRotateSessionRevokesAndIssuesAtomically(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, mutation, current := sessionFixture(t)
	next := current
	next.ID, next.CredentialDigest = "session-2", sha256.Sum256([]byte("next digest material"))
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_device_sessions WHERE tenant_id = \$1 AND credential_digest = \$2`).WillReturnRows(sessionRecordRows(current))
	mock.ExpectExec(`UPDATE controlplane_device_sessions`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_device_sessions`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(
		scope.Organization, mutation.CorrelationID+".revoke", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_tenant_audit_events`).WithArgs(
		scope.Organization, mutation.CorrelationID+".issue", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Rotate(context.Background(), scope, mutation, current.CredentialDigest, 1, next); err != nil {
		t.Fatal(err)
	}
	assertExpectations(t, mock)
}

func TestSessionStaleVersionRollsBackBeforeMutation(t *testing.T) {
	repo, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, mutation, record := sessionFixture(t)
	record.Version = 2
	mock.ExpectBegin()
	expectTenantBinding(mock, scope)
	mock.ExpectQuery(`FROM controlplane_device_sessions`).WillReturnRows(sessionRecordRows(record))
	mock.ExpectRollback()
	if err := repo.Revoke(context.Background(), scope, mutation, record.CredentialDigest, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale revoke = %v", err)
	}
	assertExpectations(t, mock)
}

func sessionFixture(t *testing.T) (tenant.Scope, devicesession.Mutation, devicesession.Record) {
	t.Helper()
	scope := mustScope(t, "tenant-a")
	digest := sha256.Sum256([]byte(t.Name()))
	return scope, devicesession.Mutation{Actor: "account-1", Device: "device-1", CorrelationID: "corr-session", OccurredAt: time.UnixMilli(1234)}, devicesession.Record{
		Tenant: scope.Organization, ID: "session-1", Account: "account-1", Device: "device-1", CredentialDigest: digest,
		State: devicesession.StateActive, Version: 1, ExpiresUnix: 2000,
	}
}

func sessionRecordRows(record devicesession.Record) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "session_id", "account_id", "device_id", "credential_digest", "state", "version", "expires_unix"}).
		AddRow(record.Tenant, record.ID, record.Account, record.Device, record.CredentialDigest[:], record.State, record.Version, record.ExpiresUnix)
}

func sessionRows(record devicesession.Record) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"tenant_id", "session_id", "account_id", "device_id", "credential_digest", "state", "version", "expires_unix", "device_name", "platform", "public_key", "device_state", "device_version", "team_id", "roles", "membership_state", "membership_version", "membership_expires"}).
		AddRow(record.Tenant, record.ID, record.Account, record.Device, record.CredentialDigest[:], record.State, record.Version, record.ExpiresUnix,
			"build", tenant.PlatformLinux, []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, tenant.LifecycleActive, 1,
			"", []byte(`["developer"]`), tenant.LifecycleActive, 1, 0)
}
