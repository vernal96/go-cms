package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/vernal96/go-cms/adapters/database/postgresdb"
	"github.com/vernal96/go-cms/adapters/resource/postgresresource"
	"github.com/vernal96/go-cms/adapters/resourcefield/postgresresourcefield"
	"github.com/vernal96/go-cms/adapters/site/postgressite"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/httpserver"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/registry"
)

func main() {
	ctx := context.Background()

	database, err := postgresdb.Connect(
		ctx,
		env("GO_CMS_DATABASE_DSN", "postgres://go_cms:go_cms@localhost:5432/go_cms?sslmode=disable"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := database.Migrate(ctx); err != nil {
		log.Fatal(err)
	}

	siteRepository, err := postgressite.NewRepository(database.Pool())
	if err != nil {
		log.Fatal(err)
	}

	resourceRepository, err := postgresresource.NewRepository(database.Pool())
	if err != nil {
		log.Fatal(err)
	}

	resourceFieldValueRepository, err := postgresresourcefield.NewRepository(database.Pool())
	if err != nil {
		log.Fatal(err)
	}

	infrastructureRegistry := project.NewInfrastructureRegistry()
	siteProfileRegistry := project.NewSiteProfileRegistry()

	devInfrastructure, err := registry.RegisterDevInfrastructure(
		ctx,
		infrastructureRegistry,
		registry.DevInfrastructureConfig{
			RedisAddr:     env("GO_CMS_REDIS_ADDR", "localhost:6379"),
			RedisPassword: env("GO_CMS_REDIS_PASSWORD", ""),
			RedisDB:       envInt("GO_CMS_REDIS_DB", 0),
			StorageRoot:   env("GO_CMS_STORAGE_LOCAL_ROOT", "./var/storage"),
			KafkaBrokers:  envList("GO_CMS_KAFKA_BROKERS", []string{"localhost:9092"}),
			KafkaTopic:    env("GO_CMS_KAFKA_TOPIC", "cms-events"),
			KafkaGroupID:  env("GO_CMS_KAFKA_GROUP_ID", "go-cms"),
		},
		resourceRepository,
	)
	if err != nil {
		log.Fatal(err)
	}
	infrastructureRegistry.UseResourceFieldValueRepository(resourceFieldValueRepository)

	defer func() {
		if err := devInfrastructure.Close(); err != nil {
			log.Printf("close dev infrastructure: %v", err)
		}
	}()

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

	localSite, err := siteRepository.FindByDomain(ctx, "localhost")
	if err != nil {
		log.Fatal(err)
	}

	if _, err := resourceRepository.EnsureResource(ctx, core.Resource{
		SiteID:      localSite.ID,
		Type:        "page",
		Template:    "default",
		Title:       "Home",
		Alias:       "home",
		Path:        "/",
		IsPublished: true,
		Settings:    map[string]any{},
		SEO: map[string]any{
			"title": "GO CMS",
		},
	}); err != nil {
		log.Fatal(err)
	}

	server, err := httpserver.New(siteRepository, runtimeFactory)
	if err != nil {
		log.Fatal(err)
	}

	addr := env("GO_CMS_HTTP_ADDR", ":8080")
	app.Logger().Info("GO CMS HTTP server listening", core.LogField{
		Key:   "address",
		Value: addr,
	})
	log.Fatal(http.ListenAndServe(addr, server))
}

func env(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("environment variable %s must be an integer: %v", name, err)
	}

	return parsedValue
}

func envList(name string, fallback []string) []string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return fallback
	}

	return result
}
