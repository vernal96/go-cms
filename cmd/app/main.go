package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/configs"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/testsite"
)

func main() {
	ctx := context.Background()

	config := configs.Null()

	app, err := project.BootstrapApp(config)
	if err != nil {
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
