package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	kernel "github.com/vernal96/go-cms-kernel"
	appkernel "github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/cache"
	"github.com/vernal96/go-cms-kernel/connectors/kafkaeventbus"
	"github.com/vernal96/go-cms-kernel/connectors/localstorage"
	connectorpostgres "github.com/vernal96/go-cms-kernel/connectors/postgres"
	connectorredis "github.com/vernal96/go-cms-kernel/connectors/redis"
	"github.com/vernal96/go-cms-kernel/filesystem"
	"github.com/vernal96/go-cms-kernel/migrations"
	"github.com/vernal96/go-cms-kernel/modules/admin"
	"github.com/vernal96/go-cms-kernel/modules/core"
	corepostgres "github.com/vernal96/go-cms-kernel/modules/core/adapters/postgres"
	"github.com/vernal96/go-cms-kernel/modules/core/user/adapters/argon2id"
	"github.com/vernal96/go-cms-kernel/security/jwt"
	"github.com/vernal96/go-cms-kernel/seeds"
	"github.com/vernal96/go-cms-kernel/transport/httpserver"
	"github.com/vernal96/go-cms/internal/platform"
)

//go:embed seeds/*.sql
var seedFiles embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		// The project logger writes to a file; startup failures must also reach
		// the container log, including errors already reported by the kernel.
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (resultErr error) {
	devSeed, err := strconv.ParseBool(env("CMS_DEV_SEED", "false"))
	if err != nil {
		return fmt.Errorf("CMS_DEV_SEED: %w", err)
	}
	var projectSeeds []appkernel.ModuleSeedSource
	if devSeed {
		projectSeeds = []appkernel.ModuleSeedSource{{Module: core.ModuleCode, Source: seeds.Source{ID: "starter", Schema: "core", Tags: []seeds.Tag{"dev"}, FS: seedFiles, Path: "seeds"}}}
	}
	application, err := appkernel.New(ctx, appkernel.Definition{
		Logger:   platform.LoggerFactory{Path: env("LOGGER_FILE_PATH", "var/log/cms.log")},
		EventBus: kafkaeventbus.Factory{Config: kafkaeventbus.Config{Brokers: strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","), ClientID: "go-cms", DialTimeout: 5 * time.Second, ConsumerRetryDelay: time.Second, ShutdownTimeout: 5 * time.Second}}, PasswordHasher: argon2id.Factory{},
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: connectorpostgres.Factory{Config: connectorpostgres.Config{Code: "main", Host: env("POSTGRES_HOST", "localhost"), Port: envInt("POSTGRES_PORT", 5432), Database: env("POSTGRES_DB", "cms"), User: env("POSTGRES_USER", "cms"), Password: os.Getenv("POSTGRES_PASSWORD"), SSLMode: env("POSTGRES_SSL_MODE", "disable"), MaxConns: 10, ConnMaxLifetime: time.Hour, ConnectTimeout: 5 * time.Second}},
			Adapters:  []kernel.ModuleDatabaseFactory{corepostgres.DatabaseFactory{}},
			Seeds:     projectSeeds,
		},
		Filesystems: []filesystem.Factory{
			localstorage.Factory{Config: localstorage.Config{Code: "public", Visibility: filesystem.VisibilityPublic, Root: env("FILES_PUBLIC_ROOT", "var/files/public"), BaseURL: env("FILES_PUBLIC_BASE_URL", "http://localhost:8080")}},
			localstorage.Factory{Config: localstorage.Config{Code: "private", Visibility: filesystem.VisibilityPrivate, Root: env("FILES_PRIVATE_ROOT", "var/files/private"), BaseURL: env("FILES_PRIVATE_BASE_URL", "http://localhost:8080"), SigningKey: os.Getenv("FILES_PRIVATE_SIGNING_KEY")}},
		},
		Caches: []cache.Factory{connectorredis.Factory{Config: connectorredis.Config{
			Code: "shared", Addrs: []string{env("REDIS_ADDR", "localhost:6379")}, Password: os.Getenv("REDIS_PASSWORD"),
			DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second,
		}}},
		Profiles: []kernel.Profile{{Code: "starter", Name: "Starter", Modules: []kernel.ProfileModule{
			{Module: core.Module{}, Caches: []cache.Binding{{Alias: core.DurableCacheAlias, Code: "shared"}, {Alias: core.HotCacheAlias, Code: "shared"}}},
			{Module: admin.Module{}},
		}}},
		AvatarStorage: "private", MaxUploadSize: 100 << 20, UploadTimeout: 10 * time.Minute, AvatarMaxSize: 5 << 20,
	})
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, application.Close()) }()
	slog.SetDefault(application.Logger())
	if err = migrations.NewManager().UpAll(ctx, application.MigrationPlans()); err != nil {
		return err
	}
	if err = seeds.NewManager().UpAll(ctx, application.SeedPlans()); err != nil {
		return err
	}
	if err = application.Boot(ctx); err != nil {
		return err
	}
	accessTokens, err := jwt.New(jwt.Config{SigningKey: os.Getenv("JWT_SIGNING_KEY"), Issuer: env("JWT_ISSUER", "go-cms"), Audience: env("JWT_AUDIENCE", "go-cms-api"), AccessTTL: envDuration("JWT_ACCESS_TTL", 30*time.Minute), ClockSkew: envDuration("JWT_CLOCK_SKEW", 30*time.Second)}, jwt.WithSessions(application.Services().Sessions))
	if err != nil {
		return err
	}
	if len(os.Args) > 1 {
		if len(os.Args) != 2 || os.Args[1] != "bootstrap-admin" {
			return errors.New("usage: server [bootstrap-admin]")
		}
		return bootstrapAdmin(ctx, application)
	}

	handler, err := httpserver.NewHandler(application, httpserver.WithAccessTokens(accessTokens))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/", handler)
	server, err := httpserver.NewServer(httpserver.Config{Address: net.JoinHostPort(env("SERVER_HOST", "0.0.0.0"), strconv.Itoa(envInt("SERVER_PORT", 8080))), ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, ShutdownTimeout: 5 * time.Second}, mux, application.Logger())
	if err != nil {
		return err
	}
	return server.Run(ctx)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}
func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(env(key, fallback.String()))
	if err != nil {
		return fallback
	}
	return value
}
