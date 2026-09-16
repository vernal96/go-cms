package seeds_test

import (
	projectseeds "github.com/vernal96/go-cms/internal/seeds"
	"github.com/vernal96/go-cms/kernel/seeds"
	"io/fs"
	"testing"
)

func TestProjectSeedSource(t *testing.T) {
	source := projectseeds.Dev()
	if err := seeds.ValidateSource(source); err != nil {
		t.Fatal(err)
	}
	if source.ID != "sites_dev" || source.Schema != "core" || len(source.Tags) != 1 || source.Tags[0] != "dev" {
		t.Fatalf("source=%+v", source)
	}
	entries, err := fs.ReadDir(source.FS, source.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Fatalf("files=%v", entries)
	}
	source.Tags[0] = "changed"
	if projectseeds.Dev().Tags[0] != "dev" {
		t.Fatal("shared tags")
	}
}
