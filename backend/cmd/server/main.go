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

	appkernel "github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/migrations"
	"github.com/vernal96/go-cms-kernel/security/jwt"
	"github.com/vernal96/go-cms-kernel/seeds"
	"github.com/vernal96/go-cms-kernel/transport/httpserver"
	"github.com/vernal96/go-cms/internal/infrastructure"
	"github.com/vernal96/go-cms/internal/profile"
	"github.com/vernal96/go-cms/internal/settings"
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
	infra := infrastructure.Config{
		KafkaBrokers:        strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","),
		PostgresHost:        env("POSTGRES_HOST", "localhost"),
		PostgresPort:        envInt("POSTGRES_PORT", 5432),
		PostgresDatabase:    env("POSTGRES_DB", "cms"),
		PostgresUser:        env("POSTGRES_USER", "cms"),
		PostgresPassword:    os.Getenv("POSTGRES_PASSWORD"),
		PostgresSSLMode:     env("POSTGRES_SSL_MODE", "disable"),
		PublicFilesRoot:     env("FILES_PUBLIC_ROOT", "var/files/public"),
		PublicFilesBaseURL:  env("FILES_PUBLIC_BASE_URL", "http://localhost:8080"),
		PrivateFilesRoot:    env("FILES_PRIVATE_ROOT", "var/files/private"),
		PrivateFilesBaseURL: env("FILES_PRIVATE_BASE_URL", "http://localhost:8080"),
		PrivateFilesSignKey: os.Getenv("FILES_PRIVATE_SIGNING_KEY"),
		RedisAddress:        env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       os.Getenv("REDIS_PASSWORD"),
	}
	definition := settings.Config{
		LoggerPath:     env("LOGGER_FILE_PATH", "var/log/cms.log"),
		Infrastructure: infra.Definition(),
		Profile:        profile.Starter,
		DevSeed:        devSeed,
		SeedFiles:      seedFiles,
	}.Definition()
	application, err := appkernel.New(ctx, definition)
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
