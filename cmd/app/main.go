package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/configs"
	"github.com/vernal96/go-cms/internal/project"
)

func main() {
	ctx := context.Background()

	infrastructureConfig := configs.DevInfrastructure()
	siteProfileConfig := configs.DevSiteProfiles()

	app, err := project.BootstrapApp(infrastructureConfig)
	if err != nil {
		log.Fatal(err)
	}

	profileManager, err := project.NewSiteProfileManager(siteProfileConfig)
	if err != nil {
		log.Fatal(err)
	}

	runtimeFactory := core.NewSiteRuntimeFactory(app, profileManager)

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
