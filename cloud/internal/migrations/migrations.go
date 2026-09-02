// Package migrations provides an ordered, checksummed PostgreSQL migration runner.
package migrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const maxMigrationBytes = 1 << 20

var migrationName = regexp.MustCompile(`^([0-9]{4})_([a-z][a-z0-9_]*)\.sql$`)

type Migration struct {
	Version  int
	Name     string
	SQL      string
	Checksum string
}

type Applied struct {
	Version  int
	Name     string
	Checksum string
}

type Store interface {
	Applied(context.Context) ([]Applied, error)
	ApplyTransaction(context.Context, Migration) error
}

func Load(source fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, fmt.Errorf("read migration catalog: %w", err)
	}
	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("migration catalog contains directory %q", entry.Name())
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("migration catalog contains symbolic link %q", entry.Name())
		}
		matches := migrationName.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil, fmt.Errorf("unsupported migration filename %q", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect migration %q: %w", entry.Name(), err)
		}
		if info.Size() <= 0 || info.Size() > maxMigrationBytes {
			return nil, fmt.Errorf("migration %q must contain 1..1048576 bytes", entry.Name())
		}
		raw, err := fs.ReadFile(source, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		version, _ := strconv.Atoi(matches[1])
		digest := sha256.Sum256(raw)
		migrations = append(migrations, Migration{Version: version, Name: matches[2], SQL: string(raw), Checksum: hex.EncodeToString(digest[:])})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	for index, migration := range migrations {
		if migration.Version != index+1 {
			return nil, fmt.Errorf("migration versions must be contiguous from 0001; found %04d at position %d", migration.Version, index+1)
		}
	}
	return migrations, nil
}

// Run checks the complete applied ledger before applying each missing migration.
func Run(ctx context.Context, store Store, catalog []Migration) error {
	applied, err := store.Applied(ctx)
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}
	known := make(map[int]Migration, len(catalog))
	for _, migration := range catalog {
		known[migration.Version] = migration
	}
	seen := make(map[int]bool, len(applied))
	for _, record := range applied {
		migration, ok := known[record.Version]
		if !ok {
			return fmt.Errorf("database contains unsupported migration version %04d", record.Version)
		}
		if record.Name != migration.Name || !strings.EqualFold(record.Checksum, migration.Checksum) {
			return fmt.Errorf("applied migration %04d was edited or does not match this binary", record.Version)
		}
		if seen[record.Version] {
			return fmt.Errorf("applied migration ledger repeats version %04d", record.Version)
		}
		seen[record.Version] = true
	}
	for version := 1; version <= len(applied); version++ {
		if !seen[version] {
			return fmt.Errorf("applied migration ledger has a gap before version %04d", version)
		}
	}
	for _, migration := range catalog {
		if seen[migration.Version] {
			continue
		}
		if err := store.ApplyTransaction(ctx, migration); err != nil {
			return fmt.Errorf("apply migration %04d transactionally: %w", migration.Version, err)
		}
	}
	return nil
}

var ErrNoSQLDriver = errors.New("PostgreSQL driver is not linked into this skeleton")

// Filename provides the canonical display identity without exposing SQL.
func (m Migration) Filename() string {
	return fmt.Sprintf("%04d_%s.sql", m.Version, m.Name)
}
