package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/vernal96/go-cms/adapters/database/mysqldb"
	"github.com/vernal96/go-cms/adapters/database/postgresdb"
	"github.com/vernal96/go-cms/adapters/resource/mysqlresource"
	"github.com/vernal96/go-cms/adapters/resource/postgresresource"
	"github.com/vernal96/go-cms/adapters/resourcefield/mysqlresourcefield"
	"github.com/vernal96/go-cms/adapters/resourcefield/postgresresourcefield"
	"github.com/vernal96/go-cms/adapters/site/mysqlsite"
	"github.com/vernal96/go-cms/adapters/site/postgressite"
	"github.com/vernal96/go-cms/adapters/widgetinstance/mysqlwidgetinstance"
	"github.com/vernal96/go-cms/adapters/widgetinstance/postgreswidgetinstance"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/httpserver"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/registry"
)

func main() {
	ctx := context.Background()

	repositories, err := buildDatabaseRepositories(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := repositories.close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := repositories.migrate(ctx); err != nil {
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
		repositories.resources,
	)
	if err != nil {
		log.Fatal(err)
	}
	infrastructureRegistry.UseResourceFieldValueRepository(repositories.resourceFieldValues)
	infrastructureRegistry.UseWidgetInstanceRepository(repositories.widgetInstances)

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

	if err := repositories.sites.EnsureSite(ctx, core.Site{
		ProfileCode: "main",
		Domain:      "localhost",
		Locale:      "ru",
		Settings: map[string]any{
			"name": "GO CMS Local Site",
		},
	}); err != nil {
		log.Fatal(err)
	}

	if err := repositories.sites.EnsureSite(ctx, core.Site{
		ProfileCode: "main",
		Domain:      "example.com",
		Locale:      "ru",
		Settings: map[string]any{
			"name": "GO CMS Example Site",
		},
	}); err != nil {
		log.Fatal(err)
	}

	localSite, err := repositories.sites.FindByDomain(ctx, "localhost")
	if err != nil {
		log.Fatal(err)
	}

	if _, err := repositories.resources.EnsureResource(ctx, core.Resource{
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

	server, err := httpserver.New(repositories.sites, runtimeFactory)
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

const (
	defaultPostgresDSN = "postgres://go_cms:go_cms@localhost:5432/go_cms?sslmode=disable"
	defaultMySQLDSN    = "go_cms:go_cms@tcp(localhost:3306)/go_cms?parseTime=true&multiStatements=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"
)

type siteRepository interface {
	core.SiteRepository

	EnsureSite(ctx context.Context, site core.Site) error
}

type resourceRepository interface {
	core.ResourceRepository

	EnsureResource(ctx context.Context, resource core.Resource) (core.Resource, error)
}

type databaseRepositories struct {
	close               func() error
	migrate             func(context.Context) error
	sites               siteRepository
	resources           resourceRepository
	resourceFieldValues core.ResourceFieldValueRepository
	widgetInstances     core.WidgetInstanceRepository
}

func buildDatabaseRepositories(ctx context.Context) (databaseRepositories, error) {
	driver := strings.ToLower(strings.TrimSpace(env("GO_CMS_DATABASE_DRIVER", "postgres")))

	switch driver {
	case "postgres":
		return buildPostgresRepositories(ctx)
	case "mysql":
		return buildMySQLRepositories(ctx)
	default:
		return databaseRepositories{}, fmt.Errorf(
			"unsupported database driver %q",
			driver,
		)
	}
}

func buildPostgresRepositories(ctx context.Context) (databaseRepositories, error) {
	database, err := postgresdb.Connect(
		ctx,
		env("GO_CMS_DATABASE_DSN", defaultPostgresDSN),
	)
	if err != nil {
		return databaseRepositories{}, err
	}

	sites, err := postgressite.NewRepository(database.Pool())
	if err != nil {
		database.Close()
		return databaseRepositories{}, err
	}
	resources, err := postgresresource.NewRepository(database.Pool())
	if err != nil {
		database.Close()
		return databaseRepositories{}, err
	}
	resourceFieldValues, err := postgresresourcefield.NewRepository(database.Pool())
	if err != nil {
		database.Close()
		return databaseRepositories{}, err
	}
	widgetInstances, err := postgreswidgetinstance.NewRepository(database.Pool())
	if err != nil {
		database.Close()
		return databaseRepositories{}, err
	}

	return databaseRepositories{
		close: func() error {
			database.Close()
			return nil
		},
		migrate:             database.Migrate,
		sites:               sites,
		resources:           resources,
		resourceFieldValues: resourceFieldValues,
		widgetInstances:     widgetInstances,
	}, nil
}

func buildMySQLRepositories(ctx context.Context) (databaseRepositories, error) {
	database, err := mysqldb.Connect(
		ctx,
		env("GO_CMS_DATABASE_DSN", defaultMySQLDSN),
	)
	if err != nil {
		return databaseRepositories{}, err
	}

	sites, err := mysqlsite.NewRepository(database.DB())
	if err != nil {
		_ = database.Close()
		return databaseRepositories{}, err
	}
	resources, err := mysqlresource.NewRepository(database.DB())
	if err != nil {
		_ = database.Close()
		return databaseRepositories{}, err
	}
	resourceFieldValues, err := mysqlresourcefield.NewRepository(database.DB())
	if err != nil {
		_ = database.Close()
		return databaseRepositories{}, err
	}
	widgetInstances, err := mysqlwidgetinstance.NewRepository(database.DB())
	if err != nil {
		_ = database.Close()
		return databaseRepositories{}, err
	}

	return databaseRepositories{
		close:               database.Close,
		migrate:             database.Migrate,
		sites:               sites,
		resources:           resources,
		resourceFieldValues: resourceFieldValues,
		widgetInstances:     widgetInstances,
	}, nil
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
