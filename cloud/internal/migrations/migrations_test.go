package migrations

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

type fakeStore struct {
	applied []Applied
	writes  []Migration
	failAt  int
	readErr error
}

func (f *fakeStore) Applied(context.Context) ([]Applied, error) { return f.applied, f.readErr }
func (f *fakeStore) ApplyTransaction(_ context.Context, migration Migration) error {
	if migration.Version == f.failAt {
		return errors.New("transaction rolled back")
	}
	f.writes = append(f.writes, migration)
	return nil
}

func catalog(t *testing.T) []Migration {
	t.Helper()
	migrations, err := Load(fstest.MapFS{
		"0001_service_state.sql": &fstest.MapFile{Data: []byte("CREATE TABLE service_state (id TEXT PRIMARY KEY);")},
		"0002_event_cursor.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE event_cursor (id TEXT PRIMARY KEY);")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return migrations
}

func TestRunnerOrdersAndAppliesEachMigrationTransactionally(t *testing.T) {
	migrations := catalog(t)
	store := &fakeStore{applied: []Applied{{Version: 1, Name: migrations[0].Name, Checksum: migrations[0].Checksum}}}
	if err := Run(context.Background(), store, migrations); err != nil {
		t.Fatal(err)
	}
	if len(store.writes) != 1 || store.writes[0].Version != 2 {
		t.Fatalf("writes = %+v", store.writes)
	}
}

func TestRunnerRefusesEditedAndUnsupportedAppliedMigrations(t *testing.T) {
	migrations := catalog(t)
	for name, applied := range map[string][]Applied{
		"edited":      {{Version: 1, Name: migrations[0].Name, Checksum: "different"}},
		"unsupported": {{Version: 99, Name: "future", Checksum: "digest"}},
		"ledger gap":  {{Version: 2, Name: migrations[1].Name, Checksum: migrations[1].Checksum}},
		"duplicate": {
			{Version: 1, Name: migrations[0].Name, Checksum: migrations[0].Checksum},
			{Version: 1, Name: migrations[0].Name, Checksum: migrations[0].Checksum},
		},
	} {
		t.Run(name, func(t *testing.T) {
			store := &fakeStore{applied: applied}
			if err := Run(context.Background(), store, migrations); err == nil {
				t.Fatal("unsafe ledger was accepted")
			}
			if len(store.writes) != 0 {
				t.Fatalf("writes after refusal = %+v", store.writes)
			}
		})
	}
}

func TestCatalogRefusesGapsAndUnknownFiles(t *testing.T) {
	for name, source := range map[string]fstest.MapFS{
		"gap":       {"0002_late.sql": &fstest.MapFile{Data: []byte("SELECT 1;")}},
		"unknown":   {"README.md": &fstest.MapFile{Data: []byte("no")}},
		"directory": {"0001_nested.sql": &fstest.MapFile{Mode: fs.ModeDir}},
		"symlink":   {"0001_link.sql": &fstest.MapFile{Mode: fs.ModeSymlink}},
		"empty":     {"0001_empty.sql": &fstest.MapFile{}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(source); err == nil {
				t.Fatal("invalid catalog was accepted")
			}
		})
	}
}

func TestRunnerPropagatesLedgerAndTransactionFailures(t *testing.T) {
	migrations := catalog(t)
	readFailure := errors.New("ledger unavailable")
	if err := Run(context.Background(), &fakeStore{readErr: readFailure}, migrations); !errors.Is(err, readFailure) {
		t.Fatalf("ledger failure = %v", err)
	}

	store := &fakeStore{failAt: 2}
	err := Run(context.Background(), store, migrations)
	if err == nil || !strings.Contains(err.Error(), "apply migration 0002 transactionally") {
		t.Fatalf("transaction failure = %v", err)
	}
	if len(store.writes) != 1 || store.writes[0].Version != 1 {
		t.Fatalf("writes before transactional refusal = %+v", store.writes)
	}
}

func TestSQLStoreWithoutDriverFailsClosed(t *testing.T) {
	store := SQLStore{}
	if err := store.EnsureLedger(context.Background()); !errors.Is(err, ErrNoSQLDriver) {
		t.Fatalf("EnsureLedger nil DB = %v", err)
	}
	if _, err := store.Applied(context.Background()); !errors.Is(err, ErrNoSQLDriver) {
		t.Fatalf("Applied nil DB = %v", err)
	}
	if err := store.ApplyTransaction(context.Background(), Migration{Version: 1}); !errors.Is(err, ErrNoSQLDriver) {
		t.Fatalf("ApplyTransaction nil DB = %v", err)
	}
}

func TestMigrationFilenameIsCanonical(t *testing.T) {
	if got := (Migration{Version: 7, Name: "add_devices"}).Filename(); got != "0007_add_devices.sql" {
		t.Fatalf("Filename = %q", got)
	}
}
