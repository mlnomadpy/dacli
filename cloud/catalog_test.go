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
	if len(catalog) != 4 || catalog[0].Version != 1 || catalog[1].Version != 2 || catalog[2].Version != 3 || catalog[3].Version != 4 {
		t.Fatalf("catalog = %+v", catalog)
	}
}
