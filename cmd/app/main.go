package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vernal96/go-cms/adapters/site/memorysite"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/registry"
)

func main() {
	ctx := context.Background()

	infrastructureRegistry := project.NewInfrastructureRegistry()
	siteProfileRegistry := project.NewSiteProfileRegistry()

	registry.RegisterDevInfrastructure(infrastructureRegistry)
	registry.RegisterDevSiteProfiles(siteProfileRegistry)

	app, err := project.BootstrapApp(infrastructureRegistry)
	if err != nil {
		log.Fatal(err)
	}

	profileManager, err := project.NewSiteProfileManager(siteProfileRegistry)
	if err != nil {
		log.Fatal(err)
	}

	runtimeFactory := core.NewSiteRuntimeFactory(app, profileManager)

	siteResolver, err := memorysite.NewResolver(core.Site{
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
