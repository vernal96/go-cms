package project

import (
	"log"

	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/core"
)

const FileDiskMemory core.FileDisk = "memory"

func BootstrapApp() (*core.App, error) {
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

	return app, nil
}
