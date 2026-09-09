package envelopeworker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mlnomadpy/dacli/cloud/internal/tenant"
	"github.com/mlnomadpy/dacli/internal/cloudsync"
)

func TestOpenRequiresCurrentSchema(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`SELECT COALESCE`).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(SchemaVersion))
	if _, err := Open(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	db2, mock2, _ := sqlmock.New()
	defer func() { _ = db2.Close() }()
	mock2.ExpectQuery(`SELECT COALESCE`).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(SchemaVersion - 1))
	if _, err := Open(context.Background(), db2); err == nil {
		t.Fatal("stale schema accepted")
	}
	if _, err := Open(context.Background(), nil); err == nil {
		t.Fatal("nil database accepted")
	}
}

func TestAcceptCommitsIdentityPayloadCursorAndAuditBeforeAcknowledgement(t *testing.T) {
	repository, mock, closeDB := testRepository(t)
	defer closeDB()
	identity, _, private := testIdentityAndKeys(t)
	envelope := testEnvelope(t, private)
	audit := testAudit(identity, "operation-a")
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WithArgs(identity.Scope.Organization).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_envelope_streams`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT last_sequence`).WillReturnRows(sqlmock.NewRows([]string{"last", "floor", "minimum", "maximum"}).AddRow(0, 1, 1, 1))
	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`INSERT INTO controlplane_envelope_inbox_identities`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_envelope_inbox_payloads`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT producer_sequence`).WillReturnRows(sqlmock.NewRows([]string{"sequence"}).AddRow(1))
	mock.ExpectExec(`UPDATE controlplane_envelope_streams SET last_sequence`).WithArgs(identity.Scope.Organization, envelope.ProjectID, envelope.Integrity.KeyID, uint64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_envelope_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	outcome, err := repository.Accept(context.Background(), identity, envelope, audit)
	if err != nil || outcome != cloudsync.OutcomeAccepted {
		t.Fatalf("accept=%q, %v", outcome, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptAuditsDuplicateReplayAndReorderedOutcomes(t *testing.T) {
	identity, _, private := testIdentityAndKeys(t)
	for name, tc := range map[string]struct {
		last, floor, sequence uint64
		duplicate             bool
		sequenceCollision     bool
		want                  cloudsync.Outcome
		persists              bool
	}{
		"duplicate":    {last: 1, floor: 1, sequence: 1, duplicate: true, want: cloudsync.OutcomeDuplicate},
		"replay":       {last: 5, floor: 3, sequence: 2, want: cloudsync.OutcomeReplay},
		"reordered":    {last: 5, floor: 1, sequence: 3, want: cloudsync.OutcomeReordered, persists: true},
		"equivocation": {last: 1, floor: 1, sequence: 2, sequenceCollision: true, want: cloudsync.OutcomeTampered},
	} {
		t.Run(name, func(t *testing.T) {
			repository, mock, closeDB := testRepository(t)
			defer closeDB()
			envelope := testEnvelope(t, private)
			envelope.ProducerSequence = tc.sequence
			_ = cloudsync.Sign(&envelope, private)
			mock.ExpectBegin()
			mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(`INSERT INTO controlplane_envelope_streams`).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery(`SELECT last_sequence`).WillReturnRows(sqlmock.NewRows([]string{"last", "floor", "minimum", "maximum"}).AddRow(tc.last, tc.floor, 1, 1))
			mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(tc.duplicate))
			if !tc.duplicate && tc.sequence >= tc.floor {
				mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(tc.sequenceCollision))
			}
			if tc.persists {
				mock.ExpectExec(`INSERT INTO controlplane_envelope_inbox_identities`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO controlplane_envelope_inbox_payloads`).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery(`SELECT producer_sequence`).WillReturnRows(sqlmock.NewRows([]string{"sequence"}))
			}
			mock.ExpectExec(`INSERT INTO controlplane_envelope_audit_events`).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			outcome, err := repository.Accept(context.Background(), identity, envelope, testAudit(identity, "operation-"+name))
			if err != nil || outcome != tc.want {
				t.Fatalf("outcome=%q err=%v want=%q", outcome, err, tc.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAcceptRollsBackWhenPayloadOrAuditCannotCommit(t *testing.T) {
	repository, mock, closeDB := testRepository(t)
	defer closeDB()
	identity, _, private := testIdentityAndKeys(t)
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_envelope_streams`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT last_sequence`).WillReturnRows(sqlmock.NewRows([]string{"last", "floor", "minimum", "maximum"}).AddRow(0, 1, 1, 1))
	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`INSERT INTO controlplane_envelope_inbox_identities`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO controlplane_envelope_inbox_payloads`).WillReturnError(errors.New("disk full"))
	mock.ExpectRollback()
	if outcome, err := repository.Accept(context.Background(), identity, testEnvelope(t, private), testAudit(identity, "operation-fail")); err == nil || outcome != cloudsync.OutcomeAccepted {
		t.Fatalf("failed transaction outcome=%q err=%v", outcome, err)
	}
}

func TestOutboxRepositoryClaimsTransitionsAndQueriesDeadLetters(t *testing.T) {
	repository, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, _ := tenant.NewScope("tenant-a")
	envelope := cloudsync.Envelope{TenantID: "tenant-a", ProjectID: "project-a", EventID: "event-a", IdempotencyKey: "idem-a"}
	raw, _ := json.Marshal(envelope)
	now := time.Unix(100, 0).UTC()

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`WITH due AS`).WillReturnRows(sqlmock.NewRows([]string{"tenant", "project", "id", "envelope", "attempts"}).AddRow("tenant-a", "project-a", "outbox-a", raw, 1))
	mock.ExpectCommit()
	rows, err := repository.ClaimDue(context.Background(), scope, now, now.Add(time.Minute), 10)
	if err != nil || len(rows) != 1 || rows[0].Envelope.EventID != "event-a" {
		t.Fatalf("claim=%+v err=%v", rows, err)
	}

	for _, transition := range []func(context.Context, tenant.Scope, Delivery, time.Time) error{
		func(ctx context.Context, scope tenant.Scope, row Delivery, at time.Time) error {
			return repository.MarkDelivered(ctx, scope, row, at)
		},
		func(ctx context.Context, scope tenant.Scope, row Delivery, at time.Time) error {
			return repository.Reschedule(ctx, scope, row, at, "offline")
		},
		func(ctx context.Context, scope tenant.Scope, row Delivery, at time.Time) error {
			return repository.MarkDeadLetter(ctx, scope, row, at, "offline")
		},
	} {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE controlplane_envelope_outbox SET`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		if err := transition(context.Background(), scope, rows[0], now); err != nil {
			t.Fatal(err)
		}
	}

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT project_id, outbox_id`).WillReturnRows(sqlmock.NewRows([]string{"project", "id", "idem", "attempts", "code", "created"}).AddRow("project-a", "outbox-a", "idem-a", 3, "offline", now.UnixMilli()))
	mock.ExpectCommit()
	dead, err := repository.DeadLetters(context.Background(), scope, "project-a", 10)
	if err != nil || len(dead) != 1 || dead[0].Attempts != 3 || !dead[0].CreatedAt.Equal(now) {
		t.Fatalf("dead letters=%+v err=%v", dead, err)
	}
}

func TestEnqueueIsIdempotentAndRetentionDeletesOnlyPayloads(t *testing.T) {
	repository, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, _ := tenant.NewScope("tenant-a")
	now := time.Unix(100, 0).UTC()
	_, _, private := testIdentityAndKeys(t)
	envelope := testEnvelope(t, private)
	raw, _ := json.Marshal(envelope)
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT envelope FROM controlplane_envelope_outbox`).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO controlplane_envelope_outbox`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repository.Enqueue(context.Background(), scope, "outbox-a", envelope, now); err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT envelope FROM controlplane_envelope_outbox`).WillReturnRows(sqlmock.NewRows([]string{"envelope"}).AddRow(raw))
	mock.ExpectCommit()
	if err := repository.Enqueue(context.Background(), scope, "different-local-id", envelope, now); err != nil {
		t.Fatalf("stable idempotent enqueue = %v", err)
	}
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM controlplane_envelope_inbox_payloads`).WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectCommit()
	count, err := repository.ExpirePayloads(context.Background(), scope, now, 100)
	if err != nil || count != 4 {
		t.Fatalf("expired=%d err=%v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueRejectsIdempotencyCollision(t *testing.T) {
	repository, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, _ := tenant.NewScope("tenant-a")
	_, _, private := testIdentityAndKeys(t)
	envelope := testEnvelope(t, private)
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT envelope FROM controlplane_envelope_outbox`).WillReturnRows(sqlmock.NewRows([]string{"envelope"}).AddRow([]byte(`{"different":true}`)))
	mock.ExpectRollback()
	if err := repository.Enqueue(context.Background(), scope, "outbox-a", envelope, time.Unix(1, 0)); err == nil {
		t.Fatal("idempotency collision reported success")
	}
	if err := repository.Enqueue(context.Background(), scope, "", envelope, time.Time{}); err == nil {
		t.Fatal("invalid envelope reported success")
	}
}

func testRepository(t *testing.T) (*Repository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return &Repository{db: db}, mock, func() { _ = db.Close() }
}

func testAudit(identity tenant.VerifiedIdentity, correlation string) Audit {
	return Audit{CorrelationID: correlation, Identity: identity, Project: "project-a", EventIdentity: strings.Repeat("a", 64), OccurredAt: time.Unix(100, 0).UTC()}
}

func TestRepositoryRejectsInvalidInputsAndLostLeases(t *testing.T) {
	repository, mock, closeDB := testRepository(t)
	defer closeDB()
	scope, _ := tenant.NewScope("tenant-a")
	if _, err := repository.ClaimDue(context.Background(), scope, time.Time{}, time.Time{}, 0); err == nil {
		t.Fatal("invalid claim accepted")
	}
	if _, err := repository.DeadLetters(context.Background(), scope, "project-a", 0); err == nil {
		t.Fatal("unbounded dead-letter query accepted")
	}
	if _, err := repository.ExpirePayloads(context.Background(), scope, time.Time{}, 0); err == nil {
		t.Fatal("invalid retention request accepted")
	}
	row := Delivery{Tenant: "tenant-a", Project: "project-a", ID: "outbox-a", Attempts: 1}
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE controlplane_envelope_outbox SET`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	if err := repository.MarkDelivered(context.Background(), scope, row, time.Unix(1, 0)); err == nil {
		t.Fatal("lost lease reported success")
	}
	if err := (&Repository{}).RecordDecision(context.Background(), testAudit(tenant.VerifiedIdentity{}, "bad")); err == nil {
		t.Fatal("closed repository accepted audit")
	}
}
