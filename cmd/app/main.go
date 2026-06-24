package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/vernal96/go-cms/adapters/site/postgressite"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/httpserver"
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

	siteRepository, err := postgressite.Connect(ctx, env("GO_CMS_DATABASE_DSN", "postgres://go_cms:go_cms@localhost:5432/go_cms?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer siteRepository.Close()

	if err := siteRepository.Migrate(ctx); err != nil {
		log.Fatal(err)
	}

	if err := siteRepository.EnsureSite(ctx, core.Site{
		ProfileCode: "main",
		Domain:      "localhost",
		Locale:      "ru",
		Settings: map[string]any{
			"name": "GO CMS Local Site",
		},
	}); err != nil {
		log.Fatal(err)
	}

	if err := siteRepository.EnsureSite(ctx, core.Site{
		ProfileCode: "main",
		Domain:      "example.com",
		Locale:      "ru",
		Settings: map[string]any{
			"name": "GO CMS Example Site",
		},
	}); err != nil {
		log.Fatal(err)
	}

	server, err := httpserver.New(siteRepository, runtimeFactory)
	if err != nil {
		log.Fatal(err)
	}

	addr := env("GO_CMS_HTTP_ADDR", ":8080")
	log.Printf("GO CMS HTTP server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server))
}

func env(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
