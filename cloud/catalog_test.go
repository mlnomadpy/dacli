package cloud_test

import (
	"os"
	"testing"

	"github.com/mlnomadpy/dacli/cloud/internal/migrations"
)

func TestShippedMigrationCatalogIsValid(t *testing.T) {
	catalog, err := migrations.Load(os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 2 || catalog[0].Version != 1 || catalog[1].Version != 2 {
		t.Fatalf("catalog = %+v", catalog)
	}
}
