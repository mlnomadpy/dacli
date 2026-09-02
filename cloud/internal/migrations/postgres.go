package migrations

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLStore adapts database/sql to the migration Store. A deployment supplies a
// PostgreSQL database driver; keeping that adapter out of the core preserves the
// Go 1.22, standard-library-only skeleton.
type SQLStore struct{ DB *sql.DB }

func (s SQLStore) EnsureLedger(ctx context.Context) error {
	if s.DB == nil {
		return ErrNoSQLDriver
	}
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS controlplane_schema_migrations (
version INTEGER PRIMARY KEY,
name TEXT NOT NULL,
checksum TEXT NOT NULL,
applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		return fmt.Errorf("ensure migration ledger: %w", err)
	}
	return nil
}

func (s SQLStore) Applied(ctx context.Context) ([]Applied, error) {
	if err := s.EnsureLedger(ctx); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT version, name, checksum FROM controlplane_schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("query migration ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Applied
	for rows.Next() {
		var record Applied
		if err := rows.Scan(&record.Version, &record.Name, &record.Checksum); err != nil {
			return nil, fmt.Errorf("scan migration ledger: %w", err)
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration ledger: %w", err)
	}
	return out, nil
}

func (s SQLStore) ApplyTransaction(ctx context.Context, migration Migration) error {
	if s.DB == nil {
		return ErrNoSQLDriver
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO controlplane_schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`, migration.Version, migration.Name, migration.Checksum); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	committed = true
	return nil
}
