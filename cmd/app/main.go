package main

import (
	"context"
	"fmt"
	"log"

	"95.79.129.33/go-cms/dev/core"
	"95.79.129.33/go-cms/dev/internal/testsite"
)

func main() {
	ctx := context.Background()

	app := core.NewApp(nil)

	if err := app.Boot(ctx); err != nil {
		log.Fatal(err)
	}

	profileRegistry := core.NewDefaultSiteProfileRegistry()

	if err := profileRegistry.RegisterProfile(testsite.New()); err != nil {
		log.Fatal(err)
	}

	runtimeFactory := core.NewSiteRuntimeFactory(app, profileRegistry)

	site := core.Site{
		ID:     1,
		Code:   "main",
		Domain: "example.com",
		Settings: map[string]any{
			"locale": "ru",
		},
	}

	runtime, err := runtimeFactory.Make(ctx, site)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("GO CMS started")
	fmt.Println("site runtime created:", runtime.Site().Domain)
}
