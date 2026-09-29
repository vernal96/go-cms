package settings

import (
	"testing"
	"testing/fstest"

	"github.com/vernal96/go-cms-kernel/modules/core"
)

func TestDevSeedsAreRegisteredForManualExecution(t *testing.T) {
	files := fstest.MapFS{
		"seeds/000001_starter.up.sql": {Data: []byte("SELECT 1;")},
	}
	definition := (Config{SeedFiles: files}).Definition()
	if len(definition.MainDatabase.Seeds) != 1 {
		t.Fatal("the console must always have the demo seed available")
	}
	source := definition.MainDatabase.Seeds[0]
	if source.Module != core.ModuleCode || source.Source.ID != "starter" || source.Source.Path != "seeds" ||
		len(source.Source.Tags) != 1 || source.Source.Tags[0] != "dev" {
		t.Fatal("dev seed registration changed")
	}
	if _, err := source.Source.FS.Open("seeds/000001_starter.up.sql"); err != nil {
		t.Fatalf("embedded seed files must reach the source: %v", err)
	}
}
