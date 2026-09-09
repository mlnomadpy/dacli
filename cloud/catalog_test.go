package cloud_test

import (
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/cloud/internal/envelopeworker"
	"github.com/mlnomadpy/dacli/cloud/internal/migrations"
	"github.com/mlnomadpy/dacli/cloud/internal/tenantrepo"
)

func TestShippedMigrationCatalogIsValid(t *testing.T) {
	catalog, err := migrations.Load(os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 9 || catalog[0].Version != 1 || catalog[1].Version != 2 || catalog[2].Version != 3 || catalog[3].Version != 4 || catalog[4].Version != 5 || catalog[5].Version != 6 || catalog[6].Version != 7 || catalog[7].Version != 8 || catalog[8].Version != 9 {
		t.Fatalf("catalog = %+v", catalog)
	}
	if got := catalog[len(catalog)-1].Version; got != tenantrepo.SchemaVersion {
		t.Fatalf("catalog=%d tenantrepo=%d", got, tenantrepo.SchemaVersion)
	}
	if got := catalog[len(catalog)-1].Version; got != envelopeworker.SchemaVersion {
		t.Fatalf("catalog=%d envelopeworker=%d", got, envelopeworker.SchemaVersion)
	}
}
