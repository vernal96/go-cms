package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/testsite"
)

const FileDiskMemory core.FileDisk = "memory"

func main() {
	ctx := context.Background()

	app := core.NewApp()

	if err := app.CacheManager().RegisterStore(core.CacheStoreMemory, memorycache.NewStore()); err != nil {
		log.Fatal(err)
	}

	if err := app.CacheManager().SetDefaultStore(core.CacheStoreMemory); err != nil {
		log.Fatal(err)
	}

	if err := app.Storage().RegisterDisk(FileDiskMemory, memorystorage.NewStorage()); err != nil {
		log.Fatal(err)
	}

	if err := app.Storage().SetDefaultDisk(FileDiskMemory); err != nil {
		log.Fatal(err)
	}

	profileRegistry := core.NewDefaultSiteProfileRegistry()

	if err := profileRegistry.RegisterProfile(testsite.New()); err != nil {
		log.Fatal(err)
	}

	runtimeFactory := core.NewSiteRuntimeFactory(app, profileRegistry)

	siteResolver, err := core.NewMemorySiteResolver(core.Site{
		ID:          1,
		ProfileCode: "main",
		Domain:      "example.com",
		Locale:      "ru",
		Settings:    map[string]any{},
	})
	if err != nil {
		log.Fatal(err)
	}

	site, err := siteResolver.ResolveByDomain(ctx, "example.com")
	if err != nil {
		log.Fatal(err)
	}

	runtime, err := runtimeFactory.Make(ctx, site)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("GO CMS started")
	fmt.Println("site runtime created:", runtime.Site().Domain)
}
